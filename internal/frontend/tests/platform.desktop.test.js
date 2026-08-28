import { describe, it, expect, beforeAll, beforeEach } from 'vitest';

// Drives the real desktop implementation against a stand-in for the bindings
// the desktop build generates. Without this only droppedFile ran, so nothing
// would notice saveFile resolving silently, exportEmod losing its error check,
// or parseEmod sending an envelope shape the Go service does not accept.
let desktop;
let stub;
let runtime;
beforeAll(async () => {
  stub = await import('./bindings-stub.js');
  runtime = await import('./wails-runtime-stub.js');
  desktop = await import('../desktop/platform.desktop.js');
});

beforeEach(() => {
  // The module is imported once and holds the window's name and its marker
  // between tests, so both go back to what a window that has never been named
  // or marked holds before the calls each test reads are cleared.
  desktop.setWindowTitle('');
  desktop.setWindowModified(false);
  stub.calls.length = 0;
  runtime.calls.length = 0;
  stub.answers.ParseEmod = '{"diagnostics":[],"diagram":{"nodes":[],"edges":[]}}';
  stub.answers.ExportEmod = '{"emod":"emod 1\\nmodel \\"Billing\\"\\n"}';
  stub.answers.Read = '{"name":"billing.emod","path":"/models/billing.emod","content":"emod 1\\n"}';
  stub.answers.Write = '{"name":"billing.emod","path":"/models/billing.emod"}';
  runtime.answers.OpenFile = '';
  runtime.answers.SaveFile = '';
  desktop.onFileOpened(null);
  desktop.onSaveRequested(null);
});

// The shell's menu is the only thing that asks for a file, and it asks by
// emitting this event — so a test that called an exported picker directly would
// not exercise the path the app takes.
function requestOpen() {
  return Promise.resolve(runtime.listeners['file:open-requested']()).then(flush);
}

function requestSave() {
  return Promise.resolve(runtime.listeners['file:save-requested']()).then(flush);
}

function requestSaveAs() {
  return Promise.resolve(runtime.listeners['file:save-as-requested']()).then(flush);
}

function flush() {
  return new Promise((resolve) => setTimeout(resolve, 0));
}

describe('readiness', () => {
  it('is ready immediately, because the core is linked in rather than fetched', async () => {
    expect(desktop.isReady).toBe(true);
    await expect(desktop.ready).resolves.toBeUndefined();
  });
});

describe('parse', () => {
  it('sends the source envelope the Go service expects', async () => {
    await desktop.parseEmod('emod 1\nmodel "Billing"\n');

    expect(stub.calls).toEqual([['ParseEmod', JSON.stringify({ source: 'emod 1\nmodel "Billing"\n' })]]);
  });

  it('answers the decoded document rather than the raw string', async () => {
    stub.answers.ParseEmod = '{"diagnostics":[{"message":"x"}],"diagram":{"nodes":[1],"edges":[]}}';

    await expect(desktop.parseEmod('src')).resolves.toEqual({
      diagnostics: [{ message: 'x' }],
      diagram: { nodes: [1], edges: [] },
    });
  });
});

describe('export', () => {
  it('sends the diagram document itself, not a source envelope', async () => {
    await desktop.exportEmod({ model_name: 'Billing', nodes: [], edges: [] });

    expect(stub.calls[0][0]).toBe('ExportEmod');
    expect(JSON.parse(stub.calls[0][1])).toEqual({ model_name: 'Billing', nodes: [], edges: [] });
  });

  it('unwraps the emod text', async () => {
    await expect(desktop.exportEmod({})).resolves.toBe('emod 1\nmodel "Billing"\n');
  });

  it('raises the error envelope rather than resolving with it', async () => {
    stub.answers.ExportEmod = '{"error":"importer: invalid diagram JSON"}';

    await expect(desktop.exportEmod({})).rejects.toThrow('importer: invalid diagram JSON');
  });
});

describe('window title', () => {
  it('names the native window, which is the only title a desktop user can see', () => {
    desktop.setWindowTitle('hotel.emod — Emod Diagram Viewer');

    expect(runtime.calls).toEqual([['Window.SetTitle', 'hotel.emod — Emod Diagram Viewer']]);
  });
});

// The window carries one title and two facts reach it separately, so the name
// and the marker are held here and the title is composed from both.
describe('the unsaved-edits marker', () => {
  const titles = () => runtime.calls
    .filter((call) => call[0] === 'Window.SetTitle')
    .map((call) => call[1]);

  it('puts a star ahead of the name while there are unsaved edits, and takes it away again', () => {
    desktop.setWindowTitle('hotel.emod — Emod Diagram Viewer');

    desktop.setWindowModified(true);
    desktop.setWindowModified(false);

    expect(titles()).toEqual([
      'hotel.emod — Emod Diagram Viewer',
      '* hotel.emod — Emod Diagram Viewer',
      'hotel.emod — Emod Diagram Viewer',
    ]);
  });

  it('keeps the star when the window is renamed while it is marked', () => {
    desktop.setWindowTitle('hotel.emod — Emod Diagram Viewer');
    desktop.setWindowModified(true);

    desktop.setWindowTitle('billing.emod — Emod Diagram Viewer');

    expect(titles().pop()).toBe('* billing.emod — Emod Diagram Viewer');
  });

  it('still marks a window whose name has never been set', () => {
    desktop.setWindowModified(true);

    expect(titles()).toEqual(['*']);
  });

  // The viewer states the answer on every keystroke, so a window renamed once
  // per character is what leaving this out costs.
  it('leaves the window alone for an answer that has not moved', () => {
    desktop.setWindowTitle('hotel.emod — Emod Diagram Viewer');
    desktop.setWindowModified(true);
    const settled = titles().length;

    desktop.setWindowModified(true);
    desktop.setWindowModified(true);

    expect(titles().length).toBe(settled);
  });
});

describe('opening a file', () => {
  function collectDeliveries() {
    const delivered = [];
    desktop.onFileOpened((opened) => delivered.push(opened));
    return delivered;
  }

  it('shows the picker filtered to the two extensions a model comes in', async () => {
    await requestOpen();

    expect(runtime.calls).toEqual([['Dialogs.OpenFile', {
      Title: 'Open model',
      CanChooseFiles: true,
      CanChooseDirectories: false,
      Filters: [{ DisplayName: 'emod models', Pattern: '*.emod;*.json' }],
    }]]);
  });

  it('reads what was chosen and delivers its name, path and contents', async () => {
    const delivered = collectDeliveries();
    runtime.answers.OpenFile = '/models/billing.emod';
    stub.answers.Read = '{"name":"billing.emod","path":"/models/billing.emod","content":"emod 1\\nmodel \\"Billing\\"\\n"}';

    await requestOpen();

    expect(stub.calls).toEqual([['Read', '/models/billing.emod']]);
    expect(delivered).toEqual([{
      name: 'billing.emod',
      path: '/models/billing.emod',
      content: 'emod 1\nmodel "Billing"\n',
    }]);
  });

  it('delivers nothing at all when the picker is cancelled', async () => {
    const delivered = collectDeliveries();
    runtime.answers.OpenFile = '';

    await requestOpen();

    expect(delivered).toEqual([]);
    expect(stub.calls).toEqual([]);
  });

  it('delivers the reason when the service could not read the chosen file', async () => {
    const delivered = collectDeliveries();
    runtime.answers.OpenFile = '/models/gone.emod';
    stub.answers.Read = '{"error":"reading /models/gone.emod: no such file or directory"}';

    await requestOpen();

    expect(delivered).toEqual([{ error: 'reading /models/gone.emod: no such file or directory' }]);
  });

  it('delivers the reason when the picker itself fails', async () => {
    const delivered = collectDeliveries();
    runtime.answers.OpenFile = new Error('Dialog.OpenFile failed, error getting selection');

    await requestOpen();

    expect(delivered).toEqual([{ error: 'Dialog.OpenFile failed, error getting selection' }]);
  });

  it('delivers only what the newest request chose, whichever read resolves last', async () => {
    const delivered = collectDeliveries();
    const answers = {
      '/models/first.emod': '{"name":"first.emod","path":"/models/first.emod","content":"first"}',
      '/models/second.emod': '{"name":"second.emod","path":"/models/second.emod","content":"second"}',
    };
    const pending = [];
    stub.answers.Read = null;
    const realRead = stub.FileService.Read;
    stub.FileService.Read = (path) => new Promise((resolve) => pending.push(() => resolve(answers[path])));

    runtime.answers.OpenFile = '/models/first.emod';
    const first = runtime.listeners['file:open-requested']();
    await flush();
    runtime.answers.OpenFile = '/models/second.emod';
    const second = runtime.listeners['file:open-requested']();
    await flush();

    // The first read finishes last, which is the ordering the counter exists for.
    pending[1]();
    pending[0]();
    await Promise.all([first, second]);
    await flush();
    stub.FileService.Read = realRead;

    expect(delivered.map((d) => d.name)).toEqual(['second.emod']);
  });

  it('never re-enters the viewer with the viewer\'s own error dressed up as a file failure', async () => {
    const thrown = [];
    desktop.onFileOpened((opened) => {
      thrown.push(opened);
      throw new Error('a bug inside the viewer');
    });
    runtime.answers.OpenFile = '/models/billing.emod';

    await expect(requestOpen()).rejects.toThrow('a bug inside the viewer');

    expect(thrown).toHaveLength(1);
    expect(thrown[0].error).toBeUndefined();
  });

  it('discards a request that arrives before the viewer has registered, rather than throwing', async () => {
    runtime.answers.OpenFile = '/models/billing.emod';

    await expect(requestOpen()).resolves.toBeUndefined();
  });
});

describe('saving a file', () => {
  it('writes straight to the path it was given, showing no dialog', async () => {
    const saved = await desktop.saveFile('billing.emod', 'emod 1\n', '/models/billing.emod');

    expect(stub.calls).toEqual([['Write', '/models/billing.emod', 'emod 1\n']]);
    expect(runtime.calls).toEqual([]);
    expect(saved).toEqual({ name: 'billing.emod', path: '/models/billing.emod' });
  });

  it('answers the path the service resolved, not the one it was handed', async () => {
    stub.answers.Write = '{"name":"billing.emod","path":"/models/billing.emod"}';

    const saved = await desktop.saveFile('billing.emod', 'emod 1\n', 'billing.emod');

    expect(saved.path).toBe('/models/billing.emod');
  });

  it('shows the save dialog when it is given no path, offering the suggested name', async () => {
    runtime.answers.SaveFile = '/models/hotel.emod';
    stub.answers.Write = '{"name":"hotel.emod","path":"/models/hotel.emod"}';

    await desktop.saveFile('hotel.emod', 'emod 1\n', '');

    expect(runtime.calls).toEqual([['Dialogs.SaveFile', {
      Title: 'Save model',
      Filename: 'hotel.emod',
      CanCreateDirectories: true,
      Filters: [{ DisplayName: 'emod models', Pattern: '*.emod;*.json' }],
    }]]);
  });

  it('writes to what the dialog chose, and answers that file', async () => {
    runtime.answers.SaveFile = '/models/hotel.emod';
    stub.answers.Write = '{"name":"hotel.emod","path":"/models/hotel.emod"}';

    const saved = await desktop.saveFile('hotel.emod', 'emod 1\n', '');

    expect(stub.calls).toEqual([['Write', '/models/hotel.emod', 'emod 1\n']]);
    expect(saved).toEqual({ name: 'hotel.emod', path: '/models/hotel.emod' });
  });

  it('writes nothing and answers no file when the dialog is cancelled', async () => {
    runtime.answers.SaveFile = '';

    const saved = await desktop.saveFile('hotel.emod', 'emod 1\n', '');

    expect(stub.calls).toEqual([]);
    expect(saved).toBeNull();
  });

  it('raises the reason the service gave, rather than a wording of its own', async () => {
    stub.answers.Write = '{"error":"writing /models/billing.emod: permission denied"}';

    await expect(desktop.saveFile('billing.emod', 'emod 1\n', '/models/billing.emod'))
      .rejects.toThrow('writing /models/billing.emod: permission denied');
  });

  it('raises the reason the dialog itself failed with', async () => {
    runtime.answers.SaveFile = new Error('Dialog.SaveFile failed, error getting selection');

    await expect(desktop.saveFile('hotel.emod', 'emod 1\n', ''))
      .rejects.toThrow('Dialog.SaveFile failed, error getting selection');
  });
});

describe('the shell asking for a save', () => {
  function collectRequests() {
    const asked = [];
    desktop.onSaveRequested((options) => { asked.push(options); });
    return asked;
  }

  it('passes on a Save as a request to reuse the open file', async () => {
    const asked = collectRequests();

    await requestSave();

    expect(asked).toEqual([{ chooseLocation: false }]);
  });

  it('passes on a Save As as a request to choose a location', async () => {
    const asked = collectRequests();

    await requestSaveAs();

    expect(asked).toEqual([{ chooseLocation: true }]);
  });

  // The listener's own return value, not the helper's tail: requestSave ends in
  // .then(flush), which resolves undefined whatever the listener did.
  it('discards a request that arrives before the viewer has registered, rather than throwing', async () => {
    expect(runtime.listeners['file:save-requested']()).toBeUndefined();
    expect(runtime.listeners['file:save-as-requested']()).toBeUndefined();
  });

  it('hands back what the viewer\'s save produced rather than swallowing it', async () => {
    desktop.onSaveRequested(() => Promise.reject(new Error('a bug inside the viewer')));

    await expect(Promise.resolve(runtime.listeners['file:save-requested']()))
      .rejects.toThrow('a bug inside the viewer');
  });
});

describe('starting up', () => {
  it('opens with no model, because nothing hands this window one at startup', async () => {
    await expect(desktop.initialState()).resolves.toBeNull();
  });
});

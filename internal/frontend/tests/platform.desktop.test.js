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
  stub.calls.length = 0;
  runtime.calls.length = 0;
  stub.answers.ParseEmod = '{"diagnostics":[],"diagram":{"nodes":[],"edges":[]}}';
  stub.answers.ExportEmod = '{"emod":"emod 1\\nmodel \\"Billing\\"\\n"}';
  stub.answers.Read = '{"name":"billing.emod","path":"/models/billing.emod","content":"emod 1\\n"}';
  runtime.answers.OpenFile = '';
  desktop.onFileOpened(null);
});

// The shell's menu is the only thing that asks for a file, and it asks by
// emitting this event — so a test that called an exported picker directly would
// not exercise the path the app takes.
function requestOpen() {
  return Promise.resolve(runtime.listeners['file:open-requested']()).then(flush);
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

describe('the operations this build does not have', () => {
  it('refuses to save, audibly, rather than resolving as though it had', async () => {
    await expect(desktop.saveFile('Billing.emod', 'emod 1\n')).rejects
      .toThrow('Saving is not available in this build yet');
  });

  it('opens with no model, because nothing hands this window one at startup', async () => {
    await expect(desktop.initialState()).resolves.toBeNull();
  });
});

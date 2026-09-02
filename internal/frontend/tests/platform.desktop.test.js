import { describe, it, expect, vi, beforeAll, beforeEach } from 'vitest';

// Drives the real desktop implementation against a stand-in for the bindings
// the desktop build generates. Without this only droppedFiles ran, so nothing
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

beforeEach(async () => {
  // The module is imported once and holds the window's name and its record of
  // the shell between tests, so both go back to what a freshly loaded page
  // holds — after the answer bag is clean, or the reset is answered by whatever
  // the previous test left in it, and where the record starts then depends on
  // the order the tests ran.
  stub.answers.SetModified = undefined;
  stub.answers.Record = undefined;
  stub.answers.Open = '{"name":"billing.emod","path":"/models/billing.emod","content":"emod 1\\n"}';
  desktop.setWindowTitle('');
  await desktop.setWindowModified(false);
  stub.calls.length = 0;
  stub.landed.length = 0;
  stub.recorded.length = 0;
  runtime.calls.length = 0;
  stub.answers.ParseEmod = '{"diagnostics":[],"diagram":{"nodes":[],"edges":[]}}';
  stub.answers.ExportEmod = '{"emod":"emod 1\\nmodel \\"Billing\\"\\n"}';
  stub.answers.Read = '{"name":"billing.emod","path":"/models/billing.emod","content":"emod 1\\n"}';
  stub.answers.Write = '{"name":"billing.emod","path":"/models/billing.emod"}';
  runtime.answers.OpenFile = '';
  runtime.answers.SaveFile = '';
  runtime.answers.Question = '';
  runtime.answers.IsMac = false;
  desktop.onFileOpened(null);
  desktop.onFilesDropped(null);
  desktop.onSaveRequested(null);
  desktop.onLeaveRequested(null);
});

// The shell's menu is the only thing that asks for a file, and it asks by
// emitting this event — so a test that called an exported picker directly would
// not exercise the path the app takes.
function requestOpen() {
  return Promise.resolve(runtime.listeners['file:open-requested']()).then(flush);
}

// A recent entry is chosen from the shell's native menu, which asks for it by
// emitting this event with the path — so a test fires the subscription rather
// than calling anything exported.
function requestRecent(path) {
  return Promise.resolve(runtime.listeners['file:open-recent-requested']({ data: path })).then(flush);
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

function collectDeliveries() {
  const delivered = [];
  desktop.onFileOpened((opened) => delivered.push(opened));
  return delivered;
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
// and the marker are held here and the title is composed from both — on the
// platforms whose only convention for unsaved work is the title.
describe('the unsaved-edits marker', () => {
  const titles = () => runtime.calls
    .filter((call) => call[0] === 'Window.SetTitle')
    .map((call) => call[1]);

  const told = () => stub.calls
    .filter((call) => call[0] === 'SetModified')
    .map((call) => call[1]);

  it('tells the shell every answer, so it knows whether closing would discard work', async () => {
    desktop.setWindowModified(true);
    await desktop.setWindowModified(false);

    expect(told()).toEqual([true, false]);
  });

  it('asks the shell nothing for an answer that has not moved', async () => {
    await desktop.setWindowModified(true);

    await desktop.setWindowModified(true);
    await desktop.setWindowModified(true);

    expect(told()).toEqual([true]);
  });

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

  // A window this module has never named still carries whatever the shell
  // called it, and a lone marker in place of that is the only name destroyed.
  it('leaves the shell\'s own title alone until the window has a name', async () => {
    await desktop.setWindowModified(true);

    expect(titles()).toEqual([]);
    expect(told()).toEqual([true]);
  });

  // Two calls in flight at once are served a goroutine each and land in
  // whichever order they finish, and the loser is permanent: the check that
  // skips an unchanged answer then reads the shell as already holding what it
  // does not, so every later repair is swallowed and the close guard goes quiet.
  it('never lets an older answer land on the shell after a newer one', async () => {
    let releaseFirst;
    stub.answers.SetModified = new Promise((resolve) => { releaseFirst = resolve; });

    const first = desktop.setWindowModified(true);
    // The dispatch is a turn away, so the gate has to still be in the bag when
    // it happens or the call it was meant to hold open sails straight through.
    await flush();
    stub.answers.SetModified = undefined;
    const second = desktop.setWindowModified(false);
    // A second call the queue does not hold back lands right here, ahead of the
    // first; one it does hold back cannot be sent until the first has landed.
    await flush();
    releaseFirst();
    await Promise.all([first, second]);

    // Sent in this order either way; the question is which one the shell is
    // left holding, and that is the order they arrive in.
    expect(stub.landed).toEqual([true, false]);
  });

  // A fresh page has told the shell nothing, and the shell may still be holding
  // what the page before it said — so the first answer always goes.
  it('sends the first answer a freshly loaded page gives, unchanged though it looks', async () => {
    // A reload is a fresh module registry, so the bindings it reaches are fresh
    // too and have to be imported alongside it to be observed. Sequentially:
    // two dynamic imports in flight at once resolve against different copies.
    vi.resetModules();
    const reloadedStub = await import('./bindings-stub.js');
    const reloadedPage = await import('../desktop/platform.desktop.js');

    await reloadedPage.setWindowModified(false);

    expect(reloadedStub.calls.filter((call) => call[0] === 'SetModified'))
      .toEqual([['SetModified', false]]);
  });

  // Three audit rounds each found a way for this record and the shell's to
  // diverge, so the invariant is asserted over sequences rather than over the
  // one interleaving a hand-written case happens to pick: whatever the page
  // says and however long each call takes, the last thing the shell is left
  // holding is the last thing the page said.
  it('leaves the shell holding the page\'s last answer, whatever order the calls take', async () => {
    // What the shell holds after the reset above, so a run whose answers are
    // all the value it already holds — which correctly sends nothing — still
    // has something to be judged against.
    stub.landed.length = 0;
    stub.landed.push(false);

    for (let run = 0; run < 40; run++) {
      const answers = [];
      for (let i = 0; i < 5; i++) {
        answers.push(Math.random() < 0.5);
      }

      const sent = answers.map((answer) => {
        // Each call is held open for its own length, so a later one can finish
        // its transport before an earlier one.
        stub.answers.SetModified = new Promise((resolve) => {
          setTimeout(resolve, Math.floor(Math.random() * 3));
        });

        return desktop.setWindowModified(answer);
      });
      await Promise.all(sent);

      expect(stub.landed[stub.landed.length - 1]).toBe(answers[answers.length - 1]);
    }
    stub.answers.SetModified = undefined;
  });

  // The shell reads its own copy to decide whether closing would discard work,
  // so a call it never heard must be retried — including when the answer has
  // not moved since, which is the case the de-duplication above would swallow.
  it('tells the shell the same answer again after a call it did not hear', async () => {
    stub.answers.SetModified = new Error('the shell did not answer');
    await desktop.setWindowModified(true);

    stub.answers.SetModified = undefined;
    await desktop.setWindowModified(true);

    expect(told()).toEqual([true, true]);
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

  // macOS marks an edited document with a dot in the close button, which the
  // shell puts there itself, so a star in the title would say it twice.
  describe('on macOS', () => {
    beforeEach(() => { runtime.answers.IsMac = true; });

    it('leaves the title to the name alone, and still tells the shell', async () => {
      desktop.setWindowTitle('hotel.emod — Emod Diagram Viewer');

      await desktop.setWindowModified(true);

      expect(titles()).toEqual(['hotel.emod — Emod Diagram Viewer']);
      expect(told()).toEqual([true]);
    });

    it('names a window renamed while it holds unsaved work without a star', () => {
      desktop.setWindowTitle('hotel.emod — Emod Diagram Viewer');
      desktop.setWindowModified(true);

      desktop.setWindowTitle('billing.emod — Emod Diagram Viewer');

      expect(titles().pop()).toBe('billing.emod — Emod Diagram Viewer');
    });
  });
});

describe('opening a file', () => {
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

    // The listener's own answer, not the helper's: a helper ending in flush
    // resolves undefined whatever the listener returned.
    await expect(Promise.resolve(runtime.listeners['file:open-requested']())).resolves.toBeUndefined();
    await flush();
  });
});

// A drop is resolved by the shell, not by the page: the platform's own drag
// destination consumes the drag, and what comes back is the real location of
// each file — which is the whole reason a dropped model can be saved back to
// where it came from. The shell emits them, so a test fires the subscription
// the shell fires rather than calling anything exported.
describe('files dropped on the window', () => {
  function collectDrops() {
    const dropped = [];
    desktop.onFilesDropped((files) => { dropped.push(files); return Promise.resolve(); });
    return dropped;
  }

  function dropPaths(...paths) {
    return Promise.resolve(
      runtime.listeners['file:dropped']({ name: 'file:dropped', data: paths }),
    ).then(flush);
  }

  it('names each dropped file by the last part of its path, in the order they arrived', async () => {
    const dropped = collectDrops();

    await dropPaths('/models/hotel.emod', '/notes/shopping.txt', 'C:\\models\\diagram.json');

    expect(dropped).toHaveLength(1);
    expect(dropped[0].map((file) => file.name))
      .toEqual(['hotel.emod', 'shopping.txt', 'diagram.json']);
  });

  // Naming a file and reading it are separate so the viewer can refuse a drop by
  // name without the shell going to disk for a file it will not open.
  it('reads nothing until a dropped file is asked for its contents', async () => {
    collectDrops();

    await dropPaths('/models/hotel.emod');

    expect(stub.calls).toEqual([]);
  });

  it('answers the name, path and contents the service read back', async () => {
    const dropped = collectDrops();
    stub.answers.Read = '{"name":"hotel.emod","path":"/models/hotel.emod","content":"emod 1\\nmodel \\"Hotel\\"\\n"}';

    await dropPaths('/models/hotel.emod');

    await expect(dropped[0][0].read()).resolves.toEqual({
      name: 'hotel.emod',
      path: '/models/hotel.emod',
      content: 'emod 1\nmodel "Hotel"\n',
    });
    expect(stub.calls).toEqual([['Read', '/models/hotel.emod']]);
  });

  it('answers the reason the service gave for a file it could not read', async () => {
    const dropped = collectDrops();
    stub.answers.Read = '{"error":"reading /models/hotel.emod: permission denied"}';

    await dropPaths('/models/hotel.emod');

    await expect(dropped[0][0].read())
      .resolves.toEqual({ error: 'reading /models/hotel.emod: permission denied' });
  });

  // The viewer opens whatever a read answers without catching, because a reason
  // is one of the answers — so a binding that fails has to arrive as one too.
  it('answers a reason rather than rejecting when the read itself fails', async () => {
    const dropped = collectDrops();
    const realRead = stub.FileService.Read;
    stub.FileService.Read = () => Promise.reject(new Error('binding call failed'));

    try {
      await dropPaths('/models/hotel.emod');

      await expect(dropped[0][0].read()).resolves.toEqual({ error: 'binding call failed' });
    } finally {
      stub.FileService.Read = realRead;
    }
  });

  it('delivers nothing when the drop resolved to no path at all', async () => {
    const dropped = collectDrops();

    await dropPaths();

    expect(dropped).toEqual([]);
  });

  // The shell answers nil when it resolved no file, which reaches the page as
  // null rather than as an empty list.
  it('delivers nothing when the shell answers no paths at all', async () => {
    const dropped = collectDrops();

    await Promise.resolve(
      runtime.listeners['file:dropped']({ name: 'file:dropped', data: null }),
    ).then(flush);

    expect(dropped).toEqual([]);
  });

  // An Open resolves a picker and a read before it can deliver; a drop delivers the
  // moment the shell names the paths. Numbered apart, the Open requested first lands
  // on top of the drop made after it — the model on screen, and the path a following
  // Save writes to, decided by which read finished rather than what the user did last.
  it('drops an Open whose read outlives a drop made after it', async () => {
    const opened = [];
    const dropped = [];
    desktop.onFileOpened((file) => opened.push(file));
    desktop.onFilesDropped((files) => { dropped.push(files); return Promise.resolve(); });

    let releaseRead;
    const realRead = stub.FileService.Read;
    stub.FileService.Read = () => new Promise((resolve) => {
      releaseRead = () => resolve('{"name":"chosen.emod","path":"/models/chosen.emod","content":"emod 1\\n"}');
    });
    runtime.answers.OpenFile = '/models/chosen.emod';

    try {
      const open = runtime.listeners['file:open-requested']();
      await flush();
      await dropPaths('/models/hotel.emod');
      releaseRead();
      await open;
      await flush();
    } finally {
      stub.FileService.Read = realRead;
    }

    expect(dropped).toHaveLength(1);
    expect(opened).toEqual([]);
  });

  // The listener's own answer, not the helper's: dropPaths ends in `.then(flush)`, which
  // resolves undefined whatever the listener returned, so asserting on it proves nothing.
  it('discards a drop that arrives before the viewer has registered, rather than throwing', () => {
    expect(runtime.listeners['file:dropped'](
      { name: 'file:dropped', data: ['/models/hotel.emod'] },
    )).toBeUndefined();
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

// Anything but the three the viewer knows has to answer cancel: an outcome the
// user did not choose either writes their file or throws their edits away, and
// both are past undoing from inside the app.
describe('asking what to do about unsaved edits', () => {
  const question = () => runtime.calls.find((call) => call[0] === 'Dialogs.Question')[1];

  // Which button the Return key presses is the whole of what a default means,
  // so a default on Discard would throw the edits away on a keystroke meant to
  // keep them.
  it('offers the three choices, with save the default and cancel the way out', async () => {
    await desktop.resolveUnsavedEdits();

    expect(question().Buttons).toEqual([
      { Label: 'Save', IsDefault: true },
      { Label: 'Discard' },
      { Label: 'Cancel', IsCancel: true },
    ]);
  });

  it('answers save when Save was pressed', async () => {
    runtime.answers.Question = 'Save';

    await expect(desktop.resolveUnsavedEdits()).resolves.toBe('save');
  });

  it('answers discard when Discard was pressed', async () => {
    runtime.answers.Question = 'Discard';

    await expect(desktop.resolveUnsavedEdits()).resolves.toBe('discard');
  });

  it('answers cancel when Cancel was pressed', async () => {
    runtime.answers.Question = 'Cancel';

    await expect(desktop.resolveUnsavedEdits()).resolves.toBe('cancel');
  });

  it('answers cancel for a dialog dismissed without a choice', async () => {
    runtime.answers.Question = '';

    await expect(desktop.resolveUnsavedEdits()).resolves.toBe('cancel');
  });

  it('answers cancel for a label it does not recognise', async () => {
    runtime.answers.Question = "Don't Save";

    await expect(desktop.resolveUnsavedEdits()).resolves.toBe('cancel');
  });

  it('answers cancel when the dialog itself fails', async () => {
    runtime.answers.Question = new Error('no window to attach to');

    await expect(desktop.resolveUnsavedEdits()).resolves.toBe('cancel');
  });
});

// The shell refuses a close or a quit while it holds unsaved work and asks
// here instead. What to do about the edits is the viewer's decision, so this
// module only carries the request over and acts on the answer.
describe('the shell asking to close or quit', () => {
  const leaving = () => runtime.calls
    .filter((call) => call[0] === 'Window.Close' || call[0] === 'Application.Quit')
    .map((call) => call[0]);

  const cleared = () => stub.calls.filter((call) => call[0] === 'SetModified' && call[1] === false);

  function requestClose() {
    return Promise.resolve(runtime.listeners['window:close-requested']()).then(flush);
  }

  function requestQuit() {
    return Promise.resolve(runtime.listeners['app:quit-requested']()).then(flush);
  }

  function viewerAnswers(answer) {
    const asked = [];
    desktop.onLeaveRequested(() => { asked.push(true); return Promise.resolve(answer); });
    return asked;
  }

  beforeEach(async () => {
    await desktop.setWindowModified(true);
    stub.calls.length = 0;
    runtime.calls.length = 0;
  });

  it('asks the viewer rather than deciding what to do about the edits itself', async () => {
    const asked = viewerAnswers(false);

    await requestClose();

    expect(asked).toEqual([true]);
    expect(runtime.calls.filter((call) => call[0] === 'Dialogs.Question')).toEqual([]);
  });

  it('leaves the window open and tells the shell nothing when the viewer says no', async () => {
    viewerAnswers(false);

    await requestClose();

    expect(leaving()).toEqual([]);
    expect(stub.calls).toEqual([]);
  });

  it('closes when the viewer says it may', async () => {
    viewerAnswers(true);

    await requestClose();

    expect(leaving()).toEqual(['Window.Close']);
    expect(cleared()).toHaveLength(1);
  });

  it('quits when the viewer says it may', async () => {
    viewerAnswers(true);

    await requestQuit();

    expect(leaving()).toEqual(['Application.Quit']);
    expect(cleared()).toHaveLength(1);
  });

  // The shell refused on the state it holds, so a close asked for before that
  // state has actually reached it is refused again and the window never goes.
  // Asking in the right order is not enough — the answer has to have landed.
  it('waits for the shell to have heard there is nothing unsaved before asking it to close', async () => {
    viewerAnswers(true);
    let hearIt;
    stub.answers.SetModified = new Promise((resolve) => { hearIt = resolve; });

    const closing = requestClose();
    await flush();
    expect(cleared()).toHaveLength(1);
    expect(leaving()).toEqual([]);

    hearIt();
    await closing;

    expect(leaving()).toEqual(['Window.Close']);
  });

  // A shell that never heard it still refuses, so asking would achieve nothing.
  it('leaves the window open when the shell could not be told', async () => {
    viewerAnswers(true);
    stub.answers.SetModified = new Error('the shell did not answer');

    await requestClose();

    expect(leaving()).toEqual([]);
  });

  // The window still holds the work, so it must still say so — clearing the
  // marker for a leave that never happened leaves the title lying about it.
  it('keeps saying there is unsaved work when the leave it cleared for did not happen', async () => {
    desktop.setWindowTitle('hotel.emod — Emod Diagram Viewer');
    await desktop.setWindowModified(true);
    runtime.calls.length = 0;
    viewerAnswers(true);
    stub.answers.SetModified = new Error('the shell did not answer');

    await requestClose();

    expect(runtime.calls.filter((call) => call[0] === 'Window.SetTitle')).toEqual([]);
  });

  // This module's record of what the shell holds is the page's and is reloaded
  // with it; the shell's is the process's and is not. After a reload the two
  // disagree, and a leave that trusts the record never tells the shell at all —
  // so the veto fires, is answered, and fires again with nothing able to move.
  it('tells the shell even when its own record says it already knows', async () => {
    // What a page holds once it has told the shell there is nothing unsaved —
    // which after a reload is not what the shell itself is holding.
    await desktop.setWindowModified(false);
    stub.calls.length = 0;
    viewerAnswers(true);

    await requestClose();

    expect(cleared()).toHaveLength(1);
    expect(leaving()).toEqual(['Window.Close']);
  });

  // Nobody can answer, and going anyway would discard the work the shell
  // refused to leave over.
  it('leaves the window open when no viewer has registered to answer', async () => {
    desktop.onLeaveRequested(null);

    await requestClose();

    expect(leaving()).toEqual([]);
  });
});

// An entry chosen from File ▸ Open Recent has to open exactly as one chosen in
// the dialog does — the same document, the same delivery, the same gesture
// numbering — with the read going through the list's own service, which is the
// only side that knows to forget a file that has gone.
describe('opening a recent entry', () => {
  it('reads the entry through the list\'s service and delivers its name, path and contents', async () => {
    const delivered = collectDeliveries();
    stub.answers.Open = '{"name":"billing.emod","path":"/models/billing.emod","content":"emod 1\\nmodel \\"Billing\\"\\n"}';

    await requestRecent('/models/billing.emod');

    expect(stub.calls).toEqual([['Open', '/models/billing.emod']]);
    expect(runtime.calls).toEqual([]);
    expect(delivered).toEqual([{
      name: 'billing.emod',
      path: '/models/billing.emod',
      content: 'emod 1\nmodel "Billing"\n',
    }]);
  });

  it('delivers the reason the service gave for an entry whose file has gone', async () => {
    const delivered = collectDeliveries();
    stub.answers.Open = '{"error":"gone.emod is no longer at /models/gone.emod; it has been removed from the recent files"}';

    await requestRecent('/models/gone.emod');

    expect(delivered).toEqual([{ error: 'gone.emod is no longer at /models/gone.emod; it has been removed from the recent files' }]);
  });

  it('delivers the reason when the service call itself fails', async () => {
    const delivered = collectDeliveries();
    stub.answers.Open = new Error('the shell did not answer');

    await requestRecent('/models/billing.emod');

    expect(delivered).toEqual([{ error: 'the shell did not answer' }]);
  });

  it('drops an Open whose read outlives a recent entry chosen after it', async () => {
    const delivered = collectDeliveries();
    const pending = [];
    const realRead = stub.FileService.Read;
    stub.FileService.Read = () => new Promise((resolve) => {
      pending.push(() => resolve('{"name":"chosen.emod","path":"/models/chosen.emod","content":"chosen"}'));
    });
    stub.answers.Open = '{"name":"recent.emod","path":"/models/recent.emod","content":"recent"}';

    try {
      runtime.answers.OpenFile = '/models/chosen.emod';
      const open = runtime.listeners['file:open-requested']();
      await flush();
      await requestRecent('/models/recent.emod');

      // The dialog's read finishes last, which is the ordering the counter exists for.
      pending[0]();
      await open;
      await flush();
    } finally {
      stub.FileService.Read = realRead;
    }

    expect(delivered.map((d) => d.name)).toEqual(['recent.emod']);
  });

  it('drops a recent entry whose read outlives an Open made after it', async () => {
    const delivered = collectDeliveries();
    let releaseRecent;
    stub.answers.Open = new Promise((resolve) => { releaseRecent = resolve; });
    runtime.answers.OpenFile = '/models/chosen.emod';
    stub.answers.Read = '{"name":"chosen.emod","path":"/models/chosen.emod","content":"chosen"}';

    const recent = runtime.listeners['file:open-recent-requested']({ data: '/models/recent.emod' });
    await flush();
    await requestOpen();

    releaseRecent('{"name":"recent.emod","path":"/models/recent.emod","content":"recent"}');
    await recent;
    await flush();

    expect(delivered.map((d) => d.name)).toEqual(['chosen.emod']);
  });

  it('never re-enters the viewer with the viewer\'s own error dressed up as a file failure', async () => {
    const thrown = [];
    desktop.onFileOpened((opened) => {
      thrown.push(opened);
      throw new Error('a bug inside the viewer');
    });

    await expect(requestRecent('/models/billing.emod')).rejects.toThrow('a bug inside the viewer');

    expect(thrown).toHaveLength(1);
    expect(thrown[0].error).toBeUndefined();
  });

  it('discards a request that arrives before the viewer has registered, rather than throwing', async () => {
    // The listener's own answer, not the helper's: a helper ending in flush
    // resolves undefined whatever the listener returned.
    await expect(Promise.resolve(runtime.listeners['file:open-recent-requested']({ data: '/models/billing.emod' })))
      .resolves.toBeUndefined();
    await flush();
  });
});

// The shell keeps the list of what has been opened, and the page keeps no
// record of what it has told the shell — so every file the viewer says it
// opened is sent, in the order the viewer said so.
describe('remembering an opened file', () => {
  it('sends the file\'s path to the shell\'s list', async () => {
    await desktop.rememberOpenedFile('/models/billing.emod');

    expect(stub.calls).toEqual([['Record', '/models/billing.emod']]);
  });

  it('sends the same file again when told again, keeping no record of its own', async () => {
    await desktop.rememberOpenedFile('/models/billing.emod');
    await desktop.rememberOpenedFile('/models/billing.emod');

    expect(stub.recorded).toEqual(['/models/billing.emod', '/models/billing.emod']);
  });

  it('never lets a file remembered earlier land on the shell after one remembered later', async () => {
    let releaseFirst;
    stub.answers.Record = new Promise((resolve) => { releaseFirst = resolve; });

    const first = desktop.rememberOpenedFile('/models/first.emod');
    // The dispatch is a turn away, so the gate has to still be in the bag when
    // it happens or the call it was meant to hold open sails straight through.
    await flush();
    stub.answers.Record = undefined;
    const second = desktop.rememberOpenedFile('/models/second.emod');
    // A second call the queue does not hold back lands right here, ahead of the
    // first; one it does hold back cannot be sent until the first has landed.
    await flush();
    releaseFirst();
    await Promise.all([first, second]);

    expect(stub.recorded).toEqual(['/models/first.emod', '/models/second.emod']);
  });

  it('raises the shell\'s refusal to the caller, and still sends the file after it', async () => {
    stub.answers.Record = new Error('writing recent-files.json: permission denied');

    await expect(desktop.rememberOpenedFile('/models/first.emod'))
      .rejects.toThrow('writing recent-files.json: permission denied');

    stub.answers.Record = undefined;
    await desktop.rememberOpenedFile('/models/second.emod');

    expect(stub.recorded).toEqual(['/models/second.emod']);
  });
});

describe('starting up', () => {
  it('opens with no model, because nothing hands this window one at startup', async () => {
    await expect(desktop.initialState()).resolves.toBeNull();
  });
});

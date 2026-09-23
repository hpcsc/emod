import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

class FakeWorker {
  constructor() {
    this.sent = [];
    FakeWorker.started.push(this);
  }

  postMessage(message) {
    this.sent.push(message);
  }

  answer(message) {
    this.onmessage({ data: message });
  }
}

// The module starts its worker on import, so each test imports it afresh.
async function load() {
  vi.resetModules();
  FakeWorker.started = [];
  vi.stubGlobal('Worker', FakeWorker);
  return import('../static/platform.browser.js');
}

let browser;
let worker;
beforeEach(async () => {
  browser = await load();
  worker = FakeWorker.started[0];
});

afterEach(() => {
  vi.unstubAllGlobals();
});

async function started() {
  worker.answer({ started: true });
  await browser.ready;
}

describe('before the worker reports it started', () => {
  it('reports that it is not ready', () => {
    expect(browser.isReady).toBe(false);
  });

  it('rejects a parse with a descriptive error', async () => {
    await expect(browser.parseEmod('test')).rejects.toThrow('WASM not ready yet');
  });

  it('rejects an export with a descriptive error', async () => {
    await expect(browser.exportEmod({ nodes: [], edges: [] })).rejects.toThrow('WASM not ready yet');
  });
});

describe('starting the worker', () => {
  it('is ready once the worker reports it started', async () => {
    worker.answer({ started: true });

    await expect(browser.ready).resolves.toBeUndefined();
    expect(browser.isReady).toBe(true);
  });

  it('rejects ready with the reason the worker reports', async () => {
    worker.answer({ started: false, error: 'Failed to fetch WASM: 404 Not Found' });

    await expect(browser.ready).rejects.toThrow('WASM initialization failed: Failed to fetch WASM: 404 Not Found');
  });

  it('rejects ready when the worker script does not load', async () => {
    worker.onerror({ message: '' });

    await expect(browser.ready).rejects.toThrow('WASM initialization failed: the worker did not load');
  });

  it('rejects ready when the page cannot start a worker', async () => {
    vi.resetModules();
    vi.stubGlobal('Worker', undefined);

    const withoutWorkers = await import('../static/platform.browser.js');

    await expect(withoutWorkers.ready).rejects.toThrow('WASM initialization failed');
  });
});

describe('parsing', () => {
  it('sends the worker the source and the name of the file it came from', async () => {
    await started();

    browser.parseEmod('test source', 'orders.emod');

    expect(worker.sent).toHaveLength(1);
    expect(worker.sent[0].name).toBe('parseEmod');
    expect(JSON.parse(worker.sent[0].input)).toEqual({ source: 'test source', filename: 'orders.emod' });
  });

  it('answers what the worker returned for the call', async () => {
    await started();
    const parseResult = { diagnostics: [], diagram: { nodes: [], edges: [] } };

    const parsing = browser.parseEmod('test source');
    worker.answer({ id: worker.sent[0].id, output: JSON.stringify(parseResult) });

    await expect(parsing).resolves.toEqual(parseResult);
  });

  it('gives each of two overlapping calls its own answer', async () => {
    await started();

    const first = browser.parseEmod('first source');
    const second = browser.parseEmod('second source');
    worker.answer({ id: worker.sent[1].id, output: '{"answer":"second"}' });
    worker.answer({ id: worker.sent[0].id, output: '{"answer":"first"}' });

    await expect(first).resolves.toEqual({ answer: 'first' });
    await expect(second).resolves.toEqual({ answer: 'second' });
  });

  it('rejects with the error the worker reports', async () => {
    await started();

    const parsing = browser.parseEmod('test');
    worker.answer({ id: worker.sent[0].id, error: 'parse failed' });

    await expect(parsing).rejects.toThrow('parse failed');
  });

  it('rejects an answer that is not JSON', async () => {
    await started();

    const parsing = browser.parseEmod('test');
    worker.answer({ id: worker.sent[0].id, output: 'not valid json' });

    await expect(parsing).rejects.toThrow();
  });
});

describe('exporting', () => {
  it('sends the worker the diagram and answers the text the export wrote', async () => {
    await started();
    const diagram = { model_name: 'Billing', nodes: [], edges: [] };

    const exporting = browser.exportEmod(diagram);
    worker.answer({ id: worker.sent[0].id, output: JSON.stringify({ emod: 'model "Billing" {\n}\n' }) });

    expect(worker.sent[0].name).toBe('exportEmod');
    expect(JSON.parse(worker.sent[0].input)).toEqual(diagram);
    await expect(exporting).resolves.toBe('model "Billing" {\n}\n');
  });

  it('rejects with the reason the export reports', async () => {
    await started();

    const exporting = browser.exportEmod({ nodes: [], edges: [] });
    worker.answer({ id: worker.sent[0].id, output: JSON.stringify({ error: 'no model name' }) });

    await expect(exporting).rejects.toThrow('no model name');
  });
});

describe('window title', () => {
  it('names the document, which is the browser window title', () => {
    browser.setWindowTitle('hotel.emod — Emod Diagram Viewer');

    expect(document.title).toBe('hotel.emod — Emod Diagram Viewer');
  });
});

describe('unsaved edits', () => {
  // A page has no shell dialog whose Save writes anywhere, so there is nothing
  // to ask: the browser viewer's drop replaces the model as it always has.
  it('answers discard without asking, so a drop behaves exactly as it did', async () => {
    await expect(browser.resolveUnsavedEdits()).resolves.toBe('discard');
  });
});

describe('the unsaved-edits marker', () => {
  it('leaves the page exactly as it was, having no window of its own to mark', () => {
    browser.setWindowTitle('hotel.emod — Emod Diagram Viewer');
    const body = document.body.innerHTML;

    browser.setWindowModified(true);

    expect(document.title).toBe('hotel.emod — Emod Diagram Viewer');
    expect(document.body.innerHTML).toBe(body);
  });
});

// The viewer chains on the answer, so an implementation that returned nothing
// would break every open in the browser build the moment a recording is made.
describe('remembering an opened file', () => {
  it('accepts the file and does nothing, answering a resolved promise', async () => {
    await expect(browser.rememberOpenedFile('/models/billing.emod')).resolves.toBeUndefined();
  });
});

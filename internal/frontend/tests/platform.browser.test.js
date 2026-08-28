import { describe, it, expect, vi, beforeAll } from 'vitest';

// Set up globals before platform.browser.js is imported — these cause init to fail,
// which lets us test the module's initial state and error handling.
vi.hoisted(() => {
  globalThis.Go = class Go {
    constructor() { this.importObject = {}; }
    run() {}
  };
  globalThis.WebAssembly = {
    instantiateStreaming: undefined,
  };
  globalThis.fetch = () => Promise.reject(new Error('NetworkError'));
});

// Dynamic import — vitest caches this within the file, but each test file
// gets an isolated module registry, so the init() call inside platform.browser.js will
// use the globals we set above.
let browser;
beforeAll(async () => {
  browser = await import('../static/platform.browser.js');
});

describe('before the module is ready', () => {
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

describe('initialization failure', () => {
  it('rejects ready on network error', async () => {
    await expect(browser.ready).rejects.toThrow('WASM initialization failed');
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

import { describe, it, expect, vi, beforeAll } from 'vitest';

// Set up globals before wasm.js is imported — these cause init to fail,
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
// gets an isolated module registry, so the init() call inside wasm.js will
// use the globals we set above.
let wasm;
beforeAll(async () => {
  wasm = await import('../static/wasm.js');
});

describe('before the module is ready', () => {
  it('reports that it is not ready', () => {
    expect(wasm.isReady).toBe(false);
  });

  it('rejects a parse with a descriptive error', async () => {
    await expect(wasm.parseEmod('test')).rejects.toThrow('WASM not ready yet');
  });

  it('rejects an export with a descriptive error', async () => {
    await expect(wasm.exportEmod({ nodes: [], edges: [] })).rejects.toThrow('WASM not ready yet');
  });
});

describe('initialization failure', () => {
  it('rejects ready on network error', async () => {
    await expect(wasm.ready).rejects.toThrow('WASM initialization failed');
  });
});

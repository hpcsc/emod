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
  wasm = await import('./wasm.js');
});

describe('wasm exports', () => {
  it('exports parseEmod, ready, and isReady', () => {
    expect(wasm).toHaveProperty('parseEmod');
    expect(wasm).toHaveProperty('ready');
    expect(wasm).toHaveProperty('isReady');
  });

  it('isReady is a boolean starting as false', () => {
    expect(typeof wasm.isReady).toBe('boolean');
    expect(wasm.isReady).toBe(false);
  });

  it('ready is a Promise', () => {
    expect(wasm.ready).toBeInstanceOf(Promise);
  });
});

describe('parseEmod', () => {
  it('rejects with descriptive error when WASM not ready yet', async () => {
    await expect(wasm.parseEmod('test')).rejects.toThrow('WASM not ready yet');
  });
});

describe('initialization failure', () => {
  it('rejects ready on network error', async () => {
    await expect(wasm.ready).rejects.toThrow('WASM initialization failed');
  });
});

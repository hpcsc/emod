import { describe, it, expect, vi, beforeAll } from 'vitest';

// Test non-streaming fallback when instantiateStreaming is not available
vi.hoisted(() => {
  globalThis.Go = function Go() {
    this.importObject = {};
    this.run = function () {};
  };

  const mockModule = {};
  globalThis.WebAssembly = {
    // No instantiateStreaming — forces fallback
    compile: () => Promise.resolve(mockModule),
    instantiate: (mod, importObject) =>
      Promise.resolve({ instance: { exports: {} }, module: mod }),
  };

  globalThis.fetch = () =>
    Promise.resolve({ ok: true, arrayBuffer: () => Promise.resolve(new ArrayBuffer(0)) });
});

let wasm;
beforeAll(async () => {
  wasm = await import('../static/wasm.js');
});

describe('non-streaming fallback', () => {
  it('initializes via compile+instantiate when instantiateStreaming is absent', async () => {
    await expect(wasm.ready).resolves.toBeUndefined();
    expect(wasm.isReady).toBe(true);
  });
});

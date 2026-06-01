import { describe, it, expect, vi, beforeAll } from 'vitest';

// Set up globals that cause an HTTP 404 error during WASM fetch
vi.hoisted(() => {
  const mockGoRun = function () {};
  globalThis.Go = function Go() {
    this.importObject = {};
    this.run = mockGoRun;
  };

  globalThis.WebAssembly = {
    instantiateStreaming: vi.fn ? vi.fn() : function () {},
  };

  globalThis.fetch = () =>
    Promise.resolve({
      ok: false,
      status: 404,
      statusText: 'Not Found',
    });
});

let wasm;
beforeAll(async () => {
  wasm = await import('../static/wasm.js');
});

describe('initialization HTTP error', () => {
  it('rejects ready with descriptive message including HTTP status', async () => {
    await expect(wasm.ready).rejects.toThrow(
      'WASM initialization failed: Failed to fetch WASM: 404 Not Found',
    );
  });
});

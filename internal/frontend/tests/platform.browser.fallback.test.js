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

let browser;
beforeAll(async () => {
  browser = await import('../static/platform.browser.js');
});

describe('non-streaming fallback', () => {
  it('initializes via compile+instantiate when instantiateStreaming is absent', async () => {
    await expect(browser.ready).resolves.toBeUndefined();
    expect(browser.isReady).toBe(true);
  });
});

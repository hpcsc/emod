import { describe, it, expect, vi, beforeAll } from 'vitest';

// Test non-streaming fallback when instantiateStreaming is not available
vi.hoisted(() => {
  // Distinctive so an assertion on it cannot be satisfied by any empty object:
  // instantiating against a bare {} must be a different value, not an equal one.
  globalThis.goImportObject = { env: { marker: 'from-the-go-runtime' } };
  globalThis.Go = function Go() {
    this.importObject = globalThis.goImportObject;
    this.run = function () {};
  };

  const mockModule = {};
  globalThis.compileCalls = [];
  globalThis.instantiateCalls = [];
  globalThis.WebAssembly = {
    // No instantiateStreaming — forces fallback
    compile: (bytes) => { globalThis.compileCalls.push(bytes); return Promise.resolve(mockModule); },
    instantiate: (mod, importObject) => {
      globalThis.instantiateCalls.push({ mod, importObject });
      return Promise.resolve({ instance: { exports: {} }, module: mod });
    },
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

  // Readiness alone is satisfied by any path that resolves, so it survives the
  // fallback branch being deleted. These pin the branch the name claims.
  it('reads the response body and compiles it itself', async () => {
    await browser.ready;
    expect(globalThis.compileCalls).toHaveLength(1);
    expect(globalThis.compileCalls[0]).toBeInstanceOf(ArrayBuffer);
  });

  it('instantiates the compiled module against the Go import object', async () => {
    await browser.ready;
    expect(globalThis.instantiateCalls).toHaveLength(1);
    expect(globalThis.instantiateCalls[0].importObject).toBe(globalThis.goImportObject);
  });
});

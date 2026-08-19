import { describe, it, expect, vi, beforeAll, beforeEach } from 'vitest';

// Drives the real platform.browser.js, as the four sibling files do. The
// viewer's own tests mock the seam and re-implement this lookup inside the
// mock, so without this file nothing exercises the read that gives
// `emod diagram --serve` its first paint.
vi.hoisted(() => {
  globalThis.Go = function Go() {
    this.importObject = {};
    this.run = function () {};
  };
  globalThis.WebAssembly = {
    instantiateStreaming: () => Promise.resolve({ instance: { exports: {} }, module: {} }),
  };
  globalThis.fetch = () => Promise.resolve({ ok: true });
});

let browser;
beforeAll(async () => {
  browser = await import('../static/platform.browser.js');
});

beforeEach(() => {
  delete globalThis.INITIAL_DATA;
});

describe('initial state', () => {
  it('answers nothing when the host injected no state', async () => {
    await expect(browser.initialState()).resolves.toBeNull();
  });

  it('answers nothing when the host injected an explicit null', async () => {
    globalThis.INITIAL_DATA = null;
    await expect(browser.initialState()).resolves.toBeNull();
  });

  it('answers the state the host injected', async () => {
    const injected = { diagram: { model_name: 'Billing', nodes: [], edges: [] } };
    globalThis.INITIAL_DATA = injected;
    await expect(browser.initialState()).resolves.toEqual(injected);
  });
});

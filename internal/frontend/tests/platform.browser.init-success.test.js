import { describe, it, expect, vi, beforeAll } from 'vitest';

// Set up globals for successful WASM initialization
vi.hoisted(() => {
  const mockGoRun = function () {};
  globalThis.Go = function Go() {
    this.importObject = {};
    this.run = mockGoRun;
  };

  // Mock instantiateStreaming to succeed
  globalThis.WebAssembly = {
    instantiateStreaming: (response, importObject) => {
      return Promise.resolve({
        instance: { exports: {} },
        module: {},
      });
    },
  };

  globalThis.fetch = () => Promise.resolve({ ok: true });
});

let browser;
beforeAll(async () => {
  browser = await import('../static/platform.browser.js');
});

describe('successful initialization', () => {
  it('resolves ready and sets isReady to true', async () => {
    await expect(browser.ready).resolves.toBeUndefined();
    expect(browser.isReady).toBe(true);
  });
});

describe('parseEmod after ready', () => {
  it('calls globalThis.parseEmod and returns parsed result', async () => {
    // Set up the parse function that WASM would register
    const parseResult = { diagnostics: [], diagram: { nodes: [], edges: [] } };
    globalThis.parseEmod = vi.fn().mockReturnValue(JSON.stringify(parseResult));

    const result = await browser.parseEmod('test source');
    expect(globalThis.parseEmod).toHaveBeenCalledWith('{"source":"test source"}');
    expect(result).toEqual(parseResult);
  });

  it('propagates errors from globalThis.parseEmod', async () => {
    globalThis.parseEmod = vi.fn().mockImplementation(() => {
      throw new Error('parse failed');
    });

    await expect(browser.parseEmod('test')).rejects.toThrow('parse failed');
  });

  it('propagates JSON parse errors', async () => {
    globalThis.parseEmod = vi.fn().mockReturnValue('not valid json');

    await expect(browser.parseEmod('test')).rejects.toThrow();
  });
});

// @vitest-environment node
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// Distinctive so an assertion on it cannot be satisfied by any empty object:
// instantiating against a bare {} must be a different value, not an equal one.
const goImportObject = { env: { marker: 'from-the-go-runtime' } };

const streams = {
  instantiateStreaming: () => Promise.resolve({ instance: { exports: {} }, module: {} }),
};
const fetches = () => Promise.resolve({ ok: true });

let posted;

beforeEach(() => {
  vi.resetModules();
  posted = [];
  vi.stubGlobal('Go', function Go() {
    this.importObject = goImportObject;
    this.run = function () {};
  });
  vi.stubGlobal('postMessage', (message) => posted.push(message));
});

afterEach(() => {
  vi.unstubAllGlobals();
  delete globalThis.onmessage;
});

// The worker starts the Go runtime on import, so each test imports it afresh
// against the globals it stubbed, and waits for the one message that reports
// how the start went.
async function start({ fetch, WebAssembly }) {
  vi.stubGlobal('fetch', fetch);
  vi.stubGlobal('WebAssembly', WebAssembly);
  await import('../static/wasm-worker.js');
  await vi.waitFor(() => expect(posted).toHaveLength(1));
  return posted[0];
}

describe('starting the Go runtime', () => {
  it('reports it started once the module streams in', async () => {
    expect(await start({ fetch: fetches, WebAssembly: streams })).toEqual({ started: true });
  });

  it('compiles the fetched bytes itself when the browser cannot stream a module', async () => {
    const bytes = new ArrayBuffer(0);
    const compiled = {};
    const compile = vi.fn(() => Promise.resolve(compiled));
    const instantiate = vi.fn(() => Promise.resolve({ instance: { exports: {} }, module: compiled }));

    const outcome = await start({
      fetch: () => Promise.resolve({ ok: true, arrayBuffer: () => Promise.resolve(bytes) }),
      WebAssembly: { compile, instantiate },
    });

    expect(outcome).toEqual({ started: true });
    expect(compile.mock.calls[0][0]).toBe(bytes);
    expect(instantiate.mock.calls[0][0]).toBe(compiled);
    expect(instantiate.mock.calls[0][1]).toBe(goImportObject);
  });

  it('reports the HTTP status when the module cannot be fetched', async () => {
    const outcome = await start({
      fetch: () => Promise.resolve({ ok: false, status: 404, statusText: 'Not Found' }),
      WebAssembly: streams,
    });

    expect(outcome).toEqual({ started: false, error: 'Failed to fetch WASM: 404 Not Found' });
  });

  it('reports why the fetch failed', async () => {
    const outcome = await start({
      fetch: () => Promise.reject(new Error('NetworkError')),
      WebAssembly: streams,
    });

    expect(outcome).toEqual({ started: false, error: 'NetworkError' });
  });
});

describe('answering a call', () => {
  it('answers with what the named Go function returned', async () => {
    await start({ fetch: fetches, WebAssembly: streams });
    const parseEmod = vi.fn(() => '{"diagnostics":[]}');
    vi.stubGlobal('parseEmod', parseEmod);

    globalThis.onmessage({ data: { id: 7, name: 'parseEmod', input: '{"source":"test source"}' } });

    expect(parseEmod).toHaveBeenCalledWith('{"source":"test source"}');
    expect(posted[1]).toEqual({ id: 7, output: '{"diagnostics":[]}' });
  });

  it('answers with the message of an error the Go function threw', async () => {
    await start({ fetch: fetches, WebAssembly: streams });
    vi.stubGlobal('parseEmod', () => { throw new Error('parse failed'); });

    globalThis.onmessage({ data: { id: 7, name: 'parseEmod', input: '{}' } });

    expect(posted[1]).toEqual({ id: 7, error: 'parse failed' });
  });
});

import { describe, it, expect, beforeAll, beforeEach } from 'vitest';

// Drives the real desktop implementation against a stand-in for the bindings
// the desktop build generates. Without this only droppedFile ran, so nothing
// would notice saveFile resolving silently, exportEmod losing its error check,
// or parseEmod sending an envelope shape the Go service does not accept.
let desktop;
let stub;
let runtime;
beforeAll(async () => {
  stub = await import('./bindings-stub.js');
  runtime = await import('./wails-runtime-stub.js');
  desktop = await import('../desktop/platform.desktop.js');
});

beforeEach(() => {
  stub.calls.length = 0;
  runtime.calls.length = 0;
  stub.answers.ParseEmod = '{"diagnostics":[],"diagram":{"nodes":[],"edges":[]}}';
  stub.answers.ExportEmod = '{"emod":"emod 1\\nmodel \\"Billing\\"\\n"}';
});

describe('readiness', () => {
  it('is ready immediately, because the core is linked in rather than fetched', async () => {
    expect(desktop.isReady).toBe(true);
    await expect(desktop.ready).resolves.toBeUndefined();
  });
});

describe('parse', () => {
  it('sends the source envelope the Go service expects', async () => {
    await desktop.parseEmod('emod 1\nmodel "Billing"\n');

    expect(stub.calls).toEqual([['ParseEmod', JSON.stringify({ source: 'emod 1\nmodel "Billing"\n' })]]);
  });

  it('answers the decoded document rather than the raw string', async () => {
    stub.answers.ParseEmod = '{"diagnostics":[{"message":"x"}],"diagram":{"nodes":[1],"edges":[]}}';

    await expect(desktop.parseEmod('src')).resolves.toEqual({
      diagnostics: [{ message: 'x' }],
      diagram: { nodes: [1], edges: [] },
    });
  });
});

describe('export', () => {
  it('sends the diagram document itself, not a source envelope', async () => {
    await desktop.exportEmod({ model_name: 'Billing', nodes: [], edges: [] });

    expect(stub.calls[0][0]).toBe('ExportEmod');
    expect(JSON.parse(stub.calls[0][1])).toEqual({ model_name: 'Billing', nodes: [], edges: [] });
  });

  it('unwraps the emod text', async () => {
    await expect(desktop.exportEmod({})).resolves.toBe('emod 1\nmodel "Billing"\n');
  });

  it('raises the error envelope rather than resolving with it', async () => {
    stub.answers.ExportEmod = '{"error":"importer: invalid diagram JSON"}';

    await expect(desktop.exportEmod({})).rejects.toThrow('importer: invalid diagram JSON');
  });
});

describe('window title', () => {
  it('names the native window, which is the only title a desktop user can see', () => {
    desktop.setWindowTitle('hotel.emod — Emod Diagram Viewer');

    expect(runtime.calls).toEqual([['Window.SetTitle', 'hotel.emod — Emod Diagram Viewer']]);
  });

  it('passes the name through untouched, so the shared viewer alone decides what a window is called', () => {
    desktop.setWindowTitle('Emod Diagram Viewer');

    expect(runtime.calls[0][1]).toBe('Emod Diagram Viewer');
  });
});

describe('the operations this build does not have', () => {
  it('refuses to save, audibly, rather than resolving as though it had', async () => {
    await expect(desktop.saveFile('Billing.emod', 'emod 1\n')).rejects
      .toThrow('Saving is not available in this build yet');
  });

  it('opens with no model, because nothing hands this window one at startup', async () => {
    await expect(desktop.initialState()).resolves.toBeNull();
  });
});

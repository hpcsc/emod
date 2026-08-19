import { describe, it, expect, vi } from 'vitest';

// Each test builds a fresh module graph so it controls when the WASM `ready`
// promise settles — resolving it is one-way, so a shared instance would leak
// readiness between tests.
async function freshExport() {
  vi.resetModules();
  const exportEmod = vi.fn();
  let markReady;
  const ready = new Promise((resolve) => { markReady = resolve; });
  vi.doMock('../static/wasm.js', () => ({ ready, exportEmod }));
  const { Export } = await import('../static/emod-export.js');
  return { Export, exportEmod, markReady };
}

function store() {
  return {
    modelName: 'Billing',
    nodes: [{ id: 'context-1', type: 'context', label: 'Payments', parentId: null }],
    edges: [{ source: 'command-1', target: 'event-1', type: 'flow' }],
  };
}

describe('Export.exportToEmodString', () => {
  it('sends the store to the WASM exporter as a diagram document', async () => {
    const { Export, exportEmod, markReady } = await freshExport();
    exportEmod.mockResolvedValue('model "Billing"\n');
    markReady();

    await Export.exportToEmodString(store());

    expect(exportEmod).toHaveBeenCalledWith({
      model_name: 'Billing',
      nodes: [{ id: 'context-1', type: 'context', label: 'Payments', parentId: null }],
      edges: [{ source: 'command-1', target: 'event-1', type: 'flow' }],
    });
  });

  it('resolves with the .emod text the exporter returns', async () => {
    const { Export, exportEmod, markReady } = await freshExport();
    exportEmod.mockResolvedValue('model "Billing"\n');
    markReady();

    await expect(Export.exportToEmodString(store())).resolves.toBe('model "Billing"\n');
  });

  it('rejects when the exporter reports a failure', async () => {
    const { Export, exportEmod, markReady } = await freshExport();
    exportEmod.mockRejectedValue(new Error('invalid diagram JSON'));
    markReady();

    await expect(Export.exportToEmodString(store())).rejects.toThrow('invalid diagram JSON');
  });

  it('waits for the WASM module before exporting', async () => {
    const { Export, exportEmod } = await freshExport();
    exportEmod.mockResolvedValue('model "Billing"\n');

    Export.exportToEmodString(store());
    await Promise.resolve();

    expect(exportEmod).not.toHaveBeenCalled();
  });
});

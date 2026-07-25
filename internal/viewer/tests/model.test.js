import { describe, it, expect, vi, beforeEach } from 'vitest';
import { Model } from '../static/model.js';
import { parseEmod } from '../static/wasm.js';

vi.mock('../static/wasm.js', () => ({
  parseEmod: vi.fn(),
  ready: Promise.resolve(),
  isReady: true,
}));

describe('Model', () => {
  describe('rebuildNodeIndex', () => {
    it('builds a Map from nodes array keyed by id', () => {
      const store = { nodeById: new Map(), nodes: [
        { id: 'a', label: 'Alpha' },
        { id: 'b', label: 'Beta' },
      ] };
      Model.rebuildNodeIndex(store);
      expect(store.nodeById.get('a')).toEqual({ id: 'a', label: 'Alpha' });
      expect(store.nodeById.get('b')).toEqual({ id: 'b', label: 'Beta' });
      expect(store.nodeById.size).toBe(2);
    });

    it('handles empty nodes array', () => {
      const store = { nodeById: new Map(), nodes: [] };
      Model.rebuildNodeIndex(store);
      expect(store.nodeById.size).toBe(0);
    });

    it('replaces previous index on rebuild', () => {
      const store = { nodeById: new Map([['old', { id: 'old' }]]), nodes: [{ id: 'new' }] };
      Model.rebuildNodeIndex(store);
      expect(store.nodeById.has('old')).toBe(false);
      expect(store.nodeById.get('new')).toEqual({ id: 'new' });
    });
  });

  describe('generateNodeId', () => {
    it('generates an id containing the prefix', () => {
      const store = { nodeById: new Map() };
      const id = Model.generateNodeId('cmd', store);
      expect(id).toContain('cmd');
    });

    it('generates unique ids on successive calls', () => {
      const store = { nodeById: new Map() };
      const a = Model.generateNodeId('cmd', store);
      const b = Model.generateNodeId('cmd', store);
      expect(a).not.toBe(b);
    });

    it('retries when generated id already exists', () => {
      const store = { nodeById: new Map() };
      const id = Model.generateNodeId('evt', store);
      store.nodeById.set(id, { id });
      const id2 = Model.generateNodeId('evt', store);
      expect(id2).not.toBe(id);
      expect(id2).toContain('evt');
    });
  });

  describe('moveSlice', () => {
    function makeNodes() {
      return [
        { id: 'cmd0', type: 'command', parentId: 'sl1' },
        { id: 'sl1', type: 'slice', parentId: 'agg1' },
        { id: 'sl2', type: 'slice', parentId: 'agg1' },
        { id: 'sl3', type: 'slice', parentId: 'agg1' },
        { id: 'agg1', type: 'aggregate', parentId: 'ctx1' },
      ];
    }

    it('moves a slice to position 0 within the same aggregate', () => {
      const nodes = makeNodes();
      const ok = Model.moveSlice(nodes, 'sl2', 0);
      expect(ok).toBe(true);
      const slices = nodes.filter(function(n) { return n.type === 'slice' && n.parentId === 'agg1'; });
      expect(slices[0].id).toBe('sl2');
      expect(slices[1].id).toBe('sl1');
      expect(slices[2].id).toBe('sl3');
    });

    it('moves a slice to position 2 within the same aggregate', () => {
      const nodes = makeNodes();
      const ok = Model.moveSlice(nodes, 'sl2', 2);
      expect(ok).toBe(true);
      const slices = nodes.filter(function(n) { return n.type === 'slice' && n.parentId === 'agg1'; });
      expect(slices[0].id).toBe('sl1');
      expect(slices[1].id).toBe('sl3');
      expect(slices[2].id).toBe('sl2');
    });

    it('moves a slice multiple positions left', () => {
      const nodes = makeNodes();
      Model.moveSlice(nodes, 'sl3', 0);
      const slices = nodes.filter(function(n) { return n.type === 'slice' && n.parentId === 'agg1'; });
      expect(slices[0].id).toBe('sl3');
      expect(slices[1].id).toBe('sl1');
      expect(slices[2].id).toBe('sl2');
    });

    it('moves a slice multiple positions right', () => {
      const nodes = makeNodes();
      Model.moveSlice(nodes, 'sl1', 2);
      const slices = nodes.filter(function(n) { return n.type === 'slice' && n.parentId === 'agg1'; });
      expect(slices[0].id).toBe('sl2');
      expect(slices[1].id).toBe('sl3');
      expect(slices[2].id).toBe('sl1');
    });

    it('does nothing when moving first slice left (clamped to 0)', () => {
      const nodes = makeNodes();
      const ok = Model.moveSlice(nodes, 'sl1', -1);
      expect(ok).toBe(false);
      const slices = nodes.filter(function(n) { return n.type === 'slice' && n.parentId === 'agg1'; });
      expect(slices[0].id).toBe('sl1');
    });

    it('does nothing when moving last slice right (clamped to last)', () => {
      const nodes = makeNodes();
      const ok = Model.moveSlice(nodes, 'sl3', 99);
      expect(ok).toBe(false);
      const slices = nodes.filter(function(n) { return n.type === 'slice' && n.parentId === 'agg1'; });
      expect(slices[2].id).toBe('sl3');
    });

    it('returns false for a non-existent slice id', () => {
      const nodes = makeNodes();
      expect(Model.moveSlice(nodes, 'nonexistent', 0)).toBe(false);
    });

    it('returns false for a node that is not a slice', () => {
      const nodes = makeNodes();
      expect(Model.moveSlice(nodes, 'agg1', 0)).toBe(false);
    });

    it('only reorders slices within the same aggregate', () => {
      const nodes = makeNodes();
      nodes.push(
        { id: 'sl_other', type: 'slice', parentId: 'agg2' },
        { id: 'agg2', type: 'aggregate', parentId: 'ctx1' },
      );
      Model.moveSlice(nodes, 'sl1', 1);
      const agg1Slices = nodes.filter(function(n) { return n.type === 'slice' && n.parentId === 'agg1'; });
      expect(agg1Slices[0].id).toBe('sl2');
      expect(agg1Slices[1].id).toBe('sl1');
      expect(agg1Slices[2].id).toBe('sl3');
      const agg2Slices = nodes.filter(function(n) { return n.type === 'slice' && n.parentId === 'agg2'; });
      expect(agg2Slices[0].id).toBe('sl_other');
    });
  });

  describe('generateLabel', () => {
    it('returns new-prefix when existing list is empty', () => {
      expect(Model.generateLabel('cmd', [])).toBe('new-cmd');
    });

    it('appends a number when existing list has items', () => {
      expect(Model.generateLabel('cmd', [{ id: 'a' }])).toBe('new-cmd-2');
    });

    it('increments based on count regardless of prefixes', () => {
      expect(Model.generateLabel('evt', [{ id: 'x' }, { id: 'y' }])).toBe('new-evt-3');
    });
  });

  describe('sendParse', () => {
    let store;
    let statusEl;

    beforeEach(() => {
      store = { nodeById: new Map(), nodes: [], edges: [] };
      statusEl = document.createElement('div');
      vi.clearAllMocks();
      parseEmod.mockResolvedValue({ diagnostics: [], diagram: { nodes: [], edges: [] } });
    });

    it('rejects with error and updates status when source is empty', async () => {
      await expect(Model.sendParse(store, '', statusEl)).rejects.toThrow('no source');
      expect(statusEl.textContent).toContain('Paste some .emod content first');
      expect(statusEl.className).toBe('status error');
    });

    it('calls parseEmod with raw .emod source', async () => {
      const diagram = { nodes: [{ id: 'n1', type: 'command', label: 'Test' }], edges: [] };
      const result = { diagnostics: [], diagram };
      parseEmod.mockResolvedValue(result);

      const value = await Model.sendParse(store, 'context Test {}', statusEl);
      expect(parseEmod).toHaveBeenCalledWith('context Test {}');
      expect(value).toEqual(result);
    });

    it('sets parsing status on status element during WASM parse', async () => {
      let resolvePromise;
      parseEmod.mockReturnValue(new Promise(function(resolve) { resolvePromise = resolve; }));

      const promise = Model.sendParse(store, 'context X {}', statusEl);
      expect(statusEl.textContent).toBe('⏳ Parsing...');
      expect(statusEl.className).toBe('');
      resolvePromise({ diagnostics: [], diagram: { nodes: [], edges: [] } });
      await promise;
    });

    it('propagates WASM errors', async () => {
      parseEmod.mockRejectedValue(new Error('syntax error'));

      await expect(Model.sendParse(store, 'context Bad {}', statusEl)).rejects.toThrow('syntax error');
    });

    it('returns diagram JSON directly when source has nodes array', async () => {
      const source = JSON.stringify({ nodes: [{ id: 'n1' }], edges: [] });
      const result = await Model.sendParse(store, source, statusEl);
      expect(result).toEqual({ diagnostics: [], diagram: { nodes: [{ id: 'n1' }], edges: [] } });
      expect(parseEmod).not.toHaveBeenCalled();
    });

    it('returns AST JSON directly when source has model key', async () => {
      const modelData = { name: 'TestModel', actors: [] };
      const source = JSON.stringify({ model: modelData });
      const result = await Model.sendParse(store, source, statusEl);
      expect(result).toEqual({ diagnostics: [], diagram: { model: modelData } });
      expect(parseEmod).not.toHaveBeenCalled();
    });

    it('recognises diagram JSON even when it also has a model key', async () => {
      const source = JSON.stringify({ nodes: [{ id: 'n1' }], edges: [], model: { name: 'Test' } });
      const result = await Model.sendParse(store, source, statusEl);
      expect(result).toEqual({ diagnostics: [], diagram: { nodes: [{ id: 'n1' }], edges: [], model: { name: 'Test' } } });
      expect(parseEmod).not.toHaveBeenCalled();
    });

    it('rejects empty after whitespace trimming is not required', async () => {
      await expect(Model.sendParse(store, '', statusEl)).rejects.toThrow('no source');
      await expect(Model.sendParse(store, null, statusEl)).rejects.toThrow('no source');
      await expect(Model.sendParse(store, undefined, statusEl)).rejects.toThrow('no source');
    });
  });

  describe('autoDetectEdgeType', () => {
    function storeWith(types) {
      const nodes = Object.entries(types).map(([id, type]) => ({ id, type }));
      return { nodes, nodeById: new Map(nodes.map((n) => [n.id, n])) };
    }

    // The direction each type runs in has to match what the exporter writes,
    // or the importer drops the edge when the diagram is written back out.
    const cases = [
      ['command', 'event', 'flow'],
      ['trigger', 'command', 'trigger_command'],
      ['event', 'view', 'subscription'],
      ['event', 'automation', 'automation_trigger'],
      ['automation', 'command', 'automation_command'],
      ['view', 'translation', 'reads'],
      ['translation', 'command', 'translation_command'],
    ];

    for (const [from, to, expected] of cases) {
      it(`types a ${from} to ${to} arrow as ${expected}`, () => {
        const store = storeWith({ a: from, b: to });
        expect(Model.autoDetectEdgeType(store, 'a', 'b')).toBe(expected);
      });
    }

    it('does not treat a view to event arrow as a subscription', () => {
      const store = storeWith({ a: 'view', b: 'event' });
      expect(Model.autoDetectEdgeType(store, 'a', 'b')).not.toBe('subscription');
    });

    it('falls back to flow for a pairing with no defined direction', () => {
      const store = storeWith({ a: 'event', b: 'command' });
      expect(Model.autoDetectEdgeType(store, 'a', 'b')).toBe('flow');
    });

    it('falls back to flow when an endpoint is not in the model', () => {
      const store = storeWith({ a: 'command' });
      expect(Model.autoDetectEdgeType(store, 'a', 'gone')).toBe('flow');
      expect(Model.autoDetectEdgeType(store, 'gone', 'a')).toBe('flow');
    });
  });
});

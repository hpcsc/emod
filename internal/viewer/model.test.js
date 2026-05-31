import { describe, it, expect } from 'vitest';
import { Model } from './model.js';

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
});

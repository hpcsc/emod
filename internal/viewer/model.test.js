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

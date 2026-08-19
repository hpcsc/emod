import { describe, it, expect } from 'vitest';
import { Layout } from '../static/layout.js';

describe('Layout', () => {
  describe('buildTree', () => {
    it('builds tree from flat node array with parent references', () => {
      const nodes = [
        { id: 'ctx1', type: 'context' },
        { id: 'agg1', type: 'aggregate', parentId: 'ctx1' },
        { id: 'sl1', type: 'slice', parentId: 'agg1' },
        { id: 'cmd1', type: 'command', parentId: 'sl1' },
      ];
      const tree = Layout.buildTree(nodes);
      expect(tree.roots).toHaveLength(1);
      expect(tree.roots[0].id).toBe('ctx1');
      expect(tree.roots[0].children).toHaveLength(1);
      expect(tree.roots[0].children[0].id).toBe('agg1');
      expect(tree.roots[0].children[0].children[0].id).toBe('sl1');
      expect(tree.roots[0].children[0].children[0].children[0].id).toBe('cmd1');
    });

    it('treats nodes with missing parentId as roots', () => {
      const nodes = [
        { id: 'ctx1', type: 'context' },
        { id: 'ctx2', type: 'context' },
      ];
      const tree = Layout.buildTree(nodes);
      expect(tree.roots).toHaveLength(2);
    });

    it('treats orphans (parentId not found) as roots', () => {
      const nodes = [
        { id: 'orphan', type: 'command', parentId: 'nonexistent' },
      ];
      const tree = Layout.buildTree(nodes);
      expect(tree.roots).toHaveLength(1);
      expect(tree.roots[0].id).toBe('orphan');
    });

    it('returns empty tree for empty input', () => {
      const tree = Layout.buildTree([]);
      expect(tree.roots).toEqual([]);
      expect(tree.byId).toEqual({});
    });

    it('populates byId map with all nodes', () => {
      const nodes = [
        { id: 'a', type: 'context' },
        { id: 'b', type: 'aggregate', parentId: 'a' },
      ];
      const tree = Layout.buildTree(nodes);
      expect(Object.keys(tree.byId)).toEqual(['a', 'b']);
      expect(tree.byId.a.id).toBe('a');
      expect(tree.byId.b.id).toBe('b');
    });
  });

  describe('isCrossBoundary', () => {
    const nodes = [
      { id: 'agg1', type: 'aggregate', parentId: 'ctx1' },
      { id: 'agg2', type: 'aggregate', parentId: 'ctx2' },
      { id: 'agg3', type: 'aggregate', parentId: 'ctx1' },
    ];

    it('returns true when nodes have different parentIds', () => {
      expect(Layout.isCrossBoundary(nodes, 'agg1', 'agg2')).toBe(true);
    });

    it('returns false when nodes have the same parentId', () => {
      expect(Layout.isCrossBoundary(nodes, 'agg1', 'agg3')).toBe(false);
    });

    it('returns null when source node is not found', () => {
      expect(Layout.isCrossBoundary(nodes, 'unknown', 'agg2')).toBeNull();
    });

    it('returns null when target node is not found', () => {
      expect(Layout.isCrossBoundary(nodes, 'agg1', 'unknown')).toBeNull();
    });
  });

  describe('getDescendantIds', () => {
    it('returns root id plus children and grandchildren', () => {
      const nodes = [
        { id: 'ctx1' }, { id: 'agg1', parentId: 'ctx1' },
        { id: 'sl1', parentId: 'agg1' }, { id: 'sl2', parentId: 'agg1' },
        { id: 'cmd1', parentId: 'sl1' },
      ];
      const ids = Layout.getDescendantIds(nodes, 'ctx1');
      expect(ids.sort()).toEqual(['agg1', 'cmd1', 'ctx1', 'sl1', 'sl2'].sort());
    });

    it('returns just root when it has no children', () => {
      const nodes = [{ id: 'orphan' }];
      const ids = Layout.getDescendantIds(nodes, 'orphan');
      expect(ids).toEqual(['orphan']);
    });

    it('returns just root id when root is not in the node list', () => {
      const nodes = [{ id: 'other' }];
      const ids = Layout.getDescendantIds(nodes, 'missing');
      expect(ids).toEqual(['missing']);
    });
  });

  describe('getConnectedEdges', () => {
    const edges = [
      { id: 'e1', type: 'flow', source: 'a', target: 'b' },
      { id: 'e2', type: 'subscription', source: 'b', target: 'c' },
      { id: 'e3', type: 'trigger_command', source: 'a', target: 'c' },
      { id: 'e4', type: 'automation_trigger', source: 'd', target: 'e' },
      { id: 'e5', type: 'reads', source: 'f', target: 'g' },
      { id: 'e6', type: 'unknown_type', source: 'a', target: 'b' },
    ];

    it('returns edges where node is source', () => {
      const result = Layout.getConnectedEdges(edges, 'a');
      expect(result.map(e => e.id).sort()).toEqual(['e1', 'e3'].sort());
    });

    it('returns edges where node is target', () => {
      const result = Layout.getConnectedEdges(edges, 'b');
      expect(result.map(e => e.id).sort()).toEqual(['e1', 'e2'].sort());
    });

    it('excludes edge types not in the allowed list', () => {
      const result = Layout.getConnectedEdges(edges, 'a');
      expect(result.find(e => e.type === 'unknown_type')).toBeUndefined();
    });

    it('returns empty when node has no edges', () => {
      expect(Layout.getConnectedEdges(edges, 'isolated')).toEqual([]);
    });
  });

  describe('getSliceChildNodeIds', () => {
    const nodes = [
      { id: 'sl1', type: 'slice' },
      { id: 'cmd1', type: 'command', parentId: 'sl1' },
      { id: 'evt1', type: 'event', parentId: 'sl1' },
      { id: 'trg1', type: 'trigger', parentId: 'sl1' },
      { id: 'view1', type: 'view', parentId: 'sl1' },
      { id: 'automation1', type: 'automation', parentId: 'sl1' },
      { id: 'trans1', type: 'translation', parentId: 'sl1' },
      { id: 'not_slice_child', type: 'command', parentId: 'other' },
    ];

    it('returns children with allowed types', () => {
      const ids = Layout.getSliceChildNodeIds(nodes, 'sl1');
      expect(ids.sort()).toEqual(['automation1', 'cmd1', 'evt1', 'trans1', 'trg1', 'view1'].sort());
    });

    it('excludes children whose type is not in the allowed list', () => {
      const mixed = [
        ...nodes,
        { id: 'unknown_child', type: 'widget', parentId: 'sl1' },
      ];
      const ids = Layout.getSliceChildNodeIds(mixed, 'sl1');
      expect(ids).not.toContain('unknown_child');
    });

    it('returns empty when slice has no children', () => {
      expect(Layout.getSliceChildNodeIds(nodes, 'empty_sl')).toEqual([]);
    });
  });

  describe('computeArrowD', () => {
    it('returns a straight downward path for in-boundary arrow', () => {
      const src = { x: 100, y: 0, w: 200, h: 55 };
      const tgt = { x: 100, y: 200, w: 200, h: 55 };
      const d = Layout.computeArrowD(src, tgt, false);
      expect(d).toBe('M 200,55 L 200,188.5');
    });

    it('returns a backward path when target is above source', () => {
      const src = { x: 100, y: 200, w: 200, h: 55 };
      const tgt = { x: 100, y: 0, w: 200, h: 55 };
      const d = Layout.computeArrowD(src, tgt, false);
      expect(d).toBe('M 200,200 L 200,66.5');
    });

    it('returns a curved cubic bezier path for cross-boundary arrows', () => {
      const src = { x: 0, y: 100, w: 200, h: 55 };
      const tgt = { x: 500, y: 100, w: 200, h: 55 };
      const d = Layout.computeArrowD(src, tgt, true);
      expect(d.startsWith('M ')).toBe(true);
      expect(d).toContain('C ');
    });

    it('produces different paths for cross-boundary vs in-boundary', () => {
      const src = { x: 0, y: 100, w: 200, h: 55 };
      const tgt = { x: 500, y: 100, w: 200, h: 55 };
      const inBound = Layout.computeArrowD(src, tgt, false);
      const cross = Layout.computeArrowD(src, tgt, true);
      expect(cross).not.toBe(inBound);
      expect(cross).toContain('C ');
    });
  });
});

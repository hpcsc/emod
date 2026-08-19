import { describe, it, expect } from 'vitest';
import { CtxActions } from '../static/ctx-actions.js';

function createStore(nodes, edges) {
  const store = {
    nodes: nodes || [],
    edges: edges || [],
    nodeById: new Map(),
    interaction: { ctxMenu: null },
  };
  store.nodes.forEach(function(n) { store.nodeById.set(n.id, n); });
  return store;
}

function withSlice() {
  return createStore([
    { id: 'ctx1', type: 'context', label: 'C' },
    { id: 'agg1', type: 'aggregate', label: 'A', parentId: 'ctx1' },
    { id: 'sl1', type: 'slice', label: 'S1', parentId: 'agg1' },
  ]);
}

function childrenOf(store, parentId, type) {
  return store.nodes.filter(function(n) { return n.parentId === parentId && n.type === type; });
}

describe('CtxActions.apply', () => {
  it('reports no change for an unknown action', () => {
    const store = withSlice();
    store.interaction.ctxMenu = { targetSliceId: 'sl1' };

    expect(CtxActions.apply(store, 'not-an-action')).toBe(false);
    expect(store.nodes).toHaveLength(3);
  });

  it('reports no change when no context menu is open', () => {
    const store = withSlice();

    expect(CtxActions.apply(store, 'add-command')).toBe(false);
    expect(store.nodes).toHaveLength(3);
  });
});

describe('CtxActions.apply — adding nodes', () => {
  it('adds a command under the slice the menu was opened over', () => {
    const store = withSlice();
    store.interaction.ctxMenu = { targetSliceId: 'sl1' };

    expect(CtxActions.apply(store, 'add-command')).toBe(true);

    const commands = childrenOf(store, 'sl1', 'command');
    expect(commands).toHaveLength(1);
    expect(commands[0].label).toBe('new-command');
  });

  it('adds an event under the slice the menu was opened over', () => {
    const store = withSlice();
    store.interaction.ctxMenu = { targetSliceId: 'sl1' };

    CtxActions.apply(store, 'add-event');

    expect(childrenOf(store, 'sl1', 'event')[0].label).toBe('new-event');
  });

  it('numbers each added node against its existing siblings', () => {
    const store = withSlice();
    store.interaction.ctxMenu = { targetSliceId: 'sl1' };

    CtxActions.apply(store, 'add-command');
    CtxActions.apply(store, 'add-command');
    CtxActions.apply(store, 'add-command');

    expect(childrenOf(store, 'sl1', 'command').map(function(n) { return n.label; }))
      .toEqual(['new-command', 'new-command-2', 'new-command-3']);
  });

  it('counts only siblings of the same type when numbering', () => {
    const store = withSlice();
    store.interaction.ctxMenu = { targetSliceId: 'sl1' };

    CtxActions.apply(store, 'add-command');
    CtxActions.apply(store, 'add-event');

    expect(childrenOf(store, 'sl1', 'event')[0].label).toBe('new-event');
  });

  it('gives every added node a distinct id', () => {
    const store = withSlice();
    store.interaction.ctxMenu = { targetSliceId: 'sl1' };

    CtxActions.apply(store, 'add-command');
    CtxActions.apply(store, 'add-event');

    const ids = store.nodes.map(function(n) { return n.id; });
    expect(new Set(ids).size).toBe(ids.length);
  });

  it('adds a slice under the aggregate the menu was opened over', () => {
    const store = withSlice();
    store.interaction.ctxMenu = { targetAggId: 'agg1', targetCtxId: 'ctx1' };

    CtxActions.apply(store, 'add-slice');

    expect(childrenOf(store, 'agg1', 'slice')).toHaveLength(2);
  });

  it('adds a slice under the context when the menu was opened over no aggregate', () => {
    const store = withSlice();
    store.interaction.ctxMenu = { targetCtxId: 'ctx1' };

    CtxActions.apply(store, 'add-slice');

    expect(childrenOf(store, 'ctx1', 'slice')).toHaveLength(1);
  });

  it('reports no change when the menu names no parent to add into', () => {
    const store = withSlice();
    store.interaction.ctxMenu = {};

    expect(CtxActions.apply(store, 'add-slice')).toBe(false);
    expect(CtxActions.apply(store, 'add-command')).toBe(false);
    expect(store.nodes).toHaveLength(3);
  });
});

describe('CtxActions.apply — add-flow', () => {
  it('connects a new event to the slice\'s last existing command', () => {
    const store = withSlice();
    store.nodes.push({ id: 'cmd1', type: 'command', label: 'First', parentId: 'sl1' });
    store.nodes.push({ id: 'cmd2', type: 'command', label: 'Last', parentId: 'sl1' });
    store.interaction.ctxMenu = { targetSliceId: 'sl1' };

    expect(CtxActions.apply(store, 'add-flow')).toBe(true);

    const event = childrenOf(store, 'sl1', 'event')[0];
    expect(store.edges).toEqual([{ source: 'cmd2', target: event.id, type: 'flow' }]);
  });

  it('creates a command too when the slice has none', () => {
    const store = withSlice();
    store.interaction.ctxMenu = { targetSliceId: 'sl1' };

    CtxActions.apply(store, 'add-flow');

    const commands = childrenOf(store, 'sl1', 'command');
    const event = childrenOf(store, 'sl1', 'event')[0];
    expect(commands).toHaveLength(1);
    expect(commands[0].label).toBe('new-command');
    expect(store.edges).toEqual([{ source: commands[0].id, target: event.id, type: 'flow' }]);
  });

  it('reports no change when the menu names no slice', () => {
    const store = withSlice();
    store.interaction.ctxMenu = {};

    expect(CtxActions.apply(store, 'add-flow')).toBe(false);
    expect(store.edges).toHaveLength(0);
  });
});

describe('CtxActions.apply — delete-arrow', () => {
  it('removes the edge the menu was opened over', () => {
    const store = withSlice();
    store.edges = [
      { source: 'cmd1', target: 'evt1', type: 'flow' },
      { source: 'cmd2', target: 'evt2', type: 'flow' },
    ];
    store.interaction.ctxMenu = { edgeSource: 'cmd1', edgeTarget: 'evt1' };

    expect(CtxActions.apply(store, 'delete-arrow')).toBe(true);
    expect(store.edges).toEqual([{ source: 'cmd2', target: 'evt2', type: 'flow' }]);
  });

  it('reports no change when the menu was not opened over an edge', () => {
    const store = withSlice();
    store.edges = [{ source: 'cmd1', target: 'evt1', type: 'flow' }];
    store.interaction.ctxMenu = { targetSliceId: 'sl1' };

    expect(CtxActions.apply(store, 'delete-arrow')).toBe(false);
    expect(store.edges).toHaveLength(1);
  });
});

describe('CtxActions.apply — reordering slices', () => {
  function withThreeSlices() {
    return createStore([
      { id: 'ctx1', type: 'context', label: 'C' },
      { id: 'agg1', type: 'aggregate', label: 'A', parentId: 'ctx1' },
      { id: 'sl1', type: 'slice', label: 'S1', parentId: 'agg1' },
      { id: 'sl2', type: 'slice', label: 'S2', parentId: 'agg1' },
      { id: 'sl3', type: 'slice', label: 'S3', parentId: 'agg1' },
    ]);
  }

  function sliceOrder(store) {
    return store.nodes.filter(function(n) { return n.type === 'slice'; })
      .map(function(n) { return n.label; });
  }

  it('swaps a slice with the one before it', () => {
    const store = withThreeSlices();
    store.interaction.ctxMenu = { targetSliceId: 'sl2' };

    expect(CtxActions.apply(store, 'move-slice-left')).toBe(true);
    expect(sliceOrder(store)).toEqual(['S2', 'S1', 'S3']);
  });

  it('swaps a slice with the one after it', () => {
    const store = withThreeSlices();
    store.interaction.ctxMenu = { targetSliceId: 'sl2' };

    expect(CtxActions.apply(store, 'move-slice-right')).toBe(true);
    expect(sliceOrder(store)).toEqual(['S1', 'S3', 'S2']);
  });

  it('reports no change moving the first slice further left', () => {
    const store = withThreeSlices();
    store.interaction.ctxMenu = { targetSliceId: 'sl1' };

    expect(CtxActions.apply(store, 'move-slice-left')).toBe(false);
    expect(sliceOrder(store)).toEqual(['S1', 'S2', 'S3']);
  });

  it('reports no change moving the last slice further right', () => {
    const store = withThreeSlices();
    store.interaction.ctxMenu = { targetSliceId: 'sl3' };

    expect(CtxActions.apply(store, 'move-slice-right')).toBe(false);
    expect(sliceOrder(store)).toEqual(['S1', 'S2', 'S3']);
  });

  it('reports no change for a slice id that is not in the model', () => {
    const store = withThreeSlices();
    store.interaction.ctxMenu = { targetSliceId: 'gone' };

    expect(CtxActions.apply(store, 'move-slice-left')).toBe(false);
    expect(sliceOrder(store)).toEqual(['S1', 'S2', 'S3']);
  });
});

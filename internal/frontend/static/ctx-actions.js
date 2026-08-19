import { Model } from './model.js';

// Each entry describes a node the context menu can add: the node type, the
// prefix its generated ID uses, and where the menu records the parent it was
// opened over. Labels are numbered against existing siblings of the same type.
const NODE_SPECS = {
  "add-slice":   { type: "slice",   idPrefix: "slice", parentOf: function(m) { return m.targetAggId || m.targetCtxId; } },
  "add-command": { type: "command", idPrefix: "cmd",   parentOf: function(m) { return m.targetSliceId; } },
  "add-event":   { type: "event",   idPrefix: "evt",   parentOf: function(m) { return m.targetSliceId; } },
};

function childrenOf(store, parentId, type) {
  return store.nodes.filter(function(n) {
    return n.parentId === parentId && n.type === type;
  });
}

function addNode(store, spec, parentId) {
  const node = {
    id: Model.generateNodeId(spec.idPrefix, store),
    type: spec.type,
    label: Model.generateLabel(spec.type, childrenOf(store, parentId, spec.type)),
    parentId: parentId,
  };
  store.nodes.push(node);
  return node;
}

// A flow needs both ends, so it reuses the last command in the slice and only
// creates one when the slice has none.
function addFlow(store, sliceId) {
  if (!sliceId) return false;

  const commands = childrenOf(store, sliceId, "command");
  const event = addNode(store, NODE_SPECS["add-event"], sliceId);
  const source = commands.length > 0
    ? commands[commands.length - 1]
    : addNode(store, NODE_SPECS["add-command"], sliceId);

  Model.addEdge(store, source.id, event.id, "flow");
  return true;
}

function deleteArrow(store, menu) {
  if (!menu.edgeSource) return false;
  Model.removeEdge(store, menu.edgeSource, menu.edgeTarget);
  return true;
}

function moveSliceBy(store, sliceId, offset) {
  if (!sliceId) return false;
  const slice = store.nodeById.get(sliceId);
  if (!slice) return false;

  const siblings = childrenOf(store, slice.parentId, "slice");
  const currentPos = siblings.findIndex(function(n) { return n.id === sliceId; });
  if (currentPos === -1) return false;

  const targetPos = currentPos + offset;
  if (targetPos < 0 || targetPos >= siblings.length) return false;

  return Model.moveSlice(store.nodes, sliceId, targetPos);
}

// apply runs the action the clicked menu item names against the node the menu
// was opened over, and reports whether the model actually changed. Callers use
// that to decide whether to dismiss the menu and re-render.
function apply(store, action) {
  const menu = store.interaction.ctxMenu;
  if (!menu) return false;

  const spec = NODE_SPECS[action];
  if (spec) {
    const parentId = spec.parentOf(menu);
    if (!parentId) return false;
    addNode(store, spec, parentId);
    return true;
  }

  switch (action) {
    case "add-flow":         return addFlow(store, menu.targetSliceId);
    case "delete-arrow":     return deleteArrow(store, menu);
    case "move-slice-left":  return moveSliceBy(store, menu.targetSliceId, -1);
    case "move-slice-right": return moveSliceBy(store, menu.targetSliceId, 1);
    default:                 return false;
  }
}

export const CtxActions = {
  apply,
};

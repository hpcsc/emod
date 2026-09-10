import { bus } from './bus.js';

function rebuildNodeIndex(store) {
  store.nodeById = new Map(store.nodes.map(function(n) { return [n.id, n]; }));
}

function generateNodeId(prefix, store) {
  let id;
  do {
    id = "_" + prefix + "_" + Date.now() + "_" + Math.random().toString(36).substr(2, 4);
  } while (store.nodeById.has(id));
  return id;
}

function generateLabel(prefix, existing) {
  return existing.length === 0 ? "new-" + prefix : "new-" + prefix + "-" + (existing.length + 1);
}

function setModelData(store, data) {
  loadModel(store, data, {}, {});
}

// The same model parsed again. Ids number each kind in document order, so an
// edit ahead of a node renumbers it: what the user arranged follows each node by
// what the source calls it instead.
function updateModelData(store, data) {
  const offsets = byIdentity(store.nodes, store.nodeOffsets);
  const hidden = byIdentity(store.nodes, store.hiddenNodes);
  const nodes = data.nodes || [];

  loadModel(store, data, byNodeId(nodes, offsets), byNodeId(nodes, hidden));
}

function loadModel(store, data, nodeOffsets, hiddenNodes) {
  store.modelName = data.model_name || "";
  store.nodes = data.nodes || [];
  store.edges = data.edges || [];
  store.layoutPositions = {};
  store.nodeOffsets = nodeOffsets;
  store.hiddenNodes = hiddenNodes;
  store.arrowData = [];
  rebuildNodeIndex(store);

  bus.emit('model:updated', { store });
  bus.emit('data:changed', { store });
}

function byIdentity(nodes, valuesByNodeId) {
  const identities = nodeIdentities(nodes);
  const values = new Map();
  nodes.forEach(function(n) {
    if (Object.prototype.hasOwnProperty.call(valuesByNodeId, n.id)) {
      values.set(identities.get(n.id), valuesByNodeId[n.id]);
    }
  });
  return values;
}

function byNodeId(nodes, valuesByIdentity) {
  const identities = nodeIdentities(nodes);
  const values = {};
  nodes.forEach(function(n) {
    const identity = identities.get(n.id);
    if (valuesByIdentity.has(identity)) {
      values[n.id] = valuesByIdentity.get(identity);
    }
  });
  return values;
}

function nodeIdentities(nodes) {
  const byId = new Map(nodes.map(function(n) { return [n.id, n]; }));
  const identities = new Map();
  const alike = new Map();

  function identityOf(node) {
    if (identities.has(node.id)) return identities.get(node.id);
    identities.set(node.id, "");
    const parent = byId.get(node.parentId);
    const named = (parent ? identityOf(parent) : "") + "\n" + node.type + "\t" + node.label;
    const count = (alike.get(named) || 0) + 1;
    alike.set(named, count);
    const identity = named + "\t" + count;
    identities.set(node.id, identity);
    return identity;
  }

  nodes.forEach(identityOf);
  return identities;
}

function moveSlice(nodes, sliceId, targetPos) {
  var sliceNode = null;
  for (var i = 0; i < nodes.length; i++) {
    if (nodes[i].id === sliceId) { sliceNode = nodes[i]; break; }
  }
  if (!sliceNode || sliceNode.type !== "slice") return false;

  var aggId = sliceNode.parentId;
  var sliceIndices = [];
  for (var j = 0; j < nodes.length; j++) {
    if (nodes[j].parentId === aggId && nodes[j].type === "slice") {
      sliceIndices.push(j);
    }
  }

  var currentPos = -1;
  for (var k = 0; k < sliceIndices.length; k++) {
    if (nodes[sliceIndices[k]].id === sliceId) { currentPos = k; break; }
  }
  if (currentPos === -1) return false;
  if (targetPos < 0) targetPos = 0;
  if (targetPos >= sliceIndices.length) targetPos = sliceIndices.length - 1;
  if (targetPos === currentPos) return false;

  if (currentPos < targetPos) {
    for (var p = currentPos; p < targetPos; p++) {
      var a = sliceIndices[p];
      var b = sliceIndices[p + 1];
      var temp = nodes[a];
      nodes[a] = nodes[b];
      nodes[b] = temp;
    }
  } else {
    for (var p = currentPos; p > targetPos; p--) {
      var a = sliceIndices[p];
      var b = sliceIndices[p - 1];
      var temp = nodes[a];
      nodes[a] = nodes[b];
      nodes[b] = temp;
    }
  }
  return true;
}

function sendParse(store, source, statusEl, filename) {
  if (!source || !source.trim()) {
    statusEl.textContent = "✗ Paste some .emod content first";
    statusEl.className = "status error";
    return Promise.reject(new Error("no source"));
  }
  statusEl.textContent = "⏳ Parsing...";
  statusEl.className = "";

  // Detect input format: try JSON, check for known shapes
  var parsed;
  try {
    parsed = JSON.parse(source);
    // Valid JSON — check for well-known formats
    if (Array.isArray(parsed.nodes)) {
      // Diagram-oriented JSON — use directly
      return Promise.resolve({ diagnostics: parsed.diagnostics || [], diagram: parsed });
    }
    if (parsed.model) {
      // Raw AST JSON — use directly
      return Promise.resolve({ diagnostics: parsed.diagnostics || [], diagram: parsed });
    }
  } catch (e) {
    // Not valid JSON — treat as raw .emod source below
  }

  // Raw .emod source. Reached through a dynamic import so this module can be
  // loaded without the platform — a static import here would make every
  // consumer of model.js pull the host implementation in too. It defers no
  // work in the app itself: viewer.js imports the platform statically.
  return import('./platform.js').then(function(platform) {
    return platform.ready.then(function() {
      return platform.parseEmod(source, filename);
    });
  }).then(function(data) {
    if (data.error) {
      throw new Error(data.error);
    }
    if (!data.diagram || !data.diagram.nodes || !data.diagram.edges) {
      throw new Error('invalid diagram response');
    }
    return data;
  });
}

function addEdge(store, source, target, type) {
  store.edges.push({ source: source, target: target, type: type });
}

function removeEdge(store, source, target) {
  var found = -1;
  for (var i = 0; i < store.edges.length; i++) {
    var e = store.edges[i];
    if (e.source === source && e.target === target) {
      found = i;
      break;
    }
  }
  if (found !== -1) {
    store.edges.splice(found, 1);
  }
}

// The direction each edge type runs in, keyed source>target. These must match
// the directions the exporter writes, because that is what the importer reads
// back: a subscription runs event to view, so an arrow drawn view to event is
// not a subscription and would be dropped on export.
var EDGE_TYPE_BY_ENDS = {
  "command>event": "flow",
  "trigger>command": "trigger_command",
  "event>view": "subscription",
  "event>automation": "automation_trigger",
  "automation>command": "automation_command",
  "view>translation": "reads",
  "view>trigger": "reads",
  "view>automation": "reads",
  "translation>command": "translation_command",
};

function autoDetectEdgeType(store, sourceId, targetId) {
  var src = store.nodeById.get(sourceId);
  var tgt = store.nodeById.get(targetId);
  if (!src || !tgt) return "flow";

  return EDGE_TYPE_BY_ENDS[src.type + ">" + tgt.type] || "flow";
}

export const Model = {
  rebuildNodeIndex,
  generateNodeId,
  generateLabel,
  setModelData,
  updateModelData,
  sendParse,
  moveSlice,
  addEdge,
  removeEdge,
  autoDetectEdgeType,
};

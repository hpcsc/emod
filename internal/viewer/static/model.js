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
  store.modelName = data.model_name || "";
  store.nodes = data.nodes || [];
  store.edges = data.edges || [];
  store.layoutPositions = {};
  store.nodeOffsets = {};
  store.hiddenContexts = {};
  store.arrowData = [];
  rebuildNodeIndex(store);

  bus.emit('model:updated', { store });
  bus.emit('data:changed', { store });
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

function sendParse(store, source, statusEl) {
  if (!source) {
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

  // Raw .emod source — parse via WASM (dynamic import to defer init side effects)
  return import('./wasm.js').then(function(wasm) {
    return wasm.ready.then(function() {
      return wasm.parseEmod(source);
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
  sendParse,
  moveSlice,
  addEdge,
  removeEdge,
  autoDetectEdgeType,
};

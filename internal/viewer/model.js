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

  return fetch("/parse", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ source: source }),
  })
  .then(function(resp) {
    if (!resp.ok) {
      return resp.json().then(function(err) { throw new Error(err.error || "parse failed"); });
    }
    return resp.json();
  })
  .then(function(data) {
    if (!data.diagram || !data.diagram.nodes || !data.diagram.edges) {
      throw new Error("invalid diagram response from server");
    }
    return data;
  });
}

export const Model = {
  rebuildNodeIndex,
  generateNodeId,
  generateLabel,
  setModelData,
  sendParse,
  moveSlice,
};

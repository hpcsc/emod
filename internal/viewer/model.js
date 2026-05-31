import { bus } from './bus.js';

function generateNodeId(prefix, store) {
  let id;
  do {
    id = "_" + prefix + "_" + Date.now() + "_" + Math.random().toString(36).substr(2, 4);
  } while (findNodeById(store, id));
  return id;
}

function generateLabel(prefix, existing) {
  return existing.length === 0 ? "new-" + prefix : "new-" + prefix + "-" + (existing.length + 1);
}

function findNodeById(store, id) {
  for (let i = 0; i < store.nodes.length; i++) {
    if (store.nodes[i].id === id) return store.nodes[i];
  }
  return null;
}

function setModelData(store, data) {
  store.modelName = data.model_name || "";
  store.nodes = data.nodes || [];
  store.edges = data.edges || [];
  store.layoutPositions = {};
  store.nodeOffsets = {};
  store.hiddenContexts = {};
  store.arrowData = [];

  bus.emit('model:updated', { store });
  bus.emit('data:changed', { store });
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
  generateNodeId,
  generateLabel,
  setModelData,
  sendParse,
};

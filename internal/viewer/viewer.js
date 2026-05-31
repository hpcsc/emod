import { createStore } from './store.js';
import { Layout } from './layout.js';
import { Renderer } from './renderer.js';
import { Interaction } from './interaction.js';
import { UI } from './ui.js';
import { Model } from './model.js';
import { bus } from './bus.js';

// ─── Create store and wire DOM references ────────────────────────────
const store = createStore();
store.dom.svg = document.getElementById("diagram-canvas");
store.dom.nameDisplay = document.getElementById("model-name-display");
store.dom.sourceInput = document.getElementById("source-input");
store.dom.renderBtn = document.getElementById("render-btn");
store.dom.statusEl = document.getElementById("render-status");
store.dom.statNodes = document.getElementById("stat-nodes");
store.dom.statEdges = document.getElementById("stat-edges");
store.dom.statCanvas = document.getElementById("stat-canvas");
store.dom.panel = document.getElementById("data-panel");
store.dom.panelHdr = document.getElementById("data-panel-header");
store.dom.panelBody = document.getElementById("data-panel-body");
store.dom.minimap = document.getElementById("minimap");
store.dom.minimapSvg = document.getElementById("minimap-svg");
store.dom.minimapToggle = document.getElementById("minimap-toggle");
store.dom.contextPanel = document.getElementById("context-panel");
store.dom.contextToggle = document.getElementById("context-toggle");
store.dom.contextList = document.getElementById("context-list");
store.dom.tooltip = document.getElementById("tooltip");
store.dom.detailPanel = document.getElementById("detail-panel");
store.dom.dpContent = document.getElementById("dp-content");
store.dom.ctxMenu = document.getElementById("ctx-menu");
store.dom.actorAnnotations = document.getElementById("actor-annotations");
store.dom.resetLayoutBtn = document.getElementById("reset-layout");
store.dom.fitViewBtn = document.getElementById("fit-view");

// ─── Event subscriptions ─────────────────────────────────────────────
bus.on('data:changed', function({ store: s }) {
  renderDiagram(s);
});

bus.on('model:updated', function({ store: s }) {
  s.dom.nameDisplay.textContent = s.modelName || "(unnamed)";
  document.title = s.modelName ? s.modelName + " — Emod Diagram Viewer" : "Emod Diagram Viewer";
  UI.updateStats(s);
  const btn = s.dom.resetLayoutBtn;
  if (btn) btn.disabled = true;
  if (s.dom.contextPanel && !s.dom.contextPanel.classList.contains("hidden")) {
    UI.updateContextList(s);
  }
});

bus.on('viewport:changed', function({ store: s }) {
  UI.updateMinimap(s);
});

bus.on('node:delete', function({ store: s, nodeId }) {
  for (let i = 0; i < s.nodes.length; i++) {
    if (s.nodes[i].id === nodeId) {
      s.nodes.splice(i, 1);
      break;
    }
  }
  s.edges = s.edges.filter(function(e) {
    return e.source !== nodeId && e.target !== nodeId;
  });
  delete s.nodeOffsets[nodeId];
  delete s.layoutPositions[nodeId];
  Model.rebuildNodeIndex(s);
  UI.hideDetailPanel(s);
  const hasOffsets = Object.keys(s.nodeOffsets).length > 0;
  if (s.dom.resetLayoutBtn) s.dom.resetLayoutBtn.disabled = !hasOffsets;
  renderDiagram(s);
});

// ─── Render orchestration ────────────────────────────────────────────
function renderDiagram(s) {
  Model.rebuildNodeIndex(s);
  bus.emit('diagram:before-render', { store: s });

  const result = Layout.computeLayout(s);
  s.layoutPositions = result.positions;

  for (const id in s.nodeOffsets) {
    if (s.layoutPositions[id]) {
      s.layoutPositions[id].x += s.nodeOffsets[id].dx;
      s.layoutPositions[id].y += s.nodeOffsets[id].dy;
    }
  }
  s.nodeOffsets = {};

  s.dom.svg.setAttribute("viewBox", "0 0 " + result.width + " " + result.height);
  UI.updateStats(s, result);

  const html = Renderer.buildSVG(s);
  Renderer.inject(s.dom.svg, html);

  bus.emit('diagram:rendered', { store: s, dims: result });
}

// ─── Before-render subscriber ───────────────────────────────────────
bus.on('diagram:before-render', function({ store: s }) {
  s.interaction.selectedNodeId = null;
  s.interaction.highlighted = {};
  s.interaction.drag = null;
  const dp = s.dom.detailPanel;
  if (dp) dp.style.display = "none";
  s.dom.svg.classList.remove("has-highlights");
  Renderer.clearSVG(s.dom.svg);
});

// ─── After-render subscriber ────────────────────────────────────────
bus.on('diagram:rendered', function({ store: s }) {
  Interaction.applyViewport(s);
  UI.renderActorAnnotations(s);
});

// ─── Init ───────────────────────────────────────────────────────────
function init() {
  Interaction.initEventListeners(store);
  UI.initDelegation(store);
  UI.initKeyboard(store);

  // ─── Render button click ──────────────────────────────────────────
  store.dom.renderBtn.addEventListener("click", function() {
    const source = store.dom.sourceInput.value.trim();
    Model.sendParse(store, source, store.dom.statusEl)
      .then(function(data) {
        Model.setModelData(store, data.diagram);
        store.dom.panel.classList.add("collapsed");
        store.dom.statusEl.textContent = "✓ Rendered";
        store.dom.statusEl.className = "status success";
        const btn = store.dom.resetLayoutBtn;
        if (btn) btn.disabled = true;
      })
      .catch(function(err) {
        store.dom.statusEl.textContent = "✗ " + err.message;
        store.dom.statusEl.className = "status error";
      });
  });

  // ─── File drag-and-drop ───────────────────────────────────────────
  store.dom.panelBody.addEventListener("dragenter", function(evt) {
    evt.preventDefault();
    evt.stopPropagation();
    store.dom.panelBody.classList.add("drag-over");
  });
  store.dom.panelBody.addEventListener("dragover", function(evt) {
    evt.preventDefault();
    evt.stopPropagation();
  });
  store.dom.panelBody.addEventListener("dragleave", function(evt) {
    evt.preventDefault();
    evt.stopPropagation();
    store.dom.panelBody.classList.remove("drag-over");
  });
  store.dom.panelBody.addEventListener("drop", function(evt) {
    evt.preventDefault();
    evt.stopPropagation();
    store.dom.panelBody.classList.remove("drag-over");
    const file = evt.dataTransfer.files[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = function(e) {
      store.dom.sourceInput.value = e.target.result;
      store.dom.renderBtn.click();
    };
    reader.readAsText(file);
  });

  // ─── Panel toggle ─────────────────────────────────────────────────
  store.dom.panelHdr.addEventListener("click", function() {
    store.dom.panel.classList.toggle("collapsed");
  });

  // ─── Reset layout button ──────────────────────────────────────────
  store.dom.resetLayoutBtn.addEventListener("click", function() {
    store.nodeOffsets = {};
    renderDiagram(store);
    store.dom.resetLayoutBtn.disabled = true;
  });

  // ─── Fit-to-view button ───────────────────────────────────────────
  store.dom.fitViewBtn.addEventListener("click", function() {
    Interaction.fitToView(store);
  });

  // ─── Minimap toggle ───────────────────────────────────────────────
  store.dom.minimapToggle.addEventListener("click", function() {
    UI.toggleMinimap(store);
  });

  // ─── Context toggle ───────────────────────────────────────────────
  store.dom.contextToggle.addEventListener("click", function() {
    UI.toggleContextPanel(store);
  });

  // ─── Minimap event listeners ──────────────────────────────────────
  let minimapDragPos = false;
  let minimapNavDrag = false;
  const minimapHandle = document.getElementById("minimap-handle");

  minimapHandle.addEventListener("mousedown", function(evt) {
    evt.preventDefault();
    evt.stopPropagation();
    minimapDragPos = true;
    let tx = 0, ty = 0;
    const mat = store.dom.minimap.style.transform;
    if (mat && mat.startsWith("translate(")) {
      const parts = mat.match(/translate\(([\d.-]+)px,\s*([\d.-]+)px\)/);
      if (parts) { tx = parseFloat(parts[1]); ty = parseFloat(parts[2]); }
    }
    store.dom.minimap.dataset.dragOffX = evt.clientX - tx;
    store.dom.minimap.dataset.dragOffY = evt.clientY - ty;
  });

  store.dom.minimap.addEventListener("mousedown", function(evt) {
    if (evt.target.closest("#minimap-close, #minimap-toggle, #minimap-handle, #context-panel, #context-toggle, #context-list")) return;
    evt.preventDefault();
    evt.stopPropagation();
    UI.minimapNavigate(store, evt);
    minimapNavDrag = true;
  });

  document.addEventListener("mousemove", function(evt) {
    if (minimapDragPos) {
      const tx = evt.clientX - parseFloat(store.dom.minimap.dataset.dragOffX);
      const ty = evt.clientY - parseFloat(store.dom.minimap.dataset.dragOffY);
      store.dom.minimap.style.transform = "translate(" + tx + "px, " + ty + "px)";
      return;
    }
    if (minimapNavDrag) {
      UI.minimapNavigate(store, evt);
      evt.preventDefault();
    }
  });

  document.addEventListener("mouseup", function() {
    minimapDragPos = false;
    minimapNavDrag = false;
  });

  minimapHandle.addEventListener("touchstart", function(evt) {
    evt.preventDefault();
    evt.stopPropagation();
    minimapDragPos = true;
    const touch = evt.touches[0];
    let tx = 0, ty = 0;
    const mat = store.dom.minimap.style.transform;
    if (mat && mat.startsWith("translate(")) {
      const parts = mat.match(/translate\(([\d.-]+)px,\s*([\d.-]+)px\)/);
      if (parts) { tx = parseFloat(parts[1]); ty = parseFloat(parts[2]); }
    }
    store.dom.minimap.dataset.dragOffX = touch.clientX - tx;
    store.dom.minimap.dataset.dragOffY = touch.clientY - ty;
  }, { passive: false });

  store.dom.minimap.addEventListener("touchstart", function(evt) {
    if (evt.target.closest("#minimap-close, #minimap-toggle, #minimap-handle, #context-panel, #context-toggle, #context-list")) return;
    evt.preventDefault();
    evt.stopPropagation();
    const touch = evt.touches[0];
    UI.minimapNavigate(store, touch);
    minimapNavDrag = true;
  }, { passive: false });

  store.dom.minimap.addEventListener("touchmove", function(evt) {
    if (minimapDragPos) {
      const touch = evt.touches[0];
      const tx = touch.clientX - parseFloat(store.dom.minimap.dataset.dragOffX);
      const ty = touch.clientY - parseFloat(store.dom.minimap.dataset.dragOffY);
      store.dom.minimap.style.transform = "translate(" + tx + "px, " + ty + "px)";
      return;
    }
    if (minimapNavDrag) {
      evt.preventDefault();
      const touch = evt.touches[0];
      UI.minimapNavigate(store, touch);
    }
  }, { passive: false });

  store.dom.minimap.addEventListener("touchend", function() {
    minimapDragPos = false;
    minimapNavDrag = false;
  });

  document.getElementById("minimap-close").addEventListener("click", function(evt) {
    evt.stopPropagation();
    UI.toggleMinimap(store, false);
  });

  // ─── Context menu item clicks ────────────────────────────────────
  store.dom.ctxMenu.addEventListener("click", function(evt) {
    const item = evt.target.closest(".ctx-menu-item");
    if (!item) return;
    const action = item.getAttribute("data-action");

    if (action === "add-slice") {
      if (!store.interaction.ctxMenu || !store.interaction.ctxMenu.targetAggId) return;
      const aggId = store.interaction.ctxMenu.targetAggId;
      const sliceId = Model.generateNodeId("slice", store);
      const existingSlices = store.nodes.filter(function(n) {
        return n.parentId === aggId && n.type === "slice";
      });
      store.nodes.push({
        id: sliceId,
        type: "slice",
        label: Model.generateLabel("slice", existingSlices),
        parentId: aggId,
      });
      UI.hideContextMenu(store);
      bus.emit('data:changed', { store });
      return;
    }

    if (action === "add-command") {
      if (!store.interaction.ctxMenu || !store.interaction.ctxMenu.targetSliceId) return;
      const sliceId = store.interaction.ctxMenu.targetSliceId;
      const cmdId = Model.generateNodeId("cmd", store);
      const existingCmds = store.nodes.filter(function(n) {
        return n.parentId === sliceId && n.type === "command";
      });
      store.nodes.push({
        id: cmdId,
        type: "command",
        label: Model.generateLabel("command", existingCmds),
        parentId: sliceId,
      });
      UI.hideContextMenu(store);
      bus.emit('data:changed', { store });
      return;
    }

    if (action === "add-event") {
      if (!store.interaction.ctxMenu || !store.interaction.ctxMenu.targetSliceId) return;
      const sliceId = store.interaction.ctxMenu.targetSliceId;
      const evtId = Model.generateNodeId("evt", store);
      const existingEvts = store.nodes.filter(function(n) {
        return n.parentId === sliceId && n.type === "event";
      });
      store.nodes.push({
        id: evtId,
        type: "event",
        label: Model.generateLabel("event", existingEvts),
        parentId: sliceId,
      });
      UI.hideContextMenu(store);
      bus.emit('data:changed', { store });
      return;
    }

    if (action === "add-flow") {
      if (!store.interaction.ctxMenu || !store.interaction.ctxMenu.targetSliceId) return;
      const sliceId = store.interaction.ctxMenu.targetSliceId;
      const cmds = store.nodes.filter(function(n) {
        return n.parentId === sliceId && n.type === "command";
      });
      const evtId = Model.generateNodeId("evt", store);
      const existingEvts = store.nodes.filter(function(n) {
        return n.parentId === sliceId && n.type === "event";
      });
      store.nodes.push({
        id: evtId,
        type: "event",
        label: Model.generateLabel("event", existingEvts),
        parentId: sliceId,
      });
      let sourceId;
      if (cmds.length > 0) {
        sourceId = cmds[cmds.length - 1].id;
      } else {
        sourceId = Model.generateNodeId("cmd", store);
        store.nodes.push({
          id: sourceId,
          type: "command",
          label: "new-command",
          parentId: sliceId,
        });
      }
      store.edges.push({
        source: sourceId,
        target: evtId,
        type: "flow",
      });
      UI.hideContextMenu(store);
      bus.emit('data:changed', { store });
      return;
    }

    if (action === "move-slice-left" || action === "move-slice-right") {
      if (!store.interaction.ctxMenu || !store.interaction.ctxMenu.targetSliceId) return;
      const moved = Model.moveSlice(store.nodes, store.interaction.ctxMenu.targetSliceId, action === "move-slice-left" ? "left" : "right");
      if (moved) {
        UI.hideContextMenu(store);
        bus.emit('data:changed', { store });
      }
      return;
    }
  });

  // ─── Dismiss context menu on outside click ──────────────────────
  document.addEventListener("click", function(evt) {
    const menu = store.dom.ctxMenu;
    if (menu && menu.style.display !== "none" && !menu.contains(evt.target)) {
      UI.hideContextMenu(store);
    }
  });

  // ─── Detail panel close button ───────────────────────────────────
  document.getElementById("dp-close").addEventListener("click", function(evt) {
    evt.stopPropagation();
    UI.hideDetailPanel(store);
  });

  // ─── Initial load ───────────────────────────────────────────────
  if (typeof INITIAL_DATA !== 'undefined' && INITIAL_DATA !== null) {
    const initData = INITIAL_DATA.diagram || INITIAL_DATA;
    Model.setModelData(store, initData);
  } else {
    store.dom.sourceInput.placeholder = 'Paste .emod content here';
    store.dom.panel.classList.remove('collapsed');
    store.dom.nameDisplay.textContent = '(no model)';
  }
}

init();

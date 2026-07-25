import { createStore } from './store.js';
import { Layout } from './layout.js';
import { Renderer } from './renderer.js';
import { Interaction } from './interaction.js';
import { UI } from './ui.js';
import { Minimap } from './minimap.js';
import { CtxActions } from './ctx-actions.js';
import { Model } from './model.js';
import { bus } from './bus.js';
import { Export } from './emod-export.js';
import { ready, isReady } from './wasm.js';

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

bus.on('diagnostics:changed', function({ store: s, diagnostics }) {
  UI.updateDiagnosticsPanel(s, diagnostics);
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
  store.dom.exportBtn = document.getElementById("export-emod");
  store.dom.diagnosticsBadge = document.getElementById("diagnostics-badge");
  store.dom.diagnosticsPanel = document.getElementById("diagnostics-panel");
  store.dom.diagnosticsList = document.getElementById("diagnostics-list");
  store.dom.diagnosticsClose = document.getElementById("diagnostics-close");

  Interaction.initEventListeners(store);
  UI.initDelegation(store);
  UI.initKeyboard(store);
  UI.initDiagnosticsDelegation(store);
  Minimap.initMinimap(store);

  // ─── Render button click ──────────────────────────────────────────
  store.dom.renderBtn.addEventListener("click", function() {
    const source = store.dom.sourceInput.value.trim();
    Model.sendParse(store, source, store.dom.statusEl)
      .then(function(data) {
        store.diagnostics = data.diagnostics || [];
        bus.emit('diagnostics:changed', { store, diagnostics: store.diagnostics });
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
    const name = file.name.toLowerCase();
    if (!name.endsWith('.emod') && !name.endsWith('.json')) {
      store.dom.statusEl.textContent = '✗ Only .emod and .json files are supported';
      store.dom.statusEl.className = 'status error';
      return;
    }
    const reader = new FileReader();
    reader.onload = function(e) {
      store.dom.sourceInput.value = e.target.result;
      store.dom.renderBtn.click();
    };
    reader.onerror = function() {
      store.dom.statusEl.textContent = '✗ Failed to read file';
      store.dom.statusEl.className = 'status error';
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

  // ─── Export .emod button ─────────────────────────────────────────
  store.dom.exportBtn.addEventListener("click", function() {
    Export.exportToEmodString(store).then(function(content) {
      var blob = new Blob([content], { type: "text/plain" });
      var url = URL.createObjectURL(blob);
      var a = document.createElement("a");
      a.href = url;
      a.download = (store.modelName || "diagram") + ".emod";
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    }).catch(function(err) {
      store.dom.statusEl.textContent = "✗ " + err.message;
      store.dom.statusEl.className = "status error";
    });
  });

  // ─── Context toggle ───────────────────────────────────────────────
  store.dom.contextToggle.addEventListener("click", function() {
    UI.toggleContextPanel(store);
  });

  // ─── Diagnostics toggle ──────────────────────────────────────────
  store.dom.diagnosticsBadge.addEventListener("click", function() {
    UI.toggleDiagnosticsPanel(store);
  });

  // ─── Diagnostics close ───────────────────────────────────────────
  store.dom.diagnosticsClose.addEventListener("click", function(evt) {
    evt.stopPropagation();
    UI.hideDiagnosticsPanel(store);
  });

  // ─── Context menu item clicks ────────────────────────────────────
  store.dom.ctxMenu.addEventListener("click", function(evt) {
    const item = evt.target.closest(".ctx-menu-item");
    if (!item) return;
    const action = item.getAttribute("data-action");

    if (!CtxActions.apply(store, action)) return;

    UI.hideContextMenu(store);
    // A deleted edge may still be the one the detail panel is showing, and
    // re-rendering alone does not clear the selection behind it.
    if (action === "delete-arrow") UI.hideDetailPanel(store);
    bus.emit('data:changed', { store });
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
    store.dom.sourceInput.placeholder = 'Paste .emod source or diagram JSON here';
    store.dom.panel.classList.remove('collapsed');
    store.dom.nameDisplay.textContent = '(no model)';

    const instructions = document.getElementById('landing-instructions');
    if (instructions) instructions.style.display = 'block';

    if (!isReady) {
      store.dom.statusEl.textContent = '⏳ Loading parser...';
      store.dom.statusEl.className = '';
    }

    ready.then(function() {
      if (store.dom.statusEl.textContent === '⏳ Loading parser...') {
        store.dom.statusEl.textContent = '✓ Ready';
        store.dom.statusEl.className = 'status success';
        setTimeout(function() {
          if (store.dom.statusEl.textContent === '✓ Ready') {
            store.dom.statusEl.textContent = '';
            store.dom.statusEl.className = 'status';
          }
        }, 1500);
      }
    }).catch(function(err) {
      if (store.dom.statusEl.textContent === '⏳ Loading parser...') {
        store.dom.statusEl.textContent = '✗ ' + (err.message || 'Parser failed to load');
        store.dom.statusEl.className = 'status error';
      }
    });
  }
}

export { init };

if (typeof process === 'undefined' || !process.env || !process.env.VITEST) {
  init();
}

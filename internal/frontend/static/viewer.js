import { createStore } from './store.js';
import { Layout } from './layout.js';
import { Renderer } from './renderer.js';
import { Interaction } from './interaction.js';
import { UI } from './ui.js';
import { Minimap } from './minimap.js';
import { Legend } from './legend.js';
import { CtxActions } from './ctx-actions.js';
import { Model } from './model.js';
import { bus } from './bus.js';
import { Export } from './emod-export.js';
import { ready, isReady, droppedFile, saveFile, setWindowTitle, setWindowModified, resolveUnsavedEdits, onFileOpened, onSaveRequested, initialState } from './platform.js';

// ─── Event subscriptions ─────────────────────────────────────────────
bus.on('data:changed', function({ store: s }) {
  renderDiagram(s);
});

// What Save writes is the source in the panel, never the model the exporter
// re-serialises from the diagram: that one canonicalises formatting and drops
// comments, so saving a file the user had not edited would rewrite it.
//
// The panel is a textarea, though, and a textarea normalises every CRLF it is
// handed to a bare LF — so its text is already not the file's bytes. The text
// the file arrived with is kept beside it: unedited source hands that back
// untouched, and edited source is put back into the same convention, so a file
// written with CRLF never comes back with every one of its lines rewritten. A
// file that mixes conventions is written wholly in the one it opened with, and
// only once it has been edited.
function sourceToSave(s) {
  const shown = s.dom.sourceInput.value;
  const arrived = s.currentFile && s.currentFile.content;
  if (typeof arrived !== 'string') {
    return shown;
  }
  if (shown === arrived.replace(/\r\n?/g, '\n')) {
    return arrived;
  }
  return arrived.indexOf('\r\n') === -1 ? shown : shown.replace(/\n/g, '\r\n');
}

// Unsaved work is what Save would write differing from what the file holds, so
// this asks sourceToSave rather than the panel: a textarea hands back LF for
// the CRLF a file arrived with, and comparing that against the file's own bytes
// would mark every CRLF model the instant it opened.
//
// A model with no file behind it is measured against nothing, so pasted source
// counts as unsaved until it has been written somewhere — which is what it is:
// Save asks it for a location, exactly as it does for a model that has never
// been saved.
function modelIsModified(s) {
  return sourceToSave(s) !== ((s.currentFile && s.currentFile.content) || '');
}

function applyWindowTitle(s) {
  const name = (s.currentFile && s.currentFile.name) || s.modelName;
  setWindowTitle(name ? name + " — Emod Diagram Viewer" : "Emod Diagram Viewer");
}

bus.on('model:updated', function({ store: s }) {
  s.dom.nameDisplay.textContent = s.modelName || "(unnamed)";
  applyWindowTitle(s);
  UI.updateStats(s);
  const btn = s.dom.resetLayoutBtn;
  if (btn) btn.disabled = true;
  if (s.dom.visibilityPanel && !s.dom.visibilityPanel.classList.contains("hidden")) {
    UI.updateVisibilityTree(s);
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
  store.dom.statFile = document.getElementById("stat-file");
  store.dom.statFilePath = document.getElementById("stat-file-path");
  store.dom.saveStatus = document.getElementById("save-status");
  store.dom.panel = document.getElementById("data-panel");
  store.dom.panelHdr = document.getElementById("data-panel-header");
  store.dom.panelBody = document.getElementById("data-panel-body");
  store.dom.minimap = document.getElementById("minimap");
  store.dom.minimapSvg = document.getElementById("minimap-svg");
  store.dom.minimapToggle = document.getElementById("minimap-toggle");
  store.dom.visibilityPanel = document.getElementById("visibility-panel");
  store.dom.visibilityToggle = document.getElementById("visibility-toggle");
  store.dom.visibilityTree = document.getElementById("visibility-tree");
  store.dom.legendPanel = document.getElementById("legend-panel");
  store.dom.legendToggle = document.getElementById("legend-toggle");
  store.dom.legendContent = document.getElementById("legend-content");
  store.dom.legendClose = document.getElementById("legend-close");
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

  // ─── Render ───────────────────────────────────────────────────────
  // Renders are numbered because a parse is genuinely concurrent on a host that
  // answers over RPC rather than in-process: without this, a slow earlier parse
  // landing after a fast later one repaints the canvas with the older model
  // while the panel shows the newer source.
  let latestRender = 0;

  // A save waits on this. The panel's text and the open file are committed at
  // different moments — the text the instant a file arrives, its identity only
  // once the parse resolves — so a save taken between the two would write the
  // arriving model's source over the departing model's path.
  let latestRenderSettled = Promise.resolve();

  // The panel's text and the render that reads it are claimed together, because
  // a drop and a host delivery both arrive asynchronously and either can land
  // mid-flight: writing the text outside this would let one entry point's source
  // be rendered under the other's identity.
  //
  // file names where the panel's source came from — an object to adopt, null to
  // forget the open file, undefined to leave it alone, which is what an ordinary
  // Render does. It is committed only once the parse resolves: a render the
  // parse rejects must leave the window naming the model still on screen, not
  // one that never appeared.
  function renderPanelSource(text, file) {
    clearSaveConfirmation();
    const previousText = store.dom.sourceInput.value;
    if (text !== undefined) {
      store.dom.sourceInput.value = text;
    }
    const source = store.dom.sourceInput.value.trim();
    const render = ++latestRender;
    latestRenderSettled = Model.sendParse(store, source, store.dom.statusEl)
      .then(function(data) {
        if (render !== latestRender) return;
        if (file !== undefined) {
          store.currentFile = file;
        }
        store.diagnostics = data.diagnostics || [];
        bus.emit('diagnostics:changed', { store, diagnostics: store.diagnostics });
        Model.setModelData(store, data.diagram);
        store.dom.panel.classList.add("collapsed");
        store.dom.statusEl.textContent = "✓ Rendered";
        store.dom.statusEl.className = "status success";
        const btn = store.dom.resetLayoutBtn;
        if (btn) btn.disabled = true;
        reportModified();
      })
      .catch(function(err) {
        if (render !== latestRender) return;
        // The panel's text was replaced for a model that never rendered, and the
        // window still names the one on screen — so putting it back is what keeps
        // the panel, the title and the path stat naming one model.
        if (text !== undefined) {
          store.dom.sourceInput.value = previousText;
        }
        store.dom.statusEl.textContent = "✗ " + err.message;
        store.dom.statusEl.className = "status error";
        reportModified();
      });

    return latestRenderSettled;
  }

  // Reporting a host failure claims a render number for the same reason a render
  // does: an older parse still in flight would otherwise resolve afterwards and
  // paint over the reason, re-collapsing the panel holding it.
  function reportHostFailure(reason) {
    latestRender++;
    store.dom.statusEl.textContent = "✗ " + reason;
    store.dom.statusEl.className = "status error";
    store.dom.panel.classList.remove("collapsed");
  }

  // Called only where the panel's text and the open file's identity are known
  // to describe one model: as the panel is typed into, once a render has
  // committed which file it belongs to, and after a save has settled. Asking
  // while a file is arriving would measure its source against the bytes of the
  // model it is replacing, because renderPanelSource writes the text
  // immediately and commits the identity only when the parse resolves.
  function reportModified() {
    setWindowModified(modelIsModified(store));
  }

  store.dom.sourceInput.addEventListener("input", reportModified);

  // The only place a different file's model is handed to renderPanelSource, so
  // every way of opening one runs the guard below and a new one has nowhere
  // else to arrive.
  function openModel(text, file) {
    return clearedToReplace().then(function(cleared) {
      return cleared ? renderPanelSource(text, file) : undefined;
    });
  }

  // Save writes the file the edits belong to before the model on screen is
  // replaced, and the answer to whether that write landed is the marker: a
  // refused save, and one whose location dialog was cancelled, both leave the
  // model modified, and neither may cost the user the edits they asked to keep.
  function clearedToReplace() {
    if (!modelIsModified(store)) {
      return Promise.resolve(true);
    }
    return resolveUnsavedEdits().then(function(outcome) {
      if (outcome === 'cancel') {
        return false;
      }
      if (outcome === 'discard') {
        return true;
      }
      return saveModel({ chooseLocation: false }).then(function() {
        return !modelIsModified(store);
      });
    });
  }

  store.dom.renderBtn.addEventListener("click", function() {
    renderPanelSource();
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
    const file = droppedFile(evt.dataTransfer);
    if (!file) return;
    const name = file.name.toLowerCase();
    if (!name.endsWith('.emod') && !name.endsWith('.json')) {
      store.dom.statusEl.textContent = '✗ Only .emod and .json files are supported';
      store.dom.statusEl.className = 'status error';
      return;
    }
    file.read().then(function(content) {
      openModel(content, null);
    }).catch(function() {
      store.dom.statusEl.textContent = '✗ Failed to read file';
      store.dom.statusEl.className = 'status error';
    });
  });

  // ─── A file the host opened ───────────────────────────────────────
  onFileOpened(function(opened) {
    // The status area lives in the panel a successful render collapses, and an
    // open request comes from outside the page — so every failure here has to
    // reveal that panel, or choosing a file from the OS dialog appears to do
    // nothing at all.
    if (opened.error) {
      reportHostFailure(opened.error);
      return;
    }
    // A file with nothing in it reads successfully and then reaches the parser's
    // own empty-source rejection, whose message is written for someone who
    // pressed Render on an empty panel rather than someone who chose a file.
    if (!opened.content.trim()) {
      reportHostFailure(opened.name + " is empty");
      return;
    }
    openModel(opened.content, { name: opened.name, path: opened.path, content: opened.content })
      .then(function() {
        if (store.dom.statusEl.className.indexOf("error") !== -1) {
          store.dom.panel.classList.remove("collapsed");
        }
      });
  });

  // ─── Saving ───────────────────────────────────────────────────────
  function clearSaveConfirmation() {
    const el = store.dom.saveStatus;
    if (!el) return;
    el.textContent = '';
    el.title = '';
    el.classList.add('hidden');
    el.classList.remove('failed');
  }

  function reportSaveOutcome(text, failed) {
    const el = store.dom.saveStatus;
    if (!el) return;
    el.textContent = text;
    el.title = text;
    el.classList.remove('hidden');
    el.classList.toggle('failed', failed);
  }

  // The bar keeps the outcome because it is the one place on screen a save can
  // be read without expanding anything; the panel carries the reason in full,
  // because a path plus a filesystem's wording does not fit in a status bar.
  // Unlike an open, this claims no render number: a save is not a render, and
  // cancelling one the user asked for would leave the canvas disagreeing with
  // the panel with nothing to say so.
  let saveFailureText = null;

  function reportSaveFailure(reason, stillOpen) {
    reportSaveOutcome('✗ ' + reason, true);
    // A save whose file is no longer the one on screen still has to be reported,
    // because its bytes did not land — but revealing the panel would interrupt a
    // model the user has already moved on to, so only the bar carries it.
    if (!stillOpen) {
      return;
    }
    saveFailureText = "✗ " + reason;
    store.dom.statusEl.textContent = saveFailureText;
    store.dom.statusEl.className = "status error";
    store.dom.panel.classList.remove("collapsed");
  }

  // A save that succeeds after one was refused has to take the refusal down with
  // it, or the window reports the same file as both saved and unsaved at once.
  function clearSaveFailure() {
    if (saveFailureText !== null && store.dom.statusEl.textContent === saveFailureText) {
      store.dom.statusEl.textContent = '';
      store.dom.statusEl.className = 'status';
    }
    saveFailureText = null;
  }

  // Saves queue behind each other and behind any render still in flight. Two
  // overlapping saves would otherwise let the slower one land last, putting
  // older text on disk under a confirmation of the newer.
  let saveQueue = Promise.resolve();

  // chooseLocation is what separates Save from Save As: Save reuses the open
  // file's path and shows no dialog, and both ask the host to choose when there
  // is no open file to write back to.
  function saveModel(options) {
    const chooseLocation = Boolean(options && options.chooseLocation);
    saveQueue = saveQueue
      .then(rendersSettled)
      .then(function() { return writeModel(chooseLocation); });
    return saveQueue;
  }

  // Waiting on one render is not enough: a file arriving while that one is in
  // flight starts another, and resuming between the two reads the panel's new
  // text beside the old file's identity. Each wait is checked against what the
  // latest render is by the time it finishes.
  function rendersSettled() {
    const waitingOn = latestRenderSettled;
    return waitingOn.then(function() {
      return waitingOn === latestRenderSettled ? undefined : rendersSettled();
    });
  }

  function writeModel(chooseLocation) {
    const openFile = store.currentFile;
    const target = chooseLocation || !openFile ? '' : openFile.path;
    const suggestedName = (openFile && openFile.name) || (store.modelName || 'diagram') + '.emod';
    const content = sourceToSave(store);

    // An empty panel is refused rather than written: the host would otherwise
    // truncate whichever model the user picked in the save dialog to nothing,
    // or empty the open file outright with no dialog at all, and confirm it.
    if (!content.trim()) {
      reportSaveFailure('There is nothing to save — the source panel is empty', true);
      return Promise.resolve();
    }

    return saveFile(suggestedName, content, target).then(function(saved) {
      // A host that answers no file is one whose dialog was cancelled, which has
      // to leave the open file, the window's name and the path on screen alone.
      if (!saved) {
        reportModified();
        return;
      }
      // A file opened while this was being written owns the window now, so
      // adopting this save's target would name a model the panel is no longer
      // showing — and the save after it would write over that model.
      if (store.currentFile === openFile) {
        store.currentFile = { name: saved.name, path: saved.path, content: content };
        applyWindowTitle(store);
        UI.updateStats(store);
        clearSaveFailure();
      }
      reportSaveOutcome('✓ Saved ' + saved.name, false);
      reportModified();
    }).catch(function(err) {
      reportSaveFailure(err.message || String(err), store.currentFile === openFile);
      reportModified();
    });
  }

  onSaveRequested(saveModel);

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
    // On the same queue as Save: an Export aimed by its own dialog at the file a
    // save is already writing would otherwise race it, and whichever landed last
    // would silently discard the other.
    saveQueue = saveQueue.then(function() {
      return Export.exportToEmodString(store).then(function(content) {
        return saveFile((store.modelName || "diagram") + ".emod", content);
      });
    }).catch(function(err) {
      store.dom.statusEl.textContent = "✗ " + err.message;
      store.dom.statusEl.className = "status error";
      // The status area lives in the panel, which a successful render collapses,
      // and Export sits outside it — so without this the reason a save failed is
      // written somewhere the user who pressed the button cannot see.
      store.dom.panel.classList.remove("collapsed");
    });
    return saveQueue;
  });

  // ─── Visibility toggle ────────────────────────────────────────────
  store.dom.visibilityToggle.addEventListener("click", function() {
    UI.toggleVisibilityPanel(store);
  });

  // ─── Legend toggle ───────────────────────────────────────────────
  store.dom.legendToggle.addEventListener("click", function() {
    Legend.toggleLegendPanel(store);
  });

  store.dom.legendClose.addEventListener("click", function(evt) {
    evt.stopPropagation();
    Legend.toggleLegendPanel(store, false);
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

    // Opening the editor changes nothing in the model, so it never reaches
    // CtxActions — which reports whether the diagram needs re-rendering.
    if (action === "open-field-editor") {
      const menu = store.interaction.ctxMenu;
      const node = menu && store.nodeById.get(menu.targetNodeId);
      UI.hideContextMenu(store);
      if (node) UI.showDetailPanel(store, node);
      return;
    }

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
  return initialState().then(function(state) {
    if (state) {
      Model.setModelData(store, state.diagram || state);
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
  }).catch(function(err) {
    // A host that cannot answer what to open would otherwise leave the window
    // blank and silent: no landing instructions, no placeholder, no status.
    store.dom.sourceInput.placeholder = 'Paste .emod source or diagram JSON here';
    store.dom.panel.classList.remove('collapsed');
    store.dom.statusEl.textContent = '✗ ' + (err.message || 'Could not load the initial model');
    store.dom.statusEl.className = 'status error';
  });
}

export { init, sourceToSave };

if (typeof process === 'undefined' || !process.env || !process.env.VITEST) {
  init();
}

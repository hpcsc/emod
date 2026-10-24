import { createStore } from './store.js';
import { Layout } from './layout.js';
import { Renderer } from './renderer.js';
import { Interaction } from './interaction.js';
import { UI } from './ui.js';
import { Minimap } from './minimap.js';
import { Legend } from './legend.js';
import { SourcePanel } from './source-panel.js';
import { CtxActions } from './ctx-actions.js';
import { Model } from './model.js';
import { bus } from './bus.js';
import { Export } from './emod-export.js';
import { REVALIDATE_PAUSE_MS } from './config.js';
import { ready, isReady, applyDiagram, droppedFiles, saveFile, setWindowTitle, setWindowModified, rememberOpenedFile, resolveUnsavedEdits, onFileOpened, onFilesDropped, onSaveRequested, onLeaveRequested, initialState } from './platform.js';

const POINTER_RELEASE_CHECK_MS = 50;

// ─── Event subscriptions ─────────────────────────────────────────────
bus.on('data:changed', function({ store: s }) {
  renderDiagram(s);
});

bus.on('diagram:edited', function({ store: s }) {
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
  if (btn) btn.disabled = Object.keys(s.nodeOffsets).length === 0;
  if (s.dom.visibilityPanel && !s.dom.visibilityPanel.classList.contains("hidden")) {
    UI.updateVisibilityTree(s);
  }
});

bus.on('diagnostics:changed', function({ store: s, diagnostics, revalidation }) {
  UI.updateDiagnosticsPanel(s, diagnostics, { reopen: !revalidation });
  SourcePanel.showDiagnostics(s);
});

bus.on('diagnostic:clicked', function({ store: s, diagnostic }) {
  if (!diagnostic.line) return;
  s.dom.panel.classList.remove("collapsed");
  SourcePanel.moveCaretTo(s, diagnostic);
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
  bus.emit('diagram:edited', { store: s });
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

bus.on('node:show', function({ store: s, nodeId }) {
  Interaction.centreOn(s, nodeId);
});

// ─── Init ───────────────────────────────────────────────────────────
function init() {
  // ─── Create store and wire DOM references ────────────────────────────
  const store = createStore();
  store.dom.svg = document.getElementById("diagram-canvas");
  store.dom.nameDisplay = document.getElementById("model-name-display");
  store.dom.sourceInput = document.getElementById("source-input");
  store.dom.sourceGutter = document.getElementById("source-gutter");
  store.dom.caretLineDiagnostics = document.getElementById("caret-line-diagnostics");
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
  store.dom.staleNotice = document.getElementById("stale-notice");
  store.dom.conflictNotice = document.getElementById("conflict-notice");
  store.dom.conflictMessage = document.getElementById("conflict-message");
  store.dom.keepDiagramEditsBtn = document.getElementById("keep-diagram-edits");
  store.dom.keepSourceEditsBtn = document.getElementById("keep-source-edits");

  Interaction.initEventListeners(store);
  UI.initDelegation(store);
  UI.initKeyboard(store);
  UI.initDiagnosticsDelegation(store);
  SourcePanel.initSourcePanel(store);
  Minimap.initMinimap(store);

  // ─── Render ───────────────────────────────────────────────────────
  // Deliveries are numbered because a drop and a file the host opened are both
  // read asynchronously, and nothing stops a second one arriving while the first
  // is still being read — so without this the model that ends up open is the one
  // whose read finished last rather than the one the user asked for last, and it
  // is that model's path a following Save writes to. The number is claimed when
  // the gesture arrives and checked before anything on screen is replaced.
  let latestDelivery = 0;

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

  let checkedSource = null;

  // The text the diagram on screen was drawn from, or null when no .emod source
  // in the panel drew it — diagram JSON, or a model the host handed the page. A
  // diagram edit is applied onto this text rather than onto the panel's, because
  // the edited nodes carry the ids a parse of this text gave them. The diagram
  // of that parse is kept as text, because diagram edits change the nodes on
  // screen in place.
  let drawnSource = null;
  let drawnDiagramText = null;

  // Diagram edits are numbered because an apply is answered asynchronously.
  let latestDiagramEdit = 0;
  let handledDiagramEdit = 0;
  let discardedDiagramEdit = 0;
  let applyingDiagramEdit = false;
  let keepDiagramEditsAsked = false;
  let latestApplySettled = Promise.resolve();

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
  function renderPanelSource(text, file, options) {
    const revalidation = Boolean(options && options.revalidation);
    if (!revalidation) {
      clearSaveConfirmation();
    }
    const previousText = store.dom.sourceInput.value;
    if (text !== undefined) {
      store.dom.sourceInput.value = text;
    }
    const opening = file !== undefined;
    const sourceFile = opening ? file : store.currentFile;
    const source = store.dom.sourceInput.value;
    const render = ++latestRender;
    const diagramEditAtStart = latestDiagramEdit;
    if (!opening && !source.trim()) {
      showEmptyPanel(source, revalidation);
      latestRenderSettled = Promise.resolve();
      return latestRenderSettled;
    }
    const overtaken = function() {
      if (!opening) {
        rendersSettled().then(revalidate);
      }
    };
    const parsingStatus = revalidation ? null : store.dom.statusEl;
    latestRenderSettled = Model.sendParse(store, source, parsingStatus, sourceFile ? sourceFile.name : undefined)
      .then(function(data) {
        if (render !== latestRender) return overtaken();
        if (!opening && latestDiagramEdit !== diagramEditAtStart) return overtaken();
        checkedSource = source;
        // A redraw would discard the diagram edits the user has not chosen
        // between yet, so only the panel's own diagnostics change.
        if (!opening && store.conflict) {
          reportDiagnostics(data.diagnostics || [], revalidation);
          showRenderOutcome("✗ Not redrawn: the diagram and the panel disagree", "status error", revalidation);
          return;
        }
        if (!opening && data.parsed === false && diagramOnScreen()) {
          reportDiagnostics(data.diagnostics || [], revalidation);
          keepStaleDiagram("the source does not parse", "✗ Not redrawn: the source does not parse", revalidation);
          return;
        }
        if (opening) {
          store.currentFile = file;
          rememberFile(file, reportRecordingRefusal);
          discardDiagramEdits();
        }
        reportDiagnostics(data.diagnostics || [], revalidation);
        UI.hideDiagramStale(store);
        drawnSource = Model.readDiagramJson(source) ? null : source;
        drawnDiagramText = JSON.stringify(data.diagram);
        if (opening) {
          Model.setModelData(store, data.diagram);
        } else {
          redrawInPlace(data.diagram);
        }
        showRenderOutcome("✓ Rendered", "status success", revalidation);
        if (!revalidation) {
          store.dom.panel.classList.add("collapsed");
          reportModified();
        }
      })
      .catch(function(err) {
        if (render !== latestRender) return overtaken();
        if (!opening && store.conflict) {
          checkedSource = source;
          showRenderOutcome("✗ " + err.message, "status error", revalidation);
          return;
        }
        if (!opening && diagramOnScreen()) {
          checkedSource = source;
          keepStaleDiagram("the source could not be drawn", "✗ " + err.message, revalidation);
          return;
        }
        // The panel's text was replaced for a model that never rendered, and the
        // window still names the one on screen — so putting it back is what keeps
        // the panel, the title and the path stat naming one model.
        if (text !== undefined) {
          store.dom.sourceInput.value = previousText;
          showFailureReason("✗ " + err.message);
          reportModified();
          return;
        }
        checkedSource = source;
        showRenderOutcome("✗ " + err.message, "status error", revalidation);
        if (!revalidation) {
          reportModified();
        }
      });
    if (parsingStatus) {
      replaceableStatus = parsingStatus.textContent;
    }

    return latestRenderSettled;
  }

  function diagramOnScreen() {
    return store.nodes.length > 0;
  }

  function redrawInPlace(diagram) {
    const outgoing = store.nodes;
    Model.setModelDataInPlace(store, diagram);
    const ids = Model.matchNodeIds(outgoing, store.nodes);
    UI.followNodeIds(store, ids);
    return ids;
  }

  function showEmptyPanel(source, revalidation) {
    checkedSource = source;
    reportDiagnostics([], revalidation);
    if (store.conflict) {
      showRenderOutcome("✗ The source panel is empty", "status error", revalidation);
      return;
    }
    if (diagramOnScreen()) {
      keepStaleDiagram("the source panel is empty", "✗ The source panel is empty", revalidation);
    } else if (!revalidation) {
      showRenderOutcome("✗ Paste some .emod content first", "status error", false);
      reportModified();
    }
  }

  function reportDiagnostics(diagnostics, revalidation) {
    store.diagnostics = diagnostics;
    bus.emit('diagnostics:changed', { store, diagnostics, revalidation });
  }

  function keepStaleDiagram(why, status, revalidation) {
    UI.showDiagramStale(store, why);
    showRenderOutcome(status, "status error", revalidation);
    if (!revalidation) {
      reportModified();
    }
  }

  let replaceableStatus = '';

  function showRenderOutcome(text, className, revalidation) {
    const shown = store.dom.statusEl.textContent;
    if (revalidation && shown !== '' && shown !== replaceableStatus) {
      return;
    }
    replaceableStatus = text;
    store.dom.statusEl.textContent = text;
    store.dom.statusEl.className = className;
  }

  function showFailureReason(text) {
    store.dom.statusEl.textContent = text;
    store.dom.statusEl.className = "status error";
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
    setWindowModified(hasUnsavedEdits());
  }

  function hasUnsavedEdits() {
    return modelIsModified(store) || diagramHasUnappliedEdit();
  }

  function diagramHasUnappliedEdit() {
    return latestDiagramEdit !== handledDiagramEdit;
  }

  // Typing over a diagram edit the panel does not hold yet puts both surfaces
  // in conflict at once, before the pause in typing redraws the diagram from
  // the panel and discards that edit.
  store.dom.sourceInput.addEventListener("input", function() {
    if (diagramHasUnappliedEdit()) {
      enterConflict();
    }
    reportModified();
    scheduleRevalidation();
  });

  let pendingRevalidation = null;

  function scheduleRevalidation() {
    clearTimeout(pendingRevalidation);
    pendingRevalidation = setTimeout(function() {
      pendingRevalidation = null;
      rendersSettled().then(revalidate);
    }, REVALIDATE_PAUSE_MS);
  }

  function revalidate() {
    if (store.dom.sourceInput.value === checkedSource) {
      return;
    }
    renderPanelSource(undefined, undefined, { revalidation: true });
  }

  // ─── Diagram edits ────────────────────────────────────────────────
  // Every init has a store of its own and the bus outlives it, so an edit is
  // applied only by the init whose store it changed.
  function countDiagramEdit({ store: s }) {
    if (s !== store || drawnSource === null) return;
    latestDiagramEdit++;
    applyUnappliedDiagramEdits();
  }

  bus.on('diagram:edited', countDiagramEdit);
  bus.on('fields:edited', countDiagramEdit);

  // A panel that no longer holds the drawn source has an edit the diagram does
  // not show, and writing the applied edit over it would discard that.
  function applyUnappliedDiagramEdits() {
    if (applyingDiagramEdit || store.conflict || !diagramHasUnappliedEdit()) {
      return undefined;
    }
    if (store.dom.sourceInput.value !== drawnSource) {
      enterConflict();
      return undefined;
    }
    latestApplySettled = sendDiagramEdit(drawnSource);
    return latestApplySettled;
  }

  function sendDiagramEdit(panelTextToReplace) {
    applyingDiagramEdit = true;
    keepDiagramEditsAsked = false;
    const edit = latestDiagramEdit;
    const source = drawnSource;
    const filename = store.currentFile ? store.currentFile.name : undefined;
    const diagram = { model_name: store.modelName, nodes: store.nodes, edges: store.edges };

    return ready.then(function() {
      return applyDiagram(source, filename, diagram);
    }).then(function(answer) {
      return pointerReleased().then(function() {
        return settleDiagramEdit(edit, source, panelTextToReplace, function() { writeAppliedSource(answer, edit); });
      });
    }, function(err) {
      return settleDiagramEdit(edit, source, panelTextToReplace, function() { reportApplyRefusal(err.message || String(err)); });
    }).catch(function(err) {
      applyingDiagramEdit = false;
      reportApplyRefusal(err.message || String(err));
    });
  }

  function pointerReleased() {
    return new Promise(function(resolve) {
      (function check() {
        if (store.interaction.drag || store.interaction.connect) {
          setTimeout(check, POINTER_RELEASE_CHECK_MS);
        } else {
          resolve();
        }
      })();
    });
  }

  function settleDiagramEdit(edit, source, panelTextToReplace, writeAnswer) {
    applyingDiagramEdit = false;
    if (edit <= discardedDiagramEdit || source !== drawnSource) {
      return applyUnappliedDiagramEdits();
    }
    if (keepDiagramEditsAsked) {
      return sendDiagramEdit(store.dom.sourceInput.value);
    }
    if (store.dom.sourceInput.value !== panelTextToReplace) {
      return undefined;
    }
    if (edit !== latestDiagramEdit) {
      return sendDiagramEdit(panelTextToReplace);
    }
    writeAnswer();
    return undefined;
  }

  function enterConflict() {
    if (!store.conflict) {
      UI.showConflict(store);
    }
    reportModified();
  }

  function keepDiagramEdits() {
    if (applyingDiagramEdit) {
      keepDiagramEditsAsked = true;
      return;
    }
    latestApplySettled = sendDiagramEdit(store.dom.sourceInput.value);
  }

  // Keep source edits puts the drawn diagram back at once, so text that does not
  // parse leaves it on screen marked out of date, and then redraws from the
  // panel the way a pause in typing does.
  function keepSourceEdits() {
    restoreDrawnDiagram();
    reportModified();
    return renderPanelSource(undefined, undefined, { revalidation: true });
  }

  function restoreDrawnDiagram() {
    discardDiagramEdits();
    redrawInPlace(JSON.parse(drawnDiagramText));
  }

  function discardDiagramEdits() {
    discardedDiagramEdit = latestDiagramEdit;
    handledDiagramEdit = latestDiagramEdit;
    keepDiagramEditsAsked = false;
    UI.hideConflict(store);
  }

  store.dom.keepDiagramEditsBtn.addEventListener("click", keepDiagramEdits);
  store.dom.keepSourceEditsBtn.addEventListener("click", keepSourceEdits);

  // The applied source claims a render number, so an older parse still in flight
  // does not draw the source that the edit replaced.
  function writeAppliedSource(answer, edit) {
    const panel = store.dom.sourceInput;
    const scrollTop = panel.scrollTop;
    panel.value = answer.emod;
    panel.scrollTop = scrollTop;
    checkedSource = answer.emod;
    latestRender++;
    handledDiagramEdit = edit;
    UI.hideConflict(store);

    reportDiagnostics(answer.diagnostics || [], true);
    if (answer.parsed === false) {
      keepStaleDiagram("the source does not parse", "✗ Not redrawn: the source does not parse", true);
    } else {
      drawnSource = answer.emod;
      drawnDiagramText = JSON.stringify(answer.diagram);
      UI.hideDiagramStale(store);
      const editor = UI.readFieldEditor(store);
      UI.restoreFieldEditor(store, editor, redrawInPlace(answer.diagram));
      showRenderOutcome("✓ Rendered", "status success", true);
    }
    reportModified();
    if (answer.reformatted) {
      reportReformat(answer.dropped_comments);
    }
  }

  function reportApplyRefusal(reason) {
    if (store.conflict) {
      UI.showConflict(store, reason);
      return;
    }
    keepStaleDiagram("the diagram edit is not in the source: " + reason, "✗ " + reason, true);
    reportModified();
  }

  function reportReformat(droppedComments) {
    const dropped = droppedComments > 0
      ? ' · ' + droppedComments + (droppedComments === 1 ? ' comment' : ' comments') + ' dropped'
      : '';
    reportInBar('Source rewritten in the emod fmt layout' + dropped, false);
  }

  // Everything that asks about unsaved edits queues here. The question and the
  // act it authorises are separated by however long the user takes to answer,
  // and the save the answer asks for reads the panel and the open file when it
  // runs rather than when the question was put — so two overlapping asks let
  // the second one's model be saved under the first one's Save, losing the
  // edits that Save existed to keep.
  let guardQueue = Promise.resolve();

  // A turn that throws must neither wedge every later one behind it nor escape
  // as an unhandled rejection. The question is the user's own gesture, so the
  // reason is shown; and the answer is that nothing may be replaced, because
  // going ahead would discard the very work the question was about.
  function guarded(work) {
    const turn = guardQueue.then(work).catch(function(err) {
      reportHostFailure(err.message || String(err));

      return false;
    });
    guardQueue = turn;

    return turn;
  }

  // The only place a different file's model is handed to renderPanelSource, so
  // every way of opening one runs the guard below and a new one has nowhere
  // else to arrive.
  function openModel(text, file) {
    return guarded(function() {
      return clearedToReplace().then(function(cleared) {
        return cleared ? renderPanelSource(text, file) : undefined;
      });
    });
  }

  // The host asks before it closes the window or quits, because a shell cannot
  // raise this question and wait for it — so the answer is decided here, on the
  // same queue and by the same policy an arriving model goes through.
  onLeaveRequested(function() {
    return guarded(clearedToReplace);
  });

  // Save writes the file the edits belong to before the model on screen is
  // replaced, and the answer to whether that write landed is the marker: a
  // refused save, and one whose location dialog was cancelled, both leave the
  // model modified, and neither may cost the user the edits they asked to keep.
  //
  // It is asked once no diagram edit is still being applied: that edit has not
  // reached the panel yet, and the panel is what Save would write.
  function clearedToReplace() {
    return appliesSettled().then(function() {
      if (!hasUnsavedEdits()) {
        return true;
      }
      return resolveUnsavedEdits().then(function(outcome) {
        if (outcome === 'cancel') {
          return false;
        }
        if (outcome === 'discard') {
          return true;
        }
        return saveModel({ chooseLocation: false }).then(function() {
          return !hasUnsavedEdits();
        });
      });
    });
  }

  // Render means "draw the panel", so it discards a diagram edit the panel does
  // not hold. An edit being applied with no conflict is on its way into the
  // panel, so Render waits for it: drawing the panel first would take the edit
  // off the canvas, and the next canvas edit would be sent without it.
  function renderClicked() {
    clearTimeout(pendingRevalidation);
    if (applyingDiagramEdit && !store.conflict) {
      appliesSettled().then(function() {
        if (!store.conflict) renderClicked();
      });
      return;
    }
    if (store.conflict || diagramHasUnappliedEdit()) {
      restoreDrawnDiagram();
    }
    renderPanelSource();
  }

  store.dom.renderBtn.addEventListener("click", renderClicked);

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
    store.dom.panelBody.classList.remove("drag-over");
    const files = droppedFiles(evt.dataTransfer);
    // A host that resolves drops itself listens above this element and reads the
    // paths its shell resolved rather than the files the page was handed, so an
    // event this page took nothing from has to keep travelling. Stopping it here
    // discards every file released over the panel on the one platform that
    // still delivers a DOM drop and resolves its paths from it.
    if (files.length > 0) {
      evt.stopPropagation();
    }
    openDroppedFiles(files);
  });

  // A host that resolves a drop itself pushes the files here instead, because a
  // native drop never reaches the page as a DOM event. Both arrive at the same
  // routine, which is what makes a dropped model open exactly as a chosen one
  // does rather than by a second copy of the policy agreeing with the first.
  onFilesDropped(openDroppedFiles);

  // A drop is one gesture and opens one model, so the files it carried are a
  // choice rather than a list: the first that names a model is opened and the
  // rest are left alone. Refusing the whole drop over an unsupported file ahead
  // of a supported one would make the order files happen to arrive in decide
  // whether the gesture works.
  function openDroppedFiles(files) {
    // A drag carrying no file — selected text, a link, or a drop a host has
    // already taken for itself and will deliver as paths — is not a model that
    // was refused, so it is nothing to report.
    if (files.length === 0) {
      return Promise.resolve();
    }

    // Claimed before the name check, so a drop the user can see was refused still
    // supersedes an older read: without it that read lands afterwards and opens
    // its model over the refusal this gesture just caused.
    const delivery = ++latestDelivery;

    const chosen = files.filter(namesAModel)[0];
    if (!chosen) {
      reportHostFailure('Only .emod and .json files are supported');

      return Promise.resolve();
    }

    return Promise.resolve(chosen.read()).then(function(opened) {
      return openDeliveredFile(opened, delivery);
    });
  }

  function namesAModel(file) {
    const name = file.name.toLowerCase();

    return name.endsWith('.emod') || name.endsWith('.json');
  }

  // ─── A file the host opened ───────────────────────────────────────
  onFileOpened(function(opened) {
    return openDeliveredFile(opened, ++latestDelivery);
  });

  function openDeliveredFile(opened, delivery) {
    // A file the user has already asked past is not worth reporting on either:
    // its reason would land on top of the model that superseded it.
    if (delivery !== latestDelivery) {
      return Promise.resolve();
    }
    // The status area lives in the panel a successful render collapses, and a
    // file the host resolved arrives from outside the page — so every failure
    // here has to reveal that panel, or choosing a file from the OS dialog, or
    // dropping one on the window, appears to do nothing at all.
    if (opened.error) {
      reportHostFailure(opened.error);

      return Promise.resolve();
    }
    // A file with nothing in it reads successfully and then reaches the parser's
    // own empty-source rejection, whose message is written for someone who
    // pressed Render on an empty panel rather than someone who chose a file.
    if (!opened.content.trim()) {
      reportHostFailure(opened.name + " is empty");

      return Promise.resolve();
    }

    return openModel(opened.content, { name: opened.name, path: opened.path, content: opened.content })
      .then(function() {
        if (store.dom.statusEl.className.indexOf("error") !== -1) {
          store.dom.panel.classList.remove("collapsed");
        }
      });
  }

  // ─── Saving ───────────────────────────────────────────────────────
  function clearSaveConfirmation() {
    const el = store.dom.saveStatus;
    if (!el) return;
    el.textContent = '';
    el.title = '';
    el.classList.add('hidden');
    el.classList.remove('failed');
  }

  function reportInBar(text, failed) {
    const el = store.dom.saveStatus;
    if (!el) return;
    el.textContent = text;
    el.title = text;
    el.classList.remove('hidden');
    el.classList.toggle('failed', failed);
  }

  // Only a file with somewhere on disk to be found again is worth remembering:
  // pasted source has no location, and a drop the page read itself carries
  // none. A recording the host refuses is not worth rearranging the window for
  // — the model did open — and what to say about it is the caller's, because
  // the bar it would go to may already be answering something the user asked
  // for.
  function rememberFile(file, onRefused) {
    if (!file || !file.path) {
      return;
    }
    rememberOpenedFile(file.path).catch(function(err) {
      onRefused(err.message || String(err));
    });
  }

  // A refused recording has one chance at the bar: the open that prompted it
  // cleared the bar, so anything written there since is something the user
  // asked for — a save's confirmation — and outranks it. The reason is back on
  // the next open, since a directory that refused this recording refuses the
  // next.
  function reportRecordingRefusal(reason) {
    const el = store.dom.saveStatus;
    if (el && el.classList.contains('hidden')) {
      reportInBar('✗ Recent files: ' + reason, true);
    }
  }

  // A save's confirmation is the answer to what the user asked for, so a
  // refused recording is added to it rather than written over it — and only
  // while the bar still shows it. Anything newer there is something the user
  // asked for since, and a directory that refused this recording refuses the
  // next, so the reason is back on the next open.
  function extendConfirmation(confirmation, reason) {
    const el = store.dom.saveStatus;
    if (el && el.textContent === confirmation) {
      reportInBar(confirmation + ' · recent files not saved: ' + reason, false);
    }
  }

  // The bar keeps the outcome because it is the one place on screen a save can
  // be read without expanding anything; the panel carries the reason in full,
  // because a path plus a filesystem's wording does not fit in a status bar.
  // Unlike an open, this claims no render number: a save is not a render, and
  // cancelling one the user asked for would leave the canvas disagreeing with
  // the panel with nothing to say so.
  let saveFailureText = null;

  function reportSaveFailure(reason, stillOpen) {
    reportInBar('✗ ' + reason, true);
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
  // latest render and the latest applied diagram edit are by the time it
  // finishes.
  function rendersSettled() {
    const rendering = latestRenderSettled;
    const applying = latestApplySettled;
    return Promise.all([rendering, applying]).then(function() {
      return rendering === latestRenderSettled && applying === latestApplySettled ? undefined : rendersSettled();
    });
  }

  function appliesSettled() {
    const applying = latestApplySettled;
    return applying.then(function() {
      return applying === latestApplySettled ? undefined : appliesSettled();
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
      const confirmation = '✓ Saved ' + saved.name;
      let newLocation = null;
      if (store.currentFile === openFile) {
        store.currentFile = { name: saved.name, path: saved.path, content: content };
        applyWindowTitle(store);
        UI.updateStats(store);
        clearSaveFailure();
        // A save back to the file already open changes nothing about where the
        // model lives; only a location it did not have before is a new one.
        if (saved.path !== (openFile ? openFile.path : '')) {
          newLocation = store.currentFile;
        }
      }
      reportInBar(confirmation, false);
      reportModified();
      rememberFile(newLocation, function(reason) {
        extendConfirmation(confirmation, reason);
      });
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
    bus.emit('diagram:edited', { store });
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

import { PROSE_KINDS } from './config.js';
import { Renderer } from './renderer.js';
import { Layout } from './layout.js';
import { Model } from './model.js';
import { bus } from './bus.js';

// ─── Tooltip ─────────────────────────────────────────────────
// A comment reaches the viewer as the source spells it, hash and all, because
// the same text has to go back into the file on a save.
function buildCommentLines(node) {
  return (node.comments || []).map(function(comment) {
    return String(comment.text || '').replace(/^#[ ]?/, '');
  });
}

function buildHeadingHtml(node) {
  return '<div class="tt-header">' + Renderer.esc(node.label) + '</div>';
}

function buildDescriptionHtml(node) {
  if (!node.description) return '';
  return '<div class="tt-description">' + Renderer.esc(node.description) + '</div>';
}

function buildCommentsHtml(node) {
  return buildCommentLines(node).map(function(line) {
    return '<div class="tt-comment">' + Renderer.esc(line) + '</div>';
  }).join('');
}

function buildFieldsHtml(node) {
  const fields = node.fields || [];
  if (fields.length === 0) return '';

  let html = '<table><thead><tr><th>Field</th><th>Type</th><th></th></tr></thead><tbody>';
  fields.forEach(function(f) {
    const mod = f.modifier ? '<span class="tf-modifier">(' + Renderer.esc(f.modifier) + ')</span>' : '';
    html += '<tr><td class="tf-name">' + Renderer.esc(f.name) + '</td><td class="tf-type">' + Renderer.esc(f.type) + '</td><td>' + mod + '</td></tr>';
  });
  return html + '</tbody></table>';
}

const proseBuilders = {
  description: buildDescriptionHtml,
  comments: buildCommentsHtml,
};

function buildProseHtml(node) {
  return PROSE_KINDS.map(function(kind) { return proseBuilders[kind](node); }).join('');
}

function buildBodyHtml(node, marker) {
  if (!marker) return buildProseHtml(node) + buildFieldsHtml(node);
  const build = proseBuilders[marker];
  return build ? build(node) : '';
}

function showTooltip(store, html, evt) {
  const el = store.dom.tooltip;
  if (!el) return;

  el.innerHTML = html;
  el.style.display = "block";
  positionTooltip(store, evt.clientX, evt.clientY);
}

function hideTooltip(store) {
  const el = store.dom.tooltip;
  if (el) el.style.display = "none";
}

function hoveredNode(store, target) {
  const nodeId = target.dataset.nodeId;
  const node = nodeId ? store.nodeById.get(nodeId) : null;
  if (!node || nodeId === store.interaction.selectedNodeId) return null;
  return node;
}

function tooltipHtmlFor(store, target) {
  const node = hoveredNode(store, target);
  if (!node) return "";

  const body = buildBodyHtml(node, target.dataset.marker);
  return body ? buildHeadingHtml(node) + body : "";
}

function updateTooltipFor(store, target, evt) {
  const html = tooltipHtmlFor(store, target);
  if (!html) {
    hideTooltip(store);
    return;
  }
  showTooltip(store, html, evt);
}

function positionTooltip(store, cx, cy) {
  const el = store.dom.tooltip;
  if (!el || el.style.display === "none") return;
  const rect = el.getBoundingClientRect();
  let x = cx + 14;
  let y = cy + 14;
  if (x + rect.width > window.innerWidth - 8) {
    x = cx - rect.width - 14;
  }
  if (y + rect.height > window.innerHeight - 8) {
    y = cy - rect.height - 14;
  }
  el.style.left = Math.max(4, x) + "px";
  el.style.top = Math.max(4, y) + "px";
}

// A marker sits inside the block it documents, so it is matched first: closest()
// walks up from the pointer and stops at whichever comes nearer. A slice's
// marker has no block around it at all and is only reachable this way.
const HOVER_TARGETS = "[data-marker], .diagram-node";

// ─── Actor annotations ──────────────────────────────────────────
function renderActorAnnotations(store) {
  const container = store.dom.actorAnnotations;
  if (!container) return;
  const actors = store.nodes.filter(function(n) { return n.type === "actor"; });
  if (actors.length === 0) {
    container.innerHTML = "";
    return;
  }
  let html = '<span class="actor-label">Actors:</span>';
  actors.forEach(function(a) {
    html += '<span class="actor-badge">' + Renderer.esc(a.label) + '</span>';
  });
  container.innerHTML = html;
}

// ─── Visibility panel ───────────────────────────────────────────
const VISIBILITY_TYPES = { context: true, aggregate: true, slice: true };

function buildVisibilityTree(nodes) {
  const childrenOf = {};
  nodes.forEach(function(n) {
    if (!VISIBILITY_TYPES[n.type] || !n.parentId) return;
    (childrenOf[n.parentId] = childrenOf[n.parentId] || []).push(n);
  });

  const rows = [];
  const descendantsOf = {};
  const parentOf = {};

  function walk(node, depth) {
    rows.push({ node: node, depth: depth });
    const ids = [];
    (childrenOf[node.id] || []).forEach(function(child) {
      parentOf[child.id] = node.id;
      walk(child, depth + 1);
      ids.push(child.id);
      Array.prototype.push.apply(ids, descendantsOf[child.id]);
    });
    descendantsOf[node.id] = ids;
  }

  nodes.forEach(function(n) { if (n.type === "context") walk(n, 0); });

  function ancestorsOf(id) {
    const ids = [];
    let cur = parentOf[id];
    while (cur) {
      ids.push(cur);
      cur = parentOf[cur];
    }
    return ids;
  }

  return { rows: rows, descendantsOf: descendantsOf, ancestorsOf: ancestorsOf };
}

function updateVisibilityTree(store) {
  const treeEl = store.dom.visibilityTree;
  if (!treeEl) return;
  const tree = buildVisibilityTree(store.nodes);
  if (tree.rows.length === 0) {
    treeEl.innerHTML = '<div class="visibility-item" style="color:#999">No contexts</div>';
    return;
  }

  let html = '';
  tree.rows.forEach(function(row) {
    const id = row.node.id;
    html += '<div class="visibility-item vis-' + row.node.type + '" data-node-id="' + id + '"';
    html += ' style="padding-left:' + (12 + row.depth * 16) + 'px">';
    html += '<input type="checkbox" id="vis-cb-' + id + '"' + (store.hiddenNodes[id] ? '' : ' checked') + '>';
    html += '<label for="vis-cb-' + id + '">' + Renderer.esc(row.node.label) + '</label>';
    html += '</div>';
  });
  treeEl.innerHTML = html;

  treeEl.querySelectorAll('.visibility-item input[type="checkbox"]').forEach(function(cb) {
    const id = cb.closest('.visibility-item').getAttribute('data-node-id');
    const descendants = tree.descendantsOf[id] || [];
    const hiddenDescendants = descendants.filter(function(d) { return store.hiddenNodes[d]; });

    // A property, not an attribute — checked already covers "all visible", so
    // this is the only way to show a context whose slices are partly hidden.
    cb.indeterminate = !store.hiddenNodes[id] && hiddenDescendants.length > 0;

    cb.addEventListener('change', function() {
      if (this.checked) {
        [id].concat(descendants, tree.ancestorsOf(id)).forEach(function(n) {
          delete store.hiddenNodes[n];
        });
      } else {
        [id].concat(descendants).forEach(function(n) { store.hiddenNodes[n] = true; });
      }
      bus.emit('data:changed', { store });
      updateVisibilityTree(store);
    });
  });
}

function toggleVisibilityPanel(store, show) {
  const panelEl = store.dom.visibilityPanel;
  const toggleEl = store.dom.visibilityToggle;
  if (show === undefined) {
    panelEl.classList.toggle("hidden");
  } else if (show) {
    panelEl.classList.remove("hidden");
  } else {
    panelEl.classList.add("hidden");
  }
  const isHidden = panelEl.classList.contains("hidden");
  toggleEl.classList.toggle("active", !isHidden);
  if (!isHidden) updateVisibilityTree(store);
}

// ─── Diagnostics panel ──────────────────────────────────────────
function updateDiagnosticsPanel(store, diagnostics) {
  const badgeEl = store.dom.diagnosticsBadge;
  const panelEl = store.dom.diagnosticsPanel;
  const listEl = store.dom.diagnosticsList;
  if (!badgeEl || !panelEl || !listEl) return;

  if (!diagnostics || diagnostics.length === 0) {
    badgeEl.style.display = "none";
    badgeEl.textContent = "";
    panelEl.classList.add("hidden");
    listEl.innerHTML = "";
    return;
  }

  // An explicit value, not "", because the stylesheet's own rule for the badge
  // is display:none — clearing the inline style just falls back to that and
  // the badge stays invisible.
  badgeEl.style.display = "inline-block";
  badgeEl.textContent = diagnostics.length + " error" + (diagnostics.length === 1 ? "" : "s");

  let html = "";
  diagnostics.forEach(function(d, idx) {
    var sev = d.severity || "error";
    var loc = (d.file || "?") + ":" + (d.line || "?");
    html += '<div class="diag-item" data-diagnostics-idx="' + idx + '">';
    html += '<span class="diag-severity ' + sev + '">' + Renderer.esc(sev) + '</span>';
    html += '<span class="diag-location">' + Renderer.esc(loc) + '</span>';
    html += '<span class="diag-message">' + Renderer.esc(d.message) + '</span>';
    html += '</div>';
  });
  listEl.innerHTML = html;

  panelEl.classList.remove("hidden");
}

function handleDiagnosticClick(store, evt) {
  const item = evt.target.closest('.diag-item');
  if (!item) return;

  // Reset not-rendered on all items
  store.dom.diagnosticsList.querySelectorAll('.diag-item.not-rendered').forEach(function(el) {
    el.classList.remove('not-rendered');
  });

  const idx = parseInt(item.getAttribute('data-diagnostics-idx'));
  if (isNaN(idx)) return;

  const diagnostics = store.diagnostics;
  if (!diagnostics || idx >= diagnostics.length) return;

  const d = diagnostics[idx];
  const file = d.file;
  const line = d.line;

  // Can't match without a specific file and line
  if (!file || !line) {
    item.classList.add('not-rendered');
    return;
  }

  // Find matching nodes by position.filename and position.line
  const matchingIds = [];
  for (var i = 0; i < store.nodes.length; i++) {
    var node = store.nodes[i];
    if (node.position && node.position.filename === file && node.position.line === line) {
      matchingIds.push(node.id);
    }
  }

  if (matchingIds.length > 0) {
    if (store.interaction.selectedNodeId) hideDetailPanel(store);
    highlightElements(store, matchingIds);
  } else {
    item.classList.add('not-rendered');
  }
}

function initDiagnosticsDelegation(store) {
  const listEl = store.dom.diagnosticsList;
  if (!listEl) return;
  listEl.addEventListener('click', function(evt) {
    handleDiagnosticClick(store, evt);
  });
}

function toggleDiagnosticsPanel(store) {
  const panelEl = store.dom.diagnosticsPanel;
  const badgeEl = store.dom.diagnosticsBadge;
  if (!panelEl || !badgeEl) return;
  panelEl.classList.toggle("hidden");
  badgeEl.classList.toggle("active", !panelEl.classList.contains("hidden"));
}

function hideDiagnosticsPanel(store) {
  const panelEl = store.dom.diagnosticsPanel;
  const badgeEl = store.dom.diagnosticsBadge;
  if (!panelEl || !badgeEl) return;
  panelEl.classList.add("hidden");
  badgeEl.classList.remove("active");
  clearHighlights(store);
  // Reset not-rendered markers on all items
  if (store.dom.diagnosticsList) {
    store.dom.diagnosticsList.querySelectorAll('.diag-item.not-rendered').forEach(function(el) {
      el.classList.remove('not-rendered');
    });
  }
}

// ─── Stats ──────────────────────────────────────────────────────
function updateStats(store, dims) {
  if (store.dom.statNodes) store.dom.statNodes.textContent = store.nodes.length;
  if (store.dom.statEdges) store.dom.statEdges.textContent = store.edges.length;
  if (store.dom.statCanvas && dims) store.dom.statCanvas.textContent = dims.width + " × " + dims.height;
}

// ─── Detail panel ───────────────────────────────────────────────
function isDeletableNodeType(type) {
  return type === "command" || type === "event" || type === "trigger" ||
         type === "view" || type === "automation" || type === "translation";
}

function showDetailPanel(store, node) {
  const el = store.dom.detailPanel;
  const content = store.dom.dpContent;
  if (!el || !content) return;

  store.interaction.selectedNodeId = node.id;
  clearHighlights(store);

  const deletable = isDeletableNodeType(node.type);
  let html = '<div class="dp-header">' + Renderer.esc(node.label) + ' <span class="dp-type">(' + node.type + ')</span></div>';
  if (deletable) {
    html += '<button class="dp-delete-btn" title="Delete node">Delete</button>';
  }

  if (node.description) {
    html += '<div class="dp-section">';
    html += '<div class="dp-section-title">Description</div>';
    html += '<div class="dp-description">' + Renderer.esc(node.description) + '</div>';
    html += '</div>';
  }

  const isFieldNode = (node.type === 'command' || node.type === 'event');
  const showFields = isFieldNode || (node.fields && node.fields.length > 0);

  if (showFields) {
    html += '<div class="dp-section">';
    html += '<div class="dp-section-title">Fields</div>';
    const fields = node.fields || [];
    if (isFieldNode) {
      html += '<table><thead><tr><th>Field</th><th>Type</th><th>Modifier</th><th></th></tr></thead><tbody>';
      fields.forEach(function(f, i) {
        html += '<tr>';
        html += '<td><input class="dp-field-input dp-field-name-input" value="' + Renderer.esc(f.name) + '" data-idx="' + i + '" data-field="name"></td>';
        html += '<td><input class="dp-field-input dp-field-type-input" value="' + Renderer.esc(f.type) + '" data-idx="' + i + '" data-field="type"></td>';
        html += '<td><input class="dp-field-input dp-field-modifier-input" value="' + Renderer.esc(f.modifier || '') + '" data-idx="' + i + '" data-field="modifier"></td>';
        html += '<td><button class="dp-field-delete" data-idx="' + i + '" title="Delete field">×</button></td>';
        html += '</tr>';
      });
      if (fields.length === 0) {
        html += '<tr><td colspan="4" style="color:#999;font-style:italic;padding:8px 0;text-align:center;">No fields defined</td></tr>';
      }
      html += '</tbody></table>';
      html += '<button class="dp-add-field">+ Add Field</button>';
    } else {
      html += '<table><thead><tr><th>Field</th><th>Type</th><th></th></tr></thead><tbody>';
      fields.forEach(function(f) {
        const mod = f.modifier ? ' <span class="dp-field-modifier">(' + Renderer.esc(f.modifier) + ')</span>' : '';
        html += '<tr><td class="dp-field-name">' + Renderer.esc(f.name) + '</td><td class="dp-field-type">' + Renderer.esc(f.type) + '</td><td>' + mod + '</td></tr>';
      });
      html += '</tbody></table>';
    }
    html += '</div>';
  }

  if (node.type === 'trigger') {
    html += '<div class="dp-section">';
    html += '<div class="dp-section-title">Trigger</div>';
    html += '<table><tbody>';
    html += '<tr><th>Actor</th><td>' + Renderer.esc(node.actor || '—') + '</td></tr>';
    html += '<tr><th>Reads</th><td>' + Renderer.esc(node.reads || '—') + '</td></tr>';
    html += '</tbody></table>';
    html += '</div>';
  }

  if (node.type === 'view') {
    if (node.subscribes && node.subscribes.length > 0) {
      html += '<div class="dp-section">';
      html += '<div class="dp-section-title">Subscribes To</div>';
      html += '<div>' + node.subscribes.map(function(s) { return Renderer.esc(s); }).join(', ') + '</div>';
      html += '</div>';
    }
  }

  if (node.type === 'automation') {
    html += '<div class="dp-section">';
    html += '<div class="dp-section-title">Automation</div>';
    html += '<table><tbody>';
    html += '<tr><th>On Event</th><td>' + Renderer.esc(node.on_event || '—') + '</td></tr>';
    html += '<tr><th>Every</th><td>' + Renderer.esc(node.every || '—') + '</td></tr>';
    html += '<tr><th>After</th><td>' + Renderer.esc(node.after || '—') + '</td></tr>';
    html += '<tr><th>Reads</th><td>' + Renderer.esc(node.reads || '—') + '</td></tr>';
    html += '<tr><th>Command</th><td>' + Renderer.esc(node.command || '—') + '</td></tr>';
    html += '<tr><th>Target Context</th><td>' + Renderer.esc(node.target_context || '—') + '</td></tr>';
    html += '</tbody></table>';
    html += '</div>';
  }

  if (node.type === 'translation') {
    html += '<div class="dp-section">';
    html += '<div class="dp-section-title">Translation</div>';
    html += '<table><tbody>';
    html += '<tr><th>External System</th><td>' + Renderer.esc(node.external_system || '—') + '</td></tr>';
    html += '<tr><th>Reads</th><td>' + Renderer.esc(node.reads || '—') + '</td></tr>';
    html += '<tr><th>Command</th><td>' + Renderer.esc(node.command || '—') + '</td></tr>';
    if (node.event) {
      html += '<tr><th>Event</th><td>' + Renderer.esc(node.event.name || '—') + '</td></tr>';
    }
    html += '</tbody></table>';
    html += '</div>';
  }

  if (node.position) {
    html += '<div class="dp-section">';
    html += '<div class="dp-section-title">Source</div>';
    html += '<div class="dp-source">' + Renderer.esc(node.position.filename || '?') + ':' + (node.position.line || '?') + '</div>';
    html += '</div>';
  }

  content.innerHTML = html;

  if (isFieldNode) {
    content.querySelectorAll('.dp-field-input').forEach(function(input) {
      input.addEventListener('blur', function() {
        const idx = parseInt(this.dataset.idx);
        const field = this.dataset.field;
        const n = store.nodeById.get( store.interaction.selectedNodeId);
        if (!n || !n.fields || !n.fields[idx]) return;
        n.fields[idx][field] = this.value;
      });
    });

    content.querySelectorAll('.dp-field-delete').forEach(function(btn) {
      btn.addEventListener('click', function(evt) {
        evt.stopPropagation();
        const idx = parseInt(this.dataset.idx);
        const n = store.nodeById.get( store.interaction.selectedNodeId);
        if (!n || !n.fields) return;
        n.fields.splice(idx, 1);
        bus.emit('data:changed', { store });
      });
    });

    const addBtn = content.querySelector('.dp-add-field');
    if (addBtn) {
      addBtn.addEventListener('click', function(evt) {
        evt.stopPropagation();
        const n = store.nodeById.get( store.interaction.selectedNodeId);
        if (!n) return;
        if (!n.fields) n.fields = [];
        n.fields.push({name: '', type: 'string', modifier: ''});
        bus.emit('data:changed', { store });
      });
    }
  }

  const delBtn = content.querySelector('.dp-delete-btn');
  if (delBtn) {
    delBtn.addEventListener('click', function(evt) {
      evt.stopPropagation();
      const nodeId = store.interaction.selectedNodeId;
      if (!nodeId) return;
      const found = store.nodeById.get( nodeId);
      if (found && isDeletableNodeType(found.type)) {
        bus.emit('node:delete', { store, nodeId });
      }
    });
  }

  el.style.display = "block";
  hideTooltip(store);
}

function showEdgeDetail(store) {
  var edge = store.interaction.selectedEdge;
  if (!edge) return;
  var srcNode = store.nodeById.get(edge.source);
  var tgtNode = store.nodeById.get(edge.target);
  var el = store.dom.detailPanel;
  var content = store.dom.dpContent;
  if (!el || !content) return;

  store.interaction.selectedNodeId = null;
  clearHighlights(store);

  var srcLabel = srcNode ? Renderer.esc(srcNode.label) : edge.source;
  var tgtLabel = tgtNode ? Renderer.esc(tgtNode.label) : edge.target;

  var html = '<div class="dp-header">Arrow: ' + Renderer.esc(edge.type) + ' <span class="dp-type">(edge)</span></div>';
  html += '<table><tbody>';
  html += '<tr><th>Source</th><td>' + srcLabel + '</td></tr>';
  html += '<tr><th>Target</th><td>' + tgtLabel + '</td></tr>';
  html += '<tr><th>Type</th><td>' + Renderer.esc(edge.type) + '</td></tr>';
  html += '</tbody></table>';
  html += '<button class="dp-delete-btn" id="dp-delete-edge" title="Delete arrow">Delete arrow</button>';
  content.innerHTML = html;

  var delBtn = content.querySelector('#dp-delete-edge');
  if (delBtn) {
    delBtn.addEventListener('click', function(evt) {
      evt.stopPropagation();
      var e = store.interaction.selectedEdge;
      if (e) {
        Model.removeEdge(store, e.source, e.target);
        store.interaction.selectedEdge = null;
        hideDetailPanel(store);
        bus.emit('data:changed', { store });
      }
    });
  }

  el.style.display = "block";
  hideTooltip(store);
}

function hideDetailPanel(store) {
  const el = store.dom.detailPanel;
  if (el) el.style.display = "none";
  store.interaction.selectedNodeId = null;
  store.interaction.selectedEdge = null;
  clearHighlights(store);
}

// ─── Highlighting ──────────────────────────────────────────────
function clearHighlights(store) {
  store.interaction.highlighted = {};
  const svgEl = store.dom.svg;
  svgEl.querySelectorAll(".hl").forEach(function(el) {
    el.classList.remove("hl");
  });
  svgEl.classList.remove("has-highlights");
}

function addHighlight(store, nodeId) {
  store.interaction.highlighted[nodeId] = true;
  const el = store.dom.svg.querySelector('[data-node-id="' + nodeId + '"]');
  if (el) el.classList.add("hl");
  store.dom.svg.classList.add("has-highlights");
}

function highlightElements(store, ids) {
  clearHighlights(store);
  ids.forEach(function(id) { addHighlight(store, id); });
}

// ─── Inline rename ─────────────────────────────────────────────
function startInlineEdit(store, nodeId) {
  const node = store.nodeById.get( nodeId);
  if (!node) return;

  cancelInlineEdit(store);

  hideDetailPanel(store);
  clearHighlights(store);

  const svgEl = store.dom.svg;
  let el = null;
  const nType = node.type;
  if (nType === "command" || nType === "event" || nType === "trigger" ||
      nType === "view" || nType === "automation" || nType === "translation") {
    el = svgEl.querySelector('[data-node-id="' + nodeId + '"]');
  } else if (nType === "context") {
    el = svgEl.querySelector('.ctx-label[data-ctx-id="' + nodeId + '"]');
  } else if (nType === "aggregate") {
    el = svgEl.querySelector('.agg-label[data-agg-id="' + nodeId + '"]');
  } else if (nType === "slice") {
    el = svgEl.querySelector('.slice-header[data-slice-id="' + nodeId + '"]');
  }
  if (!el) return;

  const rect = el.getBoundingClientRect();

  const input = document.createElement("input");
  input.type = "text";
  input.value = node.label;
  input.style.position = "fixed";
  input.style.zIndex = "999";
  input.style.border = "2px solid #3498db";
  input.style.borderRadius = "4px";
  input.style.outline = "none";
  input.style.background = "#fff";
  input.style.fontFamily = "sans-serif";
  input.style.fontSize = "13px";
  input.style.padding = "2px 6px";
  input.style.boxSizing = "border-box";
  input.style.left = Math.max(4, rect.left - 4) + "px";
  input.style.top = Math.max(4, rect.top - 4) + "px";
  input.style.width = Math.max(rect.width + 8, 60) + "px";

  const isBlock = (nType === "command" || nType === "event" || nType === "trigger" ||
                   nType === "view" || nType === "automation" || nType === "translation");
  if (isBlock) {
    const bw = Math.max(rect.width * 0.8, 80);
    input.style.left = Math.max(4, rect.left + (rect.width - bw) / 2) + "px";
    input.style.top = Math.max(4, rect.top + (rect.height - 28) / 2) + "px";
    input.style.width = bw + "px";
  }
  input.style.height = "28px";

  store.interaction.inlineEdit = {
    nodeId: nodeId,
    originalLabel: node.label,
    inputEl: input,
  };

  input.addEventListener("keydown", function(evt) {
    if (evt.key === "Enter") {
      evt.preventDefault();
      if (!store.interaction.inlineEdit) return;
      const newLabel = input.value.trim() || store.interaction.inlineEdit.originalLabel;
      const n = store.nodeById.get( store.interaction.inlineEdit.nodeId);
      if (n && newLabel !== n.label) {
        n.label = newLabel;
      }
      document.body.removeChild(input);
      store.interaction.inlineEdit = null;
      bus.emit('data:changed', { store });
    } else if (evt.key === "Escape") {
      evt.preventDefault();
      cancelInlineEdit(store);
    }
  });

  input.addEventListener("blur", function() {
    if (!store.interaction.inlineEdit) return;
    const newLabel = input.value.trim() || store.interaction.inlineEdit.originalLabel;
    const n = store.nodeById.get( store.interaction.inlineEdit.nodeId);
    if (n && newLabel !== n.label) {
      n.label = newLabel;
    }
    document.body.removeChild(input);
    store.interaction.inlineEdit = null;
    setTimeout(function() { bus.emit('data:changed', { store }); }, 0);
  });

  document.body.appendChild(input);
  input.focus();
  input.select();
}

function cancelInlineEdit(store) {
  if (!store.interaction.inlineEdit) return;
  document.body.removeChild(store.interaction.inlineEdit.inputEl);
  store.interaction.inlineEdit = null;
}

// ─── Context menu ──────────────────────────────────────────────
// target names what the menu was opened over: one of { nodeId }, { aggOrCtxId }
// or { sliceId }. Each kind gets its own items and records its own id, which is
// what the action then applies to.
function showContextMenu(store, x, y, target) {
  const el = store.dom.ctxMenu;
  if (!el) return;
  if (target.nodeId) {
    store.interaction.ctxMenu = { targetNodeId: target.nodeId };
    el.innerHTML = '<div class="ctx-menu-item" data-action="open-field-editor">Open field editor</div>';
  } else if (target.aggOrCtxId) {
    const node = store.nodeById.get(target.aggOrCtxId);
    if (node && node.type === "context") {
      store.interaction.ctxMenu = { targetCtxId: target.aggOrCtxId };
    } else {
      store.interaction.ctxMenu = { targetAggId: target.aggOrCtxId };
    }
    el.innerHTML = '<div class="ctx-menu-item" data-action="add-slice">Add Slice</div>';
  } else if (target.sliceId) {
    store.interaction.ctxMenu = { targetSliceId: target.sliceId };
    el.innerHTML = [
      '<div class="ctx-menu-item" data-action="add-command">Add Command</div>',
      '<div class="ctx-menu-item" data-action="add-event">Add Event</div>',
      '<div class="ctx-menu-item" data-action="add-flow">Add Flow</div>',
      '<div class="ctx-menu-divider"></div>',
      '<div class="ctx-menu-item" data-action="move-slice-left">Move Left</div>',
      '<div class="ctx-menu-item" data-action="move-slice-right">Move Right</div>',
    ].join('');
  }
  el.style.left = x + "px";
  el.style.top = y + "px";
  el.style.display = "block";
}

function hideContextMenu(store) {
  const el = store.dom.ctxMenu;
  if (el) el.style.display = "none";
  store.interaction.ctxMenu = null;
}

function firstAggregateIdIn(store, ctxId) {
  const agg = store.nodes.find(function(n) {
    return n.type === "aggregate" && n.parentId === ctxId;
  });
  return agg ? agg.id : null;
}

function ownerIdAt(target, attr) {
  const owner = target.closest("[" + attr + "]");
  return owner ? owner.getAttribute(attr) : null;
}

// Every part of a container's chrome names the container it belongs to — the
// band, the label drawn on it and any mark standing beside that label — so the
// menu is reached by that id. Matching the band classes instead would leave
// whatever is painted over them a patch that answers nothing.
function containerMenuIdAt(store, target) {
  const aggId = ownerIdAt(target, "data-agg-id");
  if (aggId) return aggId;

  const ctxId = ownerIdAt(target, "data-ctx-id");
  if (!ctxId) return null;
  return firstAggregateIdIn(store, ctxId) || ctxId;
}

// ─── SVG event delegation (click, dblclick, contextmenu, tooltip) ─
function initDelegation(store) {
  const svgEl = store.dom.svg;

  // Tooltip delegation
  let hovered = null;

  svgEl.addEventListener("pointerover", function(evt) {
    const target = evt.target.closest(HOVER_TARGETS);
    if (target === hovered) return;
    hovered = target;
    if (target) updateTooltipFor(store, target, evt);
  });

  svgEl.addEventListener("pointerout", function(evt) {
    const target = evt.target.closest(HOVER_TARGETS);
    if (!target) return;
    const arriving = evt.relatedTarget ? evt.relatedTarget.closest(HOVER_TARGETS) : null;
    if (arriving === target) return;
    hovered = null;
    hideTooltip(store);
  });

  svgEl.addEventListener("pointermove", function(evt) {
    if (hovered) positionTooltip(store, evt.clientX, evt.clientY);
  });

  // Arrow handle visibility on hover
  var arrowHovered = null;

  function setArrowHovered(store, edgeId) {
    if (edgeId === arrowHovered) return;
    var svg = store.dom.svg;
    if (arrowHovered) {
      svg.querySelectorAll('[data-edge-id="' + arrowHovered + '"]').forEach(function(el) {
        el.classList.remove("visible", "arrow-hover");
      });
    }
    arrowHovered = edgeId;
    if (!edgeId) return;
    svg.querySelectorAll('.arrow-handle[data-edge-id="' + edgeId + '"]').forEach(function(h) {
      h.classList.add("visible");
    });
    var arrowEl = svg.querySelector('path.arrow[data-edge-id="' + edgeId + '"]');
    if (arrowEl) arrowEl.classList.add("arrow-hover");
  }

  svgEl.addEventListener("pointerover", function(evt) {
    var arrow = evt.target.closest(".arrow-hit");
    if (arrow) {
      setArrowHovered(store, arrow.getAttribute("data-edge-id"));
    } else if (evt.target.closest('.arrow-handle, [data-port]')) {
      // Neither the handles nor the connection port they are drawn over may put
      // the arrow away: a handle is only grabbable while it is showing, and the
      // port covers the last few pixels before the pointer reaches it.
    } else {
      setArrowHovered(store, null);
    }
  });

  svgEl.addEventListener("click", function(evt) {
    if (store.interaction.pan) return;

    let target = evt.target;
    let interactive = false;

    const ctxLabel = target.closest(".ctx-label");
    if (ctxLabel) {
      const ctxId = ctxLabel.getAttribute("data-ctx-id");
      if (ctxId) {
        interactive = true;
        const ids = Layout.getDescendantIds(store.nodes, ctxId);
        if (store.interaction.selectedNodeId) hideDetailPanel(store);
        highlightElements(store, ids);
      }
    }

    if (!interactive) {
      const aggLabel = target.closest(".agg-label");
      if (aggLabel) {
        const aggId = aggLabel.getAttribute("data-agg-id");
        if (aggId) {
          interactive = true;
          const ids = Layout.getDescendantIds(store.nodes, aggId);
          if (store.interaction.selectedNodeId) hideDetailPanel(store);
          highlightElements(store, ids);
        }
      }
    }

    if (!interactive) {
      const arrow = target.closest(".arrow-hit");
      if (arrow) {
        interactive = true;
        const src = arrow.getAttribute("data-source");
        const tgt = arrow.getAttribute("data-target");
        if (src && tgt) {
          if (store.interaction.selectedNodeId) hideDetailPanel(store);
          highlightElements(store, [src, tgt]);
          // Find edge type
          for (let i = 0; i < store.edges.length; i++) {
            const e = store.edges[i];
            if (e.source === src && e.target === tgt) {
              store.interaction.selectedEdge = { source: e.source, target: e.target, type: e.type };
              break;
            }
          }
          showEdgeDetail(store);
        }
      }
    }

    // A left click on a block does nothing — the field editor opens from the
    // block's context menu now. The click is still counted as landing on
    // something, so clicking a block does not dismiss a panel being edited.
    if (!interactive && target.closest(".diagram-node")) interactive = true;

    if (!interactive) {
      const activeEl = document.activeElement;
      const panelEl = store.dom.detailPanel;
      const inputFocused = panelEl && panelEl.style.display !== "none" && activeEl &&
        panelEl.contains(activeEl) && (activeEl.tagName === "INPUT" || activeEl.tagName === "SELECT");
      if (!inputFocused) {
        hideDetailPanel(store);
        clearHighlights(store);
      }
    }
  });

  svgEl.addEventListener("dblclick", function(evt) {
    const target = evt.target;

    const block = target.closest(".diagram-node");
    if (block) {
      const nodeId = block.getAttribute("data-node-id");
      if (nodeId) { evt.preventDefault(); startInlineEdit(store, nodeId); return; }
    }

    const ctxLabel = target.closest(".ctx-label");
    if (ctxLabel) {
      const ctxId = ctxLabel.getAttribute("data-ctx-id");
      if (ctxId) { evt.preventDefault(); startInlineEdit(store, ctxId); return; }
    }

    const aggLabel = target.closest(".agg-label");
    if (aggLabel) {
      const aggId = aggLabel.getAttribute("data-agg-id");
      if (aggId) { evt.preventDefault(); startInlineEdit(store, aggId); return; }
    }

    const sliceHeader = target.closest(".slice-header");
    if (sliceHeader) {
      const sliceId = sliceHeader.getAttribute("data-slice-id");
      if (sliceId) { evt.preventDefault(); startInlineEdit(store, sliceId); return; }
    }
  });

  svgEl.addEventListener("contextmenu", function(evt) {
    const target = evt.target;

    // Check for arrow right-click
    const arrow = target.closest(".arrow-hit");
    if (arrow) {
      evt.preventDefault();
      const src = arrow.getAttribute("data-source");
      const tgt = arrow.getAttribute("data-target");
      if (src && tgt) {
        store.interaction.ctxMenu = { edgeSource: src, edgeTarget: tgt };
        const el = store.dom.ctxMenu;
        if (el) {
          el.innerHTML = '<div class="ctx-menu-item" data-action="delete-arrow">Delete arrow</div>';
          el.style.left = evt.clientX + "px";
          el.style.top = evt.clientY + "px";
          el.style.display = "block";
        }
      }
      return;
    }

    const sliceHeader = target.closest(".slice-header");
    if (sliceHeader) {
      const sliceId = sliceHeader.getAttribute("data-slice-id");
      if (sliceId) {
        evt.preventDefault();
        showContextMenu(store, evt.clientX, evt.clientY, { sliceId: sliceId });
        return;
      }
    }

    const sliceArea = target.closest(".slice-area");
    if (sliceArea) {
      const sliceId = sliceArea.getAttribute("data-slice-id");
      if (sliceId) {
        evt.preventDefault();
        showContextMenu(store, evt.clientX, evt.clientY, { sliceId: sliceId });
        return;
      }
    }

    const block = target.closest(".diagram-node");
    if (block) {
      const nodeId = block.getAttribute("data-node-id");
      if (nodeId) {
        evt.preventDefault();
        showContextMenu(store, evt.clientX, evt.clientY, { nodeId: nodeId });
      }
      return;
    }

    const containerId = containerMenuIdAt(store, target);
    if (containerId) {
      evt.preventDefault();
      showContextMenu(store, evt.clientX, evt.clientY, { aggOrCtxId: containerId });
    }
  });
}

// ─── Keyboard handler ──────────────────────────────────────────
function initKeyboard(store) {
  document.addEventListener("keydown", function(evt) {
    if (evt.key === "Escape") {
      const activeEl = document.activeElement;
      const panelEl = store.dom.detailPanel;
      if (panelEl && panelEl.style.display !== "none" && activeEl && panelEl.contains(activeEl) &&
          (activeEl.tagName === "INPUT" || activeEl.tagName === "SELECT")) {
        activeEl.blur();
        return;
      }
      if (store.interaction.inlineEdit) {
        cancelInlineEdit(store);
      } else if (store.interaction.ctxMenu) {
        hideContextMenu(store);
      } else {
        hideDetailPanel(store);
        clearHighlights(store);
      }
    } else if (evt.key === "Delete" || evt.key === "Backspace") {
      if (store.interaction.selectedEdge) {
        evt.preventDefault();
        var edge = store.interaction.selectedEdge;
        Model.removeEdge(store, edge.source, edge.target);
        store.interaction.selectedEdge = null;
        hideDetailPanel(store);
        bus.emit('data:changed', { store });
      } else if (store.interaction.selectedNodeId) {
        const delNode = store.nodeById.get( store.interaction.selectedNodeId);
        if (delNode && isDeletableNodeType(delNode.type)) {
          evt.preventDefault();
          bus.emit('node:delete', { store, nodeId: store.interaction.selectedNodeId });
        }
      }
    }
  });
}

export const UI = {
  hideTooltip,
  positionTooltip,
  renderActorAnnotations,
  updateVisibilityTree,
  toggleVisibilityPanel,
  updateDiagnosticsPanel,
  handleDiagnosticClick,
  initDiagnosticsDelegation,
  toggleDiagnosticsPanel,
  hideDiagnosticsPanel,
  updateStats,
  showDetailPanel,
  showEdgeDetail,
  hideDetailPanel,
  clearHighlights,
  addHighlight,
  highlightElements,
  startInlineEdit,
  cancelInlineEdit,
  showContextMenu,
  hideContextMenu,
  initDelegation,
  initKeyboard,
  isDeletableNodeType,
};

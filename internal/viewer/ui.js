import { MINIMAP_W, MINIMAP_H, MINIMAP_PAD } from './config.js';
import { Renderer } from './renderer.js';
import { Layout } from './layout.js';
import { Interaction } from './interaction.js';
import { bus } from './bus.js';

// ─── Tooltip ─────────────────────────────────────────────────
function showTooltip(store, node, evt) {
  const el = store.dom.tooltip;
  if (!el) return;

  let html = '<div class="tt-header">' + Renderer.esc(node.label) + '</div>';
  html += '<table><thead><tr><th>Field</th><th>Type</th><th></th></tr></thead><tbody>';
  (node.fields || []).forEach(function(f) {
    const mod = f.modifier ? '<span class="tf-modifier">(' + Renderer.esc(f.modifier) + ')</span>' : '';
    html += '<tr><td class="tf-name">' + Renderer.esc(f.name) + '</td><td class="tf-type">' + Renderer.esc(f.type) + '</td><td>' + mod + '</td></tr>';
  });
  html += '</tbody></table>';
  el.innerHTML = html;

  el.style.display = "block";
  positionTooltip(store, evt.clientX, evt.clientY);
}

function hideTooltip(store) {
  const el = store.dom.tooltip;
  if (el) el.style.display = "none";
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

let hoveredBlock = null;

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

// ─── Minimap ────────────────────────────────────────────────────
function getDiagramDims(store) {
  const svgEl = store.dom.svg;
  const vg = svgEl.querySelector("#viewport-group");
  if (vg) {
    try {
      const b = vg.getBBox();
      if (b.width > 0 && b.height > 0) return { width: b.width, height: b.height };
    } catch(e) {}
  }
  const vb = svgEl.getAttribute("viewBox");
  if (vb) {
    const parts = vb.split(/\s+/).map(Number);
    if (parts.length === 4 && parts[2] > 0 && parts[3] > 0) {
      return { width: parts[2], height: parts[3] };
    }
  }
  return null;
}

function updateMinimap(store) {
  const minimapEl = store.dom.minimap;
  const minimapSvgEl = store.dom.minimapSvg;
  if (!minimapEl || minimapEl.classList.contains("hidden")) return;
  if (!minimapSvgEl) return;

  const dims = getDiagramDims(store);
  if (!dims || dims.width <= 0 || dims.height <= 0) {
    minimapSvgEl.innerHTML = "";
    return;
  }

  const svgEl = store.dom.svg;
  const cw = svgEl.clientWidth;
  const ch = svgEl.clientHeight;

  const availableX = MINIMAP_W - MINIMAP_PAD * 2;
  const availableY = MINIMAP_H - MINIMAP_PAD * 2;
  const mmScale = Math.min(availableX / dims.width, availableY / dims.height);
  const mmOffX = (MINIMAP_W - dims.width * mmScale) / 2;
  const mmOffY = (MINIMAP_H - dims.height * mmScale) / 2;

  let html = "";
  html += '<rect class="minimap-bg" x="' + mmOffX + '" y="' + mmOffY +
    '" width="' + (dims.width * mmScale) + '" height="' + (dims.height * mmScale) +
    '" fill="#e9ecef" stroke="#adb5bd" stroke-width="0.5" rx="2"/>';

  const vpX = (-store.viewport.offsetX / store.viewport.zoomScale) * mmScale + mmOffX;
  const vpY = (-store.viewport.offsetY / store.viewport.zoomScale) * mmScale + mmOffY;
  const vpW = (cw / store.viewport.zoomScale) * mmScale;
  const vpH = (ch / store.viewport.zoomScale) * mmScale;

  html += '<rect class="minimap-viewport" x="' + vpX + '" y="' + vpY +
    '" width="' + vpW + '" height="' + vpH +
    '" fill="rgba(52,152,219,0.15)" stroke="#3498db" stroke-width="1" rx="1"/>';

  minimapSvgEl.innerHTML = html;
}

function minimapNavigate(store, evt) {
  const minimapSvgEl = store.dom.minimapSvg;
  if (!minimapSvgEl) return;
  const mRect = minimapSvgEl.getBoundingClientRect();
  const clientX = evt.clientX !== undefined ? evt.clientX : evt.pageX;
  const clientY = evt.clientY !== undefined ? evt.clientY : evt.pageY;
  const clickX = clientX - mRect.left;
  const clickY = clientY - mRect.top;

  const dims = getDiagramDims(store);
  if (!dims || dims.width <= 0 || dims.height <= 0) return;

  const availableX = MINIMAP_W - MINIMAP_PAD * 2;
  const availableY = MINIMAP_H - MINIMAP_PAD * 2;
  const mmScale = Math.min(availableX / dims.width, availableY / dims.height);
  const mmOffX = (MINIMAP_W - dims.width * mmScale) / 2;
  const mmOffY = (MINIMAP_H - dims.height * mmScale) / 2;

  const diagramX = (clickX - mmOffX) / mmScale;
  const diagramY = (clickY - mmOffY) / mmScale;

  const svgEl = store.dom.svg;
  const cw = svgEl.clientWidth;
  const ch = svgEl.clientHeight;

  store.viewport.offsetX = -diagramX * store.viewport.zoomScale + cw / 2;
  store.viewport.offsetY = -diagramY * store.viewport.zoomScale + ch / 2;

  Interaction.applyViewport(store);
}

function toggleMinimap(store, show) {
  const minimapEl = store.dom.minimap;
  const minimapToggleEl = store.dom.minimapToggle;
  if (show === undefined) {
    minimapEl.classList.toggle("hidden");
  } else if (show) {
    minimapEl.classList.remove("hidden");
  } else {
    minimapEl.classList.add("hidden");
  }
  const isHidden = minimapEl.classList.contains("hidden");
  minimapToggleEl.classList.toggle("active", !isHidden);
  if (!isHidden) updateMinimap(store);
}

// ─── Context panel ──────────────────────────────────────────────
function updateContextList(store) {
  if (!store.dom.contextList) return;
  const ctxNodes = store.nodes.filter(function(n) { return n.type === "context"; });
  if (ctxNodes.length === 0) {
    store.dom.contextList.innerHTML = '<div class="context-item" style="color:#999">No contexts</div>';
    return;
  }
  let html = '';
  ctxNodes.forEach(function(ctx) {
    const checked = !store.hiddenContexts[ctx.id];
    html += '<div class="context-item" data-ctx-id="' + ctx.id + '">';
    html += '<input type="checkbox" id="ctx-cb-' + ctx.id + '"' + (checked ? ' checked' : '') + '>';
    html += '<label for="ctx-cb-' + ctx.id + '">' + Renderer.esc(ctx.label) + '</label>';
    html += '</div>';
  });
  store.dom.contextList.innerHTML = html;

  store.dom.contextList.querySelectorAll('.context-item input[type="checkbox"]').forEach(function(cb) {
    cb.addEventListener('change', function() {
      const item = this.closest('.context-item');
      const ctxId = item.getAttribute('data-ctx-id');
      if (this.checked) {
        delete store.hiddenContexts[ctxId];
      } else {
        store.hiddenContexts[ctxId] = true;
      }
      bus.emit('data:changed', { store });
    });
  });
}

function toggleContextPanel(store, show) {
  const panelEl = store.dom.contextPanel;
  const toggleEl = store.dom.contextToggle;
  if (show === undefined) {
    panelEl.classList.toggle("hidden");
  } else if (show) {
    panelEl.classList.remove("hidden");
  } else {
    panelEl.classList.add("hidden");
  }
  const isHidden = panelEl.classList.contains("hidden");
  toggleEl.classList.toggle("active", !isHidden);
  if (!isHidden) updateContextList(store);
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
    html += '<tr><th>Kind</th><td>' + Renderer.esc(node.kind || '—') + '</td></tr>';
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
    html += '<tr><th>Trigger Event</th><td>' + Renderer.esc(node.trigger_event || '—') + '</td></tr>';
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

function hideDetailPanel(store) {
  const el = store.dom.detailPanel;
  if (el) el.style.display = "none";
  store.interaction.selectedNodeId = null;
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
function showContextMenu(store, x, y, aggId, sliceId) {
  const el = store.dom.ctxMenu;
  if (!el) return;
  if (aggId) {
    store.interaction.ctxMenu = { targetAggId: aggId };
    el.innerHTML = '<div class="ctx-menu-item" data-action="add-slice">Add Slice</div>';
  } else if (sliceId) {
    store.interaction.ctxMenu = { targetSliceId: sliceId };
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

// ─── SVG event delegation (click, dblclick, contextmenu, tooltip) ─
function initDelegation(store) {
  const svgEl = store.dom.svg;

  // Tooltip delegation
  svgEl.addEventListener("pointerover", function(evt) {
    const block = evt.target.closest(".diagram-node");
    if (block === hoveredBlock) return;
    hoveredBlock = block;
    if (!block) return;
    const nodeId = block.dataset.nodeId;
    if (!nodeId) return;
    const node = store.nodeById.get(nodeId);
    if (!node || !node.fields || node.fields.length === 0) return;
    if (nodeId === store.interaction.selectedNodeId) return;
    showTooltip(store, node, evt);
  });

  svgEl.addEventListener("pointerout", function(evt) {
    const block = evt.target.closest(".diagram-node");
    if (block && (!evt.relatedTarget || !evt.relatedTarget.closest(".diagram-node"))) {
      hoveredBlock = null;
      hideTooltip(store);
    }
  });

  svgEl.addEventListener("pointermove", function(evt) {
    if (hoveredBlock) positionTooltip(store, evt.clientX, evt.clientY);
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
      const arrow = target.closest(".flow-arrow, .sub-arrow, .auto-trg-arrow, .auto-cmd-arrow, .trg-cmd-arrow, .reads-arrow, .trans-cmd-arrow");
      if (arrow) {
        interactive = true;
        const src = arrow.getAttribute("data-source");
        const tgt = arrow.getAttribute("data-target");
        if (src && tgt) {
          if (store.interaction.selectedNodeId) hideDetailPanel(store);
          highlightElements(store, [src, tgt]);
        }
      }
    }

    if (!interactive && !store.interaction.suppressDetailClick) {
      const block = target.closest(".diagram-node");
      if (block) {
        const nodeId = block.getAttribute("data-node-id");
        if (nodeId) {
          interactive = true;
          const node = store.nodeById.get( nodeId);
          if (node) showDetailPanel(store, node);
        }
      }
    }
    store.interaction.suppressDetailClick = false;

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

    const sliceHeader = target.closest(".slice-header");
    if (sliceHeader) {
      const sliceId = sliceHeader.getAttribute("data-slice-id");
      if (sliceId) {
        evt.preventDefault();
        showContextMenu(store, evt.clientX, evt.clientY, null, sliceId);
        return;
      }
    }

    const sliceArea = target.closest(".slice-area");
    if (sliceArea) {
      const sliceId = sliceArea.getAttribute("data-slice-id");
      if (sliceId) {
        evt.preventDefault();
        showContextMenu(store, evt.clientX, evt.clientY, null, sliceId);
        return;
      }
    }

    const block = target.closest(".diagram-node");
    if (block) return;

    const aggArea = target.closest(".agg-area");
    if (aggArea) {
      const aggId = aggArea.getAttribute("data-agg-id");
      if (aggId) {
        evt.preventDefault();
        showContextMenu(store, evt.clientX, evt.clientY, aggId);
        return;
      }
    }

    const aggLabel = target.closest(".agg-label");
    if (aggLabel) {
      const aggId = aggLabel.getAttribute("data-agg-id");
      if (aggId) {
        evt.preventDefault();
        showContextMenu(store, evt.clientX, evt.clientY, aggId);
        return;
      }
    }

    const ctxHeader = target.closest(".ctx-header");
    if (ctxHeader) {
      const ctxId = ctxHeader.getAttribute("data-ctx-id");
      if (ctxId) {
        for (let i = 0; i < store.nodes.length; i++) {
          if (store.nodes[i].type === "aggregate" && store.nodes[i].parentId === ctxId) {
            evt.preventDefault();
            showContextMenu(store, evt.clientX, evt.clientY, store.nodes[i].id);
            return;
          }
        }
      }
    }

    const ctxLabel = target.closest(".ctx-label");
    if (ctxLabel) {
      const ctxId = ctxLabel.getAttribute("data-ctx-id");
      if (ctxId) {
        for (let i = 0; i < store.nodes.length; i++) {
          if (store.nodes[i].type === "aggregate" && store.nodes[i].parentId === ctxId) {
            evt.preventDefault();
            showContextMenu(store, evt.clientX, evt.clientY, store.nodes[i].id);
            return;
          }
        }
      }
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
      if (store.interaction.selectedNodeId) {
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
  showTooltip,
  hideTooltip,
  positionTooltip,
  renderActorAnnotations,
  updateMinimap,
  minimapNavigate,
  toggleMinimap,
  updateContextList,
  toggleContextPanel,
  updateStats,
  showDetailPanel,
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

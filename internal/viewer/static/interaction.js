import { L, DRAG_THRESHOLD } from './config.js';
import { Layout } from './layout.js';
import { Model } from './model.js';
import { bus } from './bus.js';

function screenToDiagram(svg, viewport, screenX, screenY) {
  const pt = svg.createSVGPoint();
  pt.x = screenX;
  pt.y = screenY;
  const ctm = svg.getScreenCTM().inverse();
  const svgPt = pt.matrixTransform(ctm);
  return {
    x: (svgPt.x - viewport.offsetX) / viewport.zoomScale,
    y: (svgPt.y - viewport.offsetY) / viewport.zoomScale,
  };
}

function clampZoom(scale) {
  return Math.min(5.0, Math.max(0.1, scale));
}

// nodeAtPoint finds the node under a screen point, looking past whatever is
// drawn over it. Arrows are appended after the nodes and so paint on top of
// them, and a command-to-event arrow runs straight through the middle of the
// node below it — taking only the topmost element means a connection dropped on
// a node's centre lands on that arrow and is thrown away.
function nodeAtPoint(x, y) {
  var stack = document.elementsFromPoint
    ? document.elementsFromPoint(x, y)
    : [document.elementFromPoint(x, y)];
  for (var i = 0; i < stack.length; i++) {
    var el = stack[i];
    var block = el && el.closest ? el.closest('.diagram-node') : null;
    if (block) return block;
  }
  return null;
}

function applyViewport(store) {
  const vg = store.dom.svg.querySelector("#viewport-group");
  if (!vg) return;
  vg.setAttribute("transform",
    "translate(" + store.viewport.offsetX + ", " + store.viewport.offsetY + ") scale(" + store.viewport.zoomScale + ")");
  bus.emit('viewport:changed', { store });
}

function fitToView(store) {
  const svgEl = store.dom.svg;
  const vg = svgEl.querySelector("#viewport-group");
  if (!vg) return;

  let bbox;
  try {
    bbox = vg.getBBox();
  } catch(e) { return; }
  if (bbox.width === 0 || bbox.height === 0) return;

  const container = store.dom.svg.parentElement;
  const cw = container.clientWidth;
  const ch = container.clientHeight;
  const padding = 40;

  const scaleX = (cw - padding * 2) / bbox.width;
  const scaleY = (ch - padding * 2) / bbox.height;
  const scale = clampZoom(Math.min(scaleX, scaleY));

  store.viewport.zoomScale = scale;
  store.viewport.offsetX = (cw - bbox.width * scale) / 2 - bbox.x * scale;
  store.viewport.offsetY = (ch - bbox.height * scale) / 2 - bbox.y * scale;

  applyViewport(store);
}

function getConnectedEdges(store, nodeId) {
  return Layout.getConnectedEdges(store.edges, nodeId);
}

function edgeIdOf(source, target) {
  return source + "--" + target;
}

// An arrow is two paths — the one you see and the wide invisible one you aim
// at — so every reshape has to move both or the hit area drifts off the line.
function setArrowPath(svgEl, source, target, d) {
  svgEl.querySelectorAll('path[data-edge-id="' + edgeIdOf(source, target) + '"]')
    .forEach(function(el) { el.setAttribute("d", d); });
}

function visibleArrow(svgEl, source, target) {
  return svgEl.querySelector('path.arrow[data-edge-id="' + edgeIdOf(source, target) + '"]');
}

// Bends the real arrow to the cursor while one of its ends is being dragged,
// so what you are aiming is the arrow itself. The end being dragged is
// modelled as a zero-size box at the pointer.
function repointArrowTo(store, connect, point) {
  const fixedId = connect.handleEnd === "source" ? connect.edgeTarget : connect.edgeSource;
  const fixedPos = store.layoutPositions[fixedId];
  if (!fixedPos) return;
  const loose = { x: point.x, y: point.y, w: 0, h: 0 };
  const srcPos = connect.handleEnd === "source" ? loose : fixedPos;
  const tgtPos = connect.handleEnd === "source" ? fixedPos : loose;
  setArrowPath(store.dom.svg, connect.edgeSource, connect.edgeTarget,
    Layout.computeArrowD(srcPos, tgtPos, false));
}

function updateArrowsForNode(store, nodeId, draggedPos) {
  const svgEl = store.dom.svg;
  const connected = getConnectedEdges(store, nodeId);
  connected.forEach(function(edge) {
    const srcPos = edge.source === nodeId ? draggedPos : store.layoutPositions[edge.source];
    const tgtPos = edge.target === nodeId ? draggedPos : store.layoutPositions[edge.target];
    if (!srcPos || !tgtPos) return;
    const crossBoundary = Layout.isCrossBoundary(store.nodes, edge.source, edge.target);
    let edgeIdx = -1;
    for (let i = 0; i < store.edges.length; i++) { if (store.edges[i] === edge) { edgeIdx = i; break; } }
    const d = Layout.computeArrowD(srcPos, tgtPos, crossBoundary, edgeIdx);
    setArrowPath(svgEl, edge.source, edge.target, d);
  });
}

// A block stays inside its slice on the top and left edges. The other two
// edges need no clamp: the slice grows to whatever the block reaches, so it
// cannot be dragged out through them.
function clampIntoSlice(store, nodeId, x, y) {
  const node = store.nodeById.get(nodeId);
  const sp = node && store.layoutPositions[node.parentId];
  if (!sp) return { x: x, y: y };
  return {
    x: Math.max(sp.x + L.slicePad, x),
    y: Math.max(sp.y + L.sliceTopPad, y),
  };
}

// The box the slice needs in order to hold its blocks, with movingId taken to
// be at target rather than at its laid-out position.
function sliceBoxFor(store, sliceId, movingId, target) {
  const sp = store.layoutPositions[sliceId];
  if (!sp) return null;
  let right = sp.x + (sp.minW || sp.w);
  let bottom = sp.y + (sp.minH || sp.h);
  Layout.getSliceChildNodeIds(store.nodes, sliceId).forEach(function(id) {
    const p = store.layoutPositions[id];
    if (!p) return;
    const x = id === movingId ? target.x : p.x;
    const y = id === movingId ? target.y : p.y;
    right = Math.max(right, x + p.w + L.slicePad);
    bottom = Math.max(bottom, y + p.h + L.slicePad);
  });
  return { w: right - sp.x, h: bottom - sp.y };
}

// Resizing the drawn rects keeps the slice wrapped around the block while the
// pointer is down. Siblings stay put until the drop re-runs the layout, so the
// diagram does not reshuffle under the cursor.
function resizeSliceParts(store, sliceId, w, h) {
  const svgEl = store.dom.svg;
  const sp = store.layoutPositions[sliceId];
  const box = svgEl.querySelector('.slice-box[data-slice-id="' + sliceId + '"]');
  if (box) {
    box.setAttribute("width", w);
    box.setAttribute("height", h);
  }
  svgEl.querySelectorAll('rect.slice-header[data-slice-id="' + sliceId + '"]').forEach(function(el) {
    el.setAttribute("width", w);
  });
  const area = svgEl.querySelector('.slice-area[data-slice-id="' + sliceId + '"]');
  if (area) {
    area.setAttribute("width", w);
    area.setAttribute("height", Math.max(0, h - L.sliceHdrH));
  }
  const label = svgEl.querySelector('text.slice-header[data-slice-id="' + sliceId + '"]');
  if (label && sp) label.setAttribute("x", sp.x + w / 2);
}

function dragNodeTo(store, drag, dx, dy) {
  const base = store.layoutPositions[drag.nodeId];
  if (!base) return;
  const target = clampIntoSlice(store, drag.nodeId, base.x + dx, base.y + dy);

  const blockEl = store.dom.svg.querySelector('.diagram-node[data-node-id="' + drag.nodeId + '"]');
  if (blockEl) {
    blockEl.setAttribute("transform",
      "translate(" + (target.x - base.x) + "," + (target.y - base.y) + ")");
  }
  updateArrowsForNode(store, drag.nodeId, { x: target.x, y: target.y, w: base.w, h: base.h });

  const node = store.nodeById.get(drag.nodeId);
  const box = node && sliceBoxFor(store, node.parentId, drag.nodeId, target);
  if (box) resizeSliceParts(store, node.parentId, box.w, box.h);
}

function commitDrag(store, nodeId, dx, dy) {
  const base = store.layoutPositions[nodeId];
  if (!base) return;
  const target = clampIntoSlice(store, nodeId, base.x + dx, base.y + dy);
  const existing = store.nodeOffsets[nodeId] || { dx: 0, dy: 0 };
  store.nodeOffsets[nodeId] = {
    dx: existing.dx + (target.x - base.x),
    dy: existing.dy + (target.y - base.y),
  };
  const btn = store.dom.resetLayoutBtn;
  if (btn) btn.disabled = false;
  // Re-render rather than patching: the offset resizes the slice, which
  // reflows every slice to its right and the swimlane around them.
  bus.emit('data:changed', { store });
}

function dragSliceTo(store, drag, dx, dy) {
  const svgEl = store.dom.svg;
  if (drag.sliceGroup) {
    drag.sliceGroup.setAttribute("transform", "translate(" + dx + "," + dy + ")");
  }
  drag.nodeIds.forEach(function(nodeId) {
    if (!drag.origPositions[nodeId]) return;
    const blockEl = svgEl.querySelector('.diagram-node[data-node-id="' + nodeId + '"]');
    if (blockEl) blockEl.setAttribute("transform", "translate(" + dx + "," + dy + ")");
  });
  updateArrowsForSlice(store, drag.sliceId, dx, dy);
}

function revertSliceDrag(store, drag) {
  const svgEl = store.dom.svg;
  drag.nodeIds.forEach(function(nodeId) {
    const blockEl = svgEl.querySelector('.diagram-node[data-node-id="' + nodeId + '"]');
    if (blockEl) blockEl.removeAttribute("transform");
    if (drag.origPositions[nodeId]) {
      updateArrowsForNode(store, nodeId, store.layoutPositions[nodeId]);
    }
  });
  if (drag.sliceGroup) {
    drag.sliceGroup.removeAttribute("transform");
    drag.sliceGroup.removeAttribute("opacity");
    if (drag.swimlane) drag.swimlane.appendChild(drag.sliceGroup);
  }
}

function updateArrowsForSlice(store, sliceId, dx, dy) {
  const childIds = Layout.getSliceChildNodeIds(store.nodes, sliceId);
  // Build dragged-position lookup for all children of this slice
  const dragged = {};
  childIds.forEach(function(id) {
    const p = store.layoutPositions[id];
    if (p) dragged[id] = { x: p.x + dx, y: p.y + dy, w: p.w, h: p.h };
  });

  const svgEl = store.dom.svg;
  const seen = {};
  childIds.forEach(function(nodeId) {
    if (!dragged[nodeId]) return;
    const connected = getConnectedEdges(store, nodeId);
    connected.forEach(function(edge) {
      var key = edge.source + "\x00" + edge.target;
      if (seen[key]) return;
      seen[key] = true;

      const srcPos = dragged[edge.source] || store.layoutPositions[edge.source];
      const tgtPos = dragged[edge.target] || store.layoutPositions[edge.target];
      if (!srcPos || !tgtPos) return;

      const crossBoundary = Layout.isCrossBoundary(store.nodes, edge.source, edge.target);
      var edgeIdx = -1;
      for (var i = 0; i < store.edges.length; i++) { if (store.edges[i] === edge) { edgeIdx = i; break; } }
      const d = Layout.computeArrowD(srcPos, tgtPos, crossBoundary, edgeIdx);
      setArrowPath(svgEl, edge.source, edge.target, d);
    });
  });
}

function applyRepoint(store, connect, targetId) {
  for (var i = 0; i < store.edges.length; i++) {
    var e = store.edges[i];
    if (e.source !== connect.edgeSource || e.target !== connect.edgeTarget) continue;
    if (e.type !== connect.edgeType) continue;
    if (connect.handleEnd === "source") {
      if (targetId === connect.edgeTarget) return false;
      e.source = targetId;
    } else {
      if (targetId === connect.edgeSource) return false;
      e.target = targetId;
    }
    return true;
  }
  return false;
}

function addEdge(store, sourceId, targetId) {
  if (targetId === sourceId) return false;
  for (var i = 0; i < store.edges.length; i++) {
    if (store.edges[i].source === sourceId && store.edges[i].target === targetId) return false;
  }
  store.edges.push({
    source: sourceId,
    target: targetId,
    type: Model.autoDetectEdgeType(store, sourceId, targetId),
  });
  return true;
}

function settleConnect(store, connect, clientX, clientY) {
  var dragged = connect.startClientX !== undefined && Math.sqrt(
    Math.pow(clientX - connect.startClientX, 2) +
    Math.pow(clientY - connect.startClientY, 2)
  ) >= DRAG_THRESHOLD;

  var targetBlock = dragged ? nodeAtPoint(clientX, clientY) : null;
  var targetId = targetBlock && targetBlock.getAttribute("data-node-id");

  if (connect.repoint) {
    if (targetId) applyRepoint(store, connect, targetId);
    // Redraw whatever the drop decided: the arrow is currently bent to the
    // cursor, so an abandoned repoint would otherwise stay hanging there.
    bus.emit('data:changed', { store });
    return;
  }

  if (targetId && addEdge(store, connect.sourceId, targetId)) {
    bus.emit('data:changed', { store });
  }
}

function tryReorderSliceOnDrop(store, sliceId, dx) {
  var sl = store.nodeById.get(sliceId);
  if (!sl) return false;
  var sp = store.layoutPositions[sliceId];
  if (!sp || sp.w === 0) return false;
  if (Math.abs(dx) <= sp.w * 0.3) return false;

  var aggId = sl.parentId;
  var siblings = [];
  store.nodes.forEach(function(n) {
    if (n.parentId === aggId && n.type === "slice") {
      var pos = store.layoutPositions[n.id];
      if (pos) siblings.push({ id: n.id, x: pos.x, w: pos.w });
    }
  });
  siblings.sort(function(a, b) { return a.x - b.x; });
  if (siblings.length < 2) return false;

  var currentPos = -1;
  for (var i = 0; i < siblings.length; i++) {
    if (siblings[i].id === sliceId) { currentPos = i; break; }
  }
  if (currentPos === -1) return false;

  var dropCenter = sp.x + sp.w / 2 + dx;

  var targetPos = siblings.length - 1;
  for (var p = 0; p < siblings.length - 1; p++) {
    var thisCenter = siblings[p].x + siblings[p].w / 2;
    var nextCenter = siblings[p + 1].x + siblings[p + 1].w / 2;
    if (dropCenter < (thisCenter + nextCenter) / 2) {
      targetPos = p;
      break;
    }
  }

  if (targetPos === currentPos) return false;

  var moved = Model.moveSlice(store.nodes, sliceId, targetPos);
  if (moved) {
    bus.emit('data:changed', { store });
  }
  return moved;
}

function initEventListeners(store) {
  const svgEl = store.dom.svg;

  svgEl.addEventListener("wheel", function(evt) {
    if (store.interaction.touch) return;
    evt.preventDefault();
    const dp = screenToDiagram(svgEl, store.viewport, evt.clientX, evt.clientY);
    const oldZoom = store.viewport.zoomScale;
    const newZoom = clampZoom(oldZoom * Math.pow(1.001, -evt.deltaY));
    if (newZoom !== oldZoom) {
      store.viewport.offsetX = dp.x * (oldZoom - newZoom) + store.viewport.offsetX;
      store.viewport.offsetY = dp.y * (oldZoom - newZoom) + store.viewport.offsetY;
      store.viewport.zoomScale = newZoom;
      applyViewport(store);
    }
  }, { passive: false });

  function createPreviewLine() {
    var line = document.createElementNS("http://www.w3.org/2000/svg", "path");
    line.setAttribute("fill", "none");
    line.setAttribute("stroke", "#3498db");
    line.setAttribute("stroke-width", "2");
    line.setAttribute("stroke-dasharray", "6,4");
    line.setAttribute("class", "connect-preview");
    line.setAttribute("pointer-events", "none");
    return line;
  }

  function updatePreviewLine(lineEl, x1, y1, x2, y2) {
    lineEl.setAttribute("d", "M " + x1 + "," + y1 + " L " + x2 + "," + y2);
  }

  function removePreviewLine(store) {
    var prev = svgEl.querySelector(".connect-preview");
    if (prev) prev.remove();
  }

  function addPreviewLine(store, x1, y1, x2, y2) {
    var line = createPreviewLine();
    updatePreviewLine(line, x1, y1, x2, y2);
    var vg = svgEl.querySelector("#viewport-group");
    if (vg) vg.appendChild(line);
    return line;
  }

  svgEl.addEventListener("mousedown", function(evt) {
    if (evt.button !== 0) return;

    // Check for connection port click
    var port = evt.target.closest('[data-port]');
    if (port) {
      var nodeId = port.getAttribute("data-node-id");
      var portType = port.getAttribute("data-port");
      var pos = store.layoutPositions[nodeId];
      if (nodeId && pos) {
        var dp = screenToDiagram(svgEl, store.viewport, evt.clientX, evt.clientY);
        var cx = portType === "out" ? pos.x + pos.w : pos.x;
        var cy = pos.y + pos.h / 2;
        store.interaction.connect = {
          sourceId: nodeId,
          portType: portType,
          startX: cx,
          startY: cy,
          currentX: dp.x,
          currentY: dp.y,
          startClientX: evt.clientX,
          startClientY: evt.clientY,
        };
        var line = addPreviewLine(store, cx, cy, dp.x, dp.y);
        evt.preventDefault();
      }
      return;
    }

    // Check for arrow handle click (repoint)
    var handle = evt.target.closest('[data-arrow-handle]');
    if (handle) {
      var edgeSource = handle.getAttribute("data-edge-source");
      var edgeTarget = handle.getAttribute("data-edge-target");
      var edgeType = handle.getAttribute("data-edge-type");
      var handleEnd = handle.getAttribute("data-arrow-handle");
      if (edgeSource && edgeTarget) {
        var dp = screenToDiagram(svgEl, store.viewport, evt.clientX, evt.clientY);
        var hx = parseFloat(handle.getAttribute("cx"));
        var hy = parseFloat(handle.getAttribute("cy"));
        store.interaction.connect = {
          repoint: true,
          edgeSource: edgeSource,
          edgeTarget: edgeTarget,
          edgeType: edgeType,
          handleEnd: handleEnd,
          startX: hx,
          startY: hy,
          currentX: dp.x,
          currentY: dp.y,
          startClientX: evt.clientX,
          startClientY: evt.clientY,
        };
        evt.preventDefault();
      }
      return;
    }

    const block = evt.target.closest('.diagram-node');
    if (block) {
      const nodeId = block.getAttribute("data-node-id");
      if (nodeId && store.layoutPositions[nodeId]) {
        const dp = screenToDiagram(svgEl, store.viewport, evt.clientX, evt.clientY);
        store.interaction.drag = {
          nodeId: nodeId,
          startDiagramX: dp.x,
          startDiagramY: dp.y,
          isDragging: false,
          startClientX: evt.clientX,
          startClientY: evt.clientY,
        };
        block.classList.add("dragging");
        evt.preventDefault();
      }
      return;
    }

    const sliceHeader = evt.target.closest('.slice-header');
    if (sliceHeader) {
      const sliceId = sliceHeader.getAttribute("data-slice-id");
      if (sliceId && store.layoutPositions[sliceId]) {
        const dp = screenToDiagram(svgEl, store.viewport, evt.clientX, evt.clientY);
        const childIds = Layout.getSliceChildNodeIds(store.nodes, sliceId);
        const origPositions = {};
        childIds.forEach(function(id) {
          if (store.layoutPositions[id]) {
            origPositions[id] = { x: store.layoutPositions[id].x, y: store.layoutPositions[id].y };
          }
        });
        var sliceGroup = svgEl.querySelector('g.slice-' + sliceId);
        var swimlane = sliceGroup ? sliceGroup.parentNode : null;
        store.interaction.drag = {
          sliceId: sliceId,
          nodeIds: childIds,
          origPositions: origPositions,
          sliceGroup: sliceGroup,
          swimlane: swimlane,
          startDiagramX: dp.x,
          startDiagramY: dp.y,
          isDragging: false,
          startClientX: evt.clientX,
          startClientY: evt.clientY,
        };
        const headerEls = svgEl.querySelectorAll('.slice-header[data-slice-id="' + sliceId + '"]');
        headerEls.forEach(function(el) { el.classList.add("dragging"); });
        evt.preventDefault();
      }
      return;
    }

    if (evt.target.closest('.arrow-hit, .ctx-label, .agg-label')) return;

    store.interaction.pan = {
      startX: evt.clientX,
      startY: evt.clientY,
      startOffsetX: store.viewport.offsetX,
      startOffsetY: store.viewport.offsetY,
    };
    svgEl.classList.add("panning");
    evt.preventDefault();
  });

  document.addEventListener("mousemove", function(evt) {
    const connect = store.interaction.connect;
    if (connect) {
      var dp = screenToDiagram(svgEl, store.viewport, evt.clientX, evt.clientY);
      connect.currentX = dp.x;
      connect.currentY = dp.y;
      if (connect.repoint) {
        repointArrowTo(store, connect, dp);
      } else {
        var preview = svgEl.querySelector(".connect-preview");
        if (preview) {
          updatePreviewLine(preview, connect.startX, connect.startY, dp.x, dp.y);
        }
      }
      evt.preventDefault();
      return;
    }

    const drag = store.interaction.drag;
    if (drag) {
      const dp = screenToDiagram(svgEl, store.viewport, evt.clientX, evt.clientY);
      const dx = dp.x - drag.startDiagramX;
      const dy = dp.y - drag.startDiagramY;

      if (!drag.isDragging) {
        const dist = Math.sqrt(
          Math.pow(evt.clientX - drag.startClientX, 2) +
          Math.pow(evt.clientY - drag.startClientY, 2)
        );
        if (dist >= DRAG_THRESHOLD) {
          drag.isDragging = true;
          if (drag.sliceId && drag.sliceGroup) {
            var vg = svgEl.querySelector("#viewport-group");
            if (vg) vg.appendChild(drag.sliceGroup);
            drag.sliceGroup.setAttribute("opacity", "0.85");
            drag.nodeIds.forEach(function(id) {
              var blockEl = svgEl.querySelector('.diagram-node[data-node-id="' + id + '"]');
              if (blockEl) {
                blockEl.classList.add("dragging");
                if (vg) vg.appendChild(blockEl);
              }
            });
            var seenArrows = {};
            drag.nodeIds.forEach(function(id) {
              var connected = getConnectedEdges(store, id);
              connected.forEach(function(edge) {
                var key = edge.source + "\x00" + edge.target;
                if (seenArrows[key]) return;
                seenArrows[key] = true;
                // Only the drawn arrow is lifted; its hit path belongs under
                // the blocks and moving it would let arrows steal their clicks.
                var pathEl = visibleArrow(svgEl, edge.source, edge.target);
                if (pathEl && vg) vg.appendChild(pathEl);
              });
            });
          }
        } else {
          return;
        }
      }

      if (drag.sliceId) {
        dragSliceTo(store, drag, dx, dy);
      } else {
        dragNodeTo(store, drag, dx, dy);
      }

      evt.preventDefault();
      return;
    }

    const pan = store.interaction.pan;
    if (!pan) return;

    store.viewport.offsetX = pan.startOffsetX + (evt.clientX - pan.startX);
    store.viewport.offsetY = pan.startOffsetY + (evt.clientY - pan.startY);
    applyViewport(store);
    evt.preventDefault();
  });

  document.addEventListener("mouseup", function(evt) {
    const connect = store.interaction.connect;
    if (connect) {
      removePreviewLine(store);
      settleConnect(store, connect, evt.clientX, evt.clientY);
      store.interaction.connect = null;
      evt.preventDefault();
      return;
    }

    const drag = store.interaction.drag;
    if (drag) {
      if (drag.sliceId) {
        const headerEls = svgEl.querySelectorAll('.slice-header[data-slice-id="' + drag.sliceId + '"]');
        if (drag.isDragging) {
          const dp = screenToDiagram(svgEl, store.viewport, evt.clientX, evt.clientY);
          const dx = dp.x - drag.startDiagramX;
          const dy = dp.y - drag.startDiagramY;
          if (!tryReorderSliceOnDrop(store, drag.sliceId, dx)) {
            revertSliceDrag(store, drag);
          }
          store.interaction.suppressDetailClick = true;
        }
        drag.nodeIds.forEach(function(nodeId) {
          const blockEl = svgEl.querySelector('.diagram-node[data-node-id="' + nodeId + '"]');
          if (blockEl) blockEl.classList.remove("dragging");
        });
        headerEls.forEach(function(el) { el.classList.remove("dragging"); });
        store.interaction.drag = null;
        evt.preventDefault();
        return;
      }

      const blockEl = svgEl.querySelector('.diagram-node[data-node-id="' + drag.nodeId + '"]');
      if (drag.isDragging) {
        const dp = screenToDiagram(svgEl, store.viewport, evt.clientX, evt.clientY);
        const dx = dp.x - drag.startDiagramX;
        const dy = dp.y - drag.startDiagramY;
        commitDrag(store, drag.nodeId, dx, dy);
        store.interaction.suppressDetailClick = true;
      }
      if (blockEl) blockEl.classList.remove("dragging");
      store.interaction.drag = null;
      evt.preventDefault();
      return;
    }

    const pan = store.interaction.pan;
    if (!pan) return;

    store.interaction.pan = null;
    svgEl.classList.remove("panning");
    evt.preventDefault();
  });

  svgEl.addEventListener("touchstart", function(evt) {
    evt.preventDefault();
    const touches = evt.touches;

    if (store.interaction.touch) return;

    if (touches.length === 1) {
      // Check for connection port
      var port = evt.target.closest('[data-port]');
      if (port) {
        var nodeId = port.getAttribute("data-node-id");
        var portType = port.getAttribute("data-port");
        var pos = store.layoutPositions[nodeId];
        if (nodeId && pos) {
          var touch = touches[0];
          var dp = screenToDiagram(svgEl, store.viewport, touch.clientX, touch.clientY);
          var cx = portType === "out" ? pos.x + pos.w : pos.x;
          var cy = pos.y + pos.h / 2;
          var t = touches[0];
          store.interaction.connect = {
            sourceId: nodeId,
            portType: portType,
            startX: cx,
            startY: cy,
            currentX: dp.x,
            currentY: dp.y,
            startClientX: t.clientX,
            startClientY: t.clientY,
          };
          var line = addPreviewLine(store, cx, cy, dp.x, dp.y);
        }
        return;
      }

      // Check for arrow handle (repoint)
      var handle = evt.target.closest('[data-arrow-handle]');
      if (handle) {
        var edgeSource = handle.getAttribute("data-edge-source");
        var edgeTarget = handle.getAttribute("data-edge-target");
        var edgeType = handle.getAttribute("data-edge-type");
        var handleEnd = handle.getAttribute("data-arrow-handle");
        if (edgeSource && edgeTarget) {
          var t = touches[0];
          var dp = screenToDiagram(svgEl, store.viewport, t.clientX, t.clientY);
          var hx = parseFloat(handle.getAttribute("cx"));
          var hy = parseFloat(handle.getAttribute("cy"));
          store.interaction.connect = {
            repoint: true,
            edgeSource: edgeSource,
            edgeTarget: edgeTarget,
            edgeType: edgeType,
            handleEnd: handleEnd,
            startX: hx,
            startY: hy,
            currentX: dp.x,
            currentY: dp.y,
            startClientX: t.clientX,
            startClientY: t.clientY,
          };
          }
        return;
      }

      const block = evt.target.closest('.diagram-node');
      if (block) {
        const nodeId = block.getAttribute("data-node-id");
        if (nodeId && store.layoutPositions[nodeId]) {
          const touch = touches[0];
          const dp = screenToDiagram(svgEl, store.viewport, touch.clientX, touch.clientY);
          store.interaction.drag = {
            nodeId: nodeId,
            startDiagramX: dp.x,
            startDiagramY: dp.y,
            isDragging: false,
            startClientX: touch.clientX,
            startClientY: touch.clientY,
          };
          block.classList.add("dragging");
        }
        return;
      }

      const sliceHeader = evt.target.closest('.slice-header');
      if (sliceHeader) {
        const sliceId = sliceHeader.getAttribute("data-slice-id");
        if (sliceId && store.layoutPositions[sliceId]) {
          const touch = touches[0];
          const dp = screenToDiagram(svgEl, store.viewport, touch.clientX, touch.clientY);
          const childIds = Layout.getSliceChildNodeIds(store.nodes, sliceId);
          const origPositions = {};
          childIds.forEach(function(id) {
            if (store.layoutPositions[id]) {
              origPositions[id] = { x: store.layoutPositions[id].x, y: store.layoutPositions[id].y };
            }
          });
          var sliceGroup = svgEl.querySelector('g.slice-' + sliceId);
          var swimlane = sliceGroup ? sliceGroup.parentNode : null;
          const headerEls = svgEl.querySelectorAll('.slice-header[data-slice-id="' + sliceId + '"]');
          headerEls.forEach(function(el) { el.classList.add("dragging"); });
          store.interaction.drag = {
            sliceId: sliceId,
            nodeIds: childIds,
            origPositions: origPositions,
            sliceGroup: sliceGroup,
            swimlane: swimlane,
            startDiagramX: dp.x,
            startDiagramY: dp.y,
            isDragging: false,
            startClientX: touch.clientX,
            startClientY: touch.clientY,
          };
        }
        return;
      }

      if (evt.target.closest('.arrow-hit, .ctx-label, .agg-label')) return;
      store.interaction.touch = {
        mode: "pan",
        startX: touches[0].clientX,
        startY: touches[0].clientY,
        startOffsetX: store.viewport.offsetX,
        startOffsetY: store.viewport.offsetY,
      };
      svgEl.classList.add("panning");
    } else if (touches.length === 2) {
      svgEl.classList.remove("panning");
      const t1 = touches[0], t2 = touches[1];
      const dx = t2.clientX - t1.clientX;
      const dy = t2.clientY - t1.clientY;
      store.interaction.touch = {
        mode: "pinch",
        startDist: Math.sqrt(dx * dx + dy * dy),
        startZoom: store.viewport.zoomScale,
        startOffsetX: store.viewport.offsetX,
        startOffsetY: store.viewport.offsetY,
      };
    }
  }, { passive: false });

  svgEl.addEventListener("touchmove", function(evt) {
    const connect = store.interaction.connect;
    if (connect) {
      evt.preventDefault();
      const touch = evt.touches[0];
      if (!touch) return;
      const dp = screenToDiagram(svgEl, store.viewport, touch.clientX, touch.clientY);
      connect.currentX = dp.x;
      connect.currentY = dp.y;
      if (connect.repoint) {
        repointArrowTo(store, connect, dp);
      } else {
        var preview = svgEl.querySelector(".connect-preview");
        if (preview) {
          updatePreviewLine(preview, connect.startX, connect.startY, dp.x, dp.y);
        }
      }
      return;
    }

    const drag = store.interaction.drag;
    if (drag) {
      evt.preventDefault();
      const touch = evt.touches[0];
      if (!touch) return;

      const dp = screenToDiagram(svgEl, store.viewport, touch.clientX, touch.clientY);
      const dx = dp.x - drag.startDiagramX;
      const dy = dp.y - drag.startDiagramY;

      if (!drag.isDragging) {
        const dist = Math.sqrt(
          Math.pow(touch.clientX - drag.startClientX, 2) +
          Math.pow(touch.clientY - drag.startClientY, 2)
        );
        if (dist >= DRAG_THRESHOLD) {
          drag.isDragging = true;
          if (drag.sliceId && drag.sliceGroup) {
            var vg = svgEl.querySelector("#viewport-group");
            if (vg) vg.appendChild(drag.sliceGroup);
            drag.sliceGroup.setAttribute("opacity", "0.85");
            drag.nodeIds.forEach(function(id) {
              var blockEl = svgEl.querySelector('.diagram-node[data-node-id="' + id + '"]');
              if (blockEl) {
                blockEl.classList.add("dragging");
                if (vg) vg.appendChild(blockEl);
              }
            });
            var seenArrows = {};
            drag.nodeIds.forEach(function(id) {
              var connected = getConnectedEdges(store, id);
              connected.forEach(function(edge) {
                var key = edge.source + "\x00" + edge.target;
                if (seenArrows[key]) return;
                seenArrows[key] = true;
                // Only the drawn arrow is lifted; its hit path belongs under
                // the blocks and moving it would let arrows steal their clicks.
                var pathEl = visibleArrow(svgEl, edge.source, edge.target);
                if (pathEl && vg) vg.appendChild(pathEl);
              });
            });
          }
        } else {
          return;
        }
      }

      if (drag.sliceId) {
        dragSliceTo(store, drag, dx, dy);
      } else {
        dragNodeTo(store, drag, dx, dy);
      }
      return;
    }

    const touch = store.interaction.touch;
    if (!touch) return;
    evt.preventDefault();

    const touches = evt.touches;

    if (touch.mode === "pan") {
      if (touches.length === 1) {
        store.viewport.offsetX = touch.startOffsetX + (touches[0].clientX - touch.startX);
        store.viewport.offsetY = touch.startOffsetY + (touches[0].clientY - touch.startY);
        applyViewport(store);
      } else if (touches.length >= 2) {
        svgEl.classList.remove("panning");
        const t1 = touches[0], t2 = touches[1];
        const dx = t2.clientX - t1.clientX;
        const dy = t2.clientY - t1.clientY;
        store.interaction.touch = {
          mode: "pinch",
          startDist: Math.sqrt(dx * dx + dy * dy),
          startZoom: store.viewport.zoomScale,
          startOffsetX: store.viewport.offsetX,
          startOffsetY: store.viewport.offsetY,
        };
      }
    } else if (touch.mode === "pinch") {
      if (touches.length >= 2) {
        const t1 = touches[0], t2 = touches[1];
        const dx = t2.clientX - t1.clientX;
        const dy = t2.clientY - t1.clientY;
        const dist = Math.sqrt(dx * dx + dy * dy);
        if (touch.startDist === 0) return;
        const scale = dist / touch.startDist;
        const newZoom = clampZoom(touch.startZoom * scale);

        const oldZoom = store.viewport.zoomScale;
        if (newZoom !== oldZoom) {
          const midX = (t1.clientX + t2.clientX) / 2;
          const midY = (t1.clientY + t2.clientY) / 2;
          const dp = screenToDiagram(svgEl, store.viewport, midX, midY);
          store.viewport.offsetX = dp.x * (oldZoom - newZoom) + store.viewport.offsetX;
          store.viewport.offsetY = dp.y * (oldZoom - newZoom) + store.viewport.offsetY;
          store.viewport.zoomScale = newZoom;
          applyViewport(store);
        }
      } else if (touches.length === 1) {
        store.interaction.touch = {
          mode: "pan",
          startX: touches[0].clientX,
          startY: touches[0].clientY,
          startOffsetX: store.viewport.offsetX,
          startOffsetY: store.viewport.offsetY,
        };
        svgEl.classList.add("panning");
      }
    }
  }, { passive: false });

  svgEl.addEventListener("touchend", function(evt) {
    const connect = store.interaction.connect;
    if (connect) {
      evt.preventDefault();
      removePreviewLine(store);
      var ct = evt.changedTouches[0];
      if (ct) settleConnect(store, connect, ct.clientX, ct.clientY);
      store.interaction.connect = null;
      return;
    }

    const drag = store.interaction.drag;
    if (drag) {
      evt.preventDefault();
      if (drag.sliceId) {
        const headerEls = svgEl.querySelectorAll('.slice-header[data-slice-id="' + drag.sliceId + '"]');
        if (drag.isDragging) {
          const touch = evt.changedTouches[0];
          if (touch) {
            const dp = screenToDiagram(svgEl, store.viewport, touch.clientX, touch.clientY);
            const dx = dp.x - drag.startDiagramX;
            const dy = dp.y - drag.startDiagramY;
            if (!tryReorderSliceOnDrop(store, drag.sliceId, dx)) {
              revertSliceDrag(store, drag);
            }
            store.interaction.suppressDetailClick = true;
          }
        }
        drag.nodeIds.forEach(function(nodeId) {
          const blockEl = svgEl.querySelector('.diagram-node[data-node-id="' + nodeId + '"]');
          if (blockEl) blockEl.classList.remove("dragging");
        });
        headerEls.forEach(function(el) { el.classList.remove("dragging"); });
        store.interaction.drag = null;
        return;
      }

      const blockEl = svgEl.querySelector('.diagram-node[data-node-id="' + drag.nodeId + '"]');
      if (drag.isDragging) {
        const touch = evt.changedTouches[0];
        if (touch) {
          const dp = screenToDiagram(svgEl, store.viewport, touch.clientX, touch.clientY);
          const dx = dp.x - drag.startDiagramX;
          const dy = dp.y - drag.startDiagramY;
          commitDrag(store, drag.nodeId, dx, dy);
          store.interaction.suppressDetailClick = true;
        }
      }
      if (blockEl) blockEl.classList.remove("dragging");
      store.interaction.drag = null;
      return;
    }

    const touch = store.interaction.touch;
    if (!touch) return;
    evt.preventDefault();

    if (evt.touches.length === 0) {
      store.interaction.touch = null;
      svgEl.classList.remove("panning");
    } else if (evt.touches.length === 1 && touch.mode === "pinch") {
      store.interaction.touch = {
        mode: "pan",
        startX: evt.touches[0].clientX,
        startY: evt.touches[0].clientY,
        startOffsetX: store.viewport.offsetX,
        startOffsetY: store.viewport.offsetY,
      };
      svgEl.classList.add("panning");
    }
  }, { passive: false });

  svgEl.addEventListener("touchcancel", function() {
    if (store.interaction.connect) {
      removePreviewLine(store);
      store.interaction.connect = null;
    }

    const drag = store.interaction.drag;
    if (drag) {
      if (drag.sliceId) {
        const headerEls = svgEl.querySelectorAll('.slice-header[data-slice-id="' + drag.sliceId + '"]');
        if (drag.isDragging) {
          const touch = evt.changedTouches[0];
          if (touch) {
            const dp = screenToDiagram(svgEl, store.viewport, touch.clientX, touch.clientY);
            const dx = dp.x - drag.startDiagramX;
            const dy = dp.y - drag.startDiagramY;
            if (!tryReorderSliceOnDrop(store, drag.sliceId, dx)) {
              revertSliceDrag(store, drag);
            }
            store.interaction.suppressDetailClick = true;
          }
        }
        drag.nodeIds.forEach(function(nodeId) {
          const blockEl = svgEl.querySelector('.diagram-node[data-node-id="' + nodeId + '"]');
          if (blockEl) blockEl.classList.remove("dragging");
        });
        headerEls.forEach(function(el) { el.classList.remove("dragging"); });
        store.interaction.drag = null;
      } else {
        const blockEl = svgEl.querySelector('.diagram-node[data-node-id="' + drag.nodeId + '"]');
        if (blockEl) blockEl.classList.remove("dragging");
        store.interaction.drag = null;
      }
    }
    store.interaction.touch = null;
    svgEl.classList.remove("panning");
  });
}

export const Interaction = {
  initEventListeners,
  screenToDiagram,
  clampZoom,
  applyViewport,
  fitToView,
};

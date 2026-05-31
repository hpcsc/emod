import { L, DRAG_THRESHOLD, arrowClassMap } from './config.js';
import { Layout } from './layout.js';
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
    const arrowCls = arrowClassMap[edge.type] || "flow-arrow";
    const pathEl = svgEl.querySelector('.' + arrowCls + '[data-source="' + edge.source + '"][data-target="' + edge.target + '"]');
    if (pathEl) {
      pathEl.setAttribute("d", d);
    }
  });
}

function updateBlockTransform(store, nodeId) {
  const svgEl = store.dom.svg;
  const blockEl = svgEl.querySelector('.cmd-block[data-node-id="' + nodeId + '"], .evt-block[data-node-id="' + nodeId + '"], .trg-block[data-node-id="' + nodeId + '"], .view-block[data-node-id="' + nodeId + '"], .auto-block[data-node-id="' + nodeId + '"], .trans-block[data-node-id="' + nodeId + '"]');
  if (!blockEl) return;
  const off = store.nodeOffsets[nodeId];
  if (off) {
    blockEl.setAttribute("transform", "translate(" + off.dx + "," + off.dy + ")");
  } else {
    blockEl.removeAttribute("transform");
  }
}

function commitDrag(store, nodeId, dx, dy, origNodeX, origNodeY) {
  const newX = origNodeX + dx;
  const newY = origNodeY + dy;
  store.layoutPositions[nodeId].x = newX;
  store.layoutPositions[nodeId].y = newY;
  const existing = store.nodeOffsets[nodeId] || { dx: 0, dy: 0 };
  store.nodeOffsets[nodeId] = { dx: existing.dx + dx, dy: existing.dy + dy };
  updateBlockTransform(store, nodeId);
  updateArrowsForNode(store, nodeId, store.layoutPositions[nodeId]);
  const btn = store.dom.resetLayoutBtn;
  if (btn) btn.disabled = false;
}

function updateArrowsForSlice(store, sliceId, dx, dy) {
  const childIds = Layout.getSliceChildNodeIds(store.nodes, sliceId);
  childIds.forEach(function(nodeId) {
    const orig = store.layoutPositions[nodeId];
    if (!orig) return;
    const draggedPos = { x: orig.x + dx, y: orig.y + dy, w: orig.w, h: orig.h };
    updateArrowsForNode(store, nodeId, draggedPos);
  });
}

function commitSliceDrag(store, sliceId, dx, dy) {
  const childIds = Layout.getSliceChildNodeIds(store.nodes, sliceId);
  childIds.forEach(function(nodeId) {
    const orig = store.layoutPositions[nodeId];
    if (!orig) return;
    commitDrag(store, nodeId, dx, dy, orig.x, orig.y);
  });
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

  svgEl.addEventListener("mousedown", function(evt) {
    if (evt.button !== 0) return;

    const block = evt.target.closest('.cmd-block, .evt-block, .trg-block, .view-block, .auto-block, .trans-block');
    if (block) {
      const nodeId = block.getAttribute("data-node-id");
      if (nodeId && store.layoutPositions[nodeId]) {
        const dp = screenToDiagram(svgEl, store.viewport, evt.clientX, evt.clientY);
        store.interaction.drag = {
          nodeId: nodeId,
          startDiagramX: dp.x,
          startDiagramY: dp.y,
          origNodeX: store.layoutPositions[nodeId].x,
          origNodeY: store.layoutPositions[nodeId].y,
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
        store.interaction.drag = {
          sliceId: sliceId,
          nodeIds: childIds,
          origPositions: origPositions,
          startDiagramX: dp.x,
          startDiagramY: dp.y,
          isDragging: false,
          startClientX: evt.clientX,
          startClientY: evt.clientY,
        };
        childIds.forEach(function(id) {
          const blockEl = svgEl.querySelector('.cmd-block[data-node-id="' + id + '"], .evt-block[data-node-id="' + id + '"], .trg-block[data-node-id="' + id + '"], .view-block[data-node-id="' + id + '"], .auto-block[data-node-id="' + id + '"], .trans-block[data-node-id="' + id + '"]');
          if (blockEl) blockEl.classList.add("dragging");
        });
        const headerEls = svgEl.querySelectorAll('.slice-header[data-slice-id="' + sliceId + '"]');
        headerEls.forEach(function(el) { el.classList.add("dragging"); });
        evt.preventDefault();
      }
      return;
    }

    if (evt.target.closest('.flow-arrow, .sub-arrow, .auto-trg-arrow, .auto-cmd-arrow, .trg-cmd-arrow, .reads-arrow, .trans-cmd-arrow, .ctx-label, .agg-label')) return;

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
        } else {
          return;
        }
      }

      if (drag.sliceId) {
        drag.nodeIds.forEach(function(nodeId) {
          const orig = drag.origPositions[nodeId];
          if (!orig) return;
          const blockEl = svgEl.querySelector('.cmd-block[data-node-id="' + nodeId + '"], .evt-block[data-node-id="' + nodeId + '"], .trg-block[data-node-id="' + nodeId + '"], .view-block[data-node-id="' + nodeId + '"], .auto-block[data-node-id="' + nodeId + '"], .trans-block[data-node-id="' + nodeId + '"]');
          if (blockEl) {
            const existingOff = store.nodeOffsets[nodeId] || {dx: 0, dy: 0};
            blockEl.setAttribute("transform", "translate(" + (existingOff.dx + dx) + "," + (existingOff.dy + dy) + ")");
          }
        });
        updateArrowsForSlice(store, drag.sliceId, dx, dy);
      } else {
        const blockEl = svgEl.querySelector('.cmd-block[data-node-id="' + drag.nodeId + '"], .evt-block[data-node-id="' + drag.nodeId + '"], .trg-block[data-node-id="' + drag.nodeId + '"], .view-block[data-node-id="' + drag.nodeId + '"], .auto-block[data-node-id="' + drag.nodeId + '"], .trans-block[data-node-id="' + drag.nodeId + '"]');
        if (blockEl) {
          const existingOff = store.nodeOffsets[drag.nodeId] || {dx: 0, dy: 0};
          blockEl.setAttribute("transform", "translate(" + (existingOff.dx + dx) + "," + (existingOff.dy + dy) + ")");
        }
        updateArrowsForNode(store, drag.nodeId, {
          x: drag.origNodeX + dx,
          y: drag.origNodeY + dy,
          w: store.layoutPositions[drag.nodeId].w,
          h: store.layoutPositions[drag.nodeId].h,
        });
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
    const drag = store.interaction.drag;
    if (drag) {
      if (drag.sliceId) {
        const headerEls = svgEl.querySelectorAll('.slice-header[data-slice-id="' + drag.sliceId + '"]');
        if (drag.isDragging) {
          const dp = screenToDiagram(svgEl, store.viewport, evt.clientX, evt.clientY);
          const dx = dp.x - drag.startDiagramX;
          const dy = dp.y - drag.startDiagramY;
          commitSliceDrag(store, drag.sliceId, dx, dy);
          store.interaction.suppressDetailClick = true;
        }
        drag.nodeIds.forEach(function(nodeId) {
          const blockEl = svgEl.querySelector('.cmd-block[data-node-id="' + nodeId + '"], .evt-block[data-node-id="' + nodeId + '"], .trg-block[data-node-id="' + nodeId + '"], .view-block[data-node-id="' + nodeId + '"], .auto-block[data-node-id="' + nodeId + '"], .trans-block[data-node-id="' + nodeId + '"]');
          if (blockEl) blockEl.classList.remove("dragging");
        });
        headerEls.forEach(function(el) { el.classList.remove("dragging"); });
        store.interaction.drag = null;
        evt.preventDefault();
        return;
      }

      const blockEl = svgEl.querySelector('.cmd-block[data-node-id="' + drag.nodeId + '"], .evt-block[data-node-id="' + drag.nodeId + '"], .trg-block[data-node-id="' + drag.nodeId + '"], .view-block[data-node-id="' + drag.nodeId + '"], .auto-block[data-node-id="' + drag.nodeId + '"], .trans-block[data-node-id="' + drag.nodeId + '"]');
      if (drag.isDragging) {
        const dp = screenToDiagram(svgEl, store.viewport, evt.clientX, evt.clientY);
        const dx = dp.x - drag.startDiagramX;
        const dy = dp.y - drag.startDiagramY;
        commitDrag(store, drag.nodeId, dx, dy, drag.origNodeX, drag.origNodeY);
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
      const block = evt.target.closest('.cmd-block, .evt-block, .trg-block, .view-block, .auto-block, .trans-block');
      if (block) {
        const nodeId = block.getAttribute("data-node-id");
        if (nodeId && store.layoutPositions[nodeId]) {
          const touch = touches[0];
          const dp = screenToDiagram(svgEl, store.viewport, touch.clientX, touch.clientY);
          store.interaction.drag = {
            nodeId: nodeId,
            startDiagramX: dp.x,
            startDiagramY: dp.y,
            origNodeX: store.layoutPositions[nodeId].x,
            origNodeY: store.layoutPositions[nodeId].y,
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
          store.interaction.drag = {
            sliceId: sliceId,
            nodeIds: childIds,
            origPositions: origPositions,
            startDiagramX: dp.x,
            startDiagramY: dp.y,
            isDragging: false,
            startClientX: touch.clientX,
            startClientY: touch.clientY,
          };
          childIds.forEach(function(id) {
            const blockEl = svgEl.querySelector('.cmd-block[data-node-id="' + id + '"], .evt-block[data-node-id="' + id + '"], .trg-block[data-node-id="' + id + '"], .view-block[data-node-id="' + id + '"], .auto-block[data-node-id="' + id + '"], .trans-block[data-node-id="' + id + '"]');
            if (blockEl) blockEl.classList.add("dragging");
          });
          const headerEls = svgEl.querySelectorAll('.slice-header[data-slice-id="' + sliceId + '"]');
          headerEls.forEach(function(el) { el.classList.add("dragging"); });
        }
        return;
      }

      if (evt.target.closest('.flow-arrow, .sub-arrow, .auto-trg-arrow, .auto-cmd-arrow, .trg-cmd-arrow, .reads-arrow, .trans-cmd-arrow, .ctx-label, .agg-label')) return;
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
        } else {
          return;
        }
      }

      if (drag.sliceId) {
        drag.nodeIds.forEach(function(nodeId) {
          const orig = drag.origPositions[nodeId];
          if (!orig) return;
          const blockEl = svgEl.querySelector('.cmd-block[data-node-id="' + nodeId + '"], .evt-block[data-node-id="' + nodeId + '"], .trg-block[data-node-id="' + nodeId + '"], .view-block[data-node-id="' + nodeId + '"], .auto-block[data-node-id="' + nodeId + '"], .trans-block[data-node-id="' + nodeId + '"]');
          if (blockEl) {
            const existingOff = store.nodeOffsets[nodeId] || {dx: 0, dy: 0};
            blockEl.setAttribute("transform", "translate(" + (existingOff.dx + dx) + "," + (existingOff.dy + dy) + ")");
          }
        });
        updateArrowsForSlice(store, drag.sliceId, dx, dy);
      } else {
        const blockEl = svgEl.querySelector('.cmd-block[data-node-id="' + drag.nodeId + '"], .evt-block[data-node-id="' + drag.nodeId + '"], .trg-block[data-node-id="' + drag.nodeId + '"], .view-block[data-node-id="' + drag.nodeId + '"], .auto-block[data-node-id="' + drag.nodeId + '"], .trans-block[data-node-id="' + drag.nodeId + '"]');
        if (blockEl) {
          const existingOff = store.nodeOffsets[drag.nodeId] || {dx: 0, dy: 0};
          blockEl.setAttribute("transform", "translate(" + (existingOff.dx + dx) + "," + (existingOff.dy + dy) + ")");
        }
        updateArrowsForNode(store, drag.nodeId, {
          x: drag.origNodeX + dx,
          y: drag.origNodeY + dy,
          w: store.layoutPositions[drag.nodeId].w,
          h: store.layoutPositions[drag.nodeId].h,
        });
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
            commitSliceDrag(store, drag.sliceId, dx, dy);
            store.interaction.suppressDetailClick = true;
          }
        }
        drag.nodeIds.forEach(function(nodeId) {
          const blockEl = svgEl.querySelector('.cmd-block[data-node-id="' + nodeId + '"], .evt-block[data-node-id="' + nodeId + '"], .trg-block[data-node-id="' + nodeId + '"], .view-block[data-node-id="' + nodeId + '"], .auto-block[data-node-id="' + nodeId + '"], .trans-block[data-node-id="' + nodeId + '"]');
          if (blockEl) blockEl.classList.remove("dragging");
        });
        headerEls.forEach(function(el) { el.classList.remove("dragging"); });
        store.interaction.drag = null;
        return;
      }

      const blockEl = svgEl.querySelector('.cmd-block[data-node-id="' + drag.nodeId + '"], .evt-block[data-node-id="' + drag.nodeId + '"], .trg-block[data-node-id="' + drag.nodeId + '"], .view-block[data-node-id="' + drag.nodeId + '"], .auto-block[data-node-id="' + drag.nodeId + '"], .trans-block[data-node-id="' + drag.nodeId + '"]');
      if (drag.isDragging) {
        const touch = evt.changedTouches[0];
        if (touch) {
          const dp = screenToDiagram(svgEl, store.viewport, touch.clientX, touch.clientY);
          const dx = dp.x - drag.startDiagramX;
          const dy = dp.y - drag.startDiagramY;
          commitDrag(store, drag.nodeId, dx, dy, drag.origNodeX, drag.origNodeY);
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
    const drag = store.interaction.drag;
    if (drag) {
      if (drag.sliceId) {
        const headerEls = svgEl.querySelectorAll('.slice-header[data-slice-id="' + drag.sliceId + '"]');
        drag.nodeIds.forEach(function(nodeId) {
          const blockEl = svgEl.querySelector('.cmd-block[data-node-id="' + nodeId + '"], .evt-block[data-node-id="' + nodeId + '"], .trg-block[data-node-id="' + nodeId + '"], .view-block[data-node-id="' + nodeId + '"], .auto-block[data-node-id="' + nodeId + '"], .trans-block[data-node-id="' + nodeId + '"]');
          if (blockEl) blockEl.classList.remove("dragging");
        });
        headerEls.forEach(function(el) { el.classList.remove("dragging"); });
        store.interaction.drag = null;
      } else {
        const blockEl = svgEl.querySelector('.cmd-block[data-node-id="' + drag.nodeId + '"], .evt-block[data-node-id="' + drag.nodeId + '"], .trg-block[data-node-id="' + drag.nodeId + '"], .view-block[data-node-id="' + drag.nodeId + '"], .auto-block[data-node-id="' + drag.nodeId + '"], .trans-block[data-node-id="' + drag.nodeId + '"]');
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

import { L, PORT_DIRECTIONS } from './config.js';

const _measureSvg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
_measureSvg.style.position = "fixed";
_measureSvg.style.visibility = "hidden";
_measureSvg.style.pointerEvents = "none";
_measureSvg.style.width = "0";
_measureSvg.style.height = "0";
const _measureText = document.createElementNS("http://www.w3.org/2000/svg", "text");
_measureText.setAttribute("font-family", "sans-serif");
_measureText.setAttribute("font-size", "13");
_measureSvg.appendChild(_measureText);
document.body.appendChild(_measureSvg);
const _labelCache = {};

function labelWidth(label) {
  const s = label || '';
  if (_labelCache[s] !== undefined) return _labelCache[s];
  _measureText.textContent = s;
  const w = _measureText.getComputedTextLength();
  _labelCache[s] = w + 16;
  return w + 16;
}

function buildTree(nodes) {
  const byId = {};
  nodes.forEach(function(n) { byId[n.id] = Object.assign({}, n, {children: []}); });
  const roots = [];
  nodes.forEach(function(n) {
    if (!n.parentId || !byId[n.parentId]) {
      roots.push(byId[n.id]);
    } else {
      byId[n.parentId].children.push(byId[n.id]);
    }
  });
  return { byId: byId, roots: roots };
}

function computeLayout(store) {
  const tree = buildTree(store.nodes);
  const positions = {};

  let contexts = tree.roots.filter(function(n) { return n.type === "context"; });
  contexts = contexts.filter(function(ctx) { return !store.hiddenNodes[ctx.id]; });
  let currentY = L.marginY;
  let maxRight = L.marginX + 800;
  let xPos;
  let maxSliceHeight;

  function layoutSlice(sl, si, siblings, slY) {
    const triggers = sl.children.filter(function(c) { return c.type === "trigger"; });
    const commands = sl.children.filter(function(c) { return c.type === "command"; });
    const events   = sl.children.filter(function(c) { return c.type === "event"; });
    const views    = sl.children.filter(function(c) { return c.type === "view"; });
    const automations = sl.children.filter(function(c) { return c.type === "automation"; });
    const translations = sl.children.filter(function(c) { return c.type === "translation"; });

    const topRowTypes = translations.concat(automations);

    const allLabelWidths = sl.children.map(function(c) { return labelWidth(c.label); });
    if (translations.length) {
      translations.forEach(function(t) { allLabelWidths.push(labelWidth(t.external_system || t.label)); });
    }
    if (events.length > 1) {
      const eSum = events.reduce(function(s, e) { return s + labelWidth(e.label); }, 0);
      allLabelWidths.push(eSum + 20 + (events.length - 1) * L.sideGap - 40);
    }
    if (topRowTypes.length > 1) {
      const tSum = topRowTypes.reduce(function(s, t) { return s + labelWidth(t.external_system || t.label); }, 0);
      allLabelWidths.push(tSum + 20 + (topRowTypes.length - 1) * L.sideGap - 40);
    }
    const sBoxWidth = Math.max(180, ...allLabelWidths);
    const sSliceWidth = sBoxWidth + 40;

    let blockY = slY + L.sliceTopPad;
    const gap = 75;

    if (topRowTypes.length > 0) {
      const tCount = topRowTypes.length;
      const tItemWidth = tCount > 1 ? (sSliceWidth - 20 - (tCount - 1) * L.sideGap) / tCount : sBoxWidth;
      const tStartX = xPos + (sSliceWidth - tItemWidth * tCount - (tCount - 1) * L.sideGap) / 2;
      topRowTypes.forEach(function(rt, idx) {
        positions[rt.id] = {
          x: tStartX + idx * (tItemWidth + L.sideGap), y: blockY,
          w: tItemWidth, h: L.boxHeight,
          node: rt,
        };
      });
      blockY += L.boxHeight + gap;
    }

    triggers.forEach(function(trg) {
      const bx = xPos + (sSliceWidth - sBoxWidth) / 2;
      positions[trg.id] = {
        x: bx, y: blockY, w: sBoxWidth, h: L.boxHeight,
        node: trg,
      };
      blockY += L.boxHeight + gap;
    });
    commands.forEach(function(cmd) {
      const bx = xPos + (sSliceWidth - sBoxWidth) / 2;
      positions[cmd.id] = {
        x: bx, y: blockY, w: sBoxWidth, h: L.boxHeight,
        node: cmd,
      };
      blockY += L.boxHeight + gap;
    });
    if (events.length > 0) {
      const eCount = events.length;
      const eItemWidth = eCount > 1 ? (sSliceWidth - 20 - (eCount - 1) * L.sideGap) / eCount : sBoxWidth;
      const eStartX = xPos + (sSliceWidth - eItemWidth * eCount - (eCount - 1) * L.sideGap) / 2;
      events.forEach(function(evt, idx) {
        positions[evt.id] = {
          x: eStartX + idx * (eItemWidth + L.sideGap), y: blockY,
          w: eItemWidth, h: L.boxHeight,
          node: evt,
        };
      });
      blockY += L.boxHeight + gap;
    }
    views.forEach(function(view) {
      const bx = xPos + (sSliceWidth - sBoxWidth) / 2;
      positions[view.id] = {
        x: bx, y: blockY, w: sBoxWidth, h: L.boxHeight,
        node: view,
      };
      blockY += L.boxHeight + gap;
    });

    // The stacked layout above is the slice's minimum size. Dragging a block
    // towards the bottom-right pushes those edges out; minW/minH are what the
    // box shrinks back to once the block returns, and interaction.js needs
    // them to resize the slice live without re-running the whole layout.
    const minW = sSliceWidth;
    const minH = blockY - slY + gap;
    let boxRight = xPos + minW;
    let boxBottom = slY + minH;

    sl.children.forEach(function(c) {
      const p = positions[c.id];
      if (!p) return;
      const off = store.nodeOffsets[c.id];
      if (off) {
        p.x += off.dx;
        p.y += off.dy;
      }
      boxRight = Math.max(boxRight, p.x + p.w + L.slicePad);
      boxBottom = Math.max(boxBottom, p.y + p.h + L.slicePad);
    });

    const sliceHeight = boxBottom - slY;
    if (sliceHeight > maxSliceHeight) maxSliceHeight = sliceHeight;
    positions[sl.id] = {
      x: xPos, y: slY, w: boxRight - xPos, h: sliceHeight,
      minW: minW, minH: minH,
      node: sl,
    };

    xPos = boxRight;
    if (si < siblings.length - 1) xPos += L.sliceGap;
  }

  contexts.forEach(function(ctx) {
    const aggs = ctx.children.filter(function(n) { return n.type === "aggregate" && !store.hiddenNodes[n.id]; });
    const dirSlices = ctx.children.filter(function(n) { return n.type === "slice" && !store.hiddenNodes[n.id]; });
    xPos = L.marginX;
    let rightEdge = xPos;
    maxSliceHeight = 0;

    aggs.forEach(function(agg) {
      const slices = agg.children.filter(function(n) { return n.type === "slice" && !store.hiddenNodes[n.id]; });

      slices.forEach(function(sl, si) {
        layoutSlice(sl, si, slices, currentY + L.swimlaneHdr + L.aggLabelH);
      });

      positions[agg.id] = {
        x: L.marginX, y: currentY + L.swimlaneHdr,
        w: xPos - L.marginX, h: L.aggLabelH,
        node: agg,
      };
    });

    dirSlices.forEach(function(sl, si) {
      layoutSlice(sl, si, dirSlices, currentY + L.swimlaneHdr);
    });

    rightEdge = Math.max(rightEdge, xPos);
    if (rightEdge > maxRight) maxRight = rightEdge;

    const hasAggs = aggs.length > 0;
    const ctxH = L.swimlaneHdr + (hasAggs ? L.aggLabelH : 0) + maxSliceHeight + L.swimlanePad;
    positions[ctx.id] = {
      x: L.marginX, y: currentY,
      w: rightEdge - L.marginX + L.marginX,
      h: ctxH,
      node: ctx,
    };

    currentY += ctxH + L.swimlaneGap;
  });

  contexts.forEach(function(ctx) {
    const cp = positions[ctx.id];
    if (!cp) return;
    if (maxRight > cp.x + cp.w) {
      cp.w = maxRight - cp.x;
    }

    // Every swimlane is widened to the widest one, and the aggregate rows tile
    // that swimlane left to right, so without stretching the last row too the
    // band stops short of the context it is meant to fill. A context holding
    // slices of its own is left alone: that trailing space is not in any
    // aggregate, so no row should claim it.
    const hasDirectSlices = ctx.children.some(function(n) {
      return n.type === "slice" && positions[n.id];
    });
    if (hasDirectSlices) return;

    const aggs = ctx.children.filter(function(n) {
      return n.type === "aggregate" && positions[n.id];
    });
    const lastAgg = aggs[aggs.length - 1];
    if (lastAgg) {
      const ap = positions[lastAgg.id];
      ap.w = Math.max(ap.w, cp.x + cp.w - ap.x);
    }
  });

  const totalW = maxRight + L.marginX;
  const totalH = currentY - L.swimlaneGap + L.marginY;
  return { width: Math.max(totalW, 800), height: Math.max(totalH, 400), positions };
}

function isCrossBoundary(nodes, srcId, tgtId) {
  let srcNode = null, tgtNode = null;
  for (let i = 0; i < nodes.length; i++) {
    if (nodes[i].id === srcId) srcNode = nodes[i];
    if (nodes[i].id === tgtId) tgtNode = nodes[i];
  }
  return srcNode && tgtNode && srcNode.parentId !== tgtNode.parentId;
}

function getDescendantIds(nodes, rootId) {
  const ids = [rootId];
  nodes.forEach(function(n) {
    if (n.parentId === rootId) {
      ids.push(n.id);
      nodes.forEach(function(c) {
        if (c.parentId === n.id) {
          ids.push(c.id);
          nodes.forEach(function(gc) {
            if (gc.parentId === c.id) {
              ids.push(gc.id);
            }
          });
        }
      });
    }
  });
  return ids;
}

function getConnectedEdges(edges, nodeId) {
  const result = [];
  edges.forEach(function(e) {
    if ((e.type === "flow" || e.type === "trigger_command" || e.type === "subscription" || e.type === "automation_trigger" || e.type === "automation_command" || e.type === "reads" || e.type === "translation_command") && (e.source === nodeId || e.target === nodeId)) {
      result.push(e);
    }
  });
  return result;
}

function getSliceChildNodeIds(nodes, sliceId) {
  const ids = [];
  nodes.forEach(function(n) {
    if (n.parentId === sliceId && (n.type === "command" || n.type === "event" || n.type === "trigger" || n.type === "view" || n.type === "automation" || n.type === "translation")) {
      ids.push(n.id);
    }
  });
  return ids;
}

function computeArrowD(srcPos, tgtPos, crossBoundary, edgeIdx) {
  const stagger = (edgeIdx !== undefined) ? ((edgeIdx % 5 - 2) * 6) : 0;
  const gap = 1.5;
  const arrowLen = 10;
  const srcCx = srcPos.x + srcPos.w / 2;
  const srcBottom = srcPos.y + srcPos.h;
  const srcTop = srcPos.y;
  const tgtCx = tgtPos.x + tgtPos.w / 2;
  const tgtTop = tgtPos.y;
  const tgtBottom = tgtPos.y + tgtPos.h;

  if (crossBoundary) {
    const srcEdgeX = tgtCx > srcCx ? srcPos.x + srcPos.w : srcPos.x;
    const srcMidY = srcPos.y + srcPos.h / 2;
    const tgtAttachEdge = srcMidY < tgtPos.y + tgtPos.h / 2 ? tgtTop - gap : tgtBottom + gap;
    const tgtEnd = tgtAttachEdge + (tgtAttachEdge < srcMidY ? arrowLen : -arrowLen);
    const dx = Math.abs(tgtCx - srcEdgeX);
    const dy = Math.abs(tgtAttachEdge - srcMidY);
    const cp1x = srcEdgeX + (tgtCx > srcEdgeX ? 1 : -1) * Math.max(dx * 0.3, 20);
    const pullback = Math.min(Math.max(dy * 0.25, 15), 40);
    const p2y = tgtAttachEdge + (tgtAttachEdge < srcMidY ? pullback : -pullback);
    return "M " + srcEdgeX + "," + srcMidY + " C " + cp1x + "," + srcMidY + " " + tgtCx + "," + p2y + " " + tgtCx + "," + tgtEnd;
  } else {
    const downward = srcBottom <= tgtTop;
    const srcY = downward ? srcBottom : srcTop;
    const tgtEdge = downward ? tgtTop - gap : tgtBottom + gap;
    const tgtEnd = downward ? tgtEdge - arrowLen : tgtEdge + arrowLen;
    return "M " + srcCx + "," + srcY + " L " + tgtCx + "," + tgtEnd;
  }
}

function computeArrowEndpoints(srcPos, tgtPos, crossBoundary) {
  const gap = 1.5;
  const arrowLen = 10;
  const srcCx = srcPos.x + srcPos.w / 2;
  const srcBottom = srcPos.y + srcPos.h;
  const srcTop = srcPos.y;
  const tgtCx = tgtPos.x + tgtPos.w / 2;
  const tgtTop = tgtPos.y;
  const tgtBottom = tgtPos.y + tgtPos.h;
  var srcPoint, tgtPoint;

  if (crossBoundary) {
    const srcEdgeX = tgtCx > srcCx ? srcPos.x + srcPos.w : srcPos.x;
    const srcMidY = srcPos.y + srcPos.h / 2;
    const tgtAttachEdge = srcMidY < tgtPos.y + tgtPos.h / 2 ? tgtTop - gap : tgtBottom + gap;
    const tgtEnd = tgtAttachEdge + (tgtAttachEdge < srcMidY ? arrowLen : -arrowLen);
    srcPoint = { x: srcEdgeX, y: srcMidY };
    tgtPoint = { x: tgtCx, y: tgtEnd };
  } else {
    const downward = srcBottom <= tgtTop;
    const srcY = downward ? srcBottom : srcTop;
    const tgtEdge = downward ? tgtTop - gap : tgtBottom + gap;
    const tgtEnd = downward ? tgtEdge - arrowLen : tgtEdge + arrowLen;
    srcPoint = { x: srcCx, y: srcY };
    tgtPoint = { x: tgtCx, y: tgtEnd };
  }
  return { src: srcPoint, tgt: tgtPoint };
}

// Where a named port sits on a block: the midpoint of that side.
function portAnchor(pos, direction) {
  let dir = null;
  PORT_DIRECTIONS.forEach(function(d) { if (d.name === direction) dir = d; });
  if (!dir) return null;
  return {
    x: pos.x + pos.w / 2 + dir.dx * pos.w / 2,
    y: pos.y + pos.h / 2 + dir.dy * pos.h / 2,
  };
}

export const Layout = {
  labelWidth,
  portAnchor,
  buildTree,
  computeLayout,
  isCrossBoundary,
  getDescendantIds,
  getConnectedEdges,
  getSliceChildNodeIds,
  computeArrowD,
  computeArrowEndpoints,
};

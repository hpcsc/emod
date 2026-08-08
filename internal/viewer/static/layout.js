import { L, PORT_DIRECTIONS } from './config.js';

// The size the hidden measurer draws at, and the gap labelWidth adds so two
// measured labels laid side by side do not touch. Private: a caller needing
// either goes through textWidth or labelAdvance.
const MEASURE_FONT_SIZE = 13;
const LABEL_PADDING = 16;

const measureSvg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
measureSvg.style.position = "fixed";
measureSvg.style.visibility = "hidden";
measureSvg.style.pointerEvents = "none";
measureSvg.style.width = "0";
measureSvg.style.height = "0";
const measureText = document.createElementNS("http://www.w3.org/2000/svg", "text");
measureText.setAttribute("font-family", "sans-serif");
measureText.setAttribute("font-size", String(MEASURE_FONT_SIZE));
measureSvg.appendChild(measureText);
document.body.appendChild(measureSvg);
const textWidthCache = {};

// Glyph width alone, rescaled to the size it will actually be drawn at. This is
// the primitive: measuring happens at one font size, and a caller drawing at
// another needs the conversion done here rather than by restating this module's
// size and padding on its own side.
function textWidth(text, fontSize) {
  const s = text || '';
  if (textWidthCache[s] === undefined) {
    measureText.textContent = s;
    textWidthCache[s] = measureText.getComputedTextLength();
  }
  return textWidthCache[s] * (fontSize || MEASURE_FONT_SIZE) / MEASURE_FONT_SIZE;
}

function labelWidth(label) {
  return textWidth(label, MEASURE_FONT_SIZE) + LABEL_PADDING;
}

// How far a label pushes what follows it: its glyphs plus the gap labelWidth's
// padding stands for, both at the size the label is drawn.
function labelAdvance(label, fontSize) {
  const size = fontSize || MEASURE_FONT_SIZE;
  return textWidth(label, size) + LABEL_PADDING * size / MEASURE_FONT_SIZE;
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

function childrenOfType(children, type) {
  return children.filter(function(c) { return c.type === type; });
}

function processorLabel(node) {
  return node.external_system || node.label;
}

function rowSpanWidth(itemWidths) {
  const total = itemWidths.reduce(function(sum, w) { return sum + w; }, 0);
  return total + 20 + (itemWidths.length - 1) * L.sideGap - 40;
}

function computeLayout(store) {
  const tree = buildTree(store.nodes);
  const positions = {};

  const contexts = tree.roots.filter(function(n) {
    return n.type === "context" && !store.hiddenNodes[n.id];
  });
  let currentY = L.marginY;
  let maxRight = L.marginX + 800;
  let xPos;
  let maxSliceHeight;

  function layoutSlice(sl, si, siblings, slY) {
    const triggers = childrenOfType(sl.children, "trigger");
    const commands = childrenOfType(sl.children, "command");
    const events = childrenOfType(sl.children, "event");
    const views = childrenOfType(sl.children, "view");
    const translations = childrenOfType(sl.children, "translation");
    const processors = translations.concat(childrenOfType(sl.children, "automation"));

    const allLabelWidths = sl.children.map(function(c) { return labelWidth(c.label); });
    translations.forEach(function(t) { allLabelWidths.push(labelWidth(processorLabel(t))); });
    if (events.length > 1) {
      allLabelWidths.push(rowSpanWidth(events.map(function(e) { return labelWidth(e.label); })));
    }
    if (processors.length > 1) {
      allLabelWidths.push(rowSpanWidth(processors.map(function(p) { return labelWidth(processorLabel(p)); })));
    }
    const sBoxWidth = Math.max(180, ...allLabelWidths);
    const sSliceWidth = sBoxWidth + 40;

    let blockY = slY + L.sliceTopPad;
    const gap = 75;

    function placeColumn(items) {
      items.forEach(function(item) {
        positions[item.id] = {
          x: xPos + (sSliceWidth - sBoxWidth) / 2, y: blockY, w: sBoxWidth, h: L.boxHeight,
          node: item,
        };
        blockY += L.boxHeight + gap;
      });
    }

    function placeRow(items) {
      if (!items.length) return;
      const count = items.length;
      const itemWidth = count > 1 ? (sSliceWidth - 20 - (count - 1) * L.sideGap) / count : sBoxWidth;
      const startX = xPos + (sSliceWidth - itemWidth * count - (count - 1) * L.sideGap) / 2;
      items.forEach(function(item, idx) {
        positions[item.id] = {
          x: startX + idx * (itemWidth + L.sideGap), y: blockY,
          w: itemWidth, h: L.boxHeight,
          node: item,
        };
      });
      blockY += L.boxHeight + gap;
    }

    placeColumn(triggers);
    placeRow(processors);
    placeColumn(commands);
    placeRow(events);
    placeColumn(views);

    const childBoxes = [];
    sl.children.forEach(function(c) {
      const p = positions[c.id];
      if (p) childBoxes.push(p);
    });

    // The stacked layout above is the slice's minimum size. Dragging a block
    // towards the bottom-right pushes those edges out; minW/minH are what the
    // box shrinks back to once the block returns, and interaction.js needs
    // them to resize the slice live without re-running the whole layout.
    let minW = sSliceWidth;
    childBoxes.forEach(function(p) {
      minW = Math.max(minW, p.x + p.w + L.slicePad - xPos);
    });
    const minH = blockY - slY + gap;
    let boxRight = xPos + minW;
    let boxBottom = slY + minH;

    childBoxes.forEach(function(p) {
      const off = store.nodeOffsets[p.node.id];
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
  textWidth,
  labelAdvance,
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

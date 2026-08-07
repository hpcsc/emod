import { L, nodePalette, edgeConfig, arrowClassMap, PORT_DIRECTIONS } from './config.js';
import { Layout } from './layout.js';

var NS = "http://www.w3.org/2000/svg";
var clockMarking = "⏱";
var ellipsis = "…";
var labelWidthFontSize = 13;
var labelWidthPadding = 16;
var descriptionSize = 11;

var ctxHeaderStyle = {
  ownerAttr: "data-ctx-id",
  nameSize: 16,
  nameFill: "#ffffff",
  nameAttrs: "font-weight=\"bold\" class=\"ctx-label\"",
  descriptionFill: "#ced4da",
  descriptionAttrs: "class=\"ctx-desc\"",
};

var aggHeaderStyle = {
  ownerAttr: "data-agg-id",
  nameSize: 13,
  nameFill: "#495057",
  nameAttrs: "font-weight=\"600\" class=\"agg-label\"",
  descriptionFill: "#868e96",
  descriptionAttrs: "class=\"agg-desc\"",
};

// Rescaling carries Layout.labelWidth's padding along with it, and that padding
// is the gap the following text sits in — drop it and a description butts
// straight against the name it follows.
function labelAdvance(label, fontSize) {
  return Layout.labelWidth(label) * fontSize / labelWidthFontSize;
}

function drawnWidth(text) {
  return Layout.labelWidth(text) - labelWidthPadding;
}

// Empty when not even one character and the ellipsis fit: an aggregate row is
// floored at 100px wide, so a long name can leave nothing beside it.
function fitWithEllipsis(text, maxWidth) {
  if (drawnWidth(text) <= maxWidth) return text;
  for (var len = text.length - 1; len > 0; len--) {
    var candidate = text.slice(0, len) + ellipsis;
    if (drawnWidth(candidate) <= maxWidth) return candidate;
  }
  return "";
}

function esc(str) {
  return String(str).replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}

function setAttrs(el, attrs) {
  if (!attrs) return;
  var re = /(\S+)=["']([^"']*)["']/g;
  var m;
  while ((m = re.exec(attrs)) !== null) {
    el.setAttribute(m[1], m[2]);
  }
}

function svgRect(x, y, w, h, fill, stroke, attrs) {
  var el = document.createElementNS(NS, "rect");
  el.setAttribute("x", x);
  el.setAttribute("y", y);
  el.setAttribute("width", w);
  el.setAttribute("height", h);
  el.setAttribute("fill", fill);
  el.setAttribute("stroke", stroke);
  setAttrs(el, attrs);
  return el;
}

function svgText(x, y, text, fontSize, fill, attrs) {
  var el = document.createElementNS(NS, "text");
  el.setAttribute("x", x);
  el.setAttribute("y", y);
  el.setAttribute("font-family", "sans-serif");
  el.setAttribute("font-size", fontSize);
  el.setAttribute("fill", fill);
  setAttrs(el, attrs);
  el.textContent = text;
  return el;
}

function svgTitle(text) {
  var el = document.createElementNS(NS, "title");
  el.textContent = text;
  return el;
}

function centeredText(x, y, text, fontSize, fill) {
  return svgText(x, y, text, fontSize, fill,
    "text-anchor=\"middle\" dominant-baseline=\"middle\"");
}

function arrowClassForType(edgeType) {
  return arrowClassMap[edgeType] || "flow-arrow";
}

function clearSVG(svgEl) {
  var defsEl = svgEl.querySelector("defs");
  while (svgEl.firstChild) {
    svgEl.removeChild(svgEl.firstChild);
  }
  if (defsEl) svgEl.appendChild(defsEl);
}

function inject(svgEl, viewportGroup) {
  var defsEl = svgEl.querySelector("defs");
  while (svgEl.firstChild) {
    svgEl.removeChild(svgEl.firstChild);
  }
  if (defsEl) svgEl.appendChild(defsEl);
  svgEl.appendChild(viewportGroup);
}

function g(cls) {
  var el = document.createElementNS(NS, "g");
  el.setAttribute("class", cls);
  return el;
}

function path(d, cls, stroke, marker, dash, source, target, edgeId) {
  var el = document.createElementNS(NS, "path");
  el.setAttribute("d", d);
  el.setAttribute("fill", "none");
  el.setAttribute("stroke", stroke);
  el.setAttribute("stroke-width", "1.5");
  el.setAttribute("marker-end", marker);
  el.setAttribute("class", cls + " arrow");
  el.setAttribute("data-source", source);
  el.setAttribute("data-target", target);
  el.setAttribute("data-edge-id", edgeId);
  // Every pointer test goes through the matching .arrow-hit path instead, so a
  // 1.5px stroke drawn over a block can no longer swallow a click meant for it.
  el.setAttribute("pointer-events", "none");
  if (dash) el.setAttribute("stroke-dasharray", dash);
  return el;
}

// A 1.5px stroke is far too thin to aim at, so each arrow gets a transparent
// stroke wide enough to hit comfortably. It sits under the blocks, which keeps
// a block clickable where an arrow crosses it.
function arrowHitPath(d, source, target, edgeId) {
  var el = document.createElementNS(NS, "path");
  el.setAttribute("d", d);
  el.setAttribute("fill", "none");
  el.setAttribute("stroke", "transparent");
  el.setAttribute("stroke-width", String(L.arrowHitWidth));
  el.setAttribute("class", "arrow-hit");
  el.setAttribute("data-source", source);
  el.setAttribute("data-target", target);
  el.setAttribute("data-edge-id", edgeId);
  el.setAttribute("pointer-events", "stroke");
  return el;
}

// A port is what you grab to draw a connection, so it is drawn as an arrowhead
// aimed out of the block rather than as a dot: it reads as "drag from here" and
// gives the pointer a target several times the area of the old 5px circle. The
// hit circle is centred on the arrowhead rather than on the block's edge — a
// circle wide enough to aim at from the block's inside would take the middle of
// each edge away from the drag that moves the block.
function connectPort(nodeId, dir, pos) {
  var anchor = Layout.portAnchor(pos, dir.name);
  var alongX = -dir.dy, alongY = dir.dx;
  var baseX = anchor.x + dir.dx * L.portGap;
  var baseY = anchor.y + dir.dy * L.portGap;
  var tipX = anchor.x + dir.dx * (L.portGap + L.portLen);
  var tipY = anchor.y + dir.dy * (L.portGap + L.portLen);

  var portG = g("node-port port-" + dir.name);
  portG.setAttribute("data-port", dir.name);
  portG.setAttribute("data-node-id", nodeId);
  portG.setAttribute("cursor", "crosshair");

  var hit = document.createElementNS(NS, "circle");
  hit.setAttribute("cx", (baseX + tipX) / 2);
  hit.setAttribute("cy", (baseY + tipY) / 2);
  hit.setAttribute("r", String(L.portHitR));
  hit.setAttribute("fill", "transparent");
  hit.setAttribute("class", "port-hit");
  portG.appendChild(hit);

  var head = document.createElementNS(NS, "path");
  head.setAttribute("d",
    "M " + (baseX + alongX * L.portHalf) + "," + (baseY + alongY * L.portHalf) +
    " L " + tipX + "," + tipY +
    " L " + (baseX - alongX * L.portHalf) + "," + (baseY - alongY * L.portHalf) + " Z");
  head.setAttribute("fill", "#3498db");
  head.setAttribute("stroke", "#ffffff");
  head.setAttribute("stroke-width", "1.5");
  head.setAttribute("stroke-linejoin", "round");
  head.setAttribute("class", "port-head");
  portG.appendChild(head);

  return portG;
}

function appendHeaderLabels(parent, node, band, style) {
  var owner = " " + style.ownerAttr + "=\"" + node.id + "\"";
  parent.appendChild(svgText(band.x, band.baseline, node.label, style.nameSize, style.nameFill,
    style.nameAttrs + owner));
  if (!node.description) return;

  var descriptionX = band.x + labelAdvance(node.label, style.nameSize);
  var fitted = fitWithEllipsis(node.description, band.rightEdge - descriptionX);
  if (!fitted) return;
  // Prose is painted over the band that answers the right-click and the
  // highlighting click for the construct it documents, so it stays transparent to
  // the pointer and leaves that band reachable straight through it.
  parent.appendChild(svgText(descriptionX, band.baseline, fitted, descriptionSize,
    style.descriptionFill, style.descriptionAttrs + owner + " pointer-events=\"none\""));
}

function appendContextHeader(swimlane, ctx, cp) {
  swimlane.appendChild(svgRect(cp.x, cp.y, cp.w, L.swimlaneHdr, "#2c3e50", "#2c3e50",
    "rx=\"8\" class=\"ctx-header\" data-ctx-id=\"" + ctx.id + "\""));
  swimlane.appendChild(svgRect(cp.x, cp.y + L.swimlaneHdr - 8, cp.w, 8, "#2c3e50", "#2c3e50",
    "class=\"ctx-header\" data-ctx-id=\"" + ctx.id + "\""));

  appendHeaderLabels(swimlane, ctx, {
    x: cp.x + 16,
    baseline: cp.y + L.swimlaneHdr - 14,
    rightEdge: cp.x + cp.w,
  }, ctxHeaderStyle);
}

function computeAggregateRows(aggs, positions, cp) {
  var rows = [];
  var rowX = cp.x;
  aggs.forEach(function(agg) {
    var ap = positions[agg.id];
    if (!ap) return;
    var ownWidth = Math.max(100, ap.w - (rowX - cp.x));
    rows.push({ agg: agg, x: rowX, w: ownWidth });
    rowX += ownWidth;
  });
  return rows;
}

function appendAggregateRows(swimlane, rows, y) {
  rows.forEach(function(row) {
    swimlane.appendChild(svgRect(row.x, y, row.w, L.aggLabelH, "#e9ecef", "#e9ecef",
      "class=\"agg-row\" data-agg-id=\"" + row.agg.id + "\""));
  });
  // Every row rect goes down before any label, because a narrow aggregate
  // is floored at 100px wide and the next row then starts partway through
  // its neighbour's label, painting over the text.
  rows.forEach(function(row) {
    appendHeaderLabels(swimlane, row.agg, {
      x: row.x + 16,
      baseline: y + L.aggLabelH - 6,
      rightEdge: row.x + row.w,
    }, aggHeaderStyle);
  });
}

function appendAggregateAreas(swimlane, rows, y, h) {
  rows.forEach(function(row) {
    swimlane.appendChild(svgRect(row.x, y, row.w, h, "transparent", "none",
      "class=\"agg-area\" data-agg-id=\"" + row.agg.id + "\""));
  });
}

function appendBlockLabels(blockG, node, pos, stroke) {
  var midX = pos.x + pos.w / 2;
  var midY = pos.y + pos.h / 2;

  if (node.type === "translation") {
    if (node.external_system) {
      blockG.appendChild(centeredText(midX, pos.y + 16, node.external_system, 12, stroke));
      blockG.appendChild(centeredText(midX, pos.y + 32, node.label, 10, stroke));
    } else {
      blockG.appendChild(centeredText(midX, pos.y + 18, node.label, 12, stroke));
    }
    if (node.reads) {
      blockG.appendChild(centeredText(midX, pos.y + 46, "Reads: " + node.reads, 9, stroke));
    }
    return;
  }

  if (node.type === "automation" && node.every) {
    // A group's tooltip is its first <title> child, and it carries the cadence a
    // second time so a cron expression wider than the box stays readable on hover.
    blockG.insertBefore(svgTitle(node.every), blockG.firstChild);
    blockG.appendChild(centeredText(midX, midY - 9, node.label, 13, stroke));
    blockG.appendChild(centeredText(midX, midY + 11, clockMarking + " " + node.every, 10, stroke));
    return;
  }

  blockG.appendChild(centeredText(midX, midY, node.label, 13, stroke));
}

function buildSVG(store) {
  var positions = store.layoutPositions;
  var nodes = store.nodes;
  var edges = store.edges;

  var vg = g("");

  var ctxNodes = nodes.filter(function(n) { return n.type === "context"; });

  ctxNodes.forEach(function(ctx) {
    var cp = positions[ctx.id];
    if (!cp) return;

    var aggs = nodes.filter(function(n) {
      return n.type === "aggregate" && n.parentId === ctx.id;
    });

    var swimlane = g("swimlane-" + ctx.id);
    swimlane.appendChild(svgRect(cp.x, cp.y, cp.w, cp.h, "#f1f3f5", "#2c3e50",
      "rx=\"8\" stroke-width=\"2\""));
    appendContextHeader(swimlane, ctx, cp);

    var aggRows = computeAggregateRows(aggs, positions, cp);
    appendAggregateRows(swimlane, aggRows, cp.y + L.swimlaneHdr);

    var aggAreaY = cp.y + L.swimlaneHdr + L.aggLabelH;
    appendAggregateAreas(swimlane, aggRows, aggAreaY, (cp.y + cp.h) - aggAreaY);

    function renderSlice(sl) {
      var sp = positions[sl.id];
      if (!sp) return;

      var sliceG = g("slice-" + sl.id);
      sliceG.appendChild(svgRect(sp.x, sp.y, sp.w, sp.h, "#ffffff", "#adb5bd",
        "rx=\"4\" stroke-width=\"1.5\" stroke-dasharray=\"6,3\" class=\"slice-box\" data-slice-id=\"" + sl.id + "\""));
      sliceG.appendChild(svgRect(sp.x, sp.y, sp.w, 28, "#f8f9fa", "none",
        "rx=\"4\" class=\"slice-header\" data-slice-id=\"" + sl.id + "\""));
      sliceG.appendChild(svgRect(sp.x, sp.y + 24, sp.w, 4, "#f8f9fa", "none",
        "class=\"slice-header\" data-slice-id=\"" + sl.id + "\""));
      sliceG.appendChild(svgText(sp.x + sp.w / 2, sp.y + 18, sl.label, 12, "#495057",
        "text-anchor=\"middle\" font-weight=\"500\" class=\"slice-header\" data-slice-id=\"" + sl.id + "\""));
      sliceG.appendChild(svgRect(sp.x, sp.y + 28, sp.w, sp.h - 28, "transparent", "none",
        "class=\"slice-area\" data-slice-id=\"" + sl.id + "\""));
      swimlane.appendChild(sliceG);
    }

    aggs.forEach(function(agg) {
      var slices = nodes.filter(function(n) {
        return n.type === "slice" && n.parentId === agg.id;
      });
      slices.forEach(renderSlice);
    });

    var dcbSlices = nodes.filter(function(n) {
      return n.type === "slice" && n.parentId === ctx.id;
    });
    dcbSlices.forEach(renderSlice);

    vg.appendChild(swimlane);
  });

  // Appended before the blocks so the arrows' hit areas paint underneath them.
  var hitLayer = g("arrow-hits");
  vg.appendChild(hitLayer);

  nodes.forEach(function(n) {
    if (n.type !== "command" && n.type !== "event" &&
        n.type !== "trigger" && n.type !== "view" &&
        n.type !== "automation" && n.type !== "translation") return;
    var np = positions[n.id];
    if (!np) return;

    var palette = nodePalette[n.type];
    if (!palette) {
      return;
    }
    var fill = palette.fill;
    var stroke = palette.stroke;
    var cls = n.type + '-block';

    var blockG = g(cls + " diagram-node");
    blockG.setAttribute("data-node-id", n.id);
    blockG.appendChild(svgRect(np.x, np.y, np.w, np.h, fill, stroke,
      "rx=\"4\" stroke-width=\"1.5\""));

    if (n.type === "trigger") {
      // A trigger is drawn as a screen: a small header bar inside the top edge
      // of the box, matching the Go renderers. The main rect stays first so
      // drawnBoxes(svg) keeps reading the box itself.
      blockG.appendChild(svgRect(np.x + 8, np.y + 6, np.w - 16, 6, stroke, stroke,
        "rx=\"0\" stroke-width=\"1\""));
    }

    PORT_DIRECTIONS.forEach(function(dir) {
      blockG.appendChild(connectPort(n.id, dir, np));
    });

    appendBlockLabels(blockG, n, np, stroke);
    vg.appendChild(blockG);
  });

  store.arrowData = [];
  edges.forEach(function(edge, ei) {
    var cfg = edgeConfig[edge.type];
    if (!cfg) return;
    var srcPos = positions[edge.source];
    var tgtPos = positions[edge.target];
    if (!srcPos || !tgtPos) return;

    var crossBoundary = Layout.isCrossBoundary(store.nodes, edge.source, edge.target);
    var d = Layout.computeArrowD(srcPos, tgtPos, crossBoundary, ei);

    store.arrowData.push({ source: edge.source, target: edge.target, path: d });

    var edgeId = edge.source + "--" + edge.target;

    hitLayer.appendChild(arrowHitPath(d, edge.source, edge.target, edgeId));
    vg.appendChild(path(d, cfg.cls, cfg.stroke, cfg.marker, cfg.dash, edge.source, edge.target, edgeId));

    // Arrow endpoint handles (for repointing)
    var eps = Layout.computeArrowEndpoints(srcPos, tgtPos, crossBoundary);

    var srcHandle = document.createElementNS(NS, "circle");
    srcHandle.setAttribute("cx", eps.src.x);
    srcHandle.setAttribute("cy", eps.src.y);
    srcHandle.setAttribute("r", "6");
    srcHandle.setAttribute("fill", "transparent");
    srcHandle.setAttribute("stroke", "transparent");
    srcHandle.setAttribute("stroke-width", "2");
    srcHandle.setAttribute("class", "arrow-handle");
    srcHandle.setAttribute("data-arrow-handle", "source");
    srcHandle.setAttribute("data-edge-source", edge.source);
    srcHandle.setAttribute("data-edge-target", edge.target);
    srcHandle.setAttribute("data-edge-type", edge.type);
    srcHandle.setAttribute("data-edge-id", edgeId);
    srcHandle.setAttribute("cursor", "pointer");
    vg.appendChild(srcHandle);

    var tgtHandle = document.createElementNS(NS, "circle");
    tgtHandle.setAttribute("cx", eps.tgt.x);
    tgtHandle.setAttribute("cy", eps.tgt.y);
    tgtHandle.setAttribute("r", "6");
    tgtHandle.setAttribute("fill", "transparent");
    tgtHandle.setAttribute("stroke", "transparent");
    tgtHandle.setAttribute("stroke-width", "2");
    tgtHandle.setAttribute("class", "arrow-handle");
    tgtHandle.setAttribute("data-arrow-handle", "target");
    tgtHandle.setAttribute("data-edge-source", edge.source);
    tgtHandle.setAttribute("data-edge-target", edge.target);
    tgtHandle.setAttribute("data-edge-type", edge.type);
    tgtHandle.setAttribute("data-edge-id", edgeId);
    tgtHandle.setAttribute("cursor", "pointer");
    vg.appendChild(tgtHandle);
  });

  vg.setAttribute("id", "viewport-group");
  return vg;
}

export const Renderer = {
  esc,
  svgRect,
  svgText,
  arrowClassForType,
  clearSVG,
  inject,
  buildSVG,
};

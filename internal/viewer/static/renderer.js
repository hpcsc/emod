import { L, nodePalette, edgeConfig, arrowClassMap, PORT_DIRECTIONS, PROSE_KINDS } from './config.js';
import { Layout } from './layout.js';

var NS = "http://www.w3.org/2000/svg";
var clockMarking = "⏱";
var proseMarkings = {
  description: {
    glyph: "ⓘ",
    carriedBy: function(node) { return Boolean(node.description); },
  },
  comments: {
    glyph: "❞",
    carriedBy: function(node) { return (node.comments || []).length > 0; },
  },
};
var markingSize = 11;
var markingInset = 12;
var markingStep = 26;
// A container reads its description off its own header, so only the comments —
// which are written nowhere on the diagram — leave it anything to open.
var headerMarkingKinds = ["comments"];
var screenBar = { inset: 8, top: 6, height: 6, stroke: 1 };
var ellipsis = "…";
var labelWidthFontSize = 13;
var labelWidthPadding = 16;
var descriptionSize = 11;
var headerTextInset = 16;

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
// floored at 100px wide, so a long name can leave nothing beside it. A string
// drawn larger than Layout.labelWidth measures inks wider than the measurement
// reports, so its room is taken in that measurement's own scale.
function fitWithEllipsis(text, maxWidth, fontSize) {
  var room = maxWidth * labelWidthFontSize / (fontSize || labelWidthFontSize);
  if (drawnWidth(text) <= room) return text;
  for (var len = text.length - 1; len > 0; len--) {
    var candidate = text.slice(0, len) + ellipsis;
    if (drawnWidth(candidate) <= room) return candidate;
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
  var nameX = band.x + headerTextInset;
  var marksEdge = markingsLeftEdge(node, band, headerMarkingKinds);
  // Only a mark cuts a name short. A row is floored at 100px however long the
  // name it carries, and every row rect goes down before any label, so a name
  // is left to reach past its own row — but a mark is painted over it.
  var name = marksEdge === null ? node.label
    : fitWithEllipsis(node.label, marksEdge - nameX, style.nameSize);
  parent.appendChild(svgText(nameX, band.baseline, name, style.nameSize, style.nameFill,
    style.nameAttrs + owner));
  // The mark names its container, which is how the right-click the band under it
  // answers reaches that container through the mark as well. It wears none of the
  // band's classes: those would highlight the whole context when it was clicked
  // and go deaf to the drag that pans the canvas.
  appendMarkings(parent, node, band, style.nameFill, owner, headerMarkingKinds);
  if (!node.description) return;

  var descriptionX = nameX + labelAdvance(name, style.nameSize);
  var descriptionEdge = marksEdge === null ? band.x + band.w : marksEdge;
  var fitted = fitWithEllipsis(node.description, descriptionEdge - descriptionX);
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
    x: cp.x,
    w: cp.w,
    y: cp.y + L.swimlaneHdr / 2,
    baseline: cp.y + L.swimlaneHdr - 14,
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
      x: row.x,
      w: row.w,
      y: y + L.aggLabelH / 2,
      baseline: y + L.aggLabelH - 6,
    }, aggHeaderStyle);
  });
}

function appendAggregateAreas(swimlane, rows, y, h) {
  rows.forEach(function(row) {
    swimlane.appendChild(svgRect(row.x, y, row.w, h, "transparent", "none",
      "class=\"agg-area\" data-agg-id=\"" + row.agg.id + "\""));
  });
}

// Every mark tiles back from the corner into its own slot, and each one's centre
// keeps its own half-width clear of what it stands against: the block's right
// edge for the mark in the corner, the block's midline for the last one in. A
// block placed in a row can be a third the width of a stacked one, so the slots
// close up on each other rather than reaching past the middle of the block.
function markingPitch(width, count) {
  var room = width / 2 - markingSize / 2;
  if (count < 2 || markingInset + (count - 1) * markingStep <= room) {
    return { inset: markingInset, step: markingStep };
  }
  var inset = Math.max(markingSize / 2, room - (count - 1) * markingStep);
  return { inset: inset, step: (room - inset) / (count - 1) };
}

function markingSlotsX(band, count) {
  var pitch = markingPitch(band.w, count);
  var slotsX = [];
  for (var slot = 0; slot < count; slot++) {
    slotsX.push(band.x + band.w - pitch.inset - slot * pitch.step);
  }
  return slotsX;
}

function markingPlacements(node, band, kinds) {
  var carried = (kinds || PROSE_KINDS).filter(function(kind) {
    return proseMarkings[kind].carriedBy(node);
  });
  var slotsX = markingSlotsX(band, carried.length);
  return carried.map(function(kind, slot) { return { kind: kind, x: slotsX[slot] }; });
}

// Where the marks start eating into the band, and null when the band carries
// none, so a header's strings stop short of them rather than running under.
function markingsLeftEdge(node, band, kinds) {
  var marks = markingPlacements(node, band, kinds);
  if (!marks.length) return null;
  var innermost = marks[marks.length - 1];
  return innermost.x - drawnWidth(proseMarkings[innermost.kind].glyph) / 2;
}

// The prose itself is read out on hover by the viewer's own tooltip, which is
// why a mark carries no <title>: a native one would open beside it.
function appendMarkings(parent, node, band, fill, attrs, kinds) {
  markingPlacements(node, band, kinds).forEach(function(mark) {
    var el = centeredText(mark.x, band.y, proseMarkings[mark.kind].glyph, markingSize, fill);
    el.setAttribute("data-marker", mark.kind);
    el.setAttribute("data-node-id", node.id);
    setAttrs(el, attrs);
    parent.appendChild(el);
  });
}

function sliceNameX(pos) {
  return pos.x + pos.w / 2;
}

// A drag stretches a slice around the block it is moving and leaves the header
// at whatever width the pointer stopped at, so the band's strings are placed
// through here on that path as well as on the render.
function placeSliceHeaderTexts(root, sliceId, x, w) {
  var band = { x: x, w: w };
  var marks = [];
  root.querySelectorAll('text.slice-header[data-slice-id="' + sliceId + '"]').forEach(function(el) {
    if (el.hasAttribute("data-marker")) marks.push(el);
    else el.setAttribute("x", sliceNameX(band));
  });

  var slotsX = markingSlotsX(band, marks.length);
  marks.forEach(function(el, slot) {
    el.setAttribute("x", slotsX[slot]);
  });
}

function markingBand(pos, y) {
  return { x: pos.x, w: pos.w, y: y };
}

function sliceMarkingBand(pos) {
  return markingBand(pos, pos.y + L.sliceHdrH / 2);
}

// A trigger is drawn as a screen, with a bar across the top of its box, so its
// marks drop below the bar rather than into it.
function blockMarkingBand(node, pos) {
  var belowScreenBar = pos.y + screenBar.top + screenBar.height + screenBar.stroke + markingSize / 2;
  return markingBand(pos, node.type === "trigger" ? belowScreenBar : pos.y + markingInset);
}

function buildSlice(slice, sp) {
  var owner = " data-slice-id=\"" + slice.id + "\"";
  var headerAttrs = "class=\"slice-header\"" + owner;

  var sliceG = g("slice-" + slice.id);
  sliceG.appendChild(svgRect(sp.x, sp.y, sp.w, sp.h, "#ffffff", "#adb5bd",
    "rx=\"4\" stroke-width=\"1.5\" stroke-dasharray=\"6,3\" class=\"slice-box\"" + owner));
  sliceG.appendChild(svgRect(sp.x, sp.y, sp.w, L.sliceHdrH, "#f8f9fa", "none",
    "rx=\"4\" " + headerAttrs));
  sliceG.appendChild(svgRect(sp.x, sp.y + L.sliceHdrH - 4, sp.w, 4, "#f8f9fa", "none",
    headerAttrs));
  sliceG.appendChild(svgText(sliceNameX(sp), sp.y + 18, slice.label, 12, "#495057",
    "text-anchor=\"middle\" font-weight=\"500\" " + headerAttrs));
  // The mark stands on the band that answers the drag, the rename and the
  // slice menu, so it joins the band rather than covering it: every one of
  // those reaches the slice through .slice-header.
  appendMarkings(sliceG, slice, sliceMarkingBand(sp), "#495057", headerAttrs);
  sliceG.appendChild(svgRect(sp.x, sp.y + L.sliceHdrH, sp.w, sp.h - L.sliceHdrH, "transparent", "none",
    "class=\"slice-area\"" + owner));
  return sliceG;
}

function childrenOfType(nodes, type, parentId) {
  return nodes.filter(function(n) {
    return n.type === type && n.parentId === parentId;
  });
}

function buildSwimlane(nodes, positions, ctx, cp) {
  var aggs = childrenOfType(nodes, "aggregate", ctx.id);

  var swimlane = g("swimlane-" + ctx.id);
  swimlane.appendChild(svgRect(cp.x, cp.y, cp.w, cp.h, "#f1f3f5", "#2c3e50",
    "rx=\"8\" stroke-width=\"2\""));
  appendContextHeader(swimlane, ctx, cp);

  var aggRows = computeAggregateRows(aggs, positions, cp);
  appendAggregateRows(swimlane, aggRows, cp.y + L.swimlaneHdr);

  var aggAreaY = cp.y + L.swimlaneHdr + L.aggLabelH;
  appendAggregateAreas(swimlane, aggRows, aggAreaY, (cp.y + cp.h) - aggAreaY);

  var sliceOwners = aggs.map(function(agg) { return agg.id; }).concat(ctx.id);
  sliceOwners.forEach(function(ownerId) {
    childrenOfType(nodes, "slice", ownerId).forEach(function(sl) {
      var sp = positions[sl.id];
      if (sp) swimlane.appendChild(buildSlice(sl, sp));
    });
  });

  return swimlane;
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

function buildBlock(node, pos, palette) {
  var stroke = palette.stroke;

  var blockG = g(node.type + "-block diagram-node");
  blockG.setAttribute("data-node-id", node.id);
  blockG.appendChild(svgRect(pos.x, pos.y, pos.w, pos.h, palette.fill, stroke,
    "rx=\"4\" stroke-width=\"1.5\""));

  if (node.type === "trigger") {
    // A trigger is drawn as a screen: a small header bar inside the top edge
    // of the box, matching the Go renderers. The main rect stays first so
    // drawnBoxes(svg) keeps reading the box itself.
    blockG.appendChild(svgRect(pos.x + screenBar.inset, pos.y + screenBar.top,
      pos.w - screenBar.inset * 2, screenBar.height, stroke, stroke,
      "rx=\"0\" stroke-width=\"" + screenBar.stroke + "\""));
  }

  PORT_DIRECTIONS.forEach(function(dir) {
    blockG.appendChild(connectPort(node.id, dir, pos));
  });

  appendBlockLabels(blockG, node, pos, stroke);
  appendMarkings(blockG, node, blockMarkingBand(node, pos), stroke);
  return blockG;
}

function arrowHandle(end, point, edge, edgeId) {
  var el = document.createElementNS(NS, "circle");
  el.setAttribute("cx", point.x);
  el.setAttribute("cy", point.y);
  el.setAttribute("r", "6");
  el.setAttribute("fill", "transparent");
  el.setAttribute("stroke", "transparent");
  el.setAttribute("stroke-width", "2");
  el.setAttribute("class", "arrow-handle");
  el.setAttribute("data-arrow-handle", end);
  el.setAttribute("data-edge-source", edge.source);
  el.setAttribute("data-edge-target", edge.target);
  el.setAttribute("data-edge-type", edge.type);
  el.setAttribute("data-edge-id", edgeId);
  el.setAttribute("cursor", "pointer");
  return el;
}

function appendEdge(vg, hitLayer, store, edge, ei) {
  var positions = store.layoutPositions;
  var cfg = edgeConfig[edge.type];
  var srcPos = positions[edge.source];
  var tgtPos = positions[edge.target];
  if (!cfg || !srcPos || !tgtPos) return;

  var crossBoundary = Layout.isCrossBoundary(store.nodes, edge.source, edge.target);
  var d = Layout.computeArrowD(srcPos, tgtPos, crossBoundary, ei);

  store.arrowData.push({ source: edge.source, target: edge.target, path: d });

  var edgeId = edge.source + "--" + edge.target;

  hitLayer.appendChild(arrowHitPath(d, edge.source, edge.target, edgeId));
  vg.appendChild(path(d, cfg.cls, cfg.stroke, cfg.marker, cfg.dash, edge.source, edge.target, edgeId));

  var eps = Layout.computeArrowEndpoints(srcPos, tgtPos, crossBoundary);
  vg.appendChild(arrowHandle("source", eps.src, edge, edgeId));
  vg.appendChild(arrowHandle("target", eps.tgt, edge, edgeId));
}

function buildSVG(store) {
  var positions = store.layoutPositions;
  var nodes = store.nodes;
  var edges = store.edges;

  var vg = g("");

  nodes.filter(function(n) { return n.type === "context"; }).forEach(function(ctx) {
    var cp = positions[ctx.id];
    if (cp) vg.appendChild(buildSwimlane(nodes, positions, ctx, cp));
  });

  // Appended before the blocks so the arrows' hit areas paint underneath them.
  var hitLayer = g("arrow-hits");
  vg.appendChild(hitLayer);

  nodes.forEach(function(n) {
    var palette = nodePalette[n.type];
    var np = positions[n.id];
    if (palette && np) vg.appendChild(buildBlock(n, np, palette));
  });

  store.arrowData = [];
  edges.forEach(function(edge, ei) {
    appendEdge(vg, hitLayer, store, edge, ei);
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
  placeSliceHeaderTexts,
};

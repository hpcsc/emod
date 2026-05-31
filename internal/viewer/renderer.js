import { L, edgeConfig, arrowClassMap } from './config.js';
import { Layout } from './layout.js';

var NS = "http://www.w3.org/2000/svg";

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

function path(d, cls, stroke, marker, dash, source, target) {
  var el = document.createElementNS(NS, "path");
  el.setAttribute("d", d);
  el.setAttribute("fill", "none");
  el.setAttribute("stroke", stroke);
  el.setAttribute("stroke-width", "1.5");
  el.setAttribute("marker-end", marker);
  el.setAttribute("class", cls);
  el.setAttribute("data-source", source);
  el.setAttribute("data-target", target);
  el.setAttribute("pointer-events", "stroke");
  if (dash) el.setAttribute("stroke-dasharray", dash);
  return el;
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
    swimlane.appendChild(svgRect(cp.x, cp.y, cp.w, L.swimlaneHdr, "#2c3e50", "#2c3e50",
      "rx=\"8\" class=\"ctx-header\" data-ctx-id=\"" + ctx.id + "\""));
    swimlane.appendChild(svgRect(cp.x, cp.y + L.swimlaneHdr - 8, cp.w, 8, "#2c3e50", "#2c3e50",
      "class=\"ctx-header\" data-ctx-id=\"" + ctx.id + "\""));
    swimlane.appendChild(svgText(cp.x + 16, cp.y + L.swimlaneHdr - 14, ctx.label, 16, "#ffffff",
      "font-weight=\"bold\" class=\"ctx-label\" data-ctx-id=\"" + ctx.id + "\""));

    if (aggs.length > 0) {
      var aggY = cp.y + L.swimlaneHdr;
      var aggStartX = cp.x;
      aggs.forEach(function(agg) {
        var ap = positions[agg.id];
        if (!ap) return;
        var aggOwnWidth = Math.max(100, ap.w - (aggStartX - cp.x));
        swimlane.appendChild(svgRect(aggStartX, aggY, aggOwnWidth, L.aggLabelH, "#e9ecef", "#e9ecef",
          "class=\"agg-row\" data-agg-id=\"" + agg.id + "\""));
        aggStartX += aggOwnWidth;
      });
      aggs.forEach(function(agg) {
        swimlane.appendChild(svgText(cp.x + 16, aggY + L.aggLabelH - 6, agg.label, 13, "#495057",
          "font-weight=\"600\" class=\"agg-label\" data-agg-id=\"" + agg.id + "\""));
      });
    }

    var aggAreaX = cp.x;
    aggs.forEach(function(agg) {
      var ap = positions[agg.id];
      if (!ap) return;
      var aggAreaW = Math.max(100, ap.w - (aggAreaX - cp.x));
      var aggAreaY = cp.y + L.swimlaneHdr + L.aggLabelH;
      var aggAreaH = (cp.y + cp.h) - aggAreaY;
      swimlane.appendChild(svgRect(aggAreaX, aggAreaY, aggAreaW, aggAreaH, "transparent", "none",
        "class=\"agg-area\" data-agg-id=\"" + agg.id + "\""));
      aggAreaX += aggAreaW;
    });

    aggs.forEach(function(agg) {
      var slices = nodes.filter(function(n) {
        return n.type === "slice" && n.parentId === agg.id;
      });
      slices.forEach(function(sl) {
        var sp = positions[sl.id];
        if (!sp) return;

        var sliceG = g("slice-" + sl.id);
        sliceG.appendChild(svgRect(sp.x, sp.y, sp.w, sp.h, "#ffffff", "#adb5bd",
          "rx=\"4\" stroke-width=\"1.5\" stroke-dasharray=\"6,3\""));
        sliceG.appendChild(svgRect(sp.x, sp.y, sp.w, 28, "#f8f9fa", "none",
          "rx=\"4\" class=\"slice-header\" data-slice-id=\"" + sl.id + "\""));
        sliceG.appendChild(svgRect(sp.x, sp.y + 24, sp.w, 4, "#f8f9fa", "none",
          "class=\"slice-header\" data-slice-id=\"" + sl.id + "\""));
        sliceG.appendChild(svgText(sp.x + sp.w / 2, sp.y + 18, sl.label, 12, "#495057",
          "text-anchor=\"middle\" font-weight=\"500\" class=\"slice-header\" data-slice-id=\"" + sl.id + "\""));
        sliceG.appendChild(svgRect(sp.x, sp.y + 28, sp.w, sp.h - 28, "transparent", "none",
          "class=\"slice-area\" data-slice-id=\"" + sl.id + "\""));
        swimlane.appendChild(sliceG);
      });
    });

    vg.appendChild(swimlane);
  });

  nodes.forEach(function(n) {
    if (n.type !== "command" && n.type !== "event" &&
        n.type !== "trigger" && n.type !== "view" &&
        n.type !== "automation" && n.type !== "translation") return;
    var np = positions[n.id];
    if (!np) return;

    var fill, stroke, cls;
    switch (n.type) {
      case "command":     fill = "#dae8fc"; stroke = "#6c8ebf"; cls = "cmd-block"; break;
      case "event":       fill = "#ffe6cc"; stroke = "#d79b00"; cls = "evt-block"; break;
      case "trigger":     fill = "#e1d5e7"; stroke = "#9673a6"; cls = "trg-block"; break;
      case "view":        fill = "#d5e8d4"; stroke = "#82b366"; cls = "view-block"; break;
      case "automation":  fill = "#fff2cc"; stroke = "#d6b656"; cls = "auto-block"; break;
      case "translation": fill = "#f5f5f5"; stroke = "#666666"; cls = "trans-block"; break;
    }

    var blockG = g(cls + " diagram-node");
    blockG.setAttribute("data-node-id", n.id);
    blockG.appendChild(svgRect(np.x, np.y, np.w, np.h, fill, stroke,
      "rx=\"4\" stroke-width=\"1.5\""));

    if (n.type === "translation") {
      if (n.external_system) {
        blockG.appendChild(svgText(np.x + np.w / 2, np.y + 16, n.external_system, 12, stroke,
          "text-anchor=\"middle\" dominant-baseline=\"middle\""));
        blockG.appendChild(svgText(np.x + np.w / 2, np.y + 32, n.label, 10, stroke,
          "text-anchor=\"middle\" dominant-baseline=\"middle\""));
      } else {
        blockG.appendChild(svgText(np.x + np.w / 2, np.y + 18, n.label, 12, stroke,
          "text-anchor=\"middle\" dominant-baseline=\"middle\""));
      }
      if (n.reads) {
        blockG.appendChild(svgText(np.x + np.w / 2, np.y + 46, "Reads: " + n.reads, 9, stroke,
          "text-anchor=\"middle\" dominant-baseline=\"middle\""));
      }
    } else {
      blockG.appendChild(svgText(np.x + np.w / 2, np.y + np.h / 2, n.label, 13, stroke,
        "text-anchor=\"middle\" dominant-baseline=\"middle\""));
    }
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

    vg.appendChild(path(d, cfg.cls, cfg.stroke, cfg.marker, cfg.dash, edge.source, edge.target));
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

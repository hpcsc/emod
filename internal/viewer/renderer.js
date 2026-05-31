import { L, edgeConfig, arrowClassMap } from './config.js';
import { Layout } from './layout.js';

function esc(str) {
  return String(str).replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}

function svgRect(x, y, w, h, fill, stroke, attrs) {
  const a = attrs ? " " + attrs : "";
  return `<rect x="${x}" y="${y}" width="${w}" height="${h}" fill="${fill}" stroke="${stroke}"${a}/>\n`;
}

function svgText(x, y, text, fontSize, fill, attrs) {
  const a = attrs ? " " + attrs : "";
  return `<text x="${x}" y="${y}" font-family="sans-serif" font-size="${fontSize}" fill="${fill}"${a}>${esc(text)}</text>\n`;
}

function arrowClassForType(edgeType) {
  return arrowClassMap[edgeType] || "flow-arrow";
}

function clearSVG(svgEl) {
  const defs = svgEl.querySelector("defs");
  svgEl.innerHTML = "";
  if (defs) svgEl.appendChild(defs);
}

function inject(svgEl, html) {
  const defsEl = svgEl.querySelector("defs");
  svgEl.innerHTML = html;
  if (defsEl) svgEl.insertBefore(defsEl, svgEl.firstChild);
}

function buildSVG(store) {
  const positions = store.layoutPositions;
  const nodes = store.nodes;
  const edges = store.edges;

  const ctxNodes = nodes.filter(function(n) { return n.type === "context"; });
  let html = "";

  ctxNodes.forEach(function(ctx) {
    const cp = positions[ctx.id];
    if (!cp) return;

    const aggs = nodes.filter(function(n) {
      return n.type === "aggregate" && n.parentId === ctx.id;
    });

    html += "<g class=\"swimlane-" + ctx.id + "\">\n";
    html += svgRect(cp.x, cp.y, cp.w, cp.h, "#f1f3f5", "#2c3e50", "rx=\"8\" stroke-width=\"2\"");

    html += svgRect(cp.x, cp.y, cp.w, L.swimlaneHdr, "#2c3e50", "#2c3e50", "rx=\"8\" class=\"ctx-header\" data-ctx-id=\"" + ctx.id + "\"");
    html += svgRect(cp.x, cp.y + L.swimlaneHdr - 8, cp.w, 8, "#2c3e50", "#2c3e50", "class=\"ctx-header\" data-ctx-id=\"" + ctx.id + "\"");
    html += svgText(cp.x + 16, cp.y + L.swimlaneHdr - 14, ctx.label, 16, "#ffffff", "font-weight=\"bold\" class=\"ctx-label\" data-ctx-id=\"" + ctx.id + "\"");

    if (aggs.length > 0) {
      let aggY = cp.y + L.swimlaneHdr;
      let aggStartX = cp.x;
      aggs.forEach(function(agg) {
        const ap = positions[agg.id];
        if (!ap) return;
        const aggOwnWidth = Math.max(100, ap.w - (aggStartX - cp.x));
        html += svgRect(aggStartX, aggY, aggOwnWidth, L.aggLabelH, "#e9ecef", "#e9ecef", "class=\"agg-row\" data-agg-id=\"" + agg.id + "\"");
        aggStartX += aggOwnWidth;
      });
      aggs.forEach(function(agg) {
        html += svgText(cp.x + 16, aggY + L.aggLabelH - 6, agg.label, 13, "#495057", "font-weight=\"600\" class=\"agg-label\" data-agg-id=\"" + agg.id + "\"");
      });
    }

    let aggAreaX = cp.x;
    aggs.forEach(function(agg) {
      const ap = positions[agg.id];
      if (!ap) return;
      const aggAreaW = Math.max(100, ap.w - (aggAreaX - cp.x));
      const aggAreaY = cp.y + L.swimlaneHdr + L.aggLabelH;
      const aggAreaH = (cp.y + cp.h) - aggAreaY;
      html += svgRect(aggAreaX, aggAreaY, aggAreaW, aggAreaH, "transparent", "none",
        "class=\"agg-area\" data-agg-id=\"" + agg.id + "\"");
      aggAreaX += aggAreaW;
    });

    aggs.forEach(function(agg) {
      const slices = nodes.filter(function(n) {
        return n.type === "slice" && n.parentId === agg.id;
      });
      slices.forEach(function(sl) {
        const sp = positions[sl.id];
        if (!sp) return;

        html += "<g class=\"slice-" + sl.id + "\">\n";
        html += svgRect(sp.x, sp.y, sp.w, sp.h, "#ffffff", "#adb5bd",
          "rx=\"4\" stroke-width=\"1.5\" stroke-dasharray=\"6,3\"");
        html += svgRect(sp.x, sp.y, sp.w, 28, "#f8f9fa", "none", "rx=\"4\" class=\"slice-header\" data-slice-id=\"" + sl.id + "\"");
        html += svgRect(sp.x, sp.y + 24, sp.w, 4, "#f8f9fa", "none", "class=\"slice-header\" data-slice-id=\"" + sl.id + "\"");
        html += svgText(sp.x + sp.w / 2, sp.y + 18, sl.label, 12, "#495057",
          "text-anchor=\"middle\" font-weight=\"500\" class=\"slice-header\" data-slice-id=\"" + sl.id + "\"");
        html += svgRect(sp.x, sp.y + 28, sp.w, sp.h - 28, "transparent", "none",
          "class=\"slice-area\" data-slice-id=\"" + sl.id + "\"");
        html += "</g>\n";
      });
    });

    html += "</g>\n";
  });

  nodes.forEach(function(n) {
    if (n.type !== "command" && n.type !== "event" &&
        n.type !== "trigger" && n.type !== "view" &&
        n.type !== "automation" && n.type !== "translation") return;
    const np = positions[n.id];
    if (!np) return;

    let fill, stroke, cls;
    switch (n.type) {
      case "command":     fill = "#dae8fc"; stroke = "#6c8ebf"; cls = "cmd-block"; break;
      case "event":       fill = "#ffe6cc"; stroke = "#d79b00"; cls = "evt-block"; break;
      case "trigger":     fill = "#e1d5e7"; stroke = "#9673a6"; cls = "trg-block"; break;
      case "view":        fill = "#d5e8d4"; stroke = "#82b366"; cls = "view-block"; break;
      case "automation":  fill = "#fff2cc"; stroke = "#d6b656"; cls = "auto-block"; break;
      case "translation": fill = "#f5f5f5"; stroke = "#666666"; cls = "trans-block"; break;
    }

    html += "<g class=\"" + cls + "\" data-node-id=\"" + n.id + "\">\n";
    html += svgRect(np.x, np.y, np.w, np.h, fill, stroke,
      "rx=\"4\" stroke-width=\"1.5\"");
    if (n.type === "translation") {
      if (n.external_system) {
        html += svgText(np.x + np.w / 2, np.y + 16, n.external_system, 12, stroke,
          "text-anchor=\"middle\" dominant-baseline=\"middle\"");
        html += svgText(np.x + np.w / 2, np.y + 32, n.label, 10, stroke,
          "text-anchor=\"middle\" dominant-baseline=\"middle\"");
      } else {
        html += svgText(np.x + np.w / 2, np.y + 18, n.label, 12, stroke,
          "text-anchor=\"middle\" dominant-baseline=\"middle\"");
      }
      if (n.reads) {
        html += svgText(np.x + np.w / 2, np.y + 46, "Reads: " + n.reads, 9, stroke,
          "text-anchor=\"middle\" dominant-baseline=\"middle\"");
      }
    } else {
      html += svgText(np.x + np.w / 2, np.y + np.h / 2, n.label, 13, stroke,
        "text-anchor=\"middle\" dominant-baseline=\"middle\"");
    }
    html += "</g>\n";
  });

  store.arrowData = [];
  edges.forEach(function(edge, ei) {
    const cfg = edgeConfig[edge.type];
    if (!cfg) return;
    const srcPos = positions[edge.source];
    const tgtPos = positions[edge.target];
    if (!srcPos || !tgtPos) return;

    const crossBoundary = Layout.isCrossBoundary(store.nodes, edge.source, edge.target);
    const d = Layout.computeArrowD(srcPos, tgtPos, crossBoundary, ei);

    store.arrowData.push({ source: edge.source, target: edge.target, path: d });

    const dashAttr = cfg.dash ? " stroke-dasharray=\"" + cfg.dash + "\"" : "";
    html += "<path d=\"" + d + "\" fill=\"none\" stroke=\"" + cfg.stroke + "\" stroke-width=\"1.5\" marker-end=\"" + cfg.marker + "\" class=\"" + cfg.cls + "\" data-source=\"" + edge.source + "\" data-target=\"" + edge.target + "\" pointer-events=\"stroke\"" + dashAttr + "/>\n";
  });

  html = '<g id="viewport-group">\n' + html + '</g>\n';
  return html;
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

import { Severity } from './severity.js';

function initSourcePanel(store) {
  // A hidden textarea reports no scroll, and showing it again restores its
  // scroll without a scroll event.
  if (typeof ResizeObserver === "function") {
    new ResizeObserver(function() { followScroll(store); }).observe(store.dom.sourceInput);
  }
  store.dom.sourceInput.addEventListener("scroll", function() {
    followScroll(store);
  });
}

function followScroll(store) {
  const gutterEl = store.dom.sourceGutter;
  if (!gutterEl) return;
  gutterEl.style.setProperty("--source-scroll", store.dom.sourceInput.scrollTop + "px");
}

function groupByLine(diagnostics) {
  const byLine = new Map();
  diagnostics.forEach(function(d) {
    if (!d.line) return;
    if (!byLine.has(d.line)) byLine.set(d.line, []);
    byLine.get(d.line).push(d);
  });
  return byLine;
}

function buildGutterEntry(line, diagnostics) {
  const label = "Line " + line + ": " + Severity.buildCountText(diagnostics);
  const entry = document.createElement("div");
  entry.className = "gutter-entry";
  entry.dataset.severity = Severity.mostSevere(diagnostics);
  entry.style.setProperty("--line", line);
  entry.title = label;
  entry.setAttribute("role", "img");
  entry.setAttribute("aria-label", label);
  return entry;
}

function showGutter(store, diagnostics) {
  const gutterEl = store.dom.sourceGutter;
  if (!gutterEl) return;

  gutterEl.textContent = "";
  const byLine = groupByLine(diagnostics || []);
  Array.from(byLine.keys()).sort(function(a, b) { return a - b; }).forEach(function(line) {
    gutterEl.appendChild(buildGutterEntry(line, byLine.get(line)));
  });
  followScroll(store);
}

export const SourcePanel = {
  initSourcePanel,
  showGutter,
};

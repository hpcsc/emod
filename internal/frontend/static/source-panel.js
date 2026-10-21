import { Severity } from './severity.js';

const CARET_MOVES = ["keyup", "mouseup", "input", "selectionchange"];

function initSourcePanel(store) {
  // A hidden textarea reports no scroll, and showing it again restores its
  // scroll without a scroll event.
  if (typeof ResizeObserver === "function") {
    new ResizeObserver(function() { followScroll(store); }).observe(store.dom.sourceInput);
  }
  store.dom.sourceInput.addEventListener("scroll", function() {
    followScroll(store);
  });
  CARET_MOVES.forEach(function(type) {
    store.dom.sourceInput.addEventListener(type, function() {
      showCaretLineDiagnostics(store);
    });
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

function offsetOf(text, location) {
  let lineStart = 0;
  for (let line = 1; line < location.line; line++) {
    const lineEnd = text.indexOf("\n", lineStart);
    if (lineEnd === -1) return text.length;
    lineStart = lineEnd + 1;
  }
  const lineEnd = text.indexOf("\n", lineStart);
  const lineText = text.slice(lineStart, lineEnd === -1 ? text.length : lineEnd);

  // HCL counts a column in grapheme clusters, not in code units.
  const graphemes = new Intl.Segmenter(undefined, { granularity: "grapheme" });
  let offset = lineStart;
  let column = 1;
  for (const { segment } of graphemes.segment(lineText)) {
    if (column >= (location.column || 1)) break;
    offset += segment.length;
    column++;
  }
  return offset;
}

function scrollToCaret(sourceInput, offset) {
  const style = getComputedStyle(sourceInput);
  const lineHeight = parseFloat(style.lineHeight);
  // jsdom lays nothing out, so it reports no row height.
  if (!lineHeight) return;

  const top = parseFloat(style.paddingTop) + (lineAt(sourceInput.value, offset) - 1) * lineHeight;
  if (top < sourceInput.scrollTop || top + lineHeight > sourceInput.scrollTop + sourceInput.clientHeight) {
    sourceInput.scrollTop = top + lineHeight / 2 - sourceInput.clientHeight / 2;
  }

  const before = sourceInput.value.slice(0, offset);
  const lineBefore = expandTabs(before.slice(before.lastIndexOf("\n") + 1), parseInt(style.tabSize, 10));
  const left = parseFloat(style.paddingLeft) + measureTextWidth(lineBefore, style);
  if (left < sourceInput.scrollLeft || left > sourceInput.scrollLeft + sourceInput.clientWidth) {
    sourceInput.scrollLeft = left - sourceInput.clientWidth / 2;
  }
}

// A canvas draws a tab as one space, where the textarea draws it to the next
// tab stop.
function expandTabs(text, tabSize) {
  let expanded = "";
  for (const char of text) {
    expanded += char === "\t" ? " ".repeat(tabSize - (expanded.length % tabSize)) : char;
  }
  return expanded;
}

function measureTextWidth(text, style) {
  const context = document.createElement("canvas").getContext("2d");
  context.font = style.fontStyle + " " + style.fontWeight + " " + style.fontSize + " " + style.fontFamily;
  return context.measureText(text).width;
}

function moveCaretTo(store, location) {
  const sourceInput = store.dom.sourceInput;
  const offset = offsetOf(sourceInput.value, location);
  sourceInput.focus();
  sourceInput.setSelectionRange(offset, offset);
  scrollToCaret(sourceInput, offset);
  showCaretLineDiagnostics(store);
}

function lineAt(text, offset) {
  let line = 1;
  for (let i = text.indexOf("\n"); i !== -1 && i < offset; i = text.indexOf("\n", i + 1)) {
    line++;
  }
  return line;
}

function caretOffset(sourceInput) {
  return sourceInput.selectionDirection === "backward" ? sourceInput.selectionStart : sourceInput.selectionEnd;
}

function buildCaretLineRow(diagnostic) {
  const row = document.createElement("div");
  row.className = "caret-line-row";

  const severity = Severity.of(diagnostic);
  const pill = document.createElement("span");
  pill.className = "diag-severity " + severity;
  pill.textContent = severity;
  row.appendChild(pill);
  row.appendChild(document.createTextNode(" "));

  if (diagnostic.rule_name) {
    const rule = document.createElement("span");
    rule.className = "diag-rule";
    rule.textContent = "[" + diagnostic.rule_name + "]";
    row.appendChild(rule);
    row.appendChild(document.createTextNode(" "));
  }
  row.appendChild(document.createTextNode(diagnostic.message));
  row.title = row.textContent;
  return row;
}

function showCaretLineDiagnostics(store) {
  const caretLineEl = store.dom.caretLineDiagnostics;
  if (!caretLineEl) return;

  const sourceInput = store.dom.sourceInput;
  const located = store.diagnostics.filter(function(d) { return d.line; });
  const line = located.length ? lineAt(sourceInput.value, caretOffset(sourceInput)) : 0;
  const onLine = located.filter(function(d) { return d.line === line; });
  const rows = Severity.sortMostSevereFirst(onLine).map(buildCaretLineRow);

  const text = rows.map(function(row) { return row.textContent; }).join("");
  if (text === caretLineEl.textContent) return;
  caretLineEl.replaceChildren(...rows);
}

function showGutter(store) {
  const gutterEl = store.dom.sourceGutter;
  if (!gutterEl) return;

  gutterEl.textContent = "";
  const byLine = groupByLine(store.diagnostics);
  Array.from(byLine.keys()).sort(function(a, b) { return a - b; }).forEach(function(line) {
    gutterEl.appendChild(buildGutterEntry(line, byLine.get(line)));
  });
  followScroll(store);
}

function showDiagnostics(store) {
  showGutter(store);
  showCaretLineDiagnostics(store);
}

export const SourcePanel = {
  initSourcePanel,
  moveCaretTo,
  showDiagnostics,
};

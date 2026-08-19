import { describe, it, expect, beforeEach } from 'vitest';
import { UI } from '../static/ui.js';
import { createStore } from '../static/store.js';
import { installSVGGeometry } from './svg-env.js';

function setupPanel({ nodes = [], renderNodes = [] } = {}) {
  document.body.innerHTML = `
    <div id="diagnostics-list"></div>
    <div id="diagnostics-panel"></div>
    <div id="diagnostics-badge" style="display:none"></div>
    <svg id="diagram-canvas"></svg>
  `;

  const store = createStore();
  store.dom.diagnosticsList = document.getElementById('diagnostics-list');
  store.dom.diagnosticsPanel = document.getElementById('diagnostics-panel');
  store.dom.diagnosticsBadge = document.getElementById('diagnostics-badge');
  store.dom.svg = document.getElementById('diagram-canvas');

  store.nodes = nodes;
  store.nodeById = new Map(nodes.map((n) => [n.id, n]));

  const elements = {};
  for (const id of renderNodes) {
    const el = document.createElementNS('http://www.w3.org/2000/svg', 'g');
    el.setAttribute('data-node-id', id);
    el.classList.add('diagram-node');
    store.dom.svg.appendChild(el);
    elements[id] = el;
  }

  return { store, elements };
}

function showDiagnostics(store, diagnostics) {
  store.diagnostics = diagnostics;
  UI.updateDiagnosticsPanel(store, diagnostics);
  UI.initDiagnosticsDelegation(store);
  return store.dom.diagnosticsList.querySelectorAll('.diag-item');
}

const cmdAtLine5 = { id: 'n1', type: 'command', label: 'Cmd1', position: { filename: 'test.cue', line: 5, column: 3 } };
const evtAtLine10 = { id: 'n2', type: 'event', label: 'Evt1', position: { filename: 'test.cue', line: 10, column: 3 } };

beforeEach(() => {
  installSVGGeometry();
  document.body.innerHTML = '';
});

describe('clicking a diagnostic', () => {
  it('highlights the diagram node declared at that line', () => {
    const { store } = setupPanel({ nodes: [cmdAtLine5, evtAtLine10] });
    const items = showDiagnostics(store, [
      { file: 'test.cue', line: 5, message: 'Error on line 5', severity: 'error' },
    ]);

    items[0].click();

    expect(store.interaction.highlighted).toEqual({ n1: true });
    expect(items[0].classList.contains('not-rendered')).toBe(false);
  });

  it('marks the diagnostic as not rendered when no node matches its location', () => {
    const { store } = setupPanel({
      nodes: [{ ...cmdAtLine5, position: { filename: 'other.cue', line: 99, column: 3 } }],
    });
    const items = showDiagnostics(store, [
      { file: 'test.cue', line: 5, message: 'Unmatched error', severity: 'error' },
    ]);

    items[0].click();

    expect(items[0].classList.contains('not-rendered')).toBe(true);
    expect(store.interaction.highlighted).toEqual({});
  });

  it('marks the diagnostic as not rendered when it carries no location', () => {
    const { store } = setupPanel({ nodes: [cmdAtLine5] });
    const items = showDiagnostics(store, [
      { file: '', line: 0, message: 'No location', severity: 'error' },
    ]);

    items[0].click();

    expect(items[0].classList.contains('not-rendered')).toBe(true);
  });

  it('replaces the previous highlight rather than adding to it', () => {
    const { store, elements } = setupPanel({
      nodes: [cmdAtLine5, evtAtLine10],
      renderNodes: ['n1', 'n2'],
    });
    const items = showDiagnostics(store, [
      { file: 'test.cue', line: 5, message: 'Error 1', severity: 'error' },
      { file: 'test.cue', line: 10, message: 'Error 2', severity: 'error' },
    ]);
    expect(items).toHaveLength(2);

    items[0].click();
    expect(store.interaction.highlighted).toEqual({ n1: true });

    items[1].click();

    expect(store.interaction.highlighted).toEqual({ n2: true });
    expect(elements.n1.classList.contains('hl')).toBe(false);
    expect(elements.n2.classList.contains('hl')).toBe(true);
  });
});

describe('closing the diagnostics panel', () => {
  it('clears the highlight the last click applied', () => {
    const { store } = setupPanel({ nodes: [cmdAtLine5], renderNodes: ['n1'] });
    const items = showDiagnostics(store, [
      { file: 'test.cue', line: 5, message: 'Error', severity: 'error' },
    ]);
    items[0].click();
    expect(store.interaction.highlighted).toEqual({ n1: true });

    UI.hideDiagnosticsPanel(store);

    expect(store.interaction.highlighted).toEqual({});
  });
});

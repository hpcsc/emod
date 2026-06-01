import { describe, it, expect, vi, beforeEach } from 'vitest';

// ─── Mutable wasm mock state (controlled per test) ─────────────────
let wasmReady = Promise.resolve();
let wasmIsReady = true;

vi.mock('../static/wasm.js', () => ({
  get ready() { return wasmReady; },
  get isReady() { return wasmIsReady; },
}));

vi.mock('../static/store.js', () => ({
  createStore: vi.fn(() => ({
    dom: {},
    nodes: [],
    edges: [],
    nodeById: new Map(),
    layoutPositions: {},
    nodeOffsets: {},
    hiddenContexts: {},
    arrowData: [],
    interaction: {},
    viewport: {},
    modelName: '',
    diagnostics: [],
  })),
}));

vi.mock('../static/bus.js', () => ({
  bus: {
    on: vi.fn(),
    emit: vi.fn(),
  },
}));

vi.mock('../static/layout.js', () => ({ Layout: {} }));
vi.mock('../static/renderer.js', () => ({ Renderer: { esc: function(x) { return String(x); } } }));
vi.mock('../static/interaction.js', () => ({ Interaction: { initEventListeners: vi.fn() } }));
vi.mock('../static/ui.js', () => ({ UI: { initDelegation: vi.fn(), initKeyboard: vi.fn(), hideContextMenu: vi.fn(), hideDetailPanel: vi.fn(), updateStats: vi.fn(), updateMinimap: vi.fn(), toggleMinimap: vi.fn(), toggleContextPanel: vi.fn(), minimapNavigate: vi.fn(), updateContextList: vi.fn(), renderActorAnnotations: vi.fn(), updateDiagnosticsPanel: vi.fn(), toggleDiagnosticsPanel: vi.fn(), hideDiagnosticsPanel: vi.fn(), initDiagnosticsDelegation: vi.fn() } }));
vi.mock('../static/model.js', () => ({ Model: { setModelData: vi.fn(), sendParse: vi.fn(() => Promise.resolve({ diagnostics: [], diagram: { nodes: [], edges: [] } })) } }));
vi.mock('../static/emod-export.js', () => ({ Export: {} }));

// ─── DOM helpers ───────────────────────────────────────────────────
function createRequiredElements() {
  const container = document.createElement('div');
  container.innerHTML = `
    <svg id="diagram-canvas"></svg>
    <div id="model-name-display"></div>
    <textarea id="source-input"></textarea>
    <button id="render-btn"></button>
    <span id="render-status"></span>
    <span id="stat-nodes"></span>
    <span id="stat-edges"></span>
    <span id="stat-canvas"></span>
    <div id="data-panel" class="collapsed"></div>
    <div id="data-panel-header"></div>
    <div id="data-panel-body"></div>
    <div id="minimap"></div>
    <svg id="minimap-svg"></svg>
    <button id="minimap-toggle"></button>
    <div id="context-panel"></div>
    <button id="context-toggle"></button>
    <div id="context-list"></div>
    <div id="tooltip"></div>
    <div id="detail-panel"></div>
    <div id="dp-content"></div>
    <div id="ctx-menu"></div>
    <div id="actor-annotations"></div>
    <button id="reset-layout"></button>
    <button id="fit-view"></button>
    <button id="export-emod"></button>
    <div id="minimap-handle"></div>
    <button id="minimap-close"></button>
    <div id="dp-close"></div>
    <div id="landing-instructions" style="display: none;"></div>
    <button id="diagnostics-badge" style="display:none"></button>
    <div id="diagnostics-panel" class="hidden"></div>
    <button id="diagnostics-close"></button>
    <div id="diagnostics-list"></div>
  `;
  document.body.appendChild(container);
  return container;
}

function clearDOM() {
  document.body.innerHTML = '';
}

function fireDrop(element, file) {
  const evt = new Event('drop', { bubbles: true, cancelable: true });
  Object.defineProperty(evt, 'dataTransfer', {
    value: { files: file ? [file] : [] },
  });
  element.dispatchEvent(evt);
  return evt;
}

// ─── Mock FileReader for deterministic file loading ──────────────
class MockFileReader {
  readAsText(file) {
    this.result = file._content || '';
    const self = this;
    setTimeout(function() {
      if (self.onload) self.onload({ target: self });
    }, 0);
  }
}

beforeEach(() => {
  clearDOM();
  globalThis.FileReader = MockFileReader;
  // Reset wasm mock defaults
  wasmReady = Promise.resolve();
  wasmIsReady = true;
});

// ─── Tests ─────────────────────────────────────────────────────────
describe('viewer initial state', () => {
  it('opens data panel and shows landing page when no INITIAL_DATA', async () => {
    globalThis.INITIAL_DATA = null;
    createRequiredElements();
    const { init } = await import('../static/viewer.js');
    init();

    const panel = document.getElementById('data-panel');
    const nameDisplay = document.getElementById('model-name-display');
    const sourceInput = document.getElementById('source-input');
    const instructions = document.getElementById('landing-instructions');

    expect(panel.classList.contains('collapsed')).toBe(false);
    expect(nameDisplay.textContent).toBe('(no model)');
    expect(sourceInput.placeholder).toBe('Paste .emod source or diagram JSON here');
    expect(instructions.style.display).toBe('block');
  });

  it('does not show landing page when INITIAL_DATA is present', async () => {
    const { Model } = await import('../static/model.js');
    globalThis.INITIAL_DATA = { diagram: { nodes: [{ id: 'n1' }], edges: [] } };
    createRequiredElements();
    const { init } = await import('../static/viewer.js');
    init();

    expect(Model.setModelData).toHaveBeenCalledWith(
      expect.anything(),
      expect.objectContaining({ nodes: [{ id: 'n1' }], edges: [] }),
    );

    const panel = document.getElementById('data-panel');
    const instructions = document.getElementById('landing-instructions');
    // Panel should remain collapsed when INITIAL_DATA loads
    expect(panel.classList.contains('collapsed')).toBe(true);
    expect(instructions.style.display).toBe('none');
  });
});

describe('viewer WASM loading indicator', () => {
  it('shows loading indicator when WASM is not ready', async () => {
    wasmReady = new Promise(function() {}); // never resolves
    wasmIsReady = false;

    globalThis.INITIAL_DATA = null;
    createRequiredElements();
    const { init } = await import('../static/viewer.js');
    init();

    const statusEl = document.getElementById('render-status');
    expect(statusEl.textContent).toBe('⏳ Loading parser...');
  });

  it('clears loading indicator when WASM becomes ready', async () => {
    let resolveReady;
    wasmReady = new Promise(function(resolve) {
      resolveReady = resolve;
    });
    wasmIsReady = false;

    globalThis.INITIAL_DATA = null;
    createRequiredElements();
    const { init } = await import('../static/viewer.js');
    init();

    const statusEl = document.getElementById('render-status');
    expect(statusEl.textContent).toBe('⏳ Loading parser...');

    resolveReady();
    await wasmReady;

    // Allow setTimeout to fire
    await new Promise(function(resolve) { setTimeout(resolve, 10); });
    expect(statusEl.textContent).toBe('✓ Ready');
  });
});

describe('viewer drag-and-drop', () => {
  it('rejects unsupported file types with an error', async () => {
    globalThis.INITIAL_DATA = null;
    createRequiredElements();
    const { init } = await import('../static/viewer.js');
    init();

    const panelBody = document.getElementById('data-panel-body');
    const statusEl = document.getElementById('render-status');

    const file = new File(['text'], 'notes.txt', { type: 'text/plain' });
    file._content = 'plain text';
    fireDrop(panelBody, file);

    expect(statusEl.textContent).toContain('Only .emod and .json files are supported');
    expect(statusEl.className).toContain('error');
  });

  it('accepts .emod file drops and loads content into textarea', async () => {
    globalThis.INITIAL_DATA = null;
    createRequiredElements();
    const { init } = await import('../static/viewer.js');
    init();

    const panelBody = document.getElementById('data-panel-body');
    const sourceInput = document.getElementById('source-input');
    const renderBtn = document.getElementById('render-btn');
    let clicked = false;
    renderBtn.addEventListener('click', function() { clicked = true; });

    const file = new File(['context Test {}'], 'test.emod', { type: 'text/plain' });
    file._content = 'context Test {}';
    fireDrop(panelBody, file);

    // Wait for MockFileReader's setTimeout
    await new Promise(function(resolve) { setTimeout(resolve, 10); });

    expect(sourceInput.value).toBe('context Test {}');
    expect(clicked).toBe(true);
  });

  it('does not show diagnostics badge or panel for valid file with no diagnostics', async () => {
    globalThis.INITIAL_DATA = null;
    createRequiredElements();
    const { init } = await import('../static/viewer.js');
    init();

    const badge = document.getElementById('diagnostics-badge');
    const panel = document.getElementById('diagnostics-panel');

    expect(badge.style.display).toBe('none');
    expect(panel.classList.contains('hidden')).toBe(true);
  });

  it('wires diagnostics badge click to toggleDiagnosticsPanel', async () => {
    const uiModule = await import('../static/ui.js');
    globalThis.INITIAL_DATA = null;
    createRequiredElements();
    const { init } = await import('../static/viewer.js');
    init();

    const badge = document.getElementById('diagnostics-badge');
    badge.click();

    expect(uiModule.UI.toggleDiagnosticsPanel).toHaveBeenCalledTimes(1);
  });

  it('wires diagnostics close button to hideDiagnosticsPanel', async () => {
    const uiModule = await import('../static/ui.js');
    globalThis.INITIAL_DATA = null;
    createRequiredElements();
    const { init } = await import('../static/viewer.js');
    init();

    const closeBtn = document.getElementById('diagnostics-close');
    closeBtn.click();

    expect(uiModule.UI.hideDiagnosticsPanel).toHaveBeenCalledTimes(1);
  });

  it('registers diagnostics:changed bus listener', async () => {
    const busModule = await import('../static/bus.js');
    globalThis.INITIAL_DATA = null;
    createRequiredElements();
    const { init } = await import('../static/viewer.js');
    init();

    expect(busModule.bus.on).toHaveBeenCalledWith(
      'diagnostics:changed',
      expect.any(Function),
    );
  });

  it('accepts .json file drops and loads content into textarea', async () => {
    globalThis.INITIAL_DATA = null;
    createRequiredElements();
    const { init } = await import('../static/viewer.js');
    init();

    const panelBody = document.getElementById('data-panel-body');
    const sourceInput = document.getElementById('source-input');
    const renderBtn = document.getElementById('render-btn');
    let clicked = false;
    renderBtn.addEventListener('click', function() { clicked = true; });

    const jsonContent = JSON.stringify({ nodes: [{ id: 'n1' }], edges: [] });
    const file = new File([jsonContent], 'diagram.json', { type: 'application/json' });
    file._content = jsonContent;
    fireDrop(panelBody, file);

    await new Promise(function(resolve) { setTimeout(resolve, 10); });

    expect(sourceInput.value).toBe(jsonContent);
    expect(clicked).toBe(true);
  });
});

// ─── Diagnostics click-to-highlight tests ──────────────────────────
describe('diagnostics click-to-highlight', () => {
  function setupDiagnosticsTest() {
    const container = document.createElement('div');
    container.innerHTML = `
      <div id="diagnostics-list"></div>
      <div id="diagnostics-panel"></div>
      <div id="diagnostics-badge" style="display:none"></div>
      <svg id="diagram-canvas"></svg>
    `;
    document.body.appendChild(container);

    const diagnosticsList = document.getElementById('diagnostics-list');
    const diagnosticsPanel = document.getElementById('diagnostics-panel');
    const diagnosticsBadge = document.getElementById('diagnostics-badge');
    const svg = document.getElementById('diagram-canvas');

    return { container, diagnosticsList, diagnosticsPanel, diagnosticsBadge, svg };
  }

  it('highlights matching diagram nodes when clicking a diagnostic', async () => {
    const { container, diagnosticsList, diagnosticsPanel, diagnosticsBadge, svg } = setupDiagnosticsTest();

    const uiModule = await vi.importActual('../static/ui.js');
    const storeModule = await vi.importActual('../static/store.js');
    const { UI } = uiModule;
    const store = storeModule.createStore();
    store.dom.diagnosticsList = diagnosticsList;
    store.dom.diagnosticsPanel = diagnosticsPanel;
    store.dom.diagnosticsBadge = diagnosticsBadge;
    store.dom.svg = svg;

    // Nodes with position data
    store.nodes = [
      { id: 'n1', type: 'command', label: 'Cmd1', position: { filename: 'test.cue', line: 5, column: 3 } },
      { id: 'n2', type: 'event', label: 'Evt1', position: { filename: 'test.cue', line: 10, column: 3 } },
    ];
    store.nodeById = new Map(store.nodes.map(function(n) { return [n.id, n]; }));

    const diagnostics = [
      { file: 'test.cue', line: 5, message: 'Error on line 5', severity: 'error' },
    ];
    store.diagnostics = diagnostics;

    UI.updateDiagnosticsPanel(store, diagnostics);
    UI.initDiagnosticsDelegation(store);

    const diagItem = diagnosticsList.querySelector('.diag-item');
    expect(diagItem).not.toBeNull();
    diagItem.click();

    // n1 should be highlighted
    expect(store.interaction.highlighted).toEqual({ n1: true });
    expect(diagItem.classList.contains('not-rendered')).toBe(false);
    document.body.removeChild(container);
  });

  it('shows not-rendered indicator when no matching node', async () => {
    const { container, diagnosticsList, diagnosticsPanel, diagnosticsBadge, svg } = setupDiagnosticsTest();

    const uiModule = await vi.importActual('../static/ui.js');
    const storeModule = await vi.importActual('../static/store.js');
    const { UI } = uiModule;
    const store = storeModule.createStore();
    store.dom.diagnosticsList = diagnosticsList;
    store.dom.diagnosticsPanel = diagnosticsPanel;
    store.dom.diagnosticsBadge = diagnosticsBadge;
    store.dom.svg = svg;

    // Node with position that won't match
    store.nodes = [
      { id: 'n1', type: 'command', label: 'Cmd1', position: { filename: 'other.cue', line: 99, column: 3 } },
    ];
    store.nodeById = new Map(store.nodes.map(function(n) { return [n.id, n]; }));

    const diagnostics = [
      { file: 'test.cue', line: 5, message: 'Unmatched error', severity: 'error' },
    ];
    store.diagnostics = diagnostics;

    UI.updateDiagnosticsPanel(store, diagnostics);
    UI.initDiagnosticsDelegation(store);

    const diagItem = diagnosticsList.querySelector('.diag-item');
    diagItem.click();

    expect(diagItem.classList.contains('not-rendered')).toBe(true);
    // Highlights should be empty
    expect(store.interaction.highlighted).toEqual({});
    document.body.removeChild(container);
  });

  it('shows not-rendered for diagnostic without file or line', async () => {
    const { container, diagnosticsList, diagnosticsPanel, diagnosticsBadge, svg } = setupDiagnosticsTest();

    const uiModule = await vi.importActual('../static/ui.js');
    const storeModule = await vi.importActual('../static/store.js');
    const { UI } = uiModule;
    const store = storeModule.createStore();
    store.dom.diagnosticsList = diagnosticsList;
    store.dom.diagnosticsPanel = diagnosticsPanel;
    store.dom.diagnosticsBadge = diagnosticsBadge;
    store.dom.svg = svg;

    store.nodes = [
      { id: 'n1', type: 'command', label: 'Cmd1', position: { filename: 'test.cue', line: 5, column: 3 } },
    ];
    store.nodeById = new Map(store.nodes.map(function(n) { return [n.id, n]; }));

    const diagnostics = [
      { file: '', line: 0, message: 'No location', severity: 'error' },
    ];
    store.diagnostics = diagnostics;

    UI.updateDiagnosticsPanel(store, diagnostics);
    UI.initDiagnosticsDelegation(store);

    const diagItem = diagnosticsList.querySelector('.diag-item');
    diagItem.click();

    expect(diagItem.classList.contains('not-rendered')).toBe(true);
    document.body.removeChild(container);
  });

  it('sequential clicks replace previous highlight', async () => {
    const { container, diagnosticsList, diagnosticsPanel, diagnosticsBadge, svg } = setupDiagnosticsTest();

    const uiModule = await vi.importActual('../static/ui.js');
    const storeModule = await vi.importActual('../static/store.js');
    const { UI } = uiModule;
    const store = storeModule.createStore();
    store.dom.diagnosticsList = diagnosticsList;
    store.dom.diagnosticsPanel = diagnosticsPanel;
    store.dom.diagnosticsBadge = diagnosticsBadge;
    store.dom.svg = svg;

    // Add node elements to SVG for highlight to work
    const n1El = document.createElementNS('http://www.w3.org/2000/svg', 'g');
    n1El.setAttribute('data-node-id', 'n1');
    n1El.classList.add('diagram-node');
    svg.appendChild(n1El);
    const n2El = document.createElementNS('http://www.w3.org/2000/svg', 'g');
    n2El.setAttribute('data-node-id', 'n2');
    n2El.classList.add('diagram-node');
    svg.appendChild(n2El);

    store.nodes = [
      { id: 'n1', type: 'command', label: 'Cmd1', position: { filename: 'test.cue', line: 5, column: 3 } },
      { id: 'n2', type: 'event', label: 'Evt1', position: { filename: 'test.cue', line: 10, column: 3 } },
    ];
    store.nodeById = new Map(store.nodes.map(function(n) { return [n.id, n]; }));

    const diagnostics = [
      { file: 'test.cue', line: 5, message: 'Error 1', severity: 'error' },
      { file: 'test.cue', line: 10, message: 'Error 2', severity: 'error' },
    ];
    store.diagnostics = diagnostics;

    UI.updateDiagnosticsPanel(store, diagnostics);
    UI.initDiagnosticsDelegation(store);

    const items = diagnosticsList.querySelectorAll('.diag-item');
    expect(items.length).toBe(2);

    // Click first diagnostic
    items[0].click();
    expect(store.interaction.highlighted).toEqual({ n1: true });

    // Click second diagnostic — should replace highlight
    items[1].click();
    expect(store.interaction.highlighted).toEqual({ n2: true });
    expect(n1El.classList.contains('hl')).toBe(false);
    expect(n2El.classList.contains('hl')).toBe(true);

    document.body.removeChild(container);
  });

  it('closing diagnostics panel clears highlights', async () => {
    const { container, diagnosticsList, diagnosticsPanel, diagnosticsBadge, svg } = setupDiagnosticsTest();

    const uiModule = await vi.importActual('../static/ui.js');
    const storeModule = await vi.importActual('../static/store.js');
    const { UI } = uiModule;
    const store = storeModule.createStore();
    store.dom.diagnosticsList = diagnosticsList;
    store.dom.diagnosticsPanel = diagnosticsPanel;
    store.dom.diagnosticsBadge = diagnosticsBadge;
    store.dom.svg = svg;

    const n1El = document.createElementNS('http://www.w3.org/2000/svg', 'g');
    n1El.setAttribute('data-node-id', 'n1');
    n1El.classList.add('diagram-node');
    svg.appendChild(n1El);

    store.nodes = [
      { id: 'n1', type: 'command', label: 'Cmd1', position: { filename: 'test.cue', line: 5, column: 3 } },
    ];
    store.nodeById = new Map(store.nodes.map(function(n) { return [n.id, n]; }));

    const diagnostics = [
      { file: 'test.cue', line: 5, message: 'Error', severity: 'error' },
    ];
    store.diagnostics = diagnostics;

    UI.updateDiagnosticsPanel(store, diagnostics);
    UI.initDiagnosticsDelegation(store);

    const diagItem = diagnosticsList.querySelector('.diag-item');
    diagItem.click();

    // Verify highlight is applied
    expect(store.interaction.highlighted).toEqual({ n1: true });

    // Close panel
    UI.hideDiagnosticsPanel(store);

    // Verify highlights cleared
    expect(store.interaction.highlighted).toEqual({});
    document.body.removeChild(container);
  });
});

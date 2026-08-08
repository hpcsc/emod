import { describe, it, expect, vi, beforeEach } from 'vitest';
import { installSVGGeometry } from './svg-env.js';

// Only the WASM parser is stubbed — it is the module boundary the viewer talks
// to instead of a network or filesystem. Every other module is the real one, so
// these tests assert what a user would see rather than which function ran.
let wasmReady = Promise.resolve();
let wasmIsReady = true;
let parseResult = { diagnostics: [], diagram: { nodes: [], edges: [] } };

vi.mock('../static/wasm.js', () => ({
  get ready() { return wasmReady; },
  get isReady() { return wasmIsReady; },
  parseEmod: vi.fn(() => Promise.resolve(parseResult)),
  exportEmod: vi.fn(() => Promise.resolve({ emod: '' })),
}));

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
    <div id="visibility-panel" class="hidden"></div>
    <button id="visibility-toggle"></button>
    <div id="visibility-tree"></div>
    <div id="legend-panel" class="hidden"></div>
    <button id="legend-toggle"></button>
    <div id="legend-content"></div>
    <button id="legend-close"></button>
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

function fireDrop(element, file) {
  const evt = new Event('drop', { bubbles: true, cancelable: true });
  Object.defineProperty(evt, 'dataTransfer', {
    value: { files: file ? [file] : [] },
  });
  element.dispatchEvent(evt);
  return evt;
}

function fireMouse(element, type) {
  element.dispatchEvent(new MouseEvent(type, { bubbles: true, cancelable: true }));
}

function blockFor(nodeId) {
  return document.querySelector('.diagram-node[data-node-id="' + nodeId + '"]');
}

function menuItemFor(action) {
  return document.querySelector('#ctx-menu .ctx-menu-item[data-action="' + action + '"]');
}

function fieldEditor() {
  return {
    open: document.getElementById('detail-panel').style.display === 'block',
    text: document.getElementById('dp-content').textContent,
  };
}

// MockFileReader stands in for the browser's async file read.
class MockFileReader {
  readAsText(file) {
    this.result = file._content || '';
    const self = this;
    Promise.resolve().then(function() {
      if (self.onload) self.onload({ target: self });
    });
  }
}

async function startViewer() {
  createRequiredElements();
  const { init } = await import('../static/viewer.js');
  init();
}

// billingDiagram is the node shape the exporter produces: a context holding a
// slice, which holds the elements.
function billingDiagram() {
  return {
    model_name: 'Billing',
    nodes: [
      { id: 'context-1', type: 'context', label: 'Payments', parentId: null },
      { id: 'slice-1', type: 'slice', label: 'Take Payment', parentId: 'context-1' },
      { id: 'command-1', type: 'command', label: 'TakePayment', parentId: 'slice-1' },
    ],
    edges: [],
  };
}

// flush lets queued microtasks — the file read and the parse promise — settle.
function flush() {
  return new Promise(function(resolve) { setTimeout(resolve, 0); });
}

beforeEach(() => {
  installSVGGeometry();
  document.body.innerHTML = '';
  globalThis.FileReader = MockFileReader;
  wasmReady = Promise.resolve();
  wasmIsReady = true;
  parseResult = { diagnostics: [], diagram: { nodes: [], edges: [] } };
});

describe('viewer initial state', () => {
  it('opens the data panel and shows the landing page when there is no initial data', async () => {
    globalThis.INITIAL_DATA = null;

    await startViewer();

    expect(document.getElementById('data-panel').classList.contains('collapsed')).toBe(false);
    expect(document.getElementById('model-name-display').textContent).toBe('(no model)');
    expect(document.getElementById('source-input').placeholder)
      .toBe('Paste .emod source or diagram JSON here');
    expect(document.getElementById('landing-instructions').style.display).toBe('block');
  });

  it('renders the supplied diagram and hides the landing page', async () => {
    globalThis.INITIAL_DATA = { diagram: billingDiagram() };

    await startViewer();

    expect(document.getElementById('landing-instructions').style.display).toBe('none');
    expect(document.getElementById('data-panel').classList.contains('collapsed')).toBe(true);
    expect(document.getElementById('diagram-canvas').innerHTML).toContain('TakePayment');
    expect(document.getElementById('stat-nodes').textContent).toBe('3');
  });
});

describe('viewer WASM loading indicator', () => {
  it('shows a loading indicator while the parser is still loading', async () => {
    wasmReady = new Promise(function() {}); // never resolves
    wasmIsReady = false;
    globalThis.INITIAL_DATA = null;

    await startViewer();

    expect(document.getElementById('render-status').textContent).toBe('⏳ Loading parser...');
  });

  it('clears the loading indicator once the parser is ready', async () => {
    let resolveReady;
    wasmReady = new Promise(function(resolve) { resolveReady = resolve; });
    wasmIsReady = false;
    globalThis.INITIAL_DATA = null;

    await startViewer();
    expect(document.getElementById('render-status').textContent).toBe('⏳ Loading parser...');

    resolveReady();
    await wasmReady;
    await flush();

    expect(document.getElementById('render-status').textContent).toBe('✓ Ready');
  });
});

describe('viewer drag-and-drop', () => {
  it('rejects unsupported file types with an error the user can read', async () => {
    globalThis.INITIAL_DATA = null;
    await startViewer();

    const file = new File(['text'], 'notes.txt', { type: 'text/plain' });
    file._content = 'plain text';
    fireDrop(document.getElementById('data-panel-body'), file);

    const statusEl = document.getElementById('render-status');
    expect(statusEl.textContent).toContain('Only .emod and .json files are supported');
    expect(statusEl.className).toContain('error');
  });

  it('loads a dropped .emod file into the editor and renders it', async () => {
    globalThis.INITIAL_DATA = null;
    parseResult = { diagnostics: [], diagram: billingDiagram() };
    await startViewer();

    const file = new File(['context Test {}'], 'test.emod', { type: 'text/plain' });
    file._content = 'context Test {}';
    fireDrop(document.getElementById('data-panel-body'), file);
    await flush();

    expect(document.getElementById('source-input').value).toBe('context Test {}');
    expect(document.getElementById('diagram-canvas').innerHTML).toContain('TakePayment');
  });

  it('loads a dropped diagram .json file into the editor and renders it', async () => {
    globalThis.INITIAL_DATA = null;
    await startViewer();

    const jsonContent = JSON.stringify(billingDiagram());
    const file = new File([jsonContent], 'diagram.json', { type: 'application/json' });
    file._content = jsonContent;
    fireDrop(document.getElementById('data-panel-body'), file);
    await flush();

    expect(document.getElementById('source-input').value).toBe(jsonContent);
    expect(document.getElementById('diagram-canvas').innerHTML).toContain('TakePayment');
  });
});

describe('viewer field editor', () => {
  async function showBillingDiagram() {
    globalThis.INITIAL_DATA = { diagram: billingDiagram() };
    await startViewer();
  }

  function openFieldEditorFor(nodeId) {
    fireMouse(blockFor(nodeId), 'contextmenu');
    menuItemFor('open-field-editor').click();
  }

  it('stays shut when a block is left-clicked', async () => {
    await showBillingDiagram();

    fireMouse(blockFor('command-1'), 'click');

    expect(fieldEditor()).toEqual({ open: false, text: '' });
  });

  it('opens on the block the context menu was raised over', async () => {
    await showBillingDiagram();

    fireMouse(blockFor('command-1'), 'contextmenu');
    expect(menuItemFor('open-field-editor').textContent).toBe('Open field editor');

    menuItemFor('open-field-editor').click();

    expect(fieldEditor().open).toBe(true);
    expect(fieldEditor().text).toContain('TakePayment');
    expect(document.getElementById('ctx-menu').style.display).toBe('none');
  });

  it('survives a left click on the block it is editing', async () => {
    await showBillingDiagram();
    openFieldEditorFor('command-1');

    fireMouse(blockFor('command-1'), 'click');

    expect(fieldEditor().open).toBe(true);
  });

  it('closes when the click lands on the canvas instead of a block', async () => {
    await showBillingDiagram();
    openFieldEditorFor('command-1');

    fireMouse(document.getElementById('diagram-canvas'), 'click');

    expect(fieldEditor().open).toBe(false);
  });
});

describe('viewer diagnostics panel', () => {
  it('stays hidden while the model has no diagnostics', async () => {
    globalThis.INITIAL_DATA = null;
    await startViewer();

    expect(document.getElementById('diagnostics-badge').style.display).toBe('none');
    expect(document.getElementById('diagnostics-panel').classList.contains('hidden')).toBe(true);
  });

  it('shows a badge listing the diagnostics a render produced', async () => {
    globalThis.INITIAL_DATA = null;
    parseResult = {
      diagnostics: [{ file: 'test.emod', line: 3, message: 'unrecognized keyword', severity: 'error' }],
      diagram: { nodes: [], edges: [] },
    };
    await startViewer();

    document.getElementById('source-input').value = 'foobar {}';
    document.getElementById('render-btn').click();
    await flush();

    const badge = document.getElementById('diagnostics-badge');
    expect(badge.style.display).toBe('inline-block');
    expect(badge.textContent).toBe('1 error');
    expect(document.getElementById('diagnostics-list').textContent)
      .toContain('unrecognized keyword');
  });

  it('opens the panel when the badge is clicked and closes it again', async () => {
    globalThis.INITIAL_DATA = null;
    parseResult = {
      diagnostics: [{ file: 'test.emod', line: 3, message: 'unrecognized keyword', severity: 'error' }],
      diagram: { nodes: [], edges: [] },
    };
    await startViewer();

    document.getElementById('render-btn').click();
    await flush();

    const panel = document.getElementById('diagnostics-panel');
    document.getElementById('diagnostics-badge').click();
    expect(panel.classList.contains('hidden')).toBe(false);

    document.getElementById('diagnostics-close').click();
    expect(panel.classList.contains('hidden')).toBe(true);
  });
});

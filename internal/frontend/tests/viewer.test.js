import { describe, it, expect, vi, beforeEach } from 'vitest';
import { readFileSync, readdirSync } from 'fs';
import { resolve } from 'path';
import { installSVGGeometry } from './svg-env.js';

// Only the platform seam is stubbed — it is the module boundary the viewer talks
// to instead of a network, a filesystem or a Go core. Every other module is the
// real one, so these tests assert what a user would see rather than which ran.
let platformReady = Promise.resolve();
let platformIsReady = true;
let parseResult = { diagnostics: [], diagram: { nodes: [], edges: [] } };
let dropReadFails = false;
let savedFile = null;
let exportFails = false;
let initialStateFails = false;
let parseQueue = [];
let windowTitle = '';
let deliverFile = null;

vi.mock('../static/platform.js', () => ({
  get ready() { return platformReady; },
  get isReady() { return platformIsReady; },
  parseEmod: vi.fn(() => parseQueue.length ? parseQueue.shift() : Promise.resolve(parseResult)),
  exportEmod: vi.fn((diagram) => exportFails
    ? Promise.reject(new Error('nothing to export'))
    : Promise.resolve('emod 1\nmodel "' + (diagram.model_name || '') + '"\n')),
  initialState: vi.fn(() => initialStateFails
    ? Promise.reject(new Error('host could not answer'))
    : Promise.resolve(typeof globalThis.INITIAL_DATA === 'undefined' ? null : globalThis.INITIAL_DATA)),
  saveFile: vi.fn((name, content) => { savedFile = { name, content }; return Promise.resolve(); }),
  setWindowTitle: vi.fn((title) => { windowTitle = title; }),
  onFileOpened: vi.fn((handler) => { deliverFile = handler; }),
  droppedFile: vi.fn((dataTransfer) => {
    const file = dataTransfer.files[0];
    if (!file) return null;
    return {
      name: file.name,
      path: '',
      read: () => dropReadFails
        ? Promise.reject(new Error('Failed to read file'))
        : Promise.resolve(file._content || ''),
    };
  }),
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
    <span id="stat-file">File: <span class="stat-value" id="stat-file-path"></span></span>
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

async function startViewer() {
  createRequiredElements();
  const { init } = await import('../static/viewer.js');
  await init();
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
  platformReady = Promise.resolve();
  platformIsReady = true;
  parseResult = { diagnostics: [], diagram: { nodes: [], edges: [] } };
  dropReadFails = false;
  savedFile = null;
  exportFails = false;
  initialStateFails = false;
  parseQueue = [];
  windowTitle = '';
  deliverFile = null;
});

describe('the window is named through the host, not by assigning document.title', () => {
  it('names the window after the model that rendered', async () => {
    globalThis.INITIAL_DATA = { diagram: billingDiagram() };

    await startViewer();

    expect(windowTitle).toBe('Billing — Emod Diagram Viewer');
  });

  it('falls back to the viewer name when the model has none', async () => {
    globalThis.INITIAL_DATA = { diagram: { ...billingDiagram(), model_name: '' } };

    await startViewer();

    expect(windowTitle).toBe('Emod Diagram Viewer');
  });
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

describe('viewer initial load failure', () => {
  it('reports a host that cannot say what to open, rather than opening blank', async () => {
    initialStateFails = true;

    await startViewer();

    const statusEl = document.getElementById('render-status');
    expect(statusEl.textContent).toContain('host could not answer');
    expect(statusEl.className).toContain('error');
    expect(document.getElementById('data-panel').classList.contains('collapsed')).toBe(false);
    expect(document.getElementById('source-input').placeholder)
      .toBe('Paste .emod source or diagram JSON here');
  });
});

describe('viewer parser loading indicator', () => {
  it('shows a loading indicator while the parser is still loading', async () => {
    platformReady = new Promise(function() {}); // never resolves
    platformIsReady = false;
    globalThis.INITIAL_DATA = null;

    await startViewer();

    expect(document.getElementById('render-status').textContent).toBe('⏳ Loading parser...');
  });

  it('clears the loading indicator once the parser is ready', async () => {
    let resolveReady;
    platformReady = new Promise(function(resolve) { resolveReady = resolve; });
    platformIsReady = false;
    globalThis.INITIAL_DATA = null;

    await startViewer();
    expect(document.getElementById('render-status').textContent).toBe('⏳ Loading parser...');

    resolveReady();
    await platformReady;
    await flush();

    expect(document.getElementById('render-status').textContent).toBe('✓ Ready');
  });
});

function diagramNamed(name) {
  return {
    model_name: name,
    nodes: [
      { id: 'context-1', type: 'context', label: 'Payments', parentId: null },
      { id: 'slice-1', type: 'slice', label: 'Take Payment', parentId: 'context-1' },
      { id: 'command-1', type: 'command', label: name + 'Cmd', parentId: 'slice-1' },
    ],
    edges: [],
  };
}

describe('viewer overlapping renders', () => {
  it('keeps the newest render when an older parse fails after it', async () => {
    globalThis.INITIAL_DATA = null;
    await startViewer();

    let rejectOlder;
    parseQueue = [
      new Promise(function(_, reject) { rejectOlder = reject; }),
      Promise.resolve({ diagnostics: [], diagram: diagramNamed('Newer') }),
    ];

    document.getElementById('source-input').value = 'context Older {}';
    document.getElementById('render-btn').click();
    await flush();

    document.getElementById('source-input').value = 'context Newer {}';
    document.getElementById('render-btn').click();
    await flush();

    rejectOlder(new Error('older source is unparseable'));
    await flush();

    // The older parse failing must not report over the render that replaced it:
    // the canvas holds Newer, so the status has to agree.
    expect(document.getElementById('render-status').textContent).toBe('✓ Rendered');
    expect(document.getElementById('render-status').className).not.toContain('error');
    expect(document.getElementById('diagram-canvas').innerHTML).toContain('NewerCmd');
  });

  it('keeps the newest render when an older parse answers after it', async () => {
    globalThis.INITIAL_DATA = null;
    await startViewer();

    let answerOlder;
    parseQueue = [
      new Promise(function(resolve) { answerOlder = resolve; }),
      Promise.resolve({ diagnostics: [], diagram: diagramNamed('Newer') }),
    ];

    document.getElementById('source-input').value = 'context Older {}';
    document.getElementById('render-btn').click();
    // Let the first render reach the platform before starting the second: both
    // go through a dynamic import, and two racing at once resolve separately.
    await flush();

    document.getElementById('source-input').value = 'context Newer {}';
    document.getElementById('render-btn').click();
    await flush();

    expect(document.getElementById('diagram-canvas').innerHTML).toContain('NewerCmd');

    answerOlder({ diagnostics: [], diagram: diagramNamed('Older') });
    await flush();

    expect(document.getElementById('diagram-canvas').innerHTML).toContain('NewerCmd');
    expect(document.getElementById('diagram-canvas').innerHTML).not.toContain('OlderCmd');
    expect(document.getElementById('model-name-display').textContent).toBe('Newer');
  });
});

describe('viewer export', () => {
  it('hands the platform the model name and the exported content', async () => {
    globalThis.INITIAL_DATA = { diagram: billingDiagram() };
    await startViewer();

    document.getElementById('export-emod').click();
    await flush();

    expect(savedFile.name).toBe('Billing.emod');
    expect(savedFile.content).toBe('emod 1\nmodel "Billing"\n');
  });

  it('falls back to diagram.emod when the model has no name', async () => {
    const unnamed = billingDiagram();
    unnamed.model_name = '';
    globalThis.INITIAL_DATA = { diagram: unnamed };
    await startViewer();

    document.getElementById('export-emod').click();
    await flush();

    expect(savedFile.name).toBe('diagram.emod');
  });

  it('reports a failed export in the status area', async () => {
    globalThis.INITIAL_DATA = { diagram: billingDiagram() };
    exportFails = true;
    await startViewer();

    document.getElementById('export-emod').click();
    await flush();

    const statusEl = document.getElementById('render-status');
    expect(statusEl.textContent).toContain('nothing to export');
    expect(statusEl.className).toContain('error');
    expect(savedFile).toBeNull();
  });

  it('reveals the panel so the reason a save failed can be read', async () => {
    globalThis.INITIAL_DATA = { diagram: billingDiagram() };
    exportFails = true;
    await startViewer();
    expect(document.getElementById('data-panel').classList.contains('collapsed')).toBe(true);

    document.getElementById('export-emod').click();
    await flush();

    expect(document.getElementById('data-panel').classList.contains('collapsed')).toBe(false);
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

  it('reports a read that fails and leaves the current diagram on screen', async () => {
    globalThis.INITIAL_DATA = { diagram: billingDiagram() };
    await startViewer();
    dropReadFails = true;

    const file = new File(['context Test {}'], 'test.emod', { type: 'text/plain' });
    fireDrop(document.getElementById('data-panel-body'), file);
    await flush();

    const statusEl = document.getElementById('render-status');
    expect(statusEl.textContent).toContain('Failed to read file');
    expect(statusEl.className).toContain('error');
    expect(document.getElementById('source-input').value).toBe('');
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

describe('a file the host opens', () => {
  const billingSource = 'emod 1\nmodel "Billing"\n';

  async function openFile(file) {
    await startViewer();
    deliverFile(file);
    await flush();
  }

  it('renders it with no further click, exactly as the same text pasted and rendered does', async () => {
    globalThis.INITIAL_DATA = null;
    parseResult = { diagnostics: [], diagram: billingDiagram() };

    await openFile({ name: 'billing.emod', path: '/models/billing.emod', content: billingSource });
    const opened = {
      canvas: document.getElementById('diagram-canvas').innerHTML,
      modelName: document.getElementById('model-name-display').textContent,
      nodes: document.getElementById('stat-nodes').textContent,
    };

    document.body.innerHTML = '';
    await startViewer();
    document.getElementById('source-input').value = billingSource;
    document.getElementById('render-btn').click();
    await flush();

    expect(opened.canvas).toBe(document.getElementById('diagram-canvas').innerHTML);
    expect(opened.modelName).toBe(document.getElementById('model-name-display').textContent);
    expect(opened.nodes).toBe(document.getElementById('stat-nodes').textContent);
    expect(opened.canvas).toContain('TakePayment');
  });

  it('puts its source in the panel, so the panel and the canvas show one model', async () => {
    globalThis.INITIAL_DATA = null;
    parseResult = { diagnostics: [], diagram: billingDiagram() };

    await openFile({ name: 'billing.emod', path: '/models/billing.emod', content: billingSource });

    expect(document.getElementById('source-input').value).toBe(billingSource);
  });

  it('opens a file with validation errors, listing them as pasted source with the same errors does', async () => {
    globalThis.INITIAL_DATA = null;
    parseResult = {
      diagnostics: [{ file: 'billing.emod', line: 3, message: 'unrecognized keyword', severity: 'error' }],
      diagram: billingDiagram(),
    };

    await openFile({ name: 'billing.emod', path: '/models/billing.emod', content: billingSource });

    expect(document.getElementById('diagram-canvas').innerHTML).toContain('TakePayment');
    expect(document.getElementById('diagnostics-badge').textContent).toBe('1 error');
    expect(document.getElementById('diagnostics-list').textContent).toContain('unrecognized keyword');
    expect(document.getElementById('diagnostics-panel').classList.contains('hidden')).toBe(false);
  });

  it('names the window after the file rather than the model inside it', async () => {
    globalThis.INITIAL_DATA = null;
    parseResult = { diagnostics: [], diagram: billingDiagram() };

    await openFile({ name: 'billing.emod', path: '/models/billing.emod', content: billingSource });

    expect(windowTitle).toBe('billing.emod — Emod Diagram Viewer');
  });

  it('shows the full path in the bar that is always on screen', async () => {
    globalThis.INITIAL_DATA = null;
    parseResult = { diagnostics: [], diagram: billingDiagram() };

    await openFile({ name: 'billing.emod', path: '/Users/me/models/billing.emod', content: billingSource });

    const bar = document.getElementById('stat-file');
    expect(bar.textContent).toContain('/Users/me/models/billing.emod');
    // The bar truncates a path wider than the space left in it, so the tooltip
    // is the only place the whole path stays readable.
    expect(bar.title).toBe('/Users/me/models/billing.emod');
    // jsdom has no layout, so being on screen can only be checked through the
    // class viewer.html styles as display:none. Asserting the text alone passes
    // with the bar hidden.
    expect(bar.classList.contains('hidden')).toBe(false);
  });

  it('keeps the bar hidden until a file is open, so the browser viewer shows no empty stat', async () => {
    globalThis.INITIAL_DATA = { diagram: billingDiagram() };

    await startViewer();

    expect(document.getElementById('stat-file').classList.contains('hidden')).toBe(true);
  });

});

describe('what the window names', () => {
  const billingSource = 'emod 1\nmodel "Billing"\n';

  it('keeps naming the model on screen when a delivered file will not render', async () => {
    globalThis.INITIAL_DATA = null;
    parseResult = { diagnostics: [], diagram: billingDiagram() };
    await startViewer();
    deliverFile({ name: 'billing.emod', path: '/models/billing.emod', content: billingSource });
    await flush();

    deliverFile({ name: 'empty.emod', path: '/models/empty.emod', content: '   ' });
    await flush();

    expect(windowTitle).toBe('billing.emod — Emod Diagram Viewer');
    expect(document.getElementById('stat-file').textContent).toContain('/models/billing.emod');
    expect(document.getElementById('diagram-canvas').innerHTML).toContain('TakePayment');
  });

  it('stops naming the opened file once a dropped model replaces it', async () => {
    globalThis.INITIAL_DATA = null;
    parseResult = { diagnostics: [], diagram: billingDiagram() };
    await startViewer();
    deliverFile({ name: 'billing.emod', path: '/models/billing.emod', content: billingSource });
    await flush();

    const dropped = new File([''], 'hotel.emod');
    dropped._content = 'emod 1\nmodel "Hotel"\n';
    parseResult = { diagnostics: [], diagram: { ...billingDiagram(), model_name: 'Hotel' } };
    fireDrop(document.getElementById('data-panel-body'), dropped);
    await flush();
    await flush();

    expect(windowTitle).toBe('Hotel — Emod Diagram Viewer');
    expect(document.getElementById('stat-file').classList.contains('hidden')).toBe(true);
  });

  // A file stays open across edits, the way it does in a text editor: the panel
  // is that file's text, whatever has been typed into it, and Save writes back
  // there. Only another file arriving replaces it.
  it('keeps naming the opened file after its source is edited and re-rendered', async () => {
    globalThis.INITIAL_DATA = null;
    parseResult = { diagnostics: [], diagram: billingDiagram() };
    await startViewer();
    deliverFile({ name: 'billing.emod', path: '/models/billing.emod', content: billingSource });
    await flush();

    parseResult = { diagnostics: [], diagram: { ...billingDiagram(), model_name: 'Hotel' } };
    document.getElementById('source-input').value = 'emod 1\nmodel "Hotel"\n';
    document.getElementById('render-btn').click();
    await flush();

    expect(windowTitle).toBe('billing.emod — Emod Diagram Viewer');
    expect(document.getElementById('stat-file').textContent).toContain('/models/billing.emod');
  });

  it('keeps naming a file whose line endings the panel rewrote', async () => {
    globalThis.INITIAL_DATA = null;
    parseResult = { diagnostics: [], diagram: billingDiagram() };
    await startViewer();
    const delivered = 'emod 1\r\nmodel "Billing"\r\n';
    deliverFile({ name: 'crlf.emod', path: '/models/crlf.emod', content: delivered });
    await flush();

    // The panel normalises CR/CRLF to LF on the way in, so its text is no longer
    // what the file delivered — which is exactly what an identity rule built on
    // comparing the two would trip over.
    expect(document.getElementById('source-input').value).not.toBe(delivered);
    expect(document.getElementById('source-input').value).toBe('emod 1\nmodel "Billing"\n');

    document.getElementById('render-btn').click();
    await flush();

    expect(windowTitle).toBe('crlf.emod — Emod Diagram Viewer');
    expect(document.getElementById('stat-file').textContent).toContain('/models/crlf.emod');
  });

  it('stops naming a file whose failed open would otherwise have claimed the panel', async () => {
    globalThis.INITIAL_DATA = null;
    parseResult = { diagnostics: [], diagram: billingDiagram() };
    await startViewer();
    deliverFile({ name: 'billing.emod', path: '/models/billing.emod', content: billingSource });
    await flush();

    parseQueue = [Promise.reject(new Error('unparseable'))];
    deliverFile({ name: 'broken.emod', path: '/models/broken.emod', content: 'emod 1\nnot a model\n' });
    await flush();

    parseResult = { diagnostics: [], diagram: { ...billingDiagram(), model_name: 'Hotel' } };
    document.getElementById('render-btn').click();
    await flush();

    // broken.emod never rendered, so it never became the open file — the render
    // that follows must not inherit its name.
    expect(windowTitle).toBe('billing.emod — Emod Diagram Viewer');
    expect(document.getElementById('stat-file').textContent).toContain('/models/billing.emod');
  });
});

describe('a file the host read but the pipeline will not render', () => {
  async function openThenDeliver(second, beforeSecond) {
    globalThis.INITIAL_DATA = null;
    parseResult = { diagnostics: [], diagram: billingDiagram() };
    await startViewer();
    deliverFile({ name: 'billing.emod', path: '/models/billing.emod', content: 'emod 1\nmodel "Billing"\n' });
    await flush();
    expect(document.getElementById('data-panel').classList.contains('collapsed')).toBe(true);
    if (beforeSecond) beforeSecond();
    deliverFile(second);
    await flush();
  }

  it('opens the panel holding the reason, so choosing a file never looks like nothing happened', async () => {
    await openThenDeliver({ name: 'empty.emod', path: '/models/empty.emod', content: '   ' });

    expect(document.getElementById('data-panel').classList.contains('collapsed')).toBe(false);
    expect(document.getElementById('render-status').className).toContain('error');
  });

  it('names the file rather than repeating a message written for someone who pressed Render', async () => {
    await openThenDeliver({ name: 'empty.emod', path: '/models/empty.emod', content: '   ' });

    expect(document.getElementById('render-status').textContent).toContain('empty.emod is empty');
    expect(document.getElementById('render-status').textContent).not.toContain('no source');
  });

  it('holds its reason against a render still in flight when the failure arrives', async () => {
    globalThis.INITIAL_DATA = null;
    parseResult = { diagnostics: [], diagram: billingDiagram() };
    await startViewer();

    let answerSlow;
    parseQueue = [new Promise(function(resolve) { answerSlow = resolve; })];
    document.getElementById('source-input').value = 'context Slow {}';
    document.getElementById('render-btn').click();
    await flush();

    deliverFile({ error: 'reading /models/gone.emod: no such file or directory' });
    await flush();

    answerSlow({ diagnostics: [], diagram: billingDiagram() });
    await flush();

    // The slow render resolving last must not paint over the failure, nor
    // re-collapse the panel that is the only place it can be read.
    expect(document.getElementById('render-status').textContent).toContain('no such file or directory');
    expect(document.getElementById('data-panel').classList.contains('collapsed')).toBe(false);
  });

  it('leaves the panel holding the model on screen, which the title and the path still name', async () => {
    await openThenDeliver({ name: 'broken.emod', path: '/models/broken.emod', content: 'emod 1\nnot a model\n' }, () => {
      parseQueue = [Promise.reject(new Error('unparseable'))];
    });

    expect(document.getElementById('source-input').value).toBe('emod 1\nmodel "Billing"\n');
    expect(windowTitle).toBe('billing.emod — Emod Diagram Viewer');
    expect(document.getElementById('stat-file').textContent).toContain('/models/billing.emod');
  });
});

describe('a file the host could not read', () => {
  async function openThenFail() {
    globalThis.INITIAL_DATA = null;
    parseResult = { diagnostics: [], diagram: billingDiagram() };
    await startViewer();
    deliverFile({ name: 'billing.emod', path: '/models/billing.emod', content: 'emod 1\n' });
    await flush();
    deliverFile({ error: 'reading /models/gone.emod: no such file or directory' });
    await flush();
  }

  it('reports the reason it was given, rather than a failure of its own wording', async () => {
    await openThenFail();

    expect(document.getElementById('render-status').textContent)
      .toContain('reading /models/gone.emod: no such file or directory');
    expect(document.getElementById('render-status').className).toContain('error');
  });

  it('opens the panel holding that reason, which a render had collapsed', async () => {
    await openThenFail();

    expect(document.getElementById('data-panel').classList.contains('collapsed')).toBe(false);
  });

  it('leaves the diagram, the model name and the path showing the model already open', async () => {
    await openThenFail();

    expect(document.getElementById('diagram-canvas').innerHTML).toContain('TakePayment');
    expect(document.getElementById('model-name-display').textContent).toBe('Billing');
    expect(document.getElementById('stat-file').textContent).toContain('/models/billing.emod');
    expect(windowTitle).toBe('billing.emod — Emod Diagram Viewer');
  });

  it('leaves the source panel holding the model already open, not the file it could not read', async () => {
    await openThenFail();

    expect(document.getElementById('source-input').value).toBe('emod 1\n');
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

  it('shows the panel on a render that reports, and toggles it from the badge', async () => {
    globalThis.INITIAL_DATA = null;
    parseResult = {
      diagnostics: [{ file: 'test.emod', line: 3, message: 'unrecognized keyword', severity: 'error' }],
      diagram: { nodes: [], edges: [] },
    };
    await startViewer();

    document.getElementById('source-input').value = 'foobar {}';
    document.getElementById('render-btn').click();
    await flush();

    const panel = document.getElementById('diagnostics-panel');
    // A render that reports opens the panel itself, so the badge toggles from
    // open. Without source the render rejects and none of this is reached.
    expect(document.getElementById('diagnostics-badge').textContent).toBe('1 error');
    expect(panel.classList.contains('hidden')).toBe(false);

    document.getElementById('diagnostics-badge').click();
    expect(panel.classList.contains('hidden')).toBe(true);

    document.getElementById('diagnostics-badge').click();
    expect(panel.classList.contains('hidden')).toBe(false);

    document.getElementById('diagnostics-close').click();
    expect(panel.classList.contains('hidden')).toBe(true);
  });
});

// The fixture above is hand-written, so it cannot notice viewer.html losing an
// element init() reaches for. `init` looks each one up unguarded and throws on
// the first miss, killing every listener wired after it — on the shipped page
// only, with the suite green. This pins the page against the code that reads it.
describe('viewer.html satisfies what init reads from it', () => {
  // init() also calls Minimap.initMinimap, which looks up two more ids and
  // dereferences both, so scanning viewer.js alone leaves the page free to lose
  // them and kill the app with the suite green.
  const sources = ['../static/viewer.js', '../static/minimap.js']
    .map((f) => readFileSync(resolve(__dirname, f), 'utf-8'))
    .join('\n');
  const viewerHtml = readFileSync(resolve(__dirname, '../static/viewer.html'), 'utf-8');
  const required = [...sources.matchAll(/getElementById\((['"])([^'"]+)\1\)/g)].map((m) => m[2]);

  it('reads more than a handful of ids, so the scan below is not matching nothing', () => {
    expect(required.length).toBeGreaterThan(10);
    expect(required).toContain('legend-close');
    // single-quoted, and the reason the pattern takes both quote styles
    expect(required).toContain('landing-instructions');
    // reached through initMinimap, and the reason more than viewer.js is scanned
    expect(required).toContain('minimap-handle');
  });

  it.each([...new Set(required)])('declares id="%s"', (id) => {
    expect(viewerHtml).toContain(`id="${id}"`);
  });
});

// The web bundle and the CLI binary ship static/ wholesale. A shared module
// reaching the Wails runtime or the generated bindings compiles and tests fine
// here, and gives the browser viewer a page that dies on a module it cannot
// fetch — with every suite green, because the desktop stubs answer for both.
describe('only the desktop implementation reaches the host', () => {
  const shared = readdirSync(resolve(__dirname, '../static'))
    .filter((f) => f.endsWith('.js'));

  it('scans every shared module, so the check below is not scanning nothing', () => {
    expect(shared).toContain('viewer.js');
    expect(shared).toContain('model.js');
    expect(shared.length).toBeGreaterThan(10);
  });

  it.each(shared)('%s imports neither the Wails runtime nor the generated bindings', (file) => {
    const src = readFileSync(resolve(__dirname, '../static', file), 'utf-8');

    expect(src).not.toContain('/wails/runtime.js');
    expect(src).not.toContain('/bindings/');
  });

  it('is checking for text the desktop implementation actually contains', () => {
    const src = readFileSync(resolve(__dirname, '../desktop/platform.desktop.js'), 'utf-8');

    expect(src).toContain('/wails/runtime.js');
    expect(src).toContain('/bindings/');
  });
});

// The seam is written three times — the contract and its two implementations —
// and the shared modules import names from it by hand. A name added to one copy
// and used by a shared module gives the distribution built from the other copy a
// blank window, with every Go and JS suite green. This reads no window, so it
// does not drive the desktop shell.
describe('the platform seam has one contract', () => {
  const exportsOf = (path) => {
    const src = readFileSync(resolve(__dirname, path), 'utf-8');
    const m = src.match(/export \{([^}]*)\}/);
    expect(m, `${path} must declare a single export block`).not.toBeNull();
    return m[1]
      .split(',')
      .map((t) => t.trim().split(/\s+/)[0])
      .filter(Boolean)
      .sort();
  };

  const contract = exportsOf('../static/platform.js');
  const browser = exportsOf('../static/platform.browser.js');
  const desktop = exportsOf('../desktop/platform.desktop.js');

  it('names every host operation the shared modules reach for', () => {
    expect(contract).toEqual(
      ['droppedFile', 'exportEmod', 'initialState', 'isReady', 'onFileOpened', 'parseEmod', 'ready',
       'saveFile', 'setWindowTitle'],
    );
  });

  it('is satisfied by the browser implementation', () => {
    expect(browser).toEqual(contract);
  });

  it('is satisfied by the desktop implementation', () => {
    expect(desktop).toEqual(contract);
  });
});

describe('viewer legend', () => {
  it('opens and closes the legend from the toolbar, and closes it from the panel', async () => {
    await startViewer();

    const panel = document.getElementById('legend-panel');
    expect(panel.classList.contains('hidden')).toBe(true);

    document.getElementById('legend-toggle').click();
    expect(panel.classList.contains('hidden')).toBe(false);
    expect(document.querySelectorAll('#legend-content .lg-row').length).toBeGreaterThan(0);

    document.getElementById('legend-close').click();
    expect(panel.classList.contains('hidden')).toBe(true);
  });
});

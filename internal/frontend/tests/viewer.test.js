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
let deliverDroppedFiles = null;
let saveAnswer = null;
let saveFails = null;
let saveHangs = null;
let savedContents = [];
let requestSave = null;
let modifiedReports = [];
let unsavedEditsAnswer = 'discard';
let unsavedEditsAsked = 0;
let requestLeave = null;
// Set by a test that has to answer several dialogs out of order.
let unsavedEditsAnswerer = null;
let remembered = [];
let rememberFails = null;
// Set by a test that has to hold a recording's answer open.
let rememberAnswerer = null;

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
  saveFile: vi.fn((name, content, path) => {
    savedFile = { name, content, path };
    savedContents.push(content);
    const settle = () => (saveFails ? Promise.reject(new Error(saveFails)) : Promise.resolve(saveAnswer));
    return saveHangs ? saveHangs.then(settle) : settle();
  }),
  onSaveRequested: vi.fn((handler) => { requestSave = handler; }),
  onLeaveRequested: vi.fn((handler) => { requestLeave = handler; }),
  setWindowTitle: vi.fn((title) => { windowTitle = title; }),
  setWindowModified: vi.fn((modified) => { modifiedReports.push(modified); }),
  rememberOpenedFile: vi.fn((path) => {
    remembered.push(path);
    if (rememberAnswerer) {
      return rememberAnswerer(path);
    }
    return rememberFails ? Promise.reject(new Error(rememberFails)) : Promise.resolve();
  }),
  resolveUnsavedEdits: vi.fn(() => {
    unsavedEditsAsked++;
    return unsavedEditsAnswerer
      ? unsavedEditsAnswerer()
      : Promise.resolve(unsavedEditsAnswer);
  }),
  onFileOpened: vi.fn((handler) => { deliverFile = handler; }),
  onFilesDropped: vi.fn((handler) => { deliverDroppedFiles = handler; }),
  droppedFiles: vi.fn((dataTransfer) => Array.from(dataTransfer.files).map((file) => ({
    name: file.name,
    read: () => Promise.resolve(dropReadFails
      ? { error: 'Failed to read file' }
      : { name: file.name, path: '', content: file._content || '' }),
  }))),
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
    <span id="save-status" class="hidden"></span>
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

function fireDrop(element, ...files) {
  const evt = new Event('drop', { bubbles: true, cancelable: true });
  Object.defineProperty(evt, 'dataTransfer', { value: { files } });
  element.dispatchEvent(evt);
  return evt;
}

function fireMouse(element, type) {
  element.dispatchEvent(new MouseEvent(type, { bubbles: true, cancelable: true }));
}

function fireMouseAt(element, type, x, y) {
  element.dispatchEvent(new MouseEvent(type, {
    bubbles: true, cancelable: true, button: 0, clientX: x, clientY: y,
  }));
}

// The first move only crosses the threshold that tells a drag from a click; the
// gesture's own distance is the second, so a caller's dx and dy are what lands.
function dragBy(element, dx, dy) {
  fireMouseAt(element, 'mousedown', 100, 100);
  fireMouseAt(document, 'mousemove', 100 + DRAG_THRESHOLD + 1, 100);
  fireMouseAt(document, 'mousemove', 100 + dx, 100 + dy);
  fireMouseAt(document, 'mouseup', 100 + dx, 100 + dy);
}

function typeIntoPanel(text) {
  const panel = document.getElementById('source-input');
  panel.value = text;
  panel.dispatchEvent(new Event('input', { bubbles: true }));
}

const marked = () => modifiedReports[modifiedReports.length - 1];

function hideFromVisibilityTree(nodeId) {
  const box = document.querySelector(
    '#visibility-tree [data-node-id="' + nodeId + '"] input[type="checkbox"]');
  box.checked = false;
  box.dispatchEvent(new Event('change'));
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

const { sourceToSave } = await import('../static/viewer.js');
const platform = await import('../static/platform.js');
const { DRAG_THRESHOLD } = await import('../static/config.js');

const billingSource = 'emod 1\nmodel "Billing"\n';
const crlfSource = 'emod 1\r\nmodel "Billing"\r\n';

async function openBilling(content) {
  globalThis.INITIAL_DATA = null;
  parseResult = { diagnostics: [], diagram: billingDiagram() };
  await startViewer();
  deliverFile({
    name: 'billing.emod',
    path: '/models/billing.emod',
    content: content === undefined ? billingSource : content,
  });
  await flush();
}

async function startEmpty() {
  globalThis.INITIAL_DATA = null;
  parseResult = { diagnostics: [], diagram: billingDiagram() };
  await startViewer();
}

function save(options) {
  return Promise.resolve(requestSave(options || { chooseLocation: false })).then(flush);
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
  deliverDroppedFiles = null;
  saveAnswer = null;
  saveFails = null;
  saveHangs = null;
  savedContents = [];
  requestSave = null;
  modifiedReports = [];
  unsavedEditsAnswer = 'discard';
  unsavedEditsAsked = 0;
  requestLeave = null;
  unsavedEditsAnswerer = null;
  remembered = [];
  rememberFails = null;
  rememberAnswerer = null;
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

  // Export writes the model re-serialised from the diagram, which has dropped
  // the author's comments. Handing it a path would overwrite the open file with
  // that, which is the one thing Save exists not to do.
  it('asks for a location rather than writing over the open file', async () => {
    globalThis.INITIAL_DATA = null;
    parseResult = { diagnostics: [], diagram: billingDiagram() };
    await startViewer();
    deliverFile({ name: 'billing.emod', path: '/models/billing.emod', content: 'emod 1\nmodel "Billing"\n' });
    await flush();

    document.getElementById('export-emod').click();
    await flush();

    expect(savedFile.path).toBeFalsy();
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

  // A drop is one gesture and opens one model, so several files are a choice
  // rather than an error — and the ones ahead of the chosen file are skipped
  // rather than refusing the whole drop.
  it('opens the first model file the drop carried, past the ones it cannot open', async () => {
    globalThis.INITIAL_DATA = null;
    parseResult = { diagnostics: [], diagram: { ...billingDiagram(), model_name: 'Hotel' } };
    await startViewer();

    const notes = new File([''], 'notes.txt');
    notes._content = 'plain text';
    const hotel = new File([''], 'hotel.emod');
    hotel._content = 'emod 1\nmodel "Hotel"\n';
    const other = new File([''], 'other.emod');
    other._content = 'emod 1\nmodel "Other"\n';
    fireDrop(document.getElementById('data-panel-body'), notes, hotel, other);
    await flush();
    await flush();

    expect(document.getElementById('source-input').value).toBe('emod 1\nmodel "Hotel"\n');
    expect(windowTitle).toBe('hotel.emod — Emod Diagram Viewer');
  });

  // The status area lives in the panel a successful render collapses, and a drop
  // comes from outside the page — so a refusal written there without revealing it
  // is a drop that appears to do nothing at all.
  it('reveals the panel holding the reason a dropped file was refused', async () => {
    globalThis.INITIAL_DATA = { diagram: billingDiagram() };
    await startViewer();
    expect(document.getElementById('data-panel').classList.contains('collapsed')).toBe(true);

    const notes = new File([''], 'notes.txt');
    notes._content = 'plain text';
    fireDrop(document.getElementById('data-panel-body'), notes);

    expect(document.getElementById('render-status').textContent)
      .toBe('✗ Only .emod and .json files are supported');
    expect(document.getElementById('data-panel').classList.contains('collapsed')).toBe(false);
    expect(document.getElementById('diagram-canvas').innerHTML).toContain('TakePayment');
  });

  // A drag carrying no file at all — selected text, a link — is not a model the
  // page refused, so refusing it would put an error on screen for a gesture that
  // was never aimed at opening anything. It is also what a drop looks like on a
  // host whose shell resolves the files itself.
  it('says nothing when the drop carried no file at all', async () => {
    globalThis.INITIAL_DATA = { diagram: billingDiagram() };
    await startViewer();

    fireDrop(document.getElementById('data-panel-body'));
    await flush();

    expect(document.getElementById('render-status').textContent).toBe('');
    expect(document.getElementById('data-panel').classList.contains('collapsed')).toBe(true);
    expect(document.getElementById('diagram-canvas').innerHTML).toContain('TakePayment');
  });

  // A host whose shell resolves drops listens above this element and reads the
  // paths it was given, not the files the page can see — so the page's own
  // listener must let an event it took nothing from carry on up. Stopping it
  // silently discards every file released over the panel on the one platform
  // that still delivers a DOM drop and resolves its paths from it.
  it('lets a drop it took no file from reach a listener above it', async () => {
    globalThis.INITIAL_DATA = { diagram: billingDiagram() };
    await startViewer();
    const reachedAbove = [];
    document.documentElement.addEventListener('drop', () => reachedAbove.push('drop'));

    fireDrop(document.getElementById('data-panel-body'));
    await flush();

    expect(reachedAbove).toEqual(['drop']);
  });

  // ...and must stop one it did open, or a host listening above would open the
  // same model a second time from its own copy of the drop.
  it('stops a drop it opened a model from', async () => {
    globalThis.INITIAL_DATA = null;
    parseResult = { diagnostics: [], diagram: billingDiagram() };
    await startViewer();
    const reachedAbove = [];
    document.documentElement.addEventListener('drop', () => reachedAbove.push('drop'));

    const file = new File([''], 'hotel.emod');
    file._content = 'emod 1\nmodel "Hotel"\n';
    fireDrop(document.getElementById('data-panel-body'), file);
    await flush();

    expect(reachedAbove).toEqual([]);
  });

  // The parser's own empty-source rejection is written for someone who pressed
  // Render on an empty panel, which is not what happened here.
  it('reports a dropped file with nothing in it by name, as the dialog route does', async () => {
    globalThis.INITIAL_DATA = { diagram: billingDiagram() };
    await startViewer();

    const empty = new File([''], 'empty.emod');
    empty._content = '   ';
    fireDrop(document.getElementById('data-panel-body'), empty);
    await flush();

    expect(document.getElementById('render-status').textContent).toBe('✗ empty.emod is empty');
    expect(document.getElementById('diagram-canvas').innerHTML).toContain('TakePayment');
  });
});

// A host that resolves a drop itself pushes the files at the page, the way it
// pushes a file its own dialog opened. Nothing pushes in the browser, so this
// drives the handler the viewer registered — which is the same routine the
// page's own drop listener feeds.
describe('files a host drops on the window', () => {
  const hotelOnDisk = {
    name: 'hotel.emod',
    path: '/models/hotel.emod',
    content: 'emod 1\nmodel "Hotel"\n',
  };

  const handleFor = (name, opened) => ({ name, read: () => Promise.resolve(opened) });

  async function startHotel() {
    globalThis.INITIAL_DATA = null;
    parseResult = { diagnostics: [], diagram: { ...billingDiagram(), model_name: 'Hotel' } };
    await startViewer();
  }

  it('renders the dropped model and names the file it opened', async () => {
    await startHotel();

    await deliverDroppedFiles([handleFor('hotel.emod', hotelOnDisk)]);
    await flush();

    expect(document.getElementById('source-input').value).toBe(hotelOnDisk.content);
    expect(windowTitle).toBe('hotel.emod — Emod Diagram Viewer');
    expect(document.getElementById('stat-file-path').textContent).toBe('/models/hotel.emod');
  });

  it('writes a following save back to the dropped file with no location dialog', async () => {
    await startHotel();
    await deliverDroppedFiles([handleFor('hotel.emod', hotelOnDisk)]);
    await flush();
    saveAnswer = { name: 'hotel.emod', path: '/models/hotel.emod' };

    await save();

    expect(savedFile).toEqual({
      name: 'hotel.emod',
      content: hotelOnDisk.content,
      path: '/models/hotel.emod',
    });
  });

  // The rest of what a pushed drop does — which of several files it opens, what
  // it says when none is a model, how it reports a read that failed, and the
  // unsaved-edits question — is the page's own drop routine, reached through the
  // same call, and is asserted where that routine is driven through the DOM.
});

// Both a drop and a file the host opened are read before anything is replaced,
// and nothing stops a second one arriving while the first read is outstanding.
describe('a second model asked for while the first is still being read', () => {
  const held = (name, path, content) => {
    let release;
    const handle = {
      name,
      read: () => new Promise(function(resolve) {
        release = () => resolve({ name, path, content });
      }),
    };

    return [handle, () => release()];
  };

  const settled = (name, path, content) => ({
    name,
    read: () => Promise.resolve({ name, path, content }),
  });

  it('opens the file dropped last, not the one whose read happened to settle last', async () => {
    await startEmpty();
    const [alpha, releaseAlpha] = held('alpha.emod', '/m/alpha.emod', 'emod 1\nmodel "Alpha"\n');

    deliverDroppedFiles([alpha]);
    await flush();
    deliverDroppedFiles([settled('bravo.emod', '/m/bravo.emod', 'emod 1\nmodel "Bravo"\n')]);
    await flush();
    await flush();
    releaseAlpha();
    await flush();
    await flush();

    expect(windowTitle).toBe('bravo.emod — Emod Diagram Viewer');
    expect(document.getElementById('stat-file-path').textContent).toBe('/m/bravo.emod');
    expect(document.getElementById('source-input').value).toBe('emod 1\nmodel "Bravo"\n');
  });

  it('lets a file the host opened supersede a drop whose read is still held', async () => {
    await startEmpty();
    const [alpha, releaseAlpha] = held('alpha.emod', '/m/alpha.emod', 'emod 1\nmodel "Alpha"\n');

    deliverDroppedFiles([alpha]);
    await flush();
    deliverFile({ name: 'chosen.emod', path: '/m/chosen.emod', content: 'emod 1\nmodel "Chosen"\n' });
    await flush();
    await flush();
    releaseAlpha();
    await flush();
    await flush();

    expect(windowTitle).toBe('chosen.emod — Emod Diagram Viewer');
    expect(document.getElementById('stat-file-path').textContent).toBe('/m/chosen.emod');
  });

  // Only this direction is reachable here: a file the host opened arrives already
  // read, so a drop cannot overlap one from the other side within the viewer. That
  // overlap belongs to the host, which reads before it delivers, and is guarded in
  // platform.desktop.test.js by 'drops an Open whose read outlives a drop made
  // after it'.

  // A drop the user can see was refused is still a gesture that moved on from
  // whatever was being read, so the refusal has to survive that read landing.
  it('keeps a refusal the user just caused, over an older read still outstanding', async () => {
    await startEmpty();
    const [alpha, releaseAlpha] = held('alpha.emod', '/m/alpha.emod', 'emod 1\nmodel "Alpha"\n');

    deliverDroppedFiles([alpha]);
    await flush();
    deliverDroppedFiles([{ name: 'notes.txt', read: () => Promise.resolve({}) }]);
    await flush();
    releaseAlpha();
    await flush();
    await flush();

    expect(document.getElementById('render-status').textContent)
      .toBe('✗ Only .emod and .json files are supported');
    expect(document.getElementById('source-input').value).toBe('');
    expect(windowTitle).not.toContain('alpha.emod');
  });

  // A superseded file's reason would otherwise land on top of the model that
  // replaced it, blaming a file the user has already moved past.
  it('says nothing about a superseded file the host could not read', async () => {
    await startEmpty();
    let refuseAlpha;
    const alpha = {
      name: 'alpha.emod',
      read: () => new Promise(function(resolve) {
        refuseAlpha = () => resolve({ error: 'reading /m/alpha.emod: permission denied' });
      }),
    };

    deliverDroppedFiles([alpha]);
    await flush();
    deliverDroppedFiles([settled('bravo.emod', '/m/bravo.emod', 'emod 1\nmodel "Bravo"\n')]);
    await flush();
    await flush();
    refuseAlpha();
    await flush();
    await flush();

    expect(document.getElementById('render-status').textContent).not.toContain('permission denied');
    expect(windowTitle).toBe('bravo.emod — Emod Diagram Viewer');
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

  // A dropped file is a file the model came from, so the window takes its name.
  // A drop the page reads itself yields contents and no location, which is what
  // the browser build gets — so there is no path to show and none to save back
  // to. A host that resolves the drop supplies one, and the leaves above cover
  // that.
  it('names a file the page read from a drop, and hides the path such a drop lacks', async () => {
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

    expect(windowTitle).toBe('hotel.emod — Emod Diagram Viewer');
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

describe('saving the model', () => {
  it('writes to the open file itself, asking the host for no location', async () => {
    await openBilling();

    await save();

    expect(savedFile.path).toBe('/models/billing.emod');
    expect(savedFile.content).toBe('emod 1\nmodel "Billing"\n');
  });

  it('hands over the panel source rather than the model re-serialised from the diagram', async () => {
    await openBilling('emod 1\n// how a payment is taken\nmodel "Billing"\n');

    await save();

    expect(savedFile.content).toContain('// how a payment is taken');
  });

  // The panel is a textarea, whose value normalises every CRLF to a bare LF, so
  // handing over what it holds would rewrite every line ending in the file.
  it('gives back exactly the bytes a CRLF file arrived with when nothing was edited', async () => {
    await openBilling(crlfSource);
    expect(document.getElementById('source-input').value).not.toBe(crlfSource);

    await save();

    expect(savedFile.content).toBe(crlfSource);
  });

  it('asks the host for a location when no file is open, and suggests a name from the model', async () => {
    await startEmpty();
    document.getElementById('source-input').value = 'emod 1\nmodel "Billing"\n';
    document.getElementById('render-btn').click();
    await flush();
    saveAnswer = { name: 'hotel.emod', path: '/models/hotel.emod' };

    await save();

    expect(savedFile.path).toBe('');
    expect(savedFile.name).toBe('Billing.emod');
  });

  it('adopts the location the host chose as the open file, naming the window and the bar', async () => {
    await startEmpty();
    document.getElementById('source-input').value = 'emod 1\nmodel "Billing"\n';
    saveAnswer = { name: 'hotel.emod', path: '/models/hotel.emod' };

    await save();

    expect(windowTitle).toBe('hotel.emod — Emod Diagram Viewer');
    expect(document.getElementById('stat-file').textContent).toContain('/models/hotel.emod');
    expect(document.getElementById('stat-file').classList.contains('hidden')).toBe(false);
  });

  it('writes to the adopted location without asking again', async () => {
    await startEmpty();
    document.getElementById('source-input').value = 'emod 1\nmodel "Billing"\n';
    saveAnswer = { name: 'hotel.emod', path: '/models/hotel.emod' };
    await save();

    saveAnswer = null;
    await save();

    expect(savedFile.path).toBe('/models/hotel.emod');
  });

  it('asks for a location on a save-to-a-new-location even with a file open', async () => {
    await openBilling();
    saveAnswer = { name: 'copy.emod', path: '/models/copy.emod' };

    await save({ chooseLocation: true });

    expect(savedFile.path).toBe('');
  });

  it('retargets every later save to what a save-to-a-new-location chose', async () => {
    await openBilling();
    saveAnswer = { name: 'copy.emod', path: '/models/copy.emod' };
    await save({ chooseLocation: true });

    saveAnswer = null;
    await save();

    expect(savedFile.path).toBe('/models/copy.emod');
    expect(windowTitle).toBe('copy.emod — Emod Diagram Viewer');
  });

  it('changes nothing when the host answers no location', async () => {
    await openBilling();
    saveAnswer = null;

    await save({ chooseLocation: true });

    expect(windowTitle).toBe('billing.emod — Emod Diagram Viewer');
    expect(document.getElementById('stat-file').textContent).toContain('/models/billing.emod');
    expect(document.getElementById('save-status').classList.contains('hidden')).toBe(true);
  });

  it('still writes to the file itself after a cancelled save-to-a-new-location', async () => {
    await openBilling();
    saveAnswer = null;
    await save({ chooseLocation: true });

    await save();

    expect(savedFile.path).toBe('/models/billing.emod');
  });

  // The status area inside the source panel is off screen whenever a successful
  // render has collapsed it, and collapsing or revealing the panel on the app's
  // most frequent keystroke would rearrange the window under the user.
  it('confirms in the bar that stays on screen, leaving the source panel as it was', async () => {
    await openBilling();
    saveAnswer = { name: 'billing.emod', path: '/models/billing.emod' };
    expect(document.getElementById('data-panel').classList.contains('collapsed')).toBe(true);

    await save();

    const status = document.getElementById('save-status');
    expect(status.textContent).toContain('Saved');
    expect(status.classList.contains('hidden')).toBe(false);
    expect(document.getElementById('data-panel').classList.contains('collapsed')).toBe(true);
  });

  it('stops claiming the model is saved once it has been rendered again', async () => {
    await openBilling();
    saveAnswer = { name: 'billing.emod', path: '/models/billing.emod' };
    await save();
    expect(document.getElementById('save-status').classList.contains('hidden')).toBe(false);

    document.getElementById('source-input').value = 'emod 1\nmodel "Hotel"\n';
    document.getElementById('render-btn').click();
    await flush();

    expect(document.getElementById('save-status').classList.contains('hidden')).toBe(true);
  });

  it('reveals the panel holding the reason a save was refused', async () => {
    await openBilling();
    saveFails = 'writing /models/billing.emod: permission denied';

    await save();

    expect(document.getElementById('data-panel').classList.contains('collapsed')).toBe(false);
    const statusEl = document.getElementById('render-status');
    expect(statusEl.textContent).toContain('permission denied');
    expect(statusEl.className).toContain('error');
  });

  // The bar is the one place a save outcome can be read without expanding
  // anything, so leaving a stale confirmation there would have the window
  // confirming a save it is simultaneously reporting as refused.
  it('refuses to write an empty panel over the open file, rather than emptying it', async () => {
    await openBilling();
    document.getElementById('source-input').value = '   ';

    await save();

    expect(savedFile).toBeNull();
    expect(document.getElementById('save-status').classList.contains('failed')).toBe(true);
    expect(document.getElementById('render-status').textContent).toContain('nothing to save');
  });

  it('refuses to ask for a location for an empty panel, so no chosen file is truncated', async () => {
    globalThis.INITIAL_DATA = null;
    parseResult = { diagnostics: [], diagram: billingDiagram() };
    await startViewer();
    saveAnswer = { name: 'chosen.emod', path: '/models/chosen.emod' };

    await save();

    expect(savedFile).toBeNull();
  });

  // The file adopted after a save has to remember the bytes that were written,
  // or the save after it treats an unedited panel as edited and rewrites every
  // line ending in a file the user never touched.
  it('remembers the bytes it wrote, so the next unedited save reproduces them', async () => {
    const crlf = 'emod 1\r\nmodel "Billing"\r\n';
    globalThis.INITIAL_DATA = null;
    parseResult = { diagnostics: [], diagram: billingDiagram() };
    await startViewer();
    deliverFile({ name: 'crlf.emod', path: '/models/crlf.emod', content: crlf });
    await flush();
    saveAnswer = { name: 'copy.emod', path: '/models/copy.emod' };
    await save({ chooseLocation: true });

    saveAnswer = { name: 'copy.emod', path: '/models/copy.emod' };
    await save();

    expect(savedFile.path).toBe('/models/copy.emod');
    expect(savedFile.content).toBe(crlf);
  });

  it('replaces the confirmation in the bar with the refusal, rather than leaving both', async () => {
    await openBilling();
    saveAnswer = { name: 'billing.emod', path: '/models/billing.emod' };
    await save();
    expect(document.getElementById('save-status').textContent).toContain('Saved');

    saveFails = 'writing /models/billing.emod: permission denied';
    await save();

    const bar = document.getElementById('save-status');
    expect(bar.textContent).not.toContain('Saved');
    expect(bar.textContent).toContain('permission denied');
    expect(bar.classList.contains('failed')).toBe(true);
    expect(bar.classList.contains('hidden')).toBe(false);
  });

  it('leaves the model, its name and the save target alone when a save is refused', async () => {
    await openBilling();
    saveFails = 'writing /models/billing.emod: permission denied';
    await save();

    saveFails = null;
    await save();

    expect(windowTitle).toBe('billing.emod — Emod Diagram Viewer');
    expect(document.getElementById('source-input').value).toBe('emod 1\nmodel "Billing"\n');
    expect(savedFile.path).toBe('/models/billing.emod');
  });
});

describe('a save that overlaps something else', () => {
  it('never writes the arriving file\'s source to the departing file\'s path', async () => {
    await openBilling();
    saveAnswer = { name: 'hotel.emod', path: '/models/hotel.emod' };

    let releaseParse;
    parseQueue = [new Promise((resolve) => { releaseParse = () => resolve({ diagnostics: [], diagram: billingDiagram() }); })];
    deliverFile({ name: 'hotel.emod', path: '/models/hotel.emod', content: 'emod 1\nmodel "Hotel"\n' });
    await flush();

    const saving = save();
    await flush();
    releaseParse();
    await saving;

    expect(savedFile.content).toBe('emod 1\nmodel "Hotel"\n');
    expect(savedFile.path).toBe('/models/hotel.emod');
  });

  it('does not let a save still in flight rename the window back to the file it was writing', async () => {
    await openBilling();
    let releaseWrite;
    saveAnswer = { name: 'billing.emod', path: '/models/billing.emod' };
    saveHangs = new Promise((resolve) => { releaseWrite = resolve; });

    const saving = save();
    await flush();
    saveHangs = null;
    deliverFile({ name: 'hotel.emod', path: '/models/hotel.emod', content: 'emod 1\nmodel "Hotel"\n' });
    await flush();
    releaseWrite();
    await saving;
    await flush();

    expect(windowTitle).toBe('hotel.emod — Emod Diagram Viewer');
    expect(document.getElementById('stat-file').textContent).toContain('/models/hotel.emod');
  });

  it('writes the newer of two overlapping saves last, so the older cannot land on top', async () => {
    await openBilling();
    saveAnswer = { name: 'billing.emod', path: '/models/billing.emod' };
    let releaseFirst;
    saveHangs = new Promise((resolve) => { releaseFirst = resolve; });

    const first = save();
    await flush();
    saveHangs = null;
    document.getElementById('source-input').value = 'emod 1\nmodel "Newer"\n';
    const second = save();
    await flush();

    // The second write must not have been issued at all yet: two writes in
    // flight together land in whichever order the host finishes them, so
    // asserting only the order they were asked for would pass either way.
    expect(savedContents).toEqual([billingSource]);

    releaseFirst();
    await Promise.all([first, second]);
    await flush();

    expect(savedContents).toEqual([billingSource, 'emod 1\nmodel "Newer"\n']);
  });

  // Waiting on a single render is not enough: the file that arrives while that
  // render is in flight starts another, and resuming between the two reads the
  // arriving text beside the departing file's path.
  it('waits for a render that only started while it was already waiting', async () => {
    await openBilling();
    saveAnswer = { name: 'hotel.emod', path: '/models/hotel.emod' };

    let releaseFirst;
    parseQueue = [new Promise((resolve) => { releaseFirst = () => resolve({ diagnostics: [], diagram: billingDiagram() }); })];
    document.getElementById('source-input').value = 'emod 1\nmodel "Edited"\n';
    document.getElementById('render-btn').click();
    await flush();

    const saving = save();
    await flush();

    let releaseSecond;
    parseQueue = [new Promise((resolve) => { releaseSecond = () => resolve({ diagnostics: [], diagram: billingDiagram() }); })];
    deliverFile({ name: 'hotel.emod', path: '/models/hotel.emod', content: 'emod 1\nmodel "Hotel"\n' });
    await flush();
    releaseFirst();
    await flush();
    releaseSecond();
    await saving;

    expect(savedFile.path).toBe('/models/hotel.emod');
    expect(savedFile.content).toBe('emod 1\nmodel "Hotel"\n');
  });

  it('does not reopen the panel for a save belonging to a file that is no longer on screen', async () => {
    await openBilling();
    let releaseWrite;
    saveHangs = new Promise((resolve) => { releaseWrite = resolve; });
    saveFails = 'writing /models/billing.emod: permission denied';

    const saving = save();
    await flush();
    saveHangs = null;
    saveFails = null;
    deliverFile({ name: 'hotel.emod', path: '/models/hotel.emod', content: 'emod 1\nmodel "Hotel"\n' });
    await flush();
    expect(document.getElementById('data-panel').classList.contains('collapsed')).toBe(true);

    saveFails = 'writing /models/billing.emod: permission denied';
    releaseWrite();
    await saving;
    await flush();

    // The bar still says the bytes did not land, naming the file they were for.
    expect(document.getElementById('save-status').textContent).toContain('billing.emod');
    expect(document.getElementById('save-status').classList.contains('failed')).toBe(true);
    expect(document.getElementById('data-panel').classList.contains('collapsed')).toBe(true);
  });

  it('takes the refusal out of the panel once a later save succeeds', async () => {
    await openBilling();
    saveFails = 'writing /models/billing.emod: permission denied';
    await save();
    expect(document.getElementById('render-status').textContent).toContain('permission denied');

    saveFails = null;
    saveAnswer = { name: 'billing.emod', path: '/models/billing.emod' };
    await save();

    expect(document.getElementById('render-status').textContent).not.toContain('permission denied');
    expect(document.getElementById('save-status').textContent).toContain('Saved');
  });

  // The save waits for renders before it starts, so the render it can still
  // collide with is one the user asks for while the write is in flight.
  it('leaves a render the user asked for running when a save is refused', async () => {
    await openBilling();
    let releaseWrite;
    saveHangs = new Promise((resolve) => { releaseWrite = resolve; });

    const saving = save();
    await flush();
    saveHangs = null;
    saveFails = 'writing /models/billing.emod: permission denied';

    let releaseRender;
    parseQueue = [new Promise((resolve) => {
      releaseRender = () => resolve({ diagnostics: [], diagram: { ...billingDiagram(), model_name: 'Rendered' } });
    })];
    document.getElementById('source-input').value = 'emod 1\nmodel "Rendered"\n';
    document.getElementById('render-btn').click();
    await flush();

    releaseWrite();
    await saving;
    await flush();
    releaseRender();
    await flush();

    // A save that claimed a render number would have discarded this render, and
    // the canvas would still be naming Billing while the panel showed Rendered.
    expect(document.getElementById('model-name-display').textContent).toBe('Rendered');
  });
});

describe('what a save writes', () => {
  const panel = (value, arrived) => ({
    dom: { sourceInput: { value: value } },
    currentFile: arrived === undefined ? null : { name: 'm.emod', path: '/m.emod', content: arrived },
  });

  it('hands back the bytes an unedited file arrived with, whatever the panel did to them', () => {
    const arrived = 'emod 1\r\nmodel "Billing"\r\n';

    expect(sourceToSave(panel('emod 1\nmodel "Billing"\n', arrived))).toBe(arrived);
  });

  it('keeps a CRLF file in its own convention once it has been edited', () => {
    expect(sourceToSave(panel('emod 1\nmodel "Hotel"\n', 'emod 1\r\nmodel "Billing"\r\n')))
      .toBe('emod 1\r\nmodel "Hotel"\r\n');
  });

  it('leaves an LF file alone once it has been edited', () => {
    expect(sourceToSave(panel('emod 1\nmodel "Hotel"\n', 'emod 1\nmodel "Billing"\n')))
      .toBe('emod 1\nmodel "Hotel"\n');
  });

  // A textarea normalises a lone CR to LF just as it does a CRLF, so an unedited
  // classic-Mac file still round-trips exactly; only editing one converts it.
  it('gives back a lone-CR file untouched when nothing was edited', () => {
    const arrived = 'emod 1\rmodel "Billing"\r';

    expect(sourceToSave(panel('emod 1\nmodel "Billing"\n', arrived))).toBe(arrived);
  });

  it('writes a lone-CR file with LF once it has been edited, having no CRLF to copy', () => {
    expect(sourceToSave(panel('emod 1\nmodel "Hotel"\n', 'emod 1\rmodel "Billing"\r')))
      .toBe('emod 1\nmodel "Hotel"\n');
  });

  it('settles a file of mixed endings on the CRLF it holds, once it has been edited', () => {
    expect(sourceToSave(panel('a\nb\nedited\n', 'a\r\nb\nc\n'))).toBe('a\r\nb\r\nedited\r\n');
  });

  it('gives back a mixed file exactly as it arrived when nothing was edited', () => {
    const arrived = 'a\r\nb\nc\n';

    expect(sourceToSave(panel('a\nb\nc\n', arrived))).toBe(arrived);
  });

  it('hands over the panel as it stands when no file was ever opened', () => {
    expect(sourceToSave(panel('emod 1\nmodel "Pasted"\n'))).toBe('emod 1\nmodel "Pasted"\n');
  });

  it('hands over what was typed into a file that arrived empty', () => {
    expect(sourceToSave(panel('emod 1\nmodel "Billing"\n', ''))).toBe('emod 1\nmodel "Billing"\n');
  });
});

// The marker answers one question: would what Save writes differ from what the
// open file holds. Save writes the source panel, so these drive the panel and
// the save, and the leaf below them drives everything that is not either.
describe('the window says whether there are unsaved edits', () => {
  it('leaves an opened file unmarked, because nothing typed is nothing to save', async () => {
    await openBilling();

    expect(modifiedReports).toEqual([false]);
  });

  // The panel is a textarea, so it holds LF for a file that arrived with CRLF.
  // Comparing what it holds against the file's own bytes would mark every such
  // file the instant it opened.
  it('leaves a CRLF file unmarked, though the panel rewrote every line ending it arrived with', async () => {
    await openBilling(crlfSource);
    expect(document.getElementById('source-input').value).not.toBe(crlfSource);

    expect(modifiedReports).toEqual([false]);
  });

  // A dropped file is on disk exactly as it arrived, so nothing about it is
  // unsaved — and when the page read it itself there is no location to write it
  // back to either.
  it('leaves a freshly dropped model unmarked, having changed nothing about it', async () => {
    globalThis.INITIAL_DATA = null;
    parseResult = { diagnostics: [], diagram: billingDiagram() };
    await startViewer();

    const dropped = new File([''], 'hotel.emod');
    dropped._content = 'emod 1\nmodel "Hotel"\n';
    fireDrop(document.getElementById('data-panel-body'), dropped);
    await flush();
    await flush();

    expect(modifiedReports).toEqual([false]);
  });

  it('marks the window as the panel is typed into, with no render asked for', async () => {
    await openBilling();
    const rendersBefore = platform.parseEmod.mock.calls.length;

    typeIntoPanel('emod 1\nmodel "Billing Edited"\n');

    expect(marked()).toBe(true);
    expect(platform.parseEmod.mock.calls.length).toBe(rendersBefore);
  });

  it('unmarks the window when the panel is edited back to what the file held', async () => {
    await openBilling();
    typeIntoPanel('emod 1\nmodel "Billing Edited"\n');
    expect(marked()).toBe(true);

    typeIntoPanel(billingSource);

    expect(marked()).toBe(false);
  });

  // A textarea hands back LF for the CRLF the file arrived with, so typing that
  // text is typing the file back exactly, and the window has to say so.
  it('unmarks a CRLF file edited back to the text the panel showed when it opened', async () => {
    await openBilling(crlfSource);
    typeIntoPanel('emod 1\nmodel "Billing Edited"\n');
    expect(marked()).toBe(true);

    typeIntoPanel(document.getElementById('source-input').value.replace('Billing Edited', 'Billing'));

    expect(marked()).toBe(false);
  });

  it('unmarks the window when a save lands', async () => {
    await openBilling();
    typeIntoPanel('emod 1\nmodel "Billing Edited"\n');
    saveAnswer = { name: 'billing.emod', path: '/models/billing.emod' };

    await save();

    expect(marked()).toBe(false);
  });

  it('leaves the window marked when the host refuses the save', async () => {
    await openBilling();
    typeIntoPanel('emod 1\nmodel "Billing Edited"\n');
    saveFails = 'permission denied';
    modifiedReports.length = 0;

    await save();

    expect(savedFile.content).toBe('emod 1\nmodel "Billing Edited"\n');
    expect(document.getElementById('save-status').textContent).toContain('permission denied');
    expect(modifiedReports).toEqual([true]);
  });

  it('leaves the window marked when the location dialog is cancelled', async () => {
    await startEmpty();
    typeIntoPanel('emod 1\nmodel "Pasted"\n');
    saveAnswer = null;
    modifiedReports.length = 0;

    await save();

    expect(savedFile.content).toBe('emod 1\nmodel "Pasted"\n');
    expect(modifiedReports).toEqual([true]);
  });

  it('marks pasted source that has never been saved anywhere', async () => {
    await startEmpty();

    typeIntoPanel('emod 1\nmodel "Pasted"\n');

    expect(marked()).toBe(true);
  });

  it('unmarks pasted source once it is saved to the location the host chose', async () => {
    await startEmpty();
    typeIntoPanel('emod 1\nmodel "Pasted"\n');
    saveAnswer = { name: 'pasted.emod', path: '/models/pasted.emod' };

    await save();

    expect(marked()).toBe(false);
  });

  // The panel's text and the open file's identity are committed at different
  // moments, so a marker taken while a file is arriving would describe the
  // arriving model against the departing model's bytes.
  it('keeps describing the model on screen when a delivered file will not render', async () => {
    await openBilling();
    typeIntoPanel('emod 1\nmodel "Billing Edited"\n');
    parseQueue = [Promise.reject(new Error('parse failed'))];

    deliverFile({ name: 'other.emod', path: '/models/other.emod', content: 'emod 1\nmodel "Other"\n' });
    await flush();

    expect(marked()).toBe(true);
    expect(document.getElementById('source-input').value).toBe('emod 1\nmodel "Billing Edited"\n');
  });

  // Each of these is a change the user can make that Save would not write, so
  // none of them may mark the window — and each ends by editing the panel,
  // which must, so a leaf cannot pass with reporting broken altogether.
  describe('a change Save would not write', () => {
    async function silentThenAudible(gesture) {
      await openBilling();
      modifiedReports.length = 0;

      await gesture(document.getElementById('diagram-canvas'));
      expect(modifiedReports).toEqual([]);

      typeIntoPanel('emod 1\nmodel "Billing Edited"\n');
      expect(modifiedReports).toEqual([true]);
    }

    // The story's own criterion: moving a node for layout alone.
    it('says nothing when a node is dragged to a new position', async () => {
      await silentThenAudible(async (canvas) => {
        dragBy(blockFor('command-1'), 40, 25);
        expect(document.getElementById('reset-layout').disabled).toBe(false);
      });
    });

    it('says nothing when the canvas is panned', async () => {
      await silentThenAudible(async (canvas) => {
        dragBy(canvas, 60, 60);
        expect(canvas.querySelector('#viewport-group').getAttribute('transform'))
          .not.toBe('translate(0, 0) scale(1)');
      });
    });

    it('says nothing when the context menu adds a node to the diagram', async () => {
      await silentThenAudible(async (canvas) => {
        const before = canvas.querySelectorAll('.diagram-node').length;
        fireMouse(canvas.querySelector('.slice-header'), 'contextmenu');
        menuItemFor('add-command').click();
        expect(canvas.querySelectorAll('.diagram-node').length).toBe(before + 1);
      });
    });

    it('says nothing when a slice is hidden from the visibility tree', async () => {
      await silentThenAudible(async () => {
        document.getElementById('visibility-toggle').click();
        hideFromVisibilityTree('slice-1');
        expect(blockFor('command-1')).toBeNull();
      });
    });

    it('says nothing when the data panel is collapsed or reopened', async () => {
      await silentThenAudible(async () => {
        document.getElementById('data-panel-header').click();
        expect(document.getElementById('data-panel').classList.contains('collapsed')).toBe(false);
      });
    });
  });
});

// Every way a different file's model reaches the screen runs one guard, so a
// new way of opening one cannot arrive unguarded.
describe('a model arriving over unsaved edits', () => {
  const otherFile = { name: 'other.emod', path: '/models/other.emod', content: 'emod 1\nmodel "Other"\n' };
  const editedSource = 'emod 1\nmodel "Billing Edited"\n';

  async function openBillingThenEdit() {
    await openBilling();
    typeIntoPanel(editedSource);
  }

  function dropOther() {
    const file = new File([''], 'other.emod');
    file._content = otherFile.content;
    fireDrop(document.getElementById('data-panel-body'), file);
    return flush();
  }

  function onScreen() {
    return {
      source: document.getElementById('source-input').value,
      title: windowTitle,
      path: document.getElementById('stat-file').textContent,
      name: document.getElementById('model-name-display').textContent,
    };
  }

  it('asks before a delivered model replaces anything on screen', async () => {
    await openBillingThenEdit();
    const before = onScreen();
    unsavedEditsAnswer = 'cancel';

    deliverFile(otherFile);
    await flush();

    expect(unsavedEditsAsked).toBe(1);
    expect(onScreen()).toEqual(before);
  });

  it('asks before a dropped model replaces anything on screen', async () => {
    await openBillingThenEdit();
    const before = onScreen();
    unsavedEditsAnswer = 'cancel';

    await dropOther();

    expect(unsavedEditsAsked).toBe(1);
    expect(onScreen()).toEqual(before);
  });

  it('leaves the window still marked and writes nothing when the answer is cancel', async () => {
    await openBillingThenEdit();
    unsavedEditsAnswer = 'cancel';

    deliverFile(otherFile);
    await flush();

    expect(savedFile).toBeNull();
    expect(modifiedReports[modifiedReports.length - 1]).toBe(true);
  });

  it('replaces the model and writes nothing when the answer is discard', async () => {
    await openBillingThenEdit();
    unsavedEditsAnswer = 'discard';

    deliverFile(otherFile);
    await flush();

    expect(savedFile).toBeNull();
    expect(onScreen().source).toBe(otherFile.content);
    expect(windowTitle).toBe('other.emod — Emod Diagram Viewer');
  });

  it('writes the edits to the file they belong to, then opens the arriving model', async () => {
    await openBillingThenEdit();
    unsavedEditsAnswer = 'save';
    saveAnswer = { name: 'billing.emod', path: '/models/billing.emod' };

    deliverFile(otherFile);
    await flush();

    expect(savedFile).toEqual({ name: 'billing.emod', content: editedSource, path: '/models/billing.emod' });
    expect(onScreen().source).toBe(otherFile.content);
  });

  it('keeps the model on screen when the save it was asked for is refused', async () => {
    await openBillingThenEdit();
    unsavedEditsAnswer = 'save';
    saveFails = 'permission denied';

    deliverFile(otherFile);
    await flush();

    expect(onScreen().source).toBe(editedSource);
    expect(windowTitle).toBe('billing.emod — Emod Diagram Viewer');
  });

  it('keeps the model on screen when the location dialog for that save is cancelled', async () => {
    await startEmpty();
    typeIntoPanel('emod 1\nmodel "Pasted"\n');
    unsavedEditsAnswer = 'save';
    saveAnswer = null;

    deliverFile(otherFile);
    await flush();

    expect(onScreen().source).toBe('emod 1\nmodel "Pasted"\n');
  });

  it('asks nothing when there is nothing unsaved to lose', async () => {
    await openBilling();

    deliverFile(otherFile);
    await flush();

    expect(unsavedEditsAsked).toBe(0);
    expect(onScreen().source).toBe(otherFile.content);
  });

  it('asks nothing when a dropped model replaces one with nothing unsaved', async () => {
    await openBilling();

    await dropOther();

    expect(unsavedEditsAsked).toBe(0);
    expect(onScreen().source).toBe(otherFile.content);
  });

  // Both are refused before anything would be replaced, so there is nothing to
  // ask about — and asking would make an unreadable file interrupt twice.
  it('reports a file it could not read without asking anything', async () => {
    await openBillingThenEdit();

    deliverFile({ error: 'reading /models/other.emod: permission denied' });
    await flush();

    expect(unsavedEditsAsked).toBe(0);
    expect(document.getElementById('render-status').textContent).toContain('permission denied');
    expect(onScreen().source).toBe(editedSource);
  });

  it('reports an empty file without asking anything', async () => {
    await openBillingThenEdit();

    deliverFile({ name: 'empty.emod', path: '/models/empty.emod', content: '   \n' });
    await flush();

    expect(unsavedEditsAsked).toBe(0);
    expect(document.getElementById('render-status').textContent).toContain('empty.emod is empty');
    expect(onScreen().source).toBe(editedSource);
  });
});

// The question and the act it authorises are separated by however long the user
// takes to answer, so two of them in flight at once is the shape that loses
// work: the save the first answer asks for reads the panel and the open file
// when it runs, not when the question was put.
describe('two models arriving at once', () => {
  const editedSource = 'emod 1\nmodel "Billing Edited"\n';

  // Each call gets its own deferred, so the dialogs can be answered in any
  // order — which is what a user with two of them on screen can do.
  function deferredDialogs() {
    const pending = [];
    unsavedEditsAnswerer = () => new Promise((resolve) => pending.push(resolve));
    return pending;
  }

  it('never saves an arriving model under the answer given for the departing one', async () => {
    await openBilling();
    typeIntoPanel(editedSource);
    const dialogs = deferredDialogs();
    saveAnswer = { name: 'billing.emod', path: '/models/billing.emod' };

    deliverFile({ name: 'b.emod', path: '/models/b.emod', content: 'emod 1\nmodel "B"\n' });
    await flush();
    deliverFile({ name: 'c.emod', path: '/models/c.emod', content: 'emod 1\nmodel "C"\n' });
    await flush();

    // Only the first ask may be on screen; the second waits its turn.
    expect(dialogs).toHaveLength(1);

    dialogs[0]('save');
    await flush();
    await flush();

    expect(savedFile.content).toBe(editedSource);
    expect(savedFile.path).toBe('/models/billing.emod');
  });

  // A rejected turn left on the queue would wedge every later open, drop, close
  // and quit behind it, which is the whole app's file handling.
  it('reports a question that could not be answered and keeps the model, then keeps answering', async () => {
    await openBilling();
    typeIntoPanel(editedSource);
    unsavedEditsAnswerer = () => Promise.reject(new Error('the host dialog exploded'));

    deliverFile({ name: 'b.emod', path: '/models/b.emod', content: 'emod 1\nmodel "B"\n' });
    await flush();
    await flush();

    expect(document.getElementById('render-status').textContent).toContain('the host dialog exploded');
    expect(document.getElementById('source-input').value).toBe(editedSource);

    unsavedEditsAnswerer = null;
    unsavedEditsAnswer = 'discard';
    deliverFile({ name: 'c.emod', path: '/models/c.emod', content: 'emod 1\nmodel "C"\n' });
    await flush();
    await flush();

    expect(document.getElementById('source-input').value).toBe('emod 1\nmodel "C"\n');
  });

  it('asks the second time against the model the first answer left on screen', async () => {
    await openBilling();
    typeIntoPanel(editedSource);
    const dialogs = deferredDialogs();

    deliverFile({ name: 'b.emod', path: '/models/b.emod', content: 'emod 1\nmodel "B"\n' });
    await flush();
    deliverFile({ name: 'c.emod', path: '/models/c.emod', content: 'emod 1\nmodel "C"\n' });
    await flush();

    dialogs[0]('discard');
    await flush();
    await flush();

    // b replaced the edited model and is itself unedited, so the second open
    // has nothing to lose and asks nothing further.
    expect(dialogs).toHaveLength(1);
    expect(document.getElementById('source-input').value).toBe('emod 1\nmodel "C"\n');
    expect(savedFile).toBeNull();
  });
});

// The host keeps the list of what has been opened, and the viewer is the only
// side that knows the moment a file becomes the model on screen — so it says so
// through the seam, from the one branch every entry point already passes.
describe('what the viewer says it has opened', () => {
  it('remembers a file the host opened, by its path, once', async () => {
    await openBilling();

    expect(remembered).toEqual(['/models/billing.emod']);
  });

  it('remembers a file the host dropped, by the real path the host resolved', async () => {
    await startEmpty();

    await deliverDroppedFiles([{
      name: 'hotel.emod',
      read: () => Promise.resolve({ name: 'hotel.emod', path: '/models/hotel.emod', content: 'emod 1\nmodel "Hotel"\n' }),
    }]);
    await flush();

    expect(remembered).toEqual(['/models/hotel.emod']);
  });

  it('remembers a file again when it is opened again, because the host decides what it already holds', async () => {
    await openBilling();

    deliverFile({ name: 'billing.emod', path: '/models/billing.emod', content: billingSource });
    await flush();

    expect(remembered).toEqual(['/models/billing.emod', '/models/billing.emod']);
  });

  it('remembers nothing for pasted source, which has no location behind it', async () => {
    await startEmpty();

    document.getElementById('source-input').value = billingSource;
    document.getElementById('render-btn').click();
    await flush();

    expect(remembered).toEqual([]);
    expect(document.getElementById('render-status').textContent).toBe('✓ Rendered');
  });

  it('remembers nothing for a drop the page read itself, which carries no location', async () => {
    await startEmpty();

    const file = new File([''], 'hotel.emod');
    file._content = 'emod 1\nmodel "Hotel"\n';
    fireDrop(document.getElementById('data-panel-body'), file);
    await flush();

    expect(document.getElementById('source-input').value).toContain('Hotel');
    expect(remembered).toEqual([]);
  });

  it('remembers nothing for a file whose render the parser rejects, because it never became the model on screen', async () => {
    globalThis.INITIAL_DATA = null;
    await startViewer();
    parseQueue.push(Promise.reject(new Error('nothing to render')));

    deliverFile({ name: 'broken.emod', path: '/models/broken.emod', content: 'emod 1\n' });
    await flush();

    expect(document.getElementById('render-status').textContent).toContain('nothing to render');
    expect(remembered).toEqual([]);
  });

  it('remembers the location a save to a new location chose, and the model is that file now', async () => {
    await openBilling();
    saveAnswer = { name: 'copy.emod', path: '/models/copy.emod' };

    await save({ chooseLocation: true });

    expect(remembered).toEqual(['/models/billing.emod', '/models/copy.emod']);
  });

  it('remembers where pasted source was first saved', async () => {
    await startEmpty();
    document.getElementById('source-input').value = billingSource;
    saveAnswer = { name: 'hotel.emod', path: '/models/hotel.emod' };

    await save();

    expect(remembered).toEqual(['/models/hotel.emod']);
  });

  it('remembers nothing for a save back to the file already open', async () => {
    await openBilling();
    saveAnswer = { name: 'billing.emod', path: '/models/billing.emod' };

    await save();

    expect(remembered).toEqual(['/models/billing.emod']);
  });

  it('remembers nothing for a save whose location dialog was cancelled', async () => {
    await openBilling();
    saveAnswer = null;

    await save({ chooseLocation: true });

    expect(remembered).toEqual(['/models/billing.emod']);
  });

  it('keeps a save confirmation first when the recording the save prompted is refused', async () => {
    await startEmpty();
    document.getElementById('source-input').value = billingSource;
    saveAnswer = { name: 'hotel.emod', path: '/models/hotel.emod' };
    rememberFails = 'writing /Users/me/Library/Application Support/emod/recent-files.json: permission denied';

    await save();

    const bar = document.getElementById('save-status');
    expect(bar.textContent.startsWith('✓ Saved hotel.emod')).toBe(true);
    expect(bar.textContent).toContain('recent files not saved');
    expect(bar.title).toContain('permission denied');
    expect(bar.classList.contains('failed')).toBe(false);
    expect(remembered).toEqual(['/models/hotel.emod']);
  });

  it('leaves a newer confirmation alone when an older recording is refused late', async () => {
    await startEmpty();
    document.getElementById('source-input').value = billingSource;
    const refusals = [];
    rememberAnswerer = () => new Promise((resolve, reject) => { refusals.push(reject); });

    saveAnswer = { name: 'hotel.emod', path: '/models/hotel.emod' };
    await save();
    saveAnswer = { name: 'copy.emod', path: '/models/copy.emod' };
    await save({ chooseLocation: true });
    // The older recording's refusal lands last, which is the order a queue
    // that answers in turn cannot produce but a slow write can.
    refusals[1](new Error('writing recent-files.json: permission denied'));
    await flush();
    refusals[0](new Error('writing recent-files.json: permission denied'));
    await flush();

    const bar = document.getElementById('save-status');
    expect(bar.textContent.startsWith('✓ Saved copy.emod')).toBe(true);
    expect(bar.textContent).not.toContain('hotel.emod');
    expect(bar.classList.contains('failed')).toBe(false);
  });

  it('leaves a save confirmation alone when the recording an earlier open prompted is refused late', async () => {
    const refusals = [];
    rememberAnswerer = () => new Promise((resolve, reject) => { refusals.push(reject); });
    await openBilling();
    saveAnswer = { name: 'billing.emod', path: '/models/billing.emod' };
    await save();

    refusals[0](new Error('writing recent-files.json: permission denied'));
    await flush();

    const bar = document.getElementById('save-status');
    expect(bar.textContent).toBe('✓ Saved billing.emod');
    expect(bar.classList.contains('failed')).toBe(false);
  });

  it('reports a recording the host refused in the bar, with the model open and the panel as it was', async () => {
    rememberFails = 'writing /Users/me/Library/Application Support/emod/recent-files.json: permission denied';

    await openBilling();

    const bar = document.getElementById('save-status');
    expect(bar.textContent).toContain('Recent files');
    expect(bar.textContent).toContain('permission denied');
    expect(bar.title).toContain('recent-files.json');
    expect(bar.classList.contains('hidden')).toBe(false);
    expect(bar.classList.contains('failed')).toBe(true);
    expect(document.getElementById('diagram-canvas').innerHTML).toContain('TakePayment');
    expect(document.getElementById('render-status').textContent).toBe('✓ Rendered');
    expect(document.getElementById('data-panel').classList.contains('collapsed')).toBe(true);
    expect(windowTitle).toBe('billing.emod — Emod Diagram Viewer');
  });
});

// The host cannot raise this question and wait for it, so it asks the viewer,
// which answers with the same policy an arriving model goes through.
describe('the host asking whether it may close', () => {
  const editedSource = 'emod 1\nmodel "Billing Edited"\n';

  it('says yes straight away when there is nothing unsaved to lose', async () => {
    await openBilling();

    await expect(requestLeave()).resolves.toBe(true);
    expect(unsavedEditsAsked).toBe(0);
  });

  // The answer a failed question gives is only observable here, and answering
  // yes would let a broken dialog authorise the close it could not ask about.
  it('says no when the question could not be put at all', async () => {
    await openBilling();
    typeIntoPanel(editedSource);
    unsavedEditsAnswerer = () => Promise.reject(new Error('the host dialog exploded'));

    await expect(requestLeave()).resolves.toBe(false);
    expect(savedFile).toBeNull();
  });

  it('says no when the answer is cancel, and writes nothing', async () => {
    await openBilling();
    typeIntoPanel(editedSource);
    unsavedEditsAnswer = 'cancel';

    await expect(requestLeave()).resolves.toBe(false);
    expect(savedFile).toBeNull();
  });

  it('says yes without writing when the answer is discard', async () => {
    await openBilling();
    typeIntoPanel(editedSource);
    unsavedEditsAnswer = 'discard';

    await expect(requestLeave()).resolves.toBe(true);
    expect(savedFile).toBeNull();
  });

  it('writes the open file and then says yes when the answer is save', async () => {
    await openBilling();
    typeIntoPanel(editedSource);
    unsavedEditsAnswer = 'save';
    saveAnswer = { name: 'billing.emod', path: '/models/billing.emod' };

    await expect(requestLeave()).resolves.toBe(true);
    expect(savedFile).toEqual({ name: 'billing.emod', content: editedSource, path: '/models/billing.emod' });
  });

  it('says no when the save it was asked for is refused', async () => {
    await openBilling();
    typeIntoPanel(editedSource);
    unsavedEditsAnswer = 'save';
    saveFails = 'permission denied';

    await expect(requestLeave()).resolves.toBe(false);
  });

  // A close arriving mid-open must answer about the model that open settles on,
  // not the one it is replacing.
  it('waits for a model already arriving before it answers', async () => {
    await openBilling();
    typeIntoPanel(editedSource);
    unsavedEditsAnswer = 'discard';

    deliverFile({ name: 'b.emod', path: '/models/b.emod', content: 'emod 1\nmodel "B"\n' });
    const leaving = requestLeave();
    await flush();
    await flush();

    await expect(leaving).resolves.toBe(true);
    expect(document.getElementById('source-input').value).toBe('emod 1\nmodel "B"\n');
    // b is unedited, so answering about it needed no second question.
    expect(unsavedEditsAsked).toBe(1);
  });
});

// The guard above is only unskippable while one function is the only way a
// different file's model reaches renderPanelSource. This reads viewer.js's own
// source, in the style of the module scan already in this suite.
describe('one function replaces the open model', () => {
  const source = readFileSync(resolve(__dirname, '../static/viewer.js'), 'utf-8');
  // The lookbehind drops the declaration, which takes the same two parameters
  // every call that opens a model passes.
  const calls = (text) => text.match(/(?<!function )renderPanelSource\(/g) || [];
  const opens = (text) => text.match(/(?<!function )renderPanelSource\(\s*[^)\s]/g) || [];

  // Each entry is one of init's own functions, so a call can be attributed to
  // the one that makes it without slicing the file by hand. The guard's own
  // function is found by what it does rather than by its name, so renaming it
  // is a rename and not a failure.
  // The split strips the `function ` keyword, so a declaration would match the
  // lookbehind above as a call; each chunk is read from its second line on. It
  // is also cut at its own closing brace — init's functions are at this indent,
  // so a chunk otherwise runs to the *next* declaration and carries whatever
  // top-level code sits between them, which would be attributed to it.
  const declarations = source.split(/\n  function /).slice(1);
  const bodyOf = (declaration) => {
    const body = declaration.slice(declaration.indexOf('\n'));
    const close = body.indexOf('\n  }');

    return close === -1 ? body : body.slice(0, close);
  };
  const opensAModel = declarations.filter((d) => opens(bodyOf(d)).length > 0);

  it('splits into init\'s functions, so the attribution below is not reading one blob', () => {
    expect(declarations.length).toBeGreaterThan(5);
  });

  it('re-renders the panel without a model too, so the count below means something', () => {
    expect(calls(source).length).toBeGreaterThan(opens(source).length);
  });

  it('hands renderPanelSource a model to open from exactly one call', () => {
    expect(opens(source)).toHaveLength(1);
  });

  it('makes that call from exactly one of init\'s own functions', () => {
    expect(opensAModel).toHaveLength(1);
  });

  it('makes it from a function that consults the unsaved-edits guard', () => {
    expect(bodyOf(opensAModel[0])).toContain('clearedToReplace');
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
      ['droppedFiles', 'exportEmod', 'initialState', 'isReady', 'onFileOpened', 'onFilesDropped',
       'onLeaveRequested', 'onSaveRequested', 'parseEmod', 'ready', 'rememberOpenedFile',
       'resolveUnsavedEdits', 'saveFile', 'setWindowModified', 'setWindowTitle'],
    );
  });

  it('is satisfied by the browser implementation', () => {
    expect(browser).toEqual(contract);
  });

  it('is satisfied by the desktop implementation', () => {
    expect(desktop).toEqual(contract);
  });
});

// A shell that resolves drops natively finds its drop target by walking up from
// whatever sits under the cursor, and discards the drop in silence when it finds
// none — so the page has to say where one may be released, and the answer is
// anywhere in the window. The panel body the browser listens on cannot be it: a
// successful render collapses the panel, which takes its body off screen. The
// attribute is inert in the builds no such shell serves.
describe('the page says where a file may be dropped', () => {
  const markup = readFileSync(resolve(__dirname, '../static/viewer.html'), 'utf-8');

  it('marks the element every other one sits inside', () => {
    expect(markup).toMatch(/<body[^>]*\sdata-file-drop-target[\s>]/);
  });

  it('marks nothing narrower, which would leave the rest of the window dead', () => {
    expect(markup.match(/data-file-drop-target/g)).toHaveLength(1);
  });
});

// A shell that resolves drops natively marks its own drop target with a class of
// its own while a file is over it, and the page's dragover listeners never run —
// so the affordance a user sees over the window is painted off that class rather
// than off the one the browser viewer sets. Both are painted by one rule, so the
// two cannot drift into two different-looking overlays.
describe('the drop affordance', () => {
  const markup = readFileSync(resolve(__dirname, '../static/viewer.html'), 'utf-8');

  // The stylesheet as a list of {selectors, declarations}, so a rule is found by
  // the whole of what it selects rather than by the first place a selector's
  // text happens to appear — which for these two is each other's selector list.
  const rules = [...markup.matchAll(/([^{}]+)\{([^{}]*)\}/g)].map((m) => ({
    selectors: m[1].trim(),
    declarations: m[2],
  }));
  const ruleSelecting = (...required) => rules.filter(
    (rule) => required.every((selector) => rule.selectors.includes(selector)),
  );

  const shared = ruleSelecting('#data-panel-body.drag-over::after', 'body.file-drop-target-active::after');
  const windowWide = ruleSelecting('body.file-drop-target-active::after')
    .filter((rule) => !rule.selectors.includes('#data-panel-body'));

  it('splits the stylesheet into rules, so the lookups below are not reading one blob', () => {
    expect(rules.length).toBeGreaterThan(50);
  });

  it('paints the window-wide overlay and the panel\'s from one rule', () => {
    expect(shared).toHaveLength(1);
  });

  it('says what may be dropped, in the wording the browser viewer has always used', () => {
    expect(shared[0].declarations).toContain('content: "Drop .emod or .json file here"');
  });

  it('lets the drop through rather than intercepting the one it announces', () => {
    expect(shared[0].declarations).toContain('pointer-events: none');
  });

  // A static flex container is not a containing block, so absolute would hand
  // the overlay the initial one and leave it behind on a resize.
  it('covers the window rather than a block inside it', () => {
    expect(windowWide).toHaveLength(1);
    expect(windowWide[0].declarations).toContain('position: fixed');
  });

  // Named rather than compared against every z-index in the file: an unrelated
  // rule in another stacking context gaining a higher value says nothing about
  // whether a drop is still announced over the chrome it can be released on top
  // of, and a leaf that failed for that would be blaming the wrong rule.
  it('sits above the chrome a file can be released on top of', () => {
    const depthOf = (declarations) => Number((declarations.match(/z-index:\s*(\d+)/) || [])[1]);
    const overlayDepth = depthOf(windowWide[0].declarations);
    const covered = ['#ctx-menu', '#detail-panel', '#tooltip', '#data-panel'];

    covered.forEach((selector) => {
      const owning = rules.filter(
        (rule) => rule.selectors.split(',').some((one) => one.trim() === selector),
      );
      expect(owning, `${selector} must have a rule of its own`).toHaveLength(1);
      expect(depthOf(owning[0].declarations), `${selector} must state a depth`).not.toBeNaN();
      expect(overlayDepth).toBeGreaterThan(depthOf(owning[0].declarations));
    });
  });

  // The browser viewer keeps its own drop region: the panel body is where its
  // listeners are and where its highlight has always been.
  it('leaves the panel body painting the browser viewer\'s own drop region', () => {
    const panelOnly = ruleSelecting('#data-panel-body.drag-over::after')
      .filter((rule) => !rule.selectors.includes('file-drop-target-active'));

    expect(panelOnly).toHaveLength(1);
    expect(panelOnly[0].declarations).toContain('position: absolute');
    expect(shared[0].declarations).toContain('inset: 0');
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

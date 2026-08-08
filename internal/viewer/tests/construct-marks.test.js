import { describe, it, expect, beforeEach } from 'vitest';
import { installSVGGeometry, installIdentityMapping } from './svg-env.js';

installSVGGeometry();

const { createStore } = await import('../static/store.js');
const { Layout } = await import('../static/layout.js');
const { Renderer } = await import('../static/renderer.js');
const { Interaction } = await import('../static/interaction.js');
const { UI } = await import('../static/ui.js');
const { DRAG_THRESHOLD } = await import('../static/config.js');

const SVG_NS = 'http://www.w3.org/2000/svg';

const sliceDescription = 'Everything it takes to agree a plan';
const commandDescription = 'Offers the customer an instalment plan';
const sliceComments = [{ text: '# Rewritten after the 2024 affordability rules' }];
const commandComments = [{ text: '# Only the plan the customer accepted' }];

function documentedSlice() {
  return [
    { id: 'ctx1', type: 'context', label: 'Collections' },
    { id: 'agg1', type: 'aggregate', label: 'Arrangement', parentId: 'ctx1' },
    {
      id: 'sl1', type: 'slice', label: 'Propose plan', parentId: 'agg1',
      description: sliceDescription,
      comments: sliceComments,
    },
    {
      id: 'cmd1', type: 'command', label: 'ProposePlan', parentId: 'sl1',
      description: commandDescription,
      comments: commandComments,
      fields: [{ name: 'planId', type: 'UUID' }],
    },
  ];
}

const contextComments = [{ text: '# Split out of Billing when the team divided' }];
const aggregateComments = [{ text: '# The plan, not the invoice that settles it' }];

// A context and an aggregate carrying notes. Neither opens a detail panel, so
// the mark on the header is the only way to read them off the diagram.
function commentedContainers() {
  return [
    { id: 'ctx1', type: 'context', label: 'Collections', comments: contextComments },
    { id: 'agg1', type: 'aggregate', label: 'Arrangement', parentId: 'ctx1', comments: aggregateComments },
    { id: 'sl1', type: 'slice', label: 'Propose plan', parentId: 'agg1' },
    { id: 'cmd1', type: 'command', label: 'ProposePlan', parentId: 'sl1' },
  ];
}

let store;

function fire(target, type, opts) {
  const evt = new MouseEvent(type, Object.assign({ bubbles: true, cancelable: true, button: 0 }, opts));
  target.dispatchEvent(evt);
  return evt;
}

const PRESS = { x: 500, y: 500 };

function pressAndMove(target, dx, dy) {
  fire(target, 'mousedown', { clientX: PRESS.x, clientY: PRESS.y });
  fire(document, 'mousemove', { clientX: PRESS.x + DRAG_THRESHOLD + 1, clientY: PRESS.y });
  fire(document, 'mousemove', { clientX: PRESS.x + dx, clientY: PRESS.y + dy });
}

function release(dx, dy) {
  fire(document, 'mouseup', { clientX: PRESS.x + dx, clientY: PRESS.y + dy });
}

function renderNodes(nodes) {
  store.nodes = nodes;
  store.nodeById = new Map(nodes.map((n) => [n.id, n]));
  store.layoutPositions = Layout.computeLayout(store).positions;
  Renderer.inject(store.dom.svg, Renderer.buildSVG(store));
}

const inSvg = (selector) => store.dom.svg.querySelector(selector);

function markOn(kind, nodeId) {
  return inSvg('[data-marker="' + kind + '"][data-node-id="' + nodeId + '"]');
}

function headerBandOf(sliceId) {
  return inSvg('rect.slice-header[data-slice-id="' + sliceId + '"]');
}

function nameIn(sliceId) {
  return inSvg('text.slice-header[data-slice-id="' + sliceId + '"]:not([data-marker])');
}

function blockOf(nodeId) {
  return inSvg('.diagram-node[data-node-id="' + nodeId + '"]');
}

const numeric = (el, attr) => Number(el.getAttribute(attr));

const rightEdgeOfBand = (band) => numeric(band, 'x') + numeric(band, 'width');

const leftEdgeOfCentred = (el) => numeric(el, 'x') - Layout.labelWidth(el.textContent) / 2;
const rightEdgeOfCentred = (el) => numeric(el, 'x') + Layout.labelWidth(el.textContent) / 2;

function menuItems() {
  return [...store.dom.ctxMenu.querySelectorAll('.ctx-menu-item')].map((el) => el.textContent);
}

function shownTooltip() {
  const el = store.dom.tooltip;
  return {
    shown: el.style.display === 'block',
    reads: [...el.children].map((part) => part.textContent),
  };
}

const panOffset = () => ({ x: store.viewport.offsetX, y: store.viewport.offsetY });

const sliceHeaderTargets = [
  ['the header band', () => headerBandOf('sl1')],
  ['the description mark drawn in it', () => markOn('description', 'sl1')],
  ['the comments mark drawn in it', () => markOn('comments', 'sl1')],
];

const containerHeaderTargets = [
  ['the swimlane header band', () => inSvg('rect.ctx-header')],
  ['the mark drawn in it', () => markOn('comments', 'ctx1')],
  ['the aggregate row band', () => inSvg('rect.agg-row')],
  ['the mark drawn in that', () => markOn('comments', 'agg1')],
];

beforeEach(() => {
  document.body.innerHTML = '';

  store = createStore();

  const svg = document.createElementNS(SVG_NS, 'svg');
  installIdentityMapping(svg);
  const tooltip = document.createElement('div');
  tooltip.style.display = 'none';
  const ctxMenu = document.createElement('div');
  document.body.append(svg, tooltip, ctxMenu);
  store.dom.svg = svg;
  store.dom.tooltip = tooltip;
  store.dom.ctxMenu = ctxMenu;

  Interaction.initEventListeners(store);
  UI.initDelegation(store);
});

describe('the marks on a documented construct', () => {
  beforeEach(() => {
    renderNodes(documentedSlice());
  });

  it.each(sliceHeaderTargets)('carries the slice with the pointer when the press lands on %s', (_, find) => {
    const sliceGroup = inSvg('.slice-sl1');

    pressAndMove(find(), 60, 30);

    expect(sliceGroup.getAttribute('transform')).toBe('translate(60,30)');
    expect(store.dom.svg.classList.contains('panning')).toBe(false);

    release(60, 30);
  });

  it('rides the header corner while a block dragged out to the right stretches the slice', () => {
    const header = headerBandOf('sl1');
    const marks = [markOn('description', 'sl1'), markOn('comments', 'sl1')];
    const restingEdge = rightEdgeOfBand(header);
    const restingInsets = marks.map((mark) => restingEdge - numeric(mark, 'x'));

    pressAndMove(blockOf('cmd1'), 300, 0);

    const stretchedEdge = rightEdgeOfBand(header);
    expect(stretchedEdge).toBeGreaterThan(restingEdge);
    expect(marks.map((mark) => stretchedEdge - numeric(mark, 'x'))).toEqual(restingInsets);
    marks.forEach((mark) => {
      expect(numeric(mark, 'x')).toBeGreaterThan(numeric(nameIn('sl1'), 'x'));
      expect(numeric(mark, 'x')).toBeLessThan(stretchedEdge);
    });
    expect(rightEdgeOfCentred(marks[1])).toBeLessThan(leftEdgeOfCentred(marks[0]));

    release(300, 0);
  });

  it('carries the block with the pointer when the press lands on the mark in its corner', () => {
    pressAndMove(markOn('description', 'cmd1'), 40, 20);
    release(40, 20);

    expect(store.nodeOffsets.cmd1).toEqual({ dx: 40, dy: 20 });
  });

  it.each(sliceHeaderTargets)('opens the slice menu when the right-click lands on %s', (_, find) => {
    fire(find(), 'contextmenu', { clientX: 120, clientY: 140 });

    expect(store.interaction.ctxMenu).toEqual({ targetSliceId: 'sl1' });
    expect(store.dom.ctxMenu.style.display).toBe('block');
    expect(menuItems()).toContain('Add Command');
  });

  it.each([
    ['a slice, which shows no prose in its header', 'sl1', 'Propose plan', sliceDescription],
    ['a block, which shows none on its face', 'cmd1', 'ProposePlan', commandDescription],
  ])('reads out the description of %s when the pointer rests on its mark', (_, nodeId, name, description) => {
    fire(markOn('description', nodeId), 'pointerover', { clientX: 120, clientY: 140 });

    expect(shownTooltip()).toEqual({ shown: true, reads: [name, description] });
  });

  it.each([
    ['a slice, which shows no prose in its header', 'sl1', 'Propose plan', 'Rewritten after the 2024 affordability rules'],
    ['a block, which shows none on its face', 'cmd1', 'ProposePlan', 'Only the plan the customer accepted'],
  ])('reads out the comments of %s when the pointer rests on its mark', (_, nodeId, name, note) => {
    fire(markOn('comments', nodeId), 'pointerover', { clientX: 120, clientY: 140 });

    expect(shownTooltip()).toEqual({ shown: true, reads: [name, note] });
  });
});

describe('the marks on a documented container', () => {
  beforeEach(() => {
    renderNodes(commentedContainers());
  });

  it.each([
    ['a context', 'ctx1', 'Collections', 'Split out of Billing when the team divided'],
    ['an aggregate', 'agg1', 'Arrangement', 'The plan, not the invoice that settles it'],
  ])('reads out the comments of %s, which opens no panel of its own, when the pointer rests on the mark in its header',
    (_, nodeId, name, note) => {
      fire(markOn('comments', nodeId), 'pointerover', { clientX: 120, clientY: 140 });

      expect(shownTooltip()).toEqual({ shown: true, reads: [name, note] });
    });

  it.each(containerHeaderTargets)('opens the menu of the container the header belongs to when the right-click lands on %s', (_, find) => {
    const evt = fire(find(), 'contextmenu', { clientX: 120, clientY: 140 });

    expect(store.interaction.ctxMenu).toEqual({ targetAggId: 'agg1' });
    expect(store.dom.ctxMenu.style.display).toBe('block');
    expect(menuItems()).toContain('Add Slice');
    // Nothing on the header may leave the browser's own menu to open instead.
    expect(evt.defaultPrevented).toBe(true);
  });

  it.each(containerHeaderTargets)('pans the canvas when the press lands on %s', (_, find) => {
    pressAndMove(find(), 60, 30);

    expect(panOffset()).toEqual({ x: 60, y: 30 });
    expect(store.dom.svg.classList.contains('panning')).toBe(true);

    release(60, 30);
    expect(store.dom.svg.classList.contains('panning')).toBe(false);
  });
});

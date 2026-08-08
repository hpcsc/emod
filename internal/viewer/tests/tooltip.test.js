import { describe, it, expect, beforeEach } from 'vitest';
import { installSVGGeometry } from './svg-env.js';

installSVGGeometry();

const { UI } = await import('../static/ui.js');

const SVG_NS = 'http://www.w3.org/2000/svg';

const ORDER_FIELDS = [
  { name: 'orderId', type: 'UUID' },
  { name: 'placedAt', type: 'Timestamp' },
];

function blockGroup(node) {
  const group = document.createElementNS(SVG_NS, 'g');
  group.setAttribute('class', node.type + '-block diagram-node');
  group.setAttribute('data-node-id', node.id);
  group.appendChild(document.createElementNS(SVG_NS, 'rect'));
  return group;
}

function createStore(nodes) {
  const tooltip = document.createElement('div');
  tooltip.style.display = 'none';
  const svg = document.createElementNS(SVG_NS, 'svg');
  nodes.filter((node) => node.type !== 'slice').forEach((node) => svg.appendChild(blockGroup(node)));
  document.body.append(tooltip, svg);

  return {
    nodeById: new Map(nodes.map((node) => [node.id, node])),
    interaction: { selectedNodeId: null, highlighted: {} },
    dom: { tooltip: tooltip, svg: svg },
  };
}

function commandNode(attrs = {}) {
  return Object.assign({ id: 'cmd1', type: 'command', label: 'Place Order' }, attrs);
}

function blockOf(store, nodeId) {
  return store.dom.svg.querySelector('.diagram-node[data-node-id="' + nodeId + '"]');
}

function rectOf(store, nodeId) {
  return blockOf(store, nodeId).querySelector('rect');
}

function pointerOver(el, at = { clientX: 40, clientY: 60 }) {
  el.dispatchEvent(new MouseEvent('pointerover', { bubbles: true, ...at }));
}

function pointerOut(el, at) {
  el.dispatchEvent(new MouseEvent('pointerout', { bubbles: true, ...at }));
}

function hover(store, nodeId, at) {
  pointerOver(rectOf(store, nodeId), at);
}

function descriptionMark(parent, nodeId) {
  const mark = document.createElementNS(SVG_NS, 'text');
  mark.setAttribute('data-marker', 'description');
  mark.setAttribute('data-node-id', nodeId);
  parent.appendChild(mark);
  return mark;
}

function markOnBlock(store, nodeId) {
  return descriptionMark(blockOf(store, nodeId), nodeId);
}

// A slice is drawn as a header band rather than as a block, so its mark hangs
// off the diagram with no block group around it to fall back on.
function markOnSliceHeader(store, nodeId) {
  return descriptionMark(store.dom.svg, nodeId);
}

function movePointer(store, fromNodeId, toNodeId) {
  pointerOut(rectOf(store, fromNodeId), {
    clientX: 140,
    clientY: 60,
    relatedTarget: rectOf(store, toNodeId),
  });
  hover(store, toNodeId, { clientX: 140, clientY: 60 });
}

function shownTooltip(store) {
  const el = store.dom.tooltip;
  const table = el.querySelector('table');
  const [heading, ...body] = [...el.children];
  return {
    shown: el.style.display === 'block',
    title: heading ? heading.textContent : '',
    prose: body.filter((part) => part !== table).map((part) => part.textContent),
    columns: [...el.querySelectorAll('th')].map((th) => th.textContent).filter((text) => text !== ''),
    fields: [...el.querySelectorAll('tbody tr')].map((row) => {
      const cells = [...row.querySelectorAll('td')];
      return { name: cells[0].textContent, type: cells[1].textContent };
    }),
  };
}

beforeEach(() => {
  document.body.innerHTML = '';
});

describe('tooltip on hovering the diagram', () => {
  it('reads out the description of a node stating no fields, and clears away over a sibling stating neither', () => {
    const description = 'Rooms still free on the requested dates';
    const store = createStore([
      { id: 'view1', type: 'view', label: 'Available Rooms', description: description },
      { id: 'view2', type: 'view', label: 'Occupancy' },
    ]);
    UI.initDelegation(store);

    hover(store, 'view1');

    expect(shownTooltip(store)).toEqual({
      shown: true,
      title: 'Available Rooms',
      prose: [description],
      columns: [],
      fields: [],
    });

    movePointer(store, 'view1', 'view2');

    expect(shownTooltip(store).shown).toBe(false);
  });

  it('reads out both the description and a row per field of a node stating each', () => {
    const description = 'Places a customer order against the stock the basket reserved';
    const store = createStore([commandNode({ description: description, fields: ORDER_FIELDS })]);
    UI.initDelegation(store);

    hover(store, 'cmd1');

    expect(shownTooltip(store)).toEqual({
      shown: true,
      title: 'Place Order',
      prose: [description],
      columns: ['Field', 'Type'],
      fields: [
        { name: 'orderId', type: 'UUID' },
        { name: 'placedAt', type: 'Timestamp' },
      ],
    });
  });

  it('adds no prose of its own over a node stating fields and no description', () => {
    const store = createStore([commandNode({ fields: ORDER_FIELDS })]);
    UI.initDelegation(store);

    hover(store, 'cmd1');

    expect(shownTooltip(store)).toEqual({
      shown: true,
      title: 'Place Order',
      prose: [],
      columns: ['Field', 'Type'],
      fields: [
        { name: 'orderId', type: 'UUID' },
        { name: 'placedAt', type: 'Timestamp' },
      ],
    });
  });

  it('reads out a description containing markup as text', () => {
    const description = 'Places an <b>&</b> order';
    const store = createStore([commandNode({ description: description })]);
    UI.initDelegation(store);

    hover(store, 'cmd1');

    expect(shownTooltip(store).prose).toEqual([description]);
    expect(store.dom.tooltip.querySelector('b')).toBeNull();
  });

  it('reads the description off a block\'s mark, without the fields the block itself shows', () => {
    const description = 'Places a customer order against the stock the basket reserved';
    const store = createStore([commandNode({ description: description, fields: ORDER_FIELDS })]);
    UI.initDelegation(store);

    pointerOver(markOnBlock(store, 'cmd1'));

    expect(shownTooltip(store)).toEqual({
      shown: true,
      title: 'Place Order',
      prose: [description],
      columns: [],
      fields: [],
    });
  });

  it('reads a slice description off the mark in its header, and clears it when the pointer leaves', () => {
    const description = 'Everything the guest does to hold a room';
    const store = createStore([
      { id: 'sl1', type: 'slice', label: 'Reserve a room', description: description },
    ]);
    UI.initDelegation(store);
    const mark = markOnSliceHeader(store, 'sl1');

    pointerOver(mark);

    expect(shownTooltip(store)).toEqual({
      shown: true,
      title: 'Reserve a room',
      prose: [description],
      columns: [],
      fields: [],
    });

    pointerOut(mark, { clientX: 300, clientY: 300 });

    expect(shownTooltip(store).shown).toBe(false);
  });

  it.each([
    ['its box', (store) => rectOf(store, 'cmd1')],
    ['the mark in its corner', (store) => markOnBlock(store, 'cmd1')],
  ])('stays away over %s while that node\'s detail panel is already open', (_, find) => {
    const store = createStore([
      commandNode({ description: 'Places a customer order', fields: ORDER_FIELDS }),
    ]);
    store.interaction.selectedNodeId = 'cmd1';
    UI.initDelegation(store);

    pointerOver(find(store));

    expect(shownTooltip(store).shown).toBe(false);
  });
});

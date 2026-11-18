import { describe, it, expect, beforeEach } from 'vitest';
import { installSVGGeometry } from './svg-env.js';

installSVGGeometry();

const { Layout } = await import('../static/layout.js');
const { Renderer } = await import('../static/renderer.js');
const { Model } = await import('../static/model.js');
const { L } = await import('../static/config.js');

function viewReadingThroughCopies() {
  return {
    nodes: [
      { id: 'ctx1', type: 'context', label: 'Payments' },
      { id: 'agg1', type: 'aggregate', label: 'Payment', parentId: 'ctx1' },
      { id: 'sl1', type: 'slice', label: 'Take Payment', parentId: 'agg1' },
      { id: 'shows1', type: 'shows', label: 'Till', parentId: 'sl1' },
      { id: 'cmd1', type: 'command', label: 'TakePayment', parentId: 'sl1' },
      { id: 'evt1', type: 'event', label: 'PaymentTaken', parentId: 'sl1' },
      { id: 'sl2', type: 'slice', label: 'Refund Payment To The Card It Came From', parentId: 'agg1' },
      { id: 'evt2', type: 'event', label: 'PaymentRefunded', parentId: 'sl2' },
      { id: 'sl3', type: 'slice', label: 'Review Payments', parentId: 'agg1' },
      { id: 'view1', type: 'view', label: 'PaymentsView', parentId: 'sl3' },
      { id: 'copy1', type: 'event_copy', label: 'PaymentTaken', parentId: 'sl3', copy_of: 'evt1' },
      { id: 'copy2', type: 'event_copy', label: 'PaymentRefunded', parentId: 'sl3', copy_of: 'evt2' },
    ],
    edges: [
      { source: 'cmd1', target: 'evt1', type: 'flow' },
      { source: 'copy1', target: 'view1', type: 'subscription' },
      { source: 'copy2', target: 'view1', type: 'subscription' },
    ],
  };
}

const fixtureByCopiedType = {
  event: {
    copyType: 'event_copy',
    original: { id: 'evt1', type: 'event', label: 'PaymentTaken' },
    joined: { id: 'view3', type: 'view', label: 'PaymentsView' },
    arrow: { source: 'copy1', target: 'view3', type: 'subscription' },
    sliceLine: 'from Take Payment At The Front Desk Till',
  },
  view: {
    copyType: 'view_copy',
    original: { id: 'view1', type: 'view', label: 'PaymentsView' },
    joined: { id: 'shows3', type: 'shows', label: 'RefundDesk' },
    arrow: { source: 'copy1', target: 'shows3', type: 'reads' },
    sliceLine: 'from Take Payment At The Front Desk Till',
  },
  command: {
    copyType: 'command_copy',
    original: { id: 'cmd1', type: 'command', label: 'TakePayment' },
    joined: { id: 'auto3', type: 'automation', label: 'RetryPayment' },
    arrow: { source: 'auto3', target: 'copy1', type: 'automation_command' },
    sliceLine: 'in Take Payment At The Front Desk Till',
  },
};

function joinedThroughCopy(copiedType) {
  const fixture = fixtureByCopiedType[copiedType];
  return {
    nodes: [
      { id: 'ctx1', type: 'context', label: 'Payments' },
      { id: 'agg1', type: 'aggregate', label: 'Payment', parentId: 'ctx1' },
      { id: 'sl1', type: 'slice', label: 'Take Payment At The Front Desk Till', parentId: 'agg1' },
      Object.assign({ parentId: 'sl1' }, fixture.original),
      { id: 'sl2', type: 'slice', label: 'Settle', parentId: 'agg1' },
      { id: 'sl3', type: 'slice', label: 'Review Payments', parentId: 'agg1' },
      Object.assign({ parentId: 'sl3' }, fixture.joined),
      { id: 'copy1', type: fixture.copyType, label: fixture.original.label, parentId: 'sl3', copy_of: fixture.original.id },
    ],
    edges: [fixture.arrow],
  };
}

function createStore({ nodes, edges }) {
  return {
    nodes: nodes,
    edges: edges,
    hiddenNodes: {},
    nodeOffsets: {},
    layoutPositions: {},
    nodeById: new Map(nodes.map((n) => [n.id, n])),
  };
}

function render(model) {
  const store = createStore(model);
  store.layoutPositions = Layout.computeLayout(store).positions;
  return { svg: Renderer.buildSVG(store), positions: store.layoutPositions };
}

function nodeGroup(svg, nodeId) {
  return svg.querySelector('.diagram-node[data-node-id="' + nodeId + '"]');
}

function boxOf(svg, nodeId) {
  return nodeGroup(svg, nodeId).querySelector('rect');
}

function drawnText(group) {
  return [...group.querySelectorAll('text')].map((el) => el.textContent);
}

function withoutNode(model, nodeId) {
  return { nodes: model.nodes.filter((n) => n.id !== nodeId), edges: model.edges };
}

beforeEach(() => {
  document.body.innerHTML = '';
});

describe('two copies of events under the view of their slice', () => {
  describe('laid out', () => {
    it('sit in a row under the view of their slice', () => {
      const { positions } = render(viewReadingThroughCopies());

      expect(positions.copy1.y).toBe(positions.copy2.y);
      expect(positions.copy1.x + positions.copy1.w).toBeLessThan(positions.copy2.x);
      expect(positions.copy1.y).toBeGreaterThan(positions.view1.y + positions.view1.h);
    });

    it('leave the view of their slice level with the command a screen stands over beside it', () => {
      const { positions } = render(viewReadingThroughCopies());

      expect(positions.view1.y).toBe(positions.cmd1.y);
    });

    it('sit level with the event under that command', () => {
      const { positions } = render(viewReadingThroughCopies());

      expect(positions.copy1.y).toBe(positions.evt1.y);
    });

    it('are each drawn wide enough for the slice they name', () => {
      const { positions } = render(viewReadingThroughCopies());

      expect([positions.copy1.w, positions.copy2.w]).toEqual([
        expect.toSatisfy((w) => w > Layout.textWidth('from Take Payment', L.copySliceLineSize)),
        expect.toSatisfy((w) => w > Layout.textWidth('from Refund Payment To The Card It Came From', L.copySliceLineSize)),
      ]);
    });
  });

  describe('drawn', () => {
    it('draw their arrows into the view from their own slice', () => {
      const { svg, positions } = render(viewReadingThroughCopies());
      const arrow = svg.querySelector('.arrow[data-source="copy1"][data-target="view1"]');
      const [start] = arrow.getAttribute('d').match(/-?[\d.]+,-?[\d.]+/g).map((pair) => pair.split(',').map(Number));

      expect(start).toEqual([positions.copy1.x + positions.copy1.w / 2, positions.copy1.y]);
    });
  });
});

describe.each([['an event', 'event'], ['a view', 'view'], ['a command', 'command']])('a copy of %s', (_, copiedType) => {
  const { original, sliceLine } = fixtureByCopiedType[copiedType];

  describe('laid out', () => {
    it('stays inside its slice', () => {
      const { positions } = render(joinedThroughCopy(copiedType));
      const slice = positions.sl3;

      expect(positions.copy1.x).toBeGreaterThanOrEqual(slice.x);
      expect(positions.copy1.x + positions.copy1.w).toBeLessThanOrEqual(slice.x + slice.w);
      expect(positions.copy1.y + positions.copy1.h).toBeLessThanOrEqual(slice.y + slice.h);
    });

    it('is drawn wide enough for the slice it names', () => {
      const { positions } = render(joinedThroughCopy(copiedType));

      expect(positions.copy1.w).toBeGreaterThan(Layout.textWidth(sliceLine, L.copySliceLineSize));
    });
  });

  describe('drawn', () => {
    it(`is outlined in the stroke of the ${copiedType} it copies`, () => {
      const { svg } = render(joinedThroughCopy(copiedType));

      expect(boxOf(svg, 'copy1').getAttribute('stroke')).toBe(boxOf(svg, original.id).getAttribute('stroke'));
    });

    it('is filled white', () => {
      const { svg } = render(joinedThroughCopy(copiedType));

      expect(boxOf(svg, 'copy1').getAttribute('fill')).toBe('#ffffff');
    });

    it(`is outlined dashed where the ${copiedType} it copies is not`, () => {
      const { svg } = render(joinedThroughCopy(copiedType));

      expect(boxOf(svg, original.id).hasAttribute('stroke-dasharray')).toBe(false);
      expect(boxOf(svg, 'copy1').getAttribute('stroke-dasharray')).toBe('5,3');
    });

    it(`writes the slice that declares the ${copiedType} under its name`, () => {
      const { svg } = render(joinedThroughCopy(copiedType));
      const [name, slice] = [...nodeGroup(svg, 'copy1').querySelectorAll('text')];

      expect(drawnText(nodeGroup(svg, 'copy1'))).toEqual([original.label, sliceLine]);
      expect(Number(slice.getAttribute('y'))).toBeGreaterThan(Number(name.getAttribute('y')));
    });

    it(`writes the name alone once the diagram no longer holds the ${copiedType} it copies`, () => {
      const { svg } = render(withoutNode(joinedThroughCopy(copiedType), original.id));

      expect(drawnText(nodeGroup(svg, 'copy1'))).toEqual([original.label]);
    });

    it('draws no port to connect from', () => {
      const { svg } = render(joinedThroughCopy(copiedType));

      expect(nodeGroup(svg, 'copy1').querySelectorAll('[data-port]')).toHaveLength(0);
      expect(nodeGroup(svg, original.id).querySelectorAll('[data-port]').length).toBeGreaterThan(0);
    });
  });

  describe('moved', () => {
    it('stays in its slice when it is dropped on another', () => {
      const { nodes } = joinedThroughCopy(copiedType);

      const moved = Model.moveToSlice(nodes, 'copy1', 'sl1');

      expect(moved).toEqual([]);
      expect(nodes.find((n) => n.id === 'copy1').parentId).toBe('sl3');
    });
  });
});

describe('copies of every kind in one slice', () => {
  function oneOfEachKind() {
    return {
      nodes: [
        { id: 'ctx1', type: 'context', label: 'Payments' },
        { id: 'agg1', type: 'aggregate', label: 'Payment', parentId: 'ctx1' },
        { id: 'sl1', type: 'slice', label: 'Take Payment', parentId: 'agg1' },
        { id: 'cmd1', type: 'command', label: 'TakePayment', parentId: 'sl1' },
        { id: 'evt1', type: 'event', label: 'PaymentTaken', parentId: 'sl1' },
        { id: 'view1', type: 'view', label: 'PaymentsView', parentId: 'sl1' },
        { id: 'sl2', type: 'slice', label: 'Settle', parentId: 'agg1' },
        { id: 'sl3', type: 'slice', label: 'Chase Payment', parentId: 'agg1' },
        { id: 'auto3', type: 'automation', label: 'RetryPayment', parentId: 'sl3' },
        { id: 'copyE', type: 'event_copy', label: 'PaymentTaken', parentId: 'sl3', copy_of: 'evt1' },
        { id: 'copyV', type: 'view_copy', label: 'PaymentsView', parentId: 'sl3', copy_of: 'view1' },
        { id: 'copyC', type: 'command_copy', label: 'TakePayment', parentId: 'sl3', copy_of: 'cmd1' },
      ],
      edges: [
        { source: 'copyE', target: 'auto3', type: 'automation_trigger' },
        { source: 'copyV', target: 'auto3', type: 'reads' },
        { source: 'auto3', target: 'copyC', type: 'automation_command' },
      ],
    };
  }

  it('sit side by side in one row inside their slice, overlapping no other', () => {
    const { positions } = render(oneOfEachKind());
    const [e, v, c] = ['copyE', 'copyV', 'copyC'].map((id) => positions[id]);
    const slice = positions.sl3;

    expect([v.y, c.y]).toEqual([e.y, e.y]);
    expect(e.x + e.w).toBeLessThan(v.x);
    expect(v.x + v.w).toBeLessThan(c.x);
    expect(e.x).toBeGreaterThanOrEqual(slice.x);
    expect(c.x + c.w).toBeLessThanOrEqual(slice.x + slice.w);
  });
});

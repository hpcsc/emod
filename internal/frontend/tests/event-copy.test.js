import { describe, it, expect, beforeEach } from 'vitest';
import { installSVGGeometry } from './svg-env.js';

installSVGGeometry();

const { Layout } = await import('../static/layout.js');
const { Renderer } = await import('../static/renderer.js');
const { Model } = await import('../static/model.js');
const { L } = await import('../static/config.js');

// Two slices that each declare an event, beside a slice whose view subscribes to
// both through the copies the exporter draws in the view's slice.
function viewReadingThroughCopies() {
  return {
    nodes: [
      { id: 'ctx1', type: 'context', label: 'Payments' },
      { id: 'agg1', type: 'aggregate', label: 'Payment', parentId: 'ctx1' },
      { id: 'sl1', type: 'slice', label: 'Take Payment', parentId: 'agg1' },
      { id: 'trg1', type: 'trigger', label: 'Till', parentId: 'sl1' },
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

describe('a copy of an event', () => {
  describe('laid out', () => {
    it('sits in a row under the view of its slice', () => {
      const { positions } = render(viewReadingThroughCopies());

      expect(positions.copy1.y).toBe(positions.copy2.y);
      expect(positions.copy1.x + positions.copy1.w).toBeLessThan(positions.copy2.x);
      expect(positions.copy1.y).toBeGreaterThan(positions.view1.y + positions.view1.h);
    });

    it('leaves the view of its slice level with the command a trigger stands over beside it', () => {
      const { positions } = render(viewReadingThroughCopies());

      expect(positions.view1.y).toBe(positions.cmd1.y);
    });

    it('sits level with the event under that command', () => {
      const { positions } = render(viewReadingThroughCopies());

      expect(positions.copy1.y).toBe(positions.evt1.y);
    });

    it('is drawn wide enough for the slice it names', () => {
      const { positions } = render(viewReadingThroughCopies());

      expect(positions.copy2.w).toBeGreaterThan(
        Layout.textWidth('from Refund Payment To The Card It Came From', L.copyFromSize));
    });

    it('stays inside its slice', () => {
      const { positions } = render(viewReadingThroughCopies());
      const slice = positions.sl3;

      ['copy1', 'copy2'].forEach((id) => {
        expect(positions[id].x).toBeGreaterThanOrEqual(slice.x);
        expect(positions[id].x + positions[id].w).toBeLessThanOrEqual(slice.x + slice.w);
        expect(positions[id].y + positions[id].h).toBeLessThanOrEqual(slice.y + slice.h);
      });
    });
  });

  describe('drawn', () => {
    it('is outlined in the stroke of the event it copies', () => {
      const { svg } = render(viewReadingThroughCopies());

      expect(boxOf(svg, 'copy1').getAttribute('stroke')).toBe(boxOf(svg, 'evt1').getAttribute('stroke'));
    });

    it('is filled white', () => {
      const { svg } = render(viewReadingThroughCopies());

      expect(boxOf(svg, 'copy1').getAttribute('fill')).toBe('#ffffff');
    });

    it('is outlined dashed where the event it copies is not', () => {
      const { svg } = render(viewReadingThroughCopies());

      expect(boxOf(svg, 'evt1').hasAttribute('stroke-dasharray')).toBe(false);
      expect(boxOf(svg, 'copy1').getAttribute('stroke-dasharray')).toBe('5,3');
    });

    it("writes the slice that declares its event under the event's name", () => {
      const { svg } = render(viewReadingThroughCopies());
      const [name, from] = [...nodeGroup(svg, 'copy1').querySelectorAll('text')];

      expect(drawnText(nodeGroup(svg, 'copy1'))).toEqual(['PaymentTaken', 'from Take Payment']);
      expect(Number(from.getAttribute('y'))).toBeGreaterThan(Number(name.getAttribute('y')));
    });

    it("writes the event's name alone once the diagram no longer holds the event it copies", () => {
      const { svg } = render(withoutNode(viewReadingThroughCopies(), 'evt1'));

      expect(drawnText(nodeGroup(svg, 'copy1'))).toEqual(['PaymentTaken']);
    });

    it('draws no port to connect from', () => {
      const { svg } = render(viewReadingThroughCopies());

      expect(nodeGroup(svg, 'copy1').querySelectorAll('[data-port]')).toHaveLength(0);
      expect(nodeGroup(svg, 'evt1').querySelectorAll('[data-port]').length).toBeGreaterThan(0);
    });

    it('draws the arrow into the view from its own slice', () => {
      const { svg, positions } = render(viewReadingThroughCopies());
      const arrow = svg.querySelector('.arrow[data-source="copy1"][data-target="view1"]');
      const [start] = arrow.getAttribute('d').match(/-?[\d.]+,-?[\d.]+/g).map((pair) => pair.split(',').map(Number));

      expect(start).toEqual([positions.copy1.x + positions.copy1.w / 2, positions.copy1.y]);
    });
  });

  describe('moved', () => {
    it('stays in its slice when it is dropped on another', () => {
      const { nodes } = viewReadingThroughCopies();

      const moved = Model.moveToSlice(nodes, 'copy1', 'sl1');

      expect(moved).toEqual([]);
      expect(nodes.find((n) => n.id === 'copy1').parentId).toBe('sl3');
    });
  });
});

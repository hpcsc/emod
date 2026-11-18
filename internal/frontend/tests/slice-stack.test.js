import { describe, it, expect, beforeEach } from 'vitest';
import { installSVGGeometry } from './svg-env.js';

installSVGGeometry();

const { Layout } = await import('../static/layout.js');
const { Renderer } = await import('../static/renderer.js');

function sliceOfEveryType() {
  return [
    { id: 'ctx1', type: 'context', label: 'Collections' },
    { id: 'agg1', type: 'aggregate', label: 'Arrangement', parentId: 'ctx1' },
    { id: 'sl1', type: 'slice', label: 'Chase arrears', parentId: 'agg1' },
    { id: 'shows1', type: 'shows', label: 'CollectorScreen', parentId: 'sl1' },
    { id: 'trans1', type: 'translation', label: 'PushToDialler', parentId: 'sl1' },
    { id: 'auto1', type: 'automation', label: 'ChaseOverdue', parentId: 'sl1' },
    { id: 'cmd1', type: 'command', label: 'ChaseInvoice', parentId: 'sl1' },
    { id: 'evt1', type: 'event', label: 'InvoiceOverdue', parentId: 'sl1' },
    { id: 'view1', type: 'view', label: 'ArrearsView', parentId: 'sl1' },
  ];
}

function sliceWithoutProcessors() {
  return sliceOfEveryType().filter((n) => n.id !== 'trans1' && n.id !== 'auto1');
}

function wiredEdges() {
  return [
    { source: 'shows1', target: 'cmd1', type: 'shows_command' },
    { source: 'cmd1', target: 'evt1', type: 'flow' },
    { source: 'evt1', target: 'view1', type: 'subscription' },
    { source: 'evt1', target: 'auto1', type: 'automation_trigger' },
    { source: 'auto1', target: 'cmd1', type: 'automation_command' },
    { source: 'view1', target: 'trans1', type: 'reads' },
    { source: 'trans1', target: 'cmd1', type: 'translation_command' },
  ];
}

function createStore(nodes, edges) {
  return {
    nodes: nodes,
    edges: edges || [],
    hiddenNodes: {},
    nodeOffsets: {},
    layoutPositions: {},
    nodeById: new Map(nodes.map((n) => [n.id, n])),
  };
}

function layOut(nodes) {
  return Layout.computeLayout(createStore(nodes)).positions;
}

function render(nodes, edges) {
  const store = createStore(nodes, edges);
  store.layoutPositions = Layout.computeLayout(store).positions;
  return { svg: Renderer.buildSVG(store), positions: store.layoutPositions };
}

// A slice of views alone beside a slice a screen starts, as a state view sits
// beside the state change whose events it reads.
function viewOnlySliceBesideStateChange() {
  return [
    { id: 'ctx1', type: 'context', label: 'Collections' },
    { id: 'agg1', type: 'aggregate', label: 'Arrangement', parentId: 'ctx1' },
    { id: 'sl1', type: 'slice', label: 'Chase invoice', parentId: 'agg1' },
    { id: 'shows1', type: 'shows', label: 'CollectorScreen', parentId: 'sl1' },
    { id: 'cmd1', type: 'command', label: 'ChaseInvoice', parentId: 'sl1' },
    { id: 'evt1', type: 'event', label: 'InvoiceChased', parentId: 'sl1' },
    { id: 'sl2', type: 'slice', label: 'See arrears', parentId: 'agg1' },
    { id: 'view1', type: 'view', label: 'ArrearsView', parentId: 'sl2' },
  ];
}

// A slice of a command and its event alone beside a slice a screen starts, as a
// command slice sits when an automation in another slice issues its command.
function commandOnlySliceBesideStateChange() {
  return [
    { id: 'ctx1', type: 'context', label: 'Collections' },
    { id: 'agg1', type: 'aggregate', label: 'Arrangement', parentId: 'ctx1' },
    { id: 'sl1', type: 'slice', label: 'Chase invoice', parentId: 'agg1' },
    { id: 'shows1', type: 'shows', label: 'CollectorScreen', parentId: 'sl1' },
    { id: 'cmd1', type: 'command', label: 'ChaseInvoice', parentId: 'sl1' },
    { id: 'evt1', type: 'event', label: 'InvoiceChased', parentId: 'sl1' },
    { id: 'sl2', type: 'slice', label: 'Send reminder', parentId: 'agg1' },
    { id: 'cmd2', type: 'command', label: 'SendReminder', parentId: 'sl2' },
    { id: 'evt2', type: 'event', label: 'ReminderSent', parentId: 'sl2' },
  ];
}

function emptySliceBesideScreenOnlySlice() {
  return [
    { id: 'ctx1', type: 'context', label: 'Collections' },
    { id: 'agg1', type: 'aggregate', label: 'Arrangement', parentId: 'ctx1' },
    { id: 'sl1', type: 'slice', label: 'Chase invoice', parentId: 'agg1' },
    { id: 'shows1', type: 'shows', label: 'CollectorScreen', parentId: 'sl1' },
    { id: 'sl2', type: 'slice', label: 'Send reminder', parentId: 'agg1' },
  ];
}

function commandsUnderAScreenAndAnAutomation() {
  return [
    { id: 'ctx1', type: 'context', label: 'Collections' },
    { id: 'agg1', type: 'aggregate', label: 'Arrangement', parentId: 'ctx1' },
    { id: 'sl1', type: 'slice', label: 'Chase invoice', parentId: 'agg1' },
    { id: 'shows1', type: 'shows', label: 'CollectorScreen', parentId: 'sl1' },
    { id: 'cmd1', type: 'command', label: 'ChaseInvoice', parentId: 'sl1' },
    { id: 'sl2', type: 'slice', label: 'Chase overdue', parentId: 'agg1' },
    { id: 'auto1', type: 'automation', label: 'ChaseOverdue', parentId: 'sl2', every: '0 9 * * 1-5' },
    { id: 'cmd2', type: 'command', label: 'SendReminder', parentId: 'sl2' },
  ];
}

const CHILD_IDS = ['shows1', 'trans1', 'auto1', 'cmd1', 'evt1', 'view1'];

function boxesOf(positions, ids) {
  return ids.map((id) => {
    const p = positions[id];
    return { id: id, x: p.x, y: p.y, w: p.w, h: p.h };
  });
}

function overlaps(a, b) {
  return a.x < b.x + b.w && b.x < a.x + a.w && a.y < b.y + b.h && b.y < a.y + a.h;
}

function stepsDown(positions, ids) {
  return ids.slice(1).map((id, i) => positions[id].y - positions[ids[i]].y);
}

function idsByY(positions, ids) {
  return [...ids].sort((a, b) => positions[a].y - positions[b].y);
}

function drawnBoxes(svg) {
  const entries = [...svg.querySelectorAll('.diagram-node')].map((group) => {
    const box = group.querySelector('rect');
    return [group.getAttribute('data-node-id'), {
      x: Number(box.getAttribute('x')),
      y: Number(box.getAttribute('y')),
      w: Number(box.getAttribute('width')),
      h: Number(box.getAttribute('height')),
    }];
  });
  return Object.fromEntries(entries);
}

function arrowBetween(svg, sourceId, targetId) {
  return svg.querySelector('.arrow[data-source="' + sourceId + '"][data-target="' + targetId + '"]');
}

function endpointsOf(arrow) {
  const points = (arrow.getAttribute('d') || '')
    .match(/-?[\d.]+,-?[\d.]+/g)
    .map((pair) => pair.split(','))
    .map(([x, y]) => ({ x: Number(x), y: Number(y) }));
  return { start: points[0], end: points[points.length - 1] };
}

function centreX(box) {
  return box.x + box.w / 2;
}

beforeEach(() => {
  document.body.innerHTML = '';
});

describe('a slice stacked top to bottom', () => {
  describe('declaring every element type', () => {
    it('puts the automation and the translation below the screen and above the command', () => {
      const positions = layOut(sliceOfEveryType());

      expect(positions.shows1.y).toBeLessThan(positions.trans1.y);
      expect(positions.shows1.y).toBeLessThan(positions.auto1.y);
      expect(positions.trans1.y).toBeLessThan(positions.cmd1.y);
      expect(positions.auto1.y).toBeLessThan(positions.cmd1.y);
      expect(positions.cmd1.y).toBeLessThan(positions.evt1.y);
      expect(positions.evt1.y).toBeLessThan(positions.view1.y);
    });

    it('keeps the translation and the automation side by side on the stack centre line', () => {
      const positions = layOut(sliceOfEveryType());
      const translation = positions.trans1;
      const automation = positions.auto1;

      expect(translation.y).toBe(automation.y);
      expect(translation.w).toBe(automation.w);
      expect(translation.x + translation.w).toBeLessThan(automation.x);

      const rowMiddle = (translation.x + automation.x + automation.w) / 2;
      expect(rowMiddle).toBe(centreX(positions.cmd1));
      expect(rowMiddle).toBe(centreX(positions.shows1));
    });

    it('gives every block its own space, overlapping none of its neighbours', () => {
      const positions = layOut(sliceOfEveryType());
      const boxes = boxesOf(positions, CHILD_IDS);

      const collisions = boxes.flatMap((a, i) =>
        boxes.slice(i + 1).filter((b) => overlaps(a, b)).map((b) => a.id + '/' + b.id));
      expect(collisions).toEqual([]);
    });

    it('sizes the slice to its minimum and still holds every block inside it', () => {
      const positions = layOut(sliceOfEveryType());
      const slice = positions.sl1;

      expect(slice.w).toBe(slice.minW);
      expect(slice.h).toBe(slice.minH);
      boxesOf(positions, CHILD_IDS).forEach((box) => {
        expect(box.x).toBeGreaterThanOrEqual(slice.x);
        expect(box.x + box.w).toBeLessThanOrEqual(slice.x + slice.w);
        expect(box.y).toBeGreaterThanOrEqual(slice.y);
        expect(box.y + box.h).toBeLessThanOrEqual(slice.y + slice.h);
      });
    });

    it('draws one arrow per edge, none of them collapsed to a point', () => {
      const edges = wiredEdges();
      const { svg } = render(sliceOfEveryType(), edges);

      const arrows = [...svg.querySelectorAll('.arrow')];
      expect(arrows).toHaveLength(edges.length);
      arrows.forEach((arrow) => {
        expect(arrow.getAttribute('d')).not.toBe('');
        const { start, end } = endpointsOf(arrow);
        [start.x, start.y, end.x, end.y].forEach((n) => expect(Number.isFinite(n)).toBe(true));
        expect(start).not.toEqual(end);
      });
    });

    it('runs the activating event up into the automation and the automation down into its command', () => {
      const { svg } = render(sliceOfEveryType(), wiredEdges());
      const boxes = drawnBoxes(svg);

      const activation = endpointsOf(arrowBetween(svg, 'evt1', 'auto1'));
      expect(activation.start).toEqual({ x: centreX(boxes.evt1), y: boxes.evt1.y });
      expect(activation.end.x).toBe(centreX(boxes.auto1));
      expect(activation.end.y).toBeGreaterThan(boxes.auto1.y + boxes.auto1.h);

      const issued = endpointsOf(arrowBetween(svg, 'auto1', 'cmd1'));
      expect(issued.start).toEqual({ x: centreX(boxes.auto1), y: boxes.auto1.y + boxes.auto1.h });
      expect(issued.end.x).toBe(centreX(boxes.cmd1));
      expect(issued.end.y).toBeLessThan(boxes.cmd1.y);
    });
  });

  describe('showing two screens', () => {
    function twoScreensBesideAScreen() {
      return [
        { id: 'ctx1', type: 'context', label: 'Shop' },
        { id: 'agg1', type: 'aggregate', label: 'Cart', parentId: 'ctx1' },
        { id: 'sl1', type: 'slice', label: 'Apply discount', parentId: 'agg1' },
        { id: 'shows1', type: 'shows', label: 'CheckoutPage', parentId: 'sl1' },
        { id: 'shows2', type: 'shows', label: 'CartDrawer', parentId: 'sl1' },
        { id: 'cmd1', type: 'command', label: 'ApplyDiscount', parentId: 'sl1' },
        { id: 'evt1', type: 'event', label: 'DiscountApplied', parentId: 'sl1' },
        { id: 'sl2', type: 'slice', label: 'Place order', parentId: 'agg1' },
        { id: 'shows3', type: 'shows', label: 'OrderForm', parentId: 'sl2' },
        { id: 'cmd2', type: 'command', label: 'PlaceOrder', parentId: 'sl2' },
        { id: 'evt2', type: 'event', label: 'OrderPlaced', parentId: 'sl2' },
      ];
    }

    it('puts the two screens side by side in the top band of the slice, with no overlap', () => {
      const positions = layOut(twoScreensBesideAScreen());
      const [first, second] = boxesOf(positions, ['shows1', 'shows2']);
      const slice = positions.sl1;

      expect(second.y).toBe(first.y);
      expect(first.y).toBe(positions.shows3.y);
      expect(overlaps(first, second)).toBe(false);
      [first, second].forEach((box) => {
        expect(box.x).toBeGreaterThanOrEqual(slice.x);
        expect(box.x + box.w).toBeLessThanOrEqual(slice.x + slice.w);
      });
    });

    it('keeps the rows under the screens level with the rows of the slice beside it', () => {
      const positions = layOut(twoScreensBesideAScreen());

      expect([positions.cmd1.y, positions.evt1.y]).toEqual([positions.cmd2.y, positions.evt2.y]);
    });

    it('makes each screen box as wide as the longest screen name, so a long name beside a short one fits its box', () => {
      const nodes = twoScreensBesideAScreen();
      nodes.find((n) => n.id === 'shows1').label = 'MemberAccountOverviewDashboardPage';

      const positions = layOut(nodes);

      expect(positions.shows1.w).toBeGreaterThanOrEqual(Layout.labelWidth('MemberAccountOverviewDashboardPage'));
      expect(positions.shows2.w).toBe(positions.shows1.w);
    });
  });

  describe('beside another slice', () => {
    it('keeps the command under an automation level with the command under a screen', () => {
      const positions = layOut(commandsUnderAScreenAndAnAutomation());

      expect(positions.cmd2.y).toBe(positions.cmd1.y);
    });
  });

  describe('declaring only views', () => {
    it('puts the view level with the command a screen stands over in the slice beside it', () => {
      const positions = layOut(viewOnlySliceBesideStateChange());

      expect(positions.view1.y).toBe(positions.cmd1.y);
    });

    it('keeps the view inside its own slice', () => {
      const positions = layOut(viewOnlySliceBesideStateChange());

      const slice = positions.sl2;
      expect(positions.view1.y + positions.view1.h).toBeLessThan(slice.y + slice.h);
    });
  });

  describe('declaring a command and its event, and nothing that issues the command', () => {
    it('puts the command level with the command a screen stands over in the slice beside it', () => {
      const positions = layOut(commandOnlySliceBesideStateChange());

      expect(positions.cmd2.y).toBe(positions.cmd1.y);
    });

    it('puts the event level with the event in the slice beside it', () => {
      const positions = layOut(commandOnlySliceBesideStateChange());

      expect(positions.evt2.y).toBe(positions.evt1.y);
    });

    it('keeps the event inside its own slice', () => {
      const positions = layOut(commandOnlySliceBesideStateChange());

      const slice = positions.sl2;
      expect(positions.evt2.y + positions.evt2.h).toBeLessThan(slice.y + slice.h);
    });
  });

  describe('declaring nothing', () => {
    it('leaves no empty row, so the slice is shorter than a slice that holds only a screen', () => {
      const positions = layOut(emptySliceBesideScreenOnlySlice());

      expect(positions.sl2.minH).toBeLessThan(positions.sl1.minH);
    });
  });

  describe('declaring neither an automation nor a translation', () => {
    it('stacks the screen, then the command, then the event, then the view, leaving no band where the row would sit', () => {
      const positions = layOut(sliceWithoutProcessors());
      const plain = stepsDown(positions, ['shows1', 'cmd1', 'evt1', 'view1']);
      const withRow = stepsDown(layOut(sliceOfEveryType()), ['shows1', 'auto1', 'cmd1']);

      expect(idsByY(positions, ['view1', 'evt1', 'cmd1', 'shows1'])).toEqual(['shows1', 'cmd1', 'evt1', 'view1']);
      expect(plain[0]).toBe(plain[1]);
      expect(plain[1]).toBe(plain[2]);
      expect(withRow).toEqual([plain[0], plain[0]]);
    });
  });
});

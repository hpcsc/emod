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
    { id: 'trg1', type: 'trigger', label: 'CollectorScreen', parentId: 'sl1' },
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
    { source: 'trg1', target: 'cmd1', type: 'trigger_command' },
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

const CHILD_IDS = ['trg1', 'trans1', 'auto1', 'cmd1', 'evt1', 'view1'];

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
    it('puts the automation and the translation below the trigger and above the command', () => {
      const positions = layOut(sliceOfEveryType());

      expect(positions.trg1.y).toBeLessThan(positions.trans1.y);
      expect(positions.trg1.y).toBeLessThan(positions.auto1.y);
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
      expect(rowMiddle).toBe(centreX(positions.trg1));
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

  describe('declaring neither an automation nor a translation', () => {
    it('stacks the trigger, command, event and view evenly, leaving no band where the row would sit', () => {
      const plain = stepsDown(layOut(sliceWithoutProcessors()), ['trg1', 'cmd1', 'evt1', 'view1']);
      const withRow = stepsDown(layOut(sliceOfEveryType()), ['trg1', 'auto1', 'cmd1']);

      expect(plain[0]).toBe(plain[1]);
      expect(plain[1]).toBe(plain[2]);
      expect(withRow).toEqual([plain[0], plain[0]]);
    });
  });
});

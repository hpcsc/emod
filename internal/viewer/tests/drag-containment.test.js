import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { installSVGGeometry } from './svg-env.js';

installSVGGeometry();

const { createStore } = await import('../static/store.js');
const { Layout } = await import('../static/layout.js');
const { Renderer } = await import('../static/renderer.js');
const { Interaction } = await import('../static/interaction.js');
const { bus } = await import('../static/bus.js');
const { L, DRAG_THRESHOLD } = await import('../static/config.js');

// jsdom has no SVG coordinate mapping, so diagram space and client space are
// made one and the same — a 10px mouse move is a 10px move in the diagram.
function installIdentityMapping(svg) {
  let pt = null;
  svg.createSVGPoint = function () {
    pt = { x: 0, y: 0, matrixTransform() { return { x: this.x, y: this.y }; } };
    return pt;
  };
  svg.getScreenCTM = function () {
    return { inverse() { return { multiply(p) { return p; } }; } };
  };
}

function oneSlice() {
  return [
    { id: 'ctx1', type: 'context', label: 'Collections' },
    { id: 'agg1', type: 'aggregate', label: 'Arrangement', parentId: 'ctx1' },
    { id: 'sl1', type: 'slice', label: 'Propose plan', parentId: 'agg1' },
    { id: 'cmd1', type: 'command', label: 'ProposePlan', parentId: 'sl1' },
    { id: 'evt1', type: 'event', label: 'PlanProposed', parentId: 'sl1' },
    { id: 'sl2', type: 'slice', label: 'Cancel plan', parentId: 'agg1' },
    { id: 'cmd2', type: 'command', label: 'CancelPlan', parentId: 'sl2' },
  ];
}

let store;
let rerender;

function render() {
  const result = Layout.computeLayout(store);
  store.layoutPositions = result.positions;
  Renderer.inject(store.dom.svg, Renderer.buildSVG(store));
}

function fire(target, type, opts) {
  const evt = new MouseEvent(type, Object.assign({ bubbles: true, cancelable: true, button: 0 }, opts));
  target.dispatchEvent(evt);
  return evt;
}

// dragBlock drives a real press-move-release over a block, clearing the drag
// threshold first so the gesture is treated as a drag and not a click.
function dragBlock(nodeId, dx, dy, opts) {
  const el = store.dom.svg.querySelector('.diagram-node[data-node-id="' + nodeId + '"]');
  const start = { x: 500, y: 500 };
  fire(el, 'mousedown', { clientX: start.x, clientY: start.y });
  fire(document, 'mousemove', { clientX: start.x + DRAG_THRESHOLD + 1, clientY: start.y });
  fire(document, 'mousemove', { clientX: start.x + dx, clientY: start.y + dy });
  if (!opts || !opts.hold) {
    fire(document, 'mouseup', { clientX: start.x + dx, clientY: start.y + dy });
  }
}

function sliceBox(sliceId) {
  const el = store.dom.svg.querySelector('.slice-box[data-slice-id="' + sliceId + '"]');
  return el && {
    x: Number(el.getAttribute('x')),
    y: Number(el.getAttribute('y')),
    w: Number(el.getAttribute('width')),
    h: Number(el.getAttribute('height')),
  };
}

beforeEach(() => {
  document.body.innerHTML = '';
  store = createStore();
  store.nodes = oneSlice();
  store.edges = [{ source: 'cmd1', target: 'evt1', type: 'flow' }];
  store.nodeById = new Map(store.nodes.map((n) => [n.id, n]));

  const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
  installIdentityMapping(svg);
  document.body.appendChild(svg);
  store.dom.svg = svg;

  rerender = ({ store: s }) => { if (s === store) render(); };
  bus.on('data:changed', rerender);

  render();
  Interaction.initEventListeners(store);
});

afterEach(() => {
  bus.off('data:changed', rerender);
});

describe('dragging a block inside its slice', () => {
  describe('leaving the block where the layout put it', () => {
    it('sizes the slice exactly as it did before drag support', () => {
      const box = sliceBox('sl1');
      const cmd = store.layoutPositions.cmd1;
      const evt = store.layoutPositions.evt1;

      expect(box.x).toBeLessThanOrEqual(cmd.x);
      expect(box.y + box.h).toBeGreaterThanOrEqual(evt.y + evt.h);
      expect(store.layoutPositions.sl1.w).toBe(store.layoutPositions.sl1.minW);
      expect(store.layoutPositions.sl1.h).toBe(store.layoutPositions.sl1.minH);
    });
  });

  describe('towards the bottom-right', () => {
    it('grows the slice to keep the block inside it', () => {
      const before = sliceBox('sl1');

      dragBlock('cmd1', 300, 400);

      const after = sliceBox('sl1');
      const cmd = store.layoutPositions.cmd1;
      expect(after.w).toBeGreaterThan(before.w);
      expect(after.h).toBeGreaterThan(before.h);
      expect(cmd.x + cmd.w).toBeLessThanOrEqual(after.x + after.w);
      expect(cmd.y + cmd.h).toBeLessThanOrEqual(after.y + after.h);
    });

    it('grows the slice while the button is still down, not just on release', () => {
      const before = sliceBox('sl1');

      dragBlock('cmd1', 300, 120, { hold: true });

      expect(sliceBox('sl1').w).toBeGreaterThan(before.w);
    });

    it('pushes the neighbouring slice further right', () => {
      const before = store.layoutPositions.sl2.x;

      dragBlock('cmd1', 300, 0);

      expect(store.layoutPositions.sl2.x).toBeGreaterThan(before);
    });

    it('grows the surrounding aggregate and context to match', () => {
      const beforeCtx = store.layoutPositions.ctx1.h;

      dragBlock('cmd1', 0, 400);

      expect(store.layoutPositions.ctx1.h).toBeGreaterThan(beforeCtx);
    });
  });

  describe('back towards where it started', () => {
    it('shrinks the slice to its original size', () => {
      const original = sliceBox('sl1');
      dragBlock('cmd1', 300, 400);

      dragBlock('cmd1', -300, -400);

      expect(sliceBox('sl1')).toEqual(original);
      expect(store.nodeOffsets.cmd1).toEqual({ dx: 0, dy: 0 });
    });
  });

  describe('towards the top-left', () => {
    it('stops the block at the slice edge instead of letting it escape', () => {
      dragBlock('cmd1', -900, -900);

      const cmd = store.layoutPositions.cmd1;
      const sl = store.layoutPositions.sl1;
      expect(cmd.x).toBe(sl.x + L.slicePad);
      expect(cmd.y).toBe(sl.y + L.sliceTopPad);
    });

    it('keeps the block clear of the slice header', () => {
      dragBlock('cmd1', 0, -900);

      const cmd = store.layoutPositions.cmd1;
      const sl = store.layoutPositions.sl1;
      expect(cmd.y).toBeGreaterThanOrEqual(sl.y + L.sliceHdrH);
    });

    it('records the clamped position, so dragging further does not build up slack', () => {
      dragBlock('cmd1', -900, -900);
      const clamped = Object.assign({}, store.nodeOffsets.cmd1);

      dragBlock('cmd1', -900, -900);

      expect(store.nodeOffsets.cmd1).toEqual(clamped);
    });
  });

  describe('across separate drags', () => {
    it('accumulates rather than starting over from the auto position', () => {
      dragBlock('cmd1', 100, 50);
      const afterFirst = store.layoutPositions.cmd1.x;

      dragBlock('cmd1', 100, 50);

      expect(store.layoutPositions.cmd1.x).toBe(afterFirst + 100);
    });

    it('survives a re-render, rather than snapping back to the auto layout', () => {
      dragBlock('cmd1', 200, 80);
      const dragged = store.layoutPositions.cmd1.x;

      render();
      render();

      expect(store.layoutPositions.cmd1.x).toBe(dragged);
    });
  });
});

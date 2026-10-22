import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { installSVGGeometry, installIdentityMapping } from './svg-env.js';

installSVGGeometry();

const { createStore } = await import('../static/store.js');
const { Layout } = await import('../static/layout.js');
const { Renderer } = await import('../static/renderer.js');
const { Interaction } = await import('../static/interaction.js');
const { bus } = await import('../static/bus.js');
const { L, DRAG_THRESHOLD } = await import('../static/config.js');

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

function slicesAcrossAggregates() {
  return [
    { id: 'ctx1', type: 'context', label: 'Collections' },
    { id: 'agg1', type: 'aggregate', label: 'Arrangement', parentId: 'ctx1' },
    { id: 'sl1', type: 'slice', label: 'Propose plan', parentId: 'agg1' },
    { id: 'trg1', type: 'trigger', label: 'Plan form', parentId: 'sl1' },
    { id: 'cmd1', type: 'command', label: 'ProposePlan', parentId: 'sl1' },
    { id: 'evt1', type: 'event', label: 'PlanProposed', parentId: 'sl1' },
    { id: 'view1', type: 'view', label: 'PlansView', parentId: 'sl1' },
    { id: 'auto1', type: 'automation', label: 'ChasePlan', parentId: 'sl1' },
    { id: 'trans1', type: 'translation', label: 'PartnerPlanImport', parentId: 'sl1', event: { name: 'PartnerPlanImported' } },
    { id: 'evt2', type: 'event', label: 'PartnerPlanImported', parentId: 'sl1' },
    { id: 'sl2', type: 'slice', label: 'Cancel plan', parentId: 'agg1' },
    { id: 'trg2', type: 'trigger', label: 'Cancel form', parentId: 'sl2' },
    { id: 'cmd2', type: 'command', label: 'CancelPlan', parentId: 'sl2' },
    { id: 'agg2', type: 'aggregate', label: 'Schedule', parentId: 'ctx1' },
    { id: 'sl3', type: 'slice', label: 'Book slot', parentId: 'agg2' },
    { id: 'cmd3', type: 'command', label: 'BookSlot', parentId: 'sl3' },
  ];
}

let store;
let rerender;
let edits;

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
  edits = 0;
  bus.on('data:changed', rerender);
  bus.on('diagram:edited', rerender);
  bus.on('diagram:edited', countEdit);

  render();
  Interaction.initEventListeners(store);
});

afterEach(() => {
  bus.off('data:changed', rerender);
  bus.off('diagram:edited', rerender);
  bus.off('diagram:edited', countEdit);
});

function countEdit({ store: s }) {
  if (s === store) edits++;
}

function useNodes(nodes) {
  store.nodes = nodes;
  store.edges = [];
  store.nodeById = new Map(store.nodes.map((n) => [n.id, n]));
  render();
}

function dropOnto(nodeId, sliceId, drive) {
  const p = store.layoutPositions[nodeId];
  const s = store.layoutPositions[sliceId];
  (drive || dragBlock)(nodeId, s.x + s.w / 2 - (p.x + p.w / 2), s.y + s.h / 2 - (p.y + p.h / 2));
}

function isInside(nodeId, sliceId) {
  const p = store.layoutPositions[nodeId];
  const s = store.layoutPositions[sliceId];
  return p.x >= s.x && p.y >= s.y && p.x + p.w <= s.x + s.w && p.y + p.h <= s.y + s.h;
}

function fireTouch(target, type, points) {
  const evt = new Event(type, { bubbles: true, cancelable: true });
  Object.defineProperty(evt, 'touches', { value: type === 'touchend' ? [] : points });
  Object.defineProperty(evt, 'changedTouches', { value: points });
  target.dispatchEvent(evt);
}

function touchDragBlock(nodeId, dx, dy) {
  const el = store.dom.svg.querySelector('.diagram-node[data-node-id="' + nodeId + '"]');
  const at = (x, y) => [{ clientX: x, clientY: y, target: el }];
  fireTouch(el, 'touchstart', at(500, 500));
  fireTouch(store.dom.svg, 'touchmove', at(500 + DRAG_THRESHOLD + 1, 500));
  fireTouch(store.dom.svg, 'touchmove', at(500 + dx, 500 + dy));
  fireTouch(store.dom.svg, 'touchend', at(500 + dx, 500 + dy));
}

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

      dragBlock('cmd1', 300, 400, { hold: true });

      expect(sliceBox('sl1').w).toBeGreaterThan(before.w);
    });

    it('pushes the neighbouring slice further right', () => {
      const before = store.layoutPositions.sl2.x;

      dragBlock('cmd1', 300, 400);

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
      dragBlock('cmd1', 200, 400);
      const dragged = store.layoutPositions.cmd1.x;

      render();
      render();

      expect(store.layoutPositions.cmd1.x).toBe(dragged);
    });
  });

  it('keeps a block released over its own slice as an offset there, announcing no edit', () => {
    dragBlock('cmd1', 10, 60);

    expect(isInside('cmd1', 'sl1')).toBe(true);
    expect(store.nodeById.get('cmd1').parentId).toBe('sl1');
    expect(store.nodeOffsets.cmd1).toEqual({ dx: 10, dy: 60 });
    expect(edits).toBe(0);
  });
});

describe('dragging a block onto another slice', () => {
  beforeEach(() => {
    useNodes(slicesAcrossAggregates());
  });

  it.each(['cmd1', 'evt1', 'view1', 'auto1', 'trg1'])('moves %s to a slice in another aggregate, drawn at its laid-out place there', (nodeId) => {
    dropOnto(nodeId, 'sl3');

    expect(store.nodeById.get(nodeId).parentId).toBe('sl3');
    expect(store.nodeOffsets[nodeId]).toBeUndefined();
    expect(isInside(nodeId, 'sl3')).toBe(true);
    expect(edits).toBe(1);
  });

  it('moves a block to the slice to the right of its own', () => {
    dropOnto('cmd1', 'sl2');

    expect(store.nodeById.get('cmd1').parentId).toBe('sl2');
    expect(isInside('cmd1', 'sl2')).toBe(true);
  });

  it('moves a block across its slice\'s left edge to the slice before it', () => {
    dropOnto('cmd2', 'sl1');

    expect(store.nodeById.get('cmd2').parentId).toBe('sl1');
    expect(isInside('cmd2', 'sl1')).toBe(true);
  });

  it('does not hold the block inside its own slice while its centre is over another one', () => {
    const p = store.layoutPositions.cmd2;
    const s = store.layoutPositions.sl1;

    dragBlock('cmd2', s.x + s.w / 2 - (p.x + p.w / 2), s.y + s.h / 2 - (p.y + p.h / 2), { hold: true });

    const shift = /translate\(([-\d.]+),/.exec(store.dom.svg.querySelector('.diagram-node[data-node-id="cmd2"]').getAttribute('transform'));
    expect(p.x + Number(shift[1])).toBeLessThan(store.layoutPositions.sl2.x);
  });

  it('drops the offset a block had in its old slice', () => {
    dragBlock('cmd1', 10, 400);
    expect(store.nodeOffsets.cmd1).toEqual({ dx: 10, dy: 400 });

    dropOnto('cmd1', 'sl3');

    expect(store.nodeOffsets.cmd1).toBeUndefined();
    expect(isInside('cmd1', 'sl3')).toBe(true);
  });

  it('leaves its own slice at its laid-out size while the block is held over another slice', () => {
    const own = sliceBox('sl1');
    const p = store.layoutPositions.cmd1;
    const s = store.layoutPositions.sl3;

    dragBlock('cmd1', s.x + s.w / 2 - (p.x + p.w / 2), s.y + s.h / 2 - (p.y + p.h / 2), { hold: true });

    expect(sliceBox('sl1')).toEqual(own);
  });

  it('moves a translation together with the event it nests', () => {
    dropOnto('trans1', 'sl3');

    expect(store.nodeById.get('trans1').parentId).toBe('sl3');
    expect(store.nodeById.get('evt2').parentId).toBe('sl3');
    expect(edits).toBe(1);
  });

  it('puts back a trigger released over a slice that has one, as it was before the drag', () => {
    const before = Object.assign({}, store.layoutPositions.trg1);

    dropOnto('trg1', 'sl2');

    expect(store.dom.svg.querySelector('.diagram-node[data-node-id="trg1"]').getAttribute('transform')).toBeNull();
    expect(store.nodeById.get('trg1').parentId).toBe('sl1');
    expect(store.nodeOffsets.trg1).toBeUndefined();
    expect(store.layoutPositions.trg1).toEqual(before);
    expect(edits).toBe(0);
  });

  it('puts back the event a translation nests, released over another slice without its translation', () => {
    const before = Object.assign({}, store.layoutPositions.evt2);

    dropOnto('evt2', 'sl3');

    expect(store.nodeById.get('evt2').parentId).toBe('sl1');
    expect(store.layoutPositions.evt2).toEqual(before);
    expect(edits).toBe(0);
  });

  it('moves a block dragged with one finger as it moves one dragged with the mouse', () => {
    dropOnto('cmd1', 'sl3', touchDragBlock);

    expect(store.nodeById.get('cmd1').parentId).toBe('sl3');
    expect(isInside('cmd1', 'sl3')).toBe(true);
  });
});

describe('dragging a slice over another aggregate', () => {
  beforeEach(() => {
    useNodes(slicesAcrossAggregates());
  });

  it('only reorders it among the slices of its own aggregate', () => {
    const header = store.dom.svg.querySelector('.slice-header[data-slice-id="sl1"]');
    const s = store.layoutPositions.sl1;
    const t = store.layoutPositions.sl3;
    fire(header, 'mousedown', { clientX: 500, clientY: 500 });
    fire(document, 'mousemove', { clientX: 500 + DRAG_THRESHOLD + 1, clientY: 500 });
    fire(document, 'mousemove', { clientX: 500 + t.x - s.x, clientY: 500 + t.y - s.y });
    fire(document, 'mouseup', { clientX: 500 + t.x - s.x, clientY: 500 + t.y - s.y });

    expect(store.nodeById.get('sl1').parentId).toBe('agg1');
    expect(store.nodes.filter((n) => n.parentId === 'agg2').map((n) => n.id)).toEqual(['sl3']);
  });
});

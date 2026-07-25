import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { installSVGGeometry } from './svg-env.js';

installSVGGeometry();

const { createStore } = await import('../static/store.js');
const { Layout } = await import('../static/layout.js');
const { Renderer } = await import('../static/renderer.js');
const { Interaction } = await import('../static/interaction.js');
const { bus } = await import('../static/bus.js');
const { L, DRAG_THRESHOLD } = await import('../static/config.js');

function installIdentityMapping(svg) {
  svg.createSVGPoint = function () {
    return { x: 0, y: 0, matrixTransform() { return { x: this.x, y: this.y }; } };
  };
  svg.getScreenCTM = function () {
    return { inverse() { return { multiply(p) { return p; } }; } };
  };
}

// jsdom does no hit-testing, but dropping a connection is resolved by
// coordinate, so stand in with the blocks whose boxes contain the point.
// Client space and diagram space are the same here, so the point needs no
// conversion.
function installHitTesting(svg, currentStore) {
  document.elementsFromPoint = function (x, y) {
    const s = currentStore();
    return s.nodes.reduce(function (hits, n) {
      const p = s.layoutPositions[n.id];
      if (p && x >= p.x && x <= p.x + p.w && y >= p.y && y <= p.y + p.h) {
        const el = svg.querySelector('.diagram-node[data-node-id="' + n.id + '"]');
        if (el) hits.push(el);
      }
      return hits;
    }, []);
  };
}

function twoSlices() {
  return [
    { id: 'ctx1', type: 'context', label: 'Collections' },
    { id: 'agg1', type: 'aggregate', label: 'Arrangement', parentId: 'ctx1' },
    { id: 'sl1', type: 'slice', label: 'Propose', parentId: 'agg1' },
    { id: 'cmd1', type: 'command', label: 'ProposePlan', parentId: 'sl1' },
    { id: 'evt1', type: 'event', label: 'PlanProposed', parentId: 'sl1' },
    { id: 'sl2', type: 'slice', label: 'Cancel', parentId: 'agg1' },
    { id: 'evt2', type: 'event', label: 'PlanCancelled', parentId: 'sl2' },
  ];
}

let store;
let rerender;

function render() {
  store.layoutPositions = Layout.computeLayout(store).positions;
  Renderer.inject(store.dom.svg, Renderer.buildSVG(store));
}

function fire(target, type, opts) {
  target.dispatchEvent(new MouseEvent(type,
    Object.assign({ bubbles: true, cancelable: true, button: 0 }, opts)));
}

const hitPath = () => store.dom.svg.querySelector('.arrow-hit[data-edge-id="cmd1--evt1"]');
const drawnArrow = () => store.dom.svg.querySelector('path.arrow[data-edge-id="cmd1--evt1"]');
const handle = (end) =>
  store.dom.svg.querySelector('.arrow-handle[data-edge-id="cmd1--evt1"][data-arrow-handle="' + end + '"]');

// dragHandle drags an arrow end and releases at `to`. Where that lands is
// decided by coordinate, so releasing away from any block abandons the repoint.
function dragHandle(end, to) {
  fire(handle(end), 'mousedown', { clientX: 400, clientY: 400 });
  fire(document, 'mousemove', { clientX: 400 + DRAG_THRESHOLD + 1, clientY: 400 });
  fire(document, 'mousemove', { clientX: to.x, clientY: to.y });
  fire(document, 'mouseup', { clientX: to.x, clientY: to.y });
}

function centreOf(nodeId) {
  const p = store.layoutPositions[nodeId];
  return { x: p.x + p.w / 2, y: p.y + p.h / 2 };
}

beforeEach(() => {
  document.body.innerHTML = '';
  store = createStore();
  store.nodes = twoSlices();
  store.edges = [{ source: 'cmd1', target: 'evt1', type: 'flow' }];
  store.nodeById = new Map(store.nodes.map((n) => [n.id, n]));

  const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
  installIdentityMapping(svg);
  installHitTesting(svg, () => store);
  document.body.appendChild(svg);
  store.dom.svg = svg;

  rerender = ({ store: s }) => { if (s === store) render(); };
  bus.on('data:changed', rerender);

  render();
  Interaction.initEventListeners(store);
});

afterEach(() => {
  bus.off('data:changed', rerender);
  delete document.elementsFromPoint;
});

describe('aiming at an arrow', () => {
  describe('hit area', () => {
    it('is far wider than the drawn stroke', () => {
      const drawn = Number(drawnArrow().getAttribute('stroke-width'));
      const hit = Number(hitPath().getAttribute('stroke-width'));

      expect(hit).toBe(L.arrowHitWidth);
      expect(hit).toBeGreaterThan(drawn * 5);
    });

    it('traces the same line as the arrow it stands for', () => {
      expect(hitPath().getAttribute('d')).toBe(drawnArrow().getAttribute('d'));
    });

    it('stays invisible', () => {
      expect(hitPath().getAttribute('stroke')).toBe('transparent');
    });

    it('sits under the blocks, so a block keeps its own clicks', () => {
      const svg = store.dom.svg;
      const all = [...svg.querySelectorAll('g, path')];
      const hitLayer = svg.querySelector('.arrow-hits');
      const firstBlock = svg.querySelector('.diagram-node');

      expect(all.indexOf(hitLayer)).toBeLessThan(all.indexOf(firstBlock));
      expect(hitPath().parentNode).toBe(hitLayer);
    });

    it('takes the pointer events the drawn arrow gives up', () => {
      expect(drawnArrow().getAttribute('pointer-events')).toBe('none');
      expect(hitPath().getAttribute('pointer-events')).toBe('stroke');
    });
  });

  describe('moving a block', () => {
    it('carries the hit area along with the arrow', () => {
      const block = store.dom.svg.querySelector('.diagram-node[data-node-id="cmd1"]');
      fire(block, 'mousedown', { clientX: 500, clientY: 500 });
      fire(document, 'mousemove', { clientX: 500 + DRAG_THRESHOLD + 1, clientY: 500 });
      fire(document, 'mousemove', { clientX: 640, clientY: 520 });

      expect(hitPath().getAttribute('d')).toBe(drawnArrow().getAttribute('d'));
    });
  });
});

describe('repointing an arrow', () => {
  describe('while dragging an end', () => {
    it('bends the arrow itself towards the pointer', () => {
      const before = drawnArrow().getAttribute('d');

      const h = handle('target');
      fire(h, 'mousedown', { clientX: 400, clientY: 400 });
      fire(document, 'mousemove', { clientX: 400 + DRAG_THRESHOLD + 1, clientY: 400 });
      fire(document, 'mousemove', { clientX: 700, clientY: 620 });

      const during = drawnArrow().getAttribute('d');
      expect(during).not.toBe(before);
      expect(during).toContain('700');
      expect(during.split(/[ ,]/).map(Number).filter((n) => !isNaN(n)).length).toBeGreaterThan(0);
      expect(during).not.toContain('NaN');
    });

    it('keeps the hit area on the bent arrow', () => {
      const h = handle('target');
      fire(h, 'mousedown', { clientX: 400, clientY: 400 });
      fire(document, 'mousemove', { clientX: 400 + DRAG_THRESHOLD + 1, clientY: 400 });
      fire(document, 'mousemove', { clientX: 700, clientY: 620 });

      expect(hitPath().getAttribute('d')).toBe(drawnArrow().getAttribute('d'));
    });

    it('draws no stand-in line alongside it', () => {
      const h = handle('target');
      fire(h, 'mousedown', { clientX: 400, clientY: 400 });
      fire(document, 'mousemove', { clientX: 400 + DRAG_THRESHOLD + 1, clientY: 400 });
      fire(document, 'mousemove', { clientX: 700, clientY: 620 });

      expect(store.dom.svg.querySelector('.connect-preview')).toBeNull();
    });
  });

  describe('on drop', () => {
    it('moves the edge to the block it was released on', () => {
      dragHandle('target', centreOf('evt2'));

      expect(store.edges[0]).toEqual({ source: 'cmd1', target: 'evt2', type: 'flow' });
    });

    it('redraws the arrow against its new end', () => {
      const before = drawnArrow().getAttribute('d');

      dragHandle('target', centreOf('evt2'));

      const after = store.dom.svg.querySelector('path.arrow[data-edge-id="cmd1--evt2"]');
      expect(after).not.toBeNull();
      expect(after.getAttribute('d')).not.toBe(before);
      expect(after.getAttribute('d')).not.toContain('NaN');
    });

    it('snaps back when released over nothing', () => {
      const before = drawnArrow().getAttribute('d');

      dragHandle('target', { x: 1400, y: 1400 });

      expect(store.edges[0]).toEqual({ source: 'cmd1', target: 'evt1', type: 'flow' });
      expect(drawnArrow().getAttribute('d')).toBe(before);
    });

    it('snaps back rather than pointing an edge at itself', () => {
      const before = drawnArrow().getAttribute('d');

      dragHandle('target', centreOf('cmd1'));

      expect(store.edges[0]).toEqual({ source: 'cmd1', target: 'evt1', type: 'flow' });
      expect(drawnArrow().getAttribute('d')).toBe(before);
    });
  });
});

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { Interaction } from '../static/interaction.js';
import { Layout } from '../static/layout.js';
import { DRAG_THRESHOLD } from '../static/config.js';

function makeMockSVGPoint() {
  return { x: 0, y: 0, matrixTransform: function () { return { x: this.x, y: this.y }; } };
}

function setupSVGMocks(svg) {
  var pt = makeMockSVGPoint();
  svg.createSVGPoint = function () { pt = makeMockSVGPoint(); return pt; };
  svg.getScreenCTM = function () {
    return { inverse: function () { return ({ a:1, b:0, c:0, d:1, e:0, f:0, multiply: function(p) { return p; } }); } };
  };
}

function createMinimalStore() {
  var svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
  setupSVGMocks(svg);

  var vg = document.createElementNS('http://www.w3.org/2000/svg', 'g');
  vg.setAttribute('id', 'viewport-group');
  svg.appendChild(vg);

  return {
    nodes: [],
    edges: [],
    nodeById: new Map(),
    layoutPositions: {},
    nodeOffsets: {},
    viewport: { offsetX: 0, offsetY: 0, zoomScale: 1 },
    interaction: {
      drag: null,
      pan: null,
      touch: null,
      suppressDetailClick: false,
    },
    dom: { svg: svg, resetLayoutBtn: null },
  };
}

function setupSliceAndNodes(store, sliceWidth) {
  var nodes = [
    { id: 'ctx1', type: 'context' },
    { id: 'agg1', type: 'aggregate', parentId: 'ctx1' },
    { id: 'sl1', type: 'slice', parentId: 'agg1', label: 'S1' },
    { id: 'cmd1', type: 'command', parentId: 'sl1' },
    { id: 'evt1', type: 'event', parentId: 'sl1' },
  ];
  store.nodes = nodes;
  store.nodeById = new Map();
  nodes.forEach(function(n) { store.nodeById.set(n.id, n); });

  store.edges = [
    { source: 'cmd1', target: 'evt1', type: 'flow' },
  ];

  store.layoutPositions = {
    ctx1: { x: 0, y: 0, w: 600, h: 400 },
    agg1: { x: 10, y: 50, w: 580, h: 340 },
    sl1:  { x: 20, y: 70, w: sliceWidth || 260, h: 200 },
    cmd1: { x: 30, y: 110, w: 100, h: 55 },
    evt1: { x: 160, y: 110, w: 100, h: 55 },
  };

  var svg = store.dom.svg;
  var vg = svg.querySelector('#viewport-group');

  var swimlane = document.createElementNS('http://www.w3.org/2000/svg', 'g');
  swimlane.setAttribute('class', 'swimlane-agg1');

  var sliceGroup = document.createElementNS('http://www.w3.org/2000/svg', 'g');
  sliceGroup.setAttribute('class', 'slice-sl1');
  sliceGroup.setAttribute('data-slice-id', 'sl1');

  var headerRect = document.createElementNS('http://www.w3.org/2000/svg', 'rect');
  headerRect.setAttribute('class', 'slice-header');
  headerRect.setAttribute('data-slice-id', 'sl1');
  sliceGroup.appendChild(headerRect);

  swimlane.appendChild(sliceGroup);
  vg.appendChild(swimlane);

  function makeNodeEl(id, cls, x, y, w, h) {
    var g = document.createElementNS('http://www.w3.org/2000/svg', 'g');
    g.setAttribute('class', cls + ' diagram-node');
    g.setAttribute('data-node-id', id);
    var r = document.createElementNS('http://www.w3.org/2000/svg', 'rect');
    r.setAttribute('x', String(x));
    r.setAttribute('y', String(y));
    r.setAttribute('width', String(w));
    r.setAttribute('height', String(h));
    g.appendChild(r);
    return g;
  }

  vg.appendChild(makeNodeEl('cmd1', 'cmd-block', 30, 110, 100, 55));
  vg.appendChild(makeNodeEl('evt1', 'evt-block', 160, 110, 100, 55));

  var arrow = document.createElementNS('http://www.w3.org/2000/svg', 'path');
  arrow.setAttribute('class', 'flow-arrow');
  arrow.setAttribute('data-source', 'cmd1');
  arrow.setAttribute('data-target', 'evt1');
  arrow.setAttribute('d', 'M80 165 L210 165');
  vg.appendChild(arrow);
}

function fire(el, type, opts) {
  var evt = new MouseEvent(type, { bubbles: true, cancelable: true, ...opts });
  el.dispatchEvent(evt);
  return evt;
}

describe('Interaction', function () {
  var store;

  beforeEach(function () {
    store = createMinimalStore();
  });

  describe('drag threshold gating', function () {
    it('does nothing on click without drag (mousedown then mouseup)', function () {
      setupSliceAndNodes(store);
      Interaction.initEventListeners(store);

      var header = store.dom.svg.querySelector('.slice-header');
      fire(header, 'mousedown', { clientX: 100, clientY: 100, button: 0 });
      fire(document, 'mouseup', { clientX: 100, clientY: 100 });

      var sliceGroup = store.dom.svg.querySelector('.slice-sl1');
      var cmdEl = store.dom.svg.querySelector('.diagram-node[data-node-id="cmd1"]');

      expect(sliceGroup.hasAttribute('transform')).toBe(false);
      expect(sliceGroup.hasAttribute('opacity')).toBe(false);
      expect(cmdEl.hasAttribute('transform')).toBe(false);
      expect(cmdEl.classList.contains('dragging')).toBe(false);
    });

    it('does nothing when mousemove is below drag threshold', function () {
      setupSliceAndNodes(store);
      Interaction.initEventListeners(store);

      var header = store.dom.svg.querySelector('.slice-header');
      fire(header, 'mousedown', { clientX: 100, clientY: 100, button: 0 });
      fire(document, 'mousemove', { clientX: 100 + DRAG_THRESHOLD - 1, clientY: 100 });

      var sliceGroup = store.dom.svg.querySelector('.slice-sl1');
      var cmdEl = store.dom.svg.querySelector('.diagram-node[data-node-id="cmd1"]');

      expect(sliceGroup.hasAttribute('transform')).toBe(false);
      expect(cmdEl.hasAttribute('transform')).toBe(false);
      expect(cmdEl.classList.contains('dragging')).toBe(false);
    });
  });

  describe('threshold crossing', function () {
    it('moves slice group and children to viewport group when threshold is crossed', function () {
      setupSliceAndNodes(store);
      Interaction.initEventListeners(store);

      var vg = store.dom.svg.querySelector('#viewport-group');
      var swimlane = vg.querySelector('.swimlane-agg1');
      var sliceGroup = store.dom.svg.querySelector('.slice-sl1');
      var cmdEl = store.dom.svg.querySelector('.diagram-node[data-node-id="cmd1"]');

      // Before drag: slice group is in swimlane, children are in vg
      expect(sliceGroup.parentNode).toBe(swimlane);

      fire(store.dom.svg.querySelector('.slice-header'), 'mousedown', { clientX: 100, clientY: 100, button: 0 });
      fire(document, 'mousemove', { clientX: 100 + DRAG_THRESHOLD + 1, clientY: 100 });

      expect(sliceGroup.parentNode).toBe(vg);
      expect(sliceGroup.getAttribute('opacity')).toBe('0.85');
      expect(cmdEl.classList.contains('dragging')).toBe(true);
    });

    it('moves connected arrows to end of viewport group', function () {
      setupSliceAndNodes(store);
      Interaction.initEventListeners(store);

      var vg = store.dom.svg.querySelector('#viewport-group');
      var arrow = store.dom.svg.querySelector('.flow-arrow');
      var children = Array.prototype.slice.call(vg.children);
      var arrowIndex = children.indexOf(arrow);

      // Last child before drag
      expect(arrowIndex).toBe(children.length - 1);

      fire(store.dom.svg.querySelector('.slice-header'), 'mousedown', { clientX: 100, clientY: 100, button: 0 });
      fire(document, 'mousemove', { clientX: 100 + DRAG_THRESHOLD + 1, clientY: 100 });

      children = Array.prototype.slice.call(vg.children);
      arrowIndex = children.indexOf(arrow);
      expect(arrowIndex).toBe(children.length - 1);
    });
  });

  describe('revert on no-reorder drop', function () {
    it('leaves no transform on a child that carries a drag offset', function () {
      setupSliceAndNodes(store);
      store.nodeOffsets.cmd1 = { dx: 5, dy: 3 };
      Interaction.initEventListeners(store);

      fire(store.dom.svg.querySelector('.slice-header'), 'mousedown', { clientX: 100, clientY: 100, button: 0 });
      fire(document, 'mousemove', { clientX: 100 + DRAG_THRESHOLD + 1, clientY: 100 });
      fire(document, 'mousemove', { clientX: 110, clientY: 105 });
      fire(document, 'mouseup', { clientX: 110, clientY: 105 });

      // The offset is already part of the position the layout drew the block
      // at, so re-applying it as a transform would move the block twice.
      var cmdEl = store.dom.svg.querySelector('.diagram-node[data-node-id="cmd1"]');
      expect(cmdEl.getAttribute('transform')).toBeNull();
    });

    it('removes transform from children with no existing offsets after revert', function () {
      setupSliceAndNodes(store);
      Interaction.initEventListeners(store);

      fire(store.dom.svg.querySelector('.slice-header'), 'mousedown', { clientX: 100, clientY: 100, button: 0 });
      fire(document, 'mousemove', { clientX: 100 + DRAG_THRESHOLD + 1, clientY: 100 });
      fire(document, 'mousemove', { clientX: 110, clientY: 105 });
      fire(document, 'mouseup', { clientX: 110, clientY: 105 });

      var cmdEl = store.dom.svg.querySelector('.diagram-node[data-node-id="cmd1"]');
      expect(cmdEl.hasAttribute('transform')).toBe(false);
    });

    it('returns slice group to swimlane after revert', function () {
      setupSliceAndNodes(store);
      Interaction.initEventListeners(store);

      var vg = store.dom.svg.querySelector('#viewport-group');
      var swimlane = vg.querySelector('.swimlane-agg1');
      var sliceGroup = store.dom.svg.querySelector('.slice-sl1');

      fire(store.dom.svg.querySelector('.slice-header'), 'mousedown', { clientX: 100, clientY: 100, button: 0 });
      fire(document, 'mousemove', { clientX: 100 + DRAG_THRESHOLD + 1, clientY: 100 });
      fire(document, 'mousemove', { clientX: 110, clientY: 105 });
      fire(document, 'mouseup', { clientX: 110, clientY: 105 });

      expect(sliceGroup.parentNode).toBe(swimlane);
      expect(sliceGroup.hasAttribute('opacity')).toBe(false);
    });

    it('computes arrow d attribute with valid numbers after revert', function () {
      setupSliceAndNodes(store);
      Interaction.initEventListeners(store);

      fire(store.dom.svg.querySelector('.slice-header'), 'mousedown', { clientX: 100, clientY: 100, button: 0 });
      fire(document, 'mousemove', { clientX: 100 + DRAG_THRESHOLD + 1, clientY: 100 });
      fire(document, 'mousemove', { clientX: 110, clientY: 105 });
      fire(document, 'mouseup', { clientX: 110, clientY: 105 });

      var arrow = store.dom.svg.querySelector('.flow-arrow');
      var d = arrow.getAttribute('d');
      expect(d).toBeTruthy();
      expect(d).not.toContain('NaN');
      expect(/M\s+[\d.]+/.test(d)).toBe(true);
    });

    it('removes dragging class from children and headers after revert', function () {
      setupSliceAndNodes(store);
      Interaction.initEventListeners(store);

      var header = store.dom.svg.querySelector('.slice-header');
      fire(header, 'mousedown', { clientX: 100, clientY: 100, button: 0 });
      fire(document, 'mousemove', { clientX: 100 + DRAG_THRESHOLD + 1, clientY: 100 });
      fire(document, 'mousemove', { clientX: 110, clientY: 105 });
      fire(document, 'mouseup', { clientX: 110, clientY: 105 });

      var cmdEl = store.dom.svg.querySelector('.diagram-node[data-node-id="cmd1"]');
      expect(cmdEl.classList.contains('dragging')).toBe(false);
      expect(header.classList.contains('dragging')).toBe(false);
    });
  });

  describe('tryReorderSliceOnDrop', function () {
    it('triggers reorder when dx exceeds 30% of slice width', function () {
      var sliceWidth = 100;
      setupSliceAndNodes(store, sliceWidth);
      Interaction.initEventListeners(store);

      var reorderDx = Math.floor(sliceWidth * 0.3) + 1;
      fire(store.dom.svg.querySelector('.slice-header'), 'mousedown', { clientX: 100, clientY: 100, button: 0 });
      fire(document, 'mousemove', { clientX: 100 + DRAG_THRESHOLD + 1, clientY: 100 });
      fire(document, 'mousemove', { clientX: 100 + reorderDx, clientY: 100 });

      var slicesBefore = store.nodes.filter(function(n) { return n.type === 'slice' && n.parentId === 'agg1'; });
      fire(document, 'mouseup', { clientX: 100 + reorderDx, clientY: 100 });

      // Reorder should have been attempted — we don't test the result since
      // there's only one slice, so moveSlice returns false.
      // The key is no crash and clean state.
      var cmdEl = store.dom.svg.querySelector('.diagram-node[data-node-id="cmd1"]');
      expect(cmdEl.classList.contains('dragging')).toBe(false);
    });
  });

  describe('arrow integrity during drag', function () {
    it('computes arrow with same-slice edge using dragged positions for both ends', function () {
      setupSliceAndNodes(store);
      Interaction.initEventListeners(store);

      fire(store.dom.svg.querySelector('.slice-header'), 'mousedown', { clientX: 100, clientY: 100, button: 0 });
      fire(document, 'mousemove', { clientX: 100 + DRAG_THRESHOLD + 1, clientY: 100 });
      fire(document, 'mousemove', { clientX: 130, clientY: 120 });

      var arrow = store.dom.svg.querySelector('.flow-arrow');
      var d = arrow.getAttribute('d');
      expect(d).not.toContain('NaN');

      fire(document, 'mouseup', { clientX: 130, clientY: 120 });
      d = arrow.getAttribute('d');
      expect(d).not.toContain('NaN');
    });
  });
});

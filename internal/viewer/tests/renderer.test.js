import { describe, it, expect, beforeEach } from 'vitest';
import { installSVGGeometry } from './svg-env.js';

installSVGGeometry();

const { Renderer } = await import('../static/renderer.js');
const { Layout } = await import('../static/layout.js');

// Two aggregates side by side in one context, each with a slice, so the
// swimlane has to place two labelled aggregate rows on the same line.
function twoAggregates() {
  return [
    { id: 'ctx1', type: 'context', label: 'Collections' },
    { id: 'agg1', type: 'aggregate', label: 'Arrangement', parentId: 'ctx1' },
    { id: 'sl1', type: 'slice', label: 'Propose plan', parentId: 'agg1' },
    { id: 'cmd1', type: 'command', label: 'ProposePlan', parentId: 'sl1' },
    { id: 'agg2', type: 'aggregate', label: 'Payment', parentId: 'ctx1' },
    { id: 'sl2', type: 'slice', label: 'Take payment', parentId: 'agg2' },
    { id: 'cmd2', type: 'command', label: 'TakePayment', parentId: 'sl2' },
  ];
}

// Two contexts of different widths. Every swimlane is drawn as wide as the
// widest, so ctxNarrow is the one that gets stretched.
function narrowAndWideContexts() {
  return [
    { id: 'ctxWide', type: 'context', label: 'Wide' },
    { id: 'aggWide', type: 'aggregate', label: 'Spread', parentId: 'ctxWide' },
    { id: 'wide1', type: 'slice', label: 'One', parentId: 'aggWide' },
    { id: 'cmdW1', type: 'command', label: 'CmdOne', parentId: 'wide1' },
    { id: 'wide2', type: 'slice', label: 'Two', parentId: 'aggWide' },
    { id: 'cmdW2', type: 'command', label: 'CmdTwo', parentId: 'wide2' },
    { id: 'wide3', type: 'slice', label: 'Three', parentId: 'aggWide' },
    { id: 'cmdW3', type: 'command', label: 'CmdThree', parentId: 'wide3' },

    { id: 'ctxNarrow', type: 'context', label: 'Narrow' },
    { id: 'aggNarrow', type: 'aggregate', label: 'Single', parentId: 'ctxNarrow' },
    { id: 'narrow1', type: 'slice', label: 'Only', parentId: 'aggNarrow' },
    { id: 'cmdN1', type: 'command', label: 'CmdOnly', parentId: 'narrow1' },
  ];
}

function createStore(nodes, hiddenNodes) {
  return {
    nodes: nodes,
    edges: [],
    hiddenNodes: hiddenNodes || {},
    nodeOffsets: {},
    layoutPositions: {},
  };
}

function render(nodes) {
  const store = createStore(nodes);
  store.layoutPositions = Layout.computeLayout(store).positions;
  return { svg: Renderer.buildSVG(store), positions: store.layoutPositions };
}

function labelFor(svg, aggId) {
  return svg.querySelector('.agg-label[data-agg-id="' + aggId + '"]');
}

function rowFor(svg, aggId) {
  return svg.querySelector('.agg-row[data-agg-id="' + aggId + '"]');
}

const numeric = (el, attr) => Number(el.getAttribute(attr));

beforeEach(() => {
  document.body.innerHTML = '';
});

describe('Renderer.buildSVG', () => {
  describe('aggregate rows', () => {
    it('places each aggregate label inside its own row rather than stacking them', () => {
      const { svg } = render(twoAggregates());

      const first = numeric(labelFor(svg, 'agg1'), 'x');
      const second = numeric(labelFor(svg, 'agg2'), 'x');
      expect(second).toBeGreaterThan(first);

      const row = rowFor(svg, 'agg2');
      const rowLeft = numeric(row, 'x');
      expect(second).toBeGreaterThanOrEqual(rowLeft);
      expect(second).toBeLessThan(rowLeft + numeric(row, 'width'));
    });

    it('lays the rows edge to edge without gaps or overlap', () => {
      const { svg } = render(twoAggregates());

      const first = rowFor(svg, 'agg1');
      const second = rowFor(svg, 'agg2');
      expect(numeric(second, 'x')).toBe(numeric(first, 'x') + numeric(first, 'width'));
    });

    it('draws every row before any label, so no row can cover its neighbour text', () => {
      const { svg } = render(twoAggregates());

      const painted = [...svg.querySelectorAll('.agg-row, .agg-label')]
        .map((el) => el.getAttribute('class'));
      expect(painted).toEqual(['agg-row', 'agg-row', 'agg-label', 'agg-label']);
    });

    it('stretches the last row to the swimlane edge when the context is widened', () => {
      const { svg, positions } = render(narrowAndWideContexts());

      // ctxNarrow gets widened to match ctxWide, so its band has to follow.
      const row = rowFor(svg, 'aggNarrow');
      const ctx = positions.ctxNarrow;
      expect(numeric(row, 'x') + numeric(row, 'width')).toBe(ctx.x + ctx.w);
    });

    it('stretches the row hit area along with the row', () => {
      const { svg, positions } = render(narrowAndWideContexts());

      const area = svg.querySelector('.agg-area[data-agg-id="aggNarrow"]');
      const ctx = positions.ctxNarrow;
      expect(numeric(area, 'x') + numeric(area, 'width')).toBe(ctx.x + ctx.w);
    });

    it('leaves the trailing space unclaimed when the context holds slices of its own', () => {
      const nodes = narrowAndWideContexts().concat([
        { id: 'loose', type: 'slice', label: 'Loose slice', parentId: 'ctxNarrow' },
        { id: 'cmdLoose', type: 'command', label: 'Loose', parentId: 'loose' },
      ]);

      const { svg, positions } = render(nodes);

      const row = rowFor(svg, 'aggNarrow');
      const ctx = positions.ctxNarrow;
      expect(numeric(row, 'x') + numeric(row, 'width')).toBeLessThan(ctx.x + ctx.w);
    });

    it('skips an aggregate the layout left out', () => {
      const store = createStore(twoAggregates(), { agg1: true });
      store.layoutPositions = Layout.computeLayout(store).positions;

      const svg = Renderer.buildSVG(store);

      expect(rowFor(svg, 'agg1')).toBeNull();
      expect(labelFor(svg, 'agg1')).toBeNull();
      expect(rowFor(svg, 'agg2')).not.toBeNull();
    });
  });
});

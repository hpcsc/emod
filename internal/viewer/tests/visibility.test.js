import { describe, it, expect, beforeEach } from 'vitest';
import { installSVGGeometry } from './svg-env.js';

installSVGGeometry();

const { UI } = await import('../static/ui.js');
const { Layout } = await import('../static/layout.js');

// One context, two aggregates, two slices each — enough that hiding one slice
// leaves siblings behind to observe.
function modelNodes() {
  return [
    { id: 'ctx1', type: 'context', label: 'Collections' },
    { id: 'agg1', type: 'aggregate', label: 'Arrangement', parentId: 'ctx1' },
    { id: 'sl1', type: 'slice', label: 'Propose plan', parentId: 'agg1' },
    { id: 'cmd1', type: 'command', label: 'ProposePlan', parentId: 'sl1' },
    { id: 'sl2', type: 'slice', label: 'Cancel plan', parentId: 'agg1' },
    { id: 'cmd2', type: 'command', label: 'CancelPlan', parentId: 'sl2' },
    { id: 'agg2', type: 'aggregate', label: 'Payment', parentId: 'ctx1' },
    { id: 'sl3', type: 'slice', label: 'Take payment', parentId: 'agg2' },
    { id: 'cmd3', type: 'command', label: 'TakePayment', parentId: 'sl3' },
  ];
}

function createStore(nodes) {
  const panel = document.createElement('div');
  panel.id = 'visibility-panel';
  panel.className = 'hidden';
  const tree = document.createElement('div');
  tree.id = 'visibility-tree';
  const toggle = document.createElement('button');
  toggle.id = 'visibility-toggle';
  document.body.append(panel, tree, toggle);

  return {
    nodes: nodes || modelNodes(),
    edges: [],
    hiddenNodes: {},
    dom: { visibilityPanel: panel, visibilityToggle: toggle, visibilityTree: tree },
  };
}

function rowFor(store, nodeId) {
  return store.dom.visibilityTree.querySelector('[data-node-id="' + nodeId + '"]');
}

function checkboxFor(store, nodeId) {
  return rowFor(store, nodeId).querySelector('input[type="checkbox"]');
}

function uncheck(store, nodeId) {
  const cb = checkboxFor(store, nodeId);
  cb.checked = false;
  cb.dispatchEvent(new Event('change'));
}

function check(store, nodeId) {
  const cb = checkboxFor(store, nodeId);
  cb.checked = true;
  cb.dispatchEvent(new Event('change'));
}

function laidOut(store) {
  return Object.keys(Layout.computeLayout(store).positions);
}

beforeEach(() => {
  document.body.innerHTML = '';
});

describe('visibility panel', () => {
  describe('tree', () => {
    it('nests slices under their aggregate under their context', () => {
      const store = createStore();

      UI.updateVisibilityTree(store);

      const labels = [...store.dom.visibilityTree.querySelectorAll('.visibility-item label')]
        .map((el) => el.textContent);
      expect(labels).toEqual([
        'Collections', 'Arrangement', 'Propose plan', 'Cancel plan', 'Payment', 'Take payment',
      ]);
    });

    it('indents each level deeper than its parent', () => {
      const store = createStore();

      UI.updateVisibilityTree(store);

      const indent = (id) => parseInt(rowFor(store, id).style.paddingLeft, 10);
      expect(indent('ctx1')).toBeLessThan(indent('agg1'));
      expect(indent('agg1')).toBeLessThan(indent('sl1'));
    });

    it('lists slices declared directly on a context', () => {
      const store = createStore([
        { id: 'ctx1', type: 'context', label: 'Collections' },
        { id: 'sl1', type: 'slice', label: 'Loose slice', parentId: 'ctx1' },
      ]);

      UI.updateVisibilityTree(store);

      expect(rowFor(store, 'sl1')).not.toBeNull();
      expect(parseInt(rowFor(store, 'sl1').style.paddingLeft, 10))
        .toBeGreaterThan(parseInt(rowFor(store, 'ctx1').style.paddingLeft, 10));
    });

    it('says so when the model has no contexts', () => {
      const store = createStore([]);

      UI.updateVisibilityTree(store);

      expect(store.dom.visibilityTree.textContent).toContain('No contexts');
    });
  });

  describe('hiding a slice', () => {
    it('drops the slice and its blocks from the layout, keeping its siblings', () => {
      const store = createStore();
      UI.updateVisibilityTree(store);

      uncheck(store, 'sl1');

      const ids = laidOut(store);
      expect(ids).not.toContain('sl1');
      expect(ids).not.toContain('cmd1');
      expect(ids).toEqual(expect.arrayContaining(['sl2', 'cmd2', 'sl3', 'agg1', 'ctx1']));
    });

    it('closes the gap so the remaining slices sit where the hidden one was', () => {
      const store = createStore();
      const before = Layout.computeLayout(store);
      UI.updateVisibilityTree(store);

      uncheck(store, 'sl1');

      const after = Layout.computeLayout(store);
      expect(after.positions.sl2.x).toBe(before.positions.sl1.x);
      expect(after.positions.sl3.x).toBeLessThan(before.positions.sl3.x);
    });

    it('marks the ancestors as partially visible without hiding them', () => {
      const store = createStore();
      UI.updateVisibilityTree(store);

      uncheck(store, 'sl1');

      expect(checkboxFor(store, 'agg1').indeterminate).toBe(true);
      expect(checkboxFor(store, 'ctx1').indeterminate).toBe(true);
      expect(checkboxFor(store, 'agg1').checked).toBe(true);
      expect(laidOut(store)).toContain('agg1');
    });

    it('leaves an unrelated aggregate fully checked', () => {
      const store = createStore();
      UI.updateVisibilityTree(store);

      uncheck(store, 'sl1');

      expect(checkboxFor(store, 'agg2').indeterminate).toBe(false);
      expect(checkboxFor(store, 'agg2').checked).toBe(true);
    });
  });

  describe('hiding an aggregate', () => {
    it('takes its slices with it', () => {
      const store = createStore();
      UI.updateVisibilityTree(store);

      uncheck(store, 'agg1');

      const ids = laidOut(store);
      expect(ids).not.toContain('agg1');
      expect(ids).not.toContain('sl1');
      expect(ids).not.toContain('sl2');
      expect(ids).toContain('sl3');
      expect(checkboxFor(store, 'sl1').checked).toBe(false);
    });

    it('restores every slice underneath when checked again', () => {
      const store = createStore();
      UI.updateVisibilityTree(store);
      uncheck(store, 'agg1');

      check(store, 'agg1');

      expect(store.hiddenNodes).toEqual({});
      expect(laidOut(store)).toEqual(expect.arrayContaining(['agg1', 'sl1', 'sl2']));
    });
  });

  describe('hiding a context', () => {
    it('removes the whole swimlane from the layout', () => {
      const store = createStore();
      UI.updateVisibilityTree(store);

      uncheck(store, 'ctx1');

      expect(laidOut(store)).toEqual([]);
    });

    it('keeps the row in the panel so it can be brought back', () => {
      const store = createStore();
      UI.updateVisibilityTree(store);

      uncheck(store, 'ctx1');

      expect(checkboxFor(store, 'ctx1').checked).toBe(false);
      check(store, 'ctx1');
      expect(laidOut(store)).toContain('ctx1');
    });
  });

  describe('re-checking a nested item', () => {
    it('reveals its hidden ancestors so the item is actually visible', () => {
      const store = createStore();
      UI.updateVisibilityTree(store);
      uncheck(store, 'ctx1');

      check(store, 'sl1');

      const ids = laidOut(store);
      expect(ids).toContain('sl1');
      expect(ids).toContain('ctx1');
      expect(ids).not.toContain('sl2');
    });
  });

  describe('panel toggle', () => {
    it('shows the panel and fills the tree on first open', () => {
      const store = createStore();

      UI.toggleVisibilityPanel(store);

      expect(store.dom.visibilityPanel.classList.contains('hidden')).toBe(false);
      expect(store.dom.visibilityToggle.classList.contains('active')).toBe(true);
      expect(rowFor(store, 'sl1')).not.toBeNull();
    });

    it('hides the panel again on the next toggle', () => {
      const store = createStore();
      UI.toggleVisibilityPanel(store);

      UI.toggleVisibilityPanel(store);

      expect(store.dom.visibilityPanel.classList.contains('hidden')).toBe(true);
      expect(store.dom.visibilityToggle.classList.contains('active')).toBe(false);
    });

    it('opens a closed panel when told to show it', () => {
      const store = createStore();

      UI.toggleVisibilityPanel(store, true);
      UI.toggleVisibilityPanel(store, true);

      expect(store.dom.visibilityPanel.classList.contains('hidden')).toBe(false);
    });
  });
});

import { describe, it, expect, beforeEach } from 'vitest';
import { installSVGGeometry } from './svg-env.js';

installSVGGeometry();

const { UI } = await import('../static/ui.js');

function createStore() {
  const panel = document.createElement('div');
  const content = document.createElement('div');
  panel.appendChild(content);
  const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
  document.body.append(panel, svg);

  return {
    nodeById: new Map(),
    interaction: { selectedNodeId: null, highlighted: {} },
    dom: { detailPanel: panel, dpContent: content, svg: svg },
  };
}

function automationNode(attrs = {}) {
  return Object.assign(
    { id: 'auto1', type: 'automation', label: 'Chase overdue invoice' },
    attrs,
  );
}

function shownRows(store) {
  return [...store.dom.dpContent.querySelectorAll('tr')].map((row) => ({
    label: row.querySelector('th').textContent,
    value: row.querySelector('td').textContent,
  }));
}

beforeEach(() => {
  document.body.innerHTML = '';
});

describe('detail panel for an automation', () => {
  it('shows the activation event, the view it reads, the command it issues and the target context', () => {
    const store = createStore();

    UI.showDetailPanel(store, automationNode({
      trigger_event: 'InvoiceOverdue',
      reads: 'OutstandingInvoices',
      command: 'ChaseInvoice',
      target_context: 'Collections',
    }));

    expect(store.dom.dpContent.textContent).toContain('Automation');
    expect(shownRows(store)).toEqual([
      { label: 'Trigger Event', value: 'InvoiceOverdue' },
      { label: 'Reads', value: 'OutstandingInvoices' },
      { label: 'Command', value: 'ChaseInvoice' },
      { label: 'Target Context', value: 'Collections' },
    ]);
  });

  it('shows a placeholder for every attribute the automation leaves undeclared', () => {
    const store = createStore();

    UI.showDetailPanel(store, automationNode());

    expect(shownRows(store)).toEqual([
      { label: 'Trigger Event', value: '—' },
      { label: 'Reads', value: '—' },
      { label: 'Command', value: '—' },
      { label: 'Target Context', value: '—' },
    ]);
  });

  it('shows a view name containing markup as text', () => {
    const store = createStore();
    const reads = 'Invoices<b>&</b>Payments';

    UI.showDetailPanel(store, automationNode({ reads }));

    expect(shownRows(store)).toContainEqual({ label: 'Reads', value: reads });
    expect(store.dom.dpContent.querySelector('b')).toBeNull();
  });
});

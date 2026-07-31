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
  it.each([
    {
      activation: 'an event',
      attrs: { on_event: 'InvoiceOverdue' },
      activationRows: [
        { label: 'On Event', value: 'InvoiceOverdue' },
        { label: 'Every', value: '—' },
      ],
    },
    {
      activation: 'a schedule',
      attrs: { every: '0 9 * * 1-5' },
      activationRows: [
        { label: 'On Event', value: '—' },
        { label: 'Every', value: '0 9 * * 1-5' },
      ],
    },
  ])('shows how an automation activated by $activation wakes up, the view it reads, the command it issues and the target context',
    ({ attrs, activationRows }) => {
      const store = createStore();

      UI.showDetailPanel(store, automationNode({
        reads: 'OutstandingInvoices',
        command: 'ChaseInvoice',
        target_context: 'Collections',
        ...attrs,
      }));

      expect(store.dom.dpContent.textContent).toContain('Automation');
      expect(shownRows(store)).toEqual(activationRows.concat([
        { label: 'Reads', value: 'OutstandingInvoices' },
        { label: 'Command', value: 'ChaseInvoice' },
        { label: 'Target Context', value: 'Collections' },
      ]));
    });

  it('shows a placeholder for every attribute the automation leaves undeclared', () => {
    const store = createStore();

    UI.showDetailPanel(store, automationNode());

    expect(shownRows(store)).toEqual([
      { label: 'On Event', value: '—' },
      { label: 'Every', value: '—' },
      { label: 'Reads', value: '—' },
      { label: 'Command', value: '—' },
      { label: 'Target Context', value: '—' },
    ]);
  });

  it.each([
    { label: 'On Event', key: 'on_event', value: 'Invoice<b>&</b>Overdue' },
    { label: 'Reads', key: 'reads', value: 'Invoices<b>&</b>Payments' },
    { label: 'Every', key: 'every', value: '0 9 * * <b>&</b>' },
  ])('shows the $label value containing markup as text', ({ label, key, value }) => {
    const store = createStore();

    UI.showDetailPanel(store, automationNode({ [key]: value }));

    expect(shownRows(store)).toContainEqual({ label, value });
    expect(store.dom.dpContent.querySelector('b')).toBeNull();
  });

  it.each([
    { label: 'On Event', key: 'trigger_event', value: 'InvoiceOverdue' },
    { label: 'Every', key: 'schedule', value: '0 9 * * 1-5' },
  ])('shows no $label for a node stating it under a key the exporter never writes', ({ label, key, value }) => {
    const store = createStore();

    UI.showDetailPanel(store, automationNode({ [key]: value }));

    expect(shownRows(store)).toContainEqual({ label, value: '—' });
  });
});

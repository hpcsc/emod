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

function triggerNode(attrs = {}) {
  return Object.assign(
    { id: 'trg1', type: 'trigger', label: 'Reservation Form' },
    attrs,
  );
}

function commandNode(attrs = {}) {
  return Object.assign(
    {
      id: 'cmd1',
      type: 'command',
      label: 'Place Order',
      fields: [{ name: 'orderId', type: 'UUID' }],
      position: { filename: 'orders.emod', line: 12 },
    },
    attrs,
  );
}

function shownRows(store) {
  return [...store.dom.dpContent.querySelectorAll('tr')].map((row) => ({
    label: row.querySelector('th').textContent,
    value: row.querySelector('td').textContent,
  }));
}

function shownSections(store) {
  return [...store.dom.dpContent.querySelectorAll('.dp-section')].map((section) => {
    const [title, ...body] = [...section.children];
    return {
      title: title.textContent,
      text: body.map((el) => el.textContent).join(''),
    };
  });
}

function sectionTitles(store) {
  return shownSections(store).map((section) => section.title);
}

function sectionText(store, title) {
  const found = shownSections(store).find((section) => section.title === title);
  return found ? found.text : null;
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

describe('detail panel for a trigger', () => {
  it('shows the actor and the reads the node states', () => {
    const store = createStore();

    UI.showDetailPanel(store, triggerNode({ actor: 'Guest', reads: 'AvailableRoomsView' }));

    expect(store.dom.dpContent.textContent).toContain('Trigger');
    expect(shownRows(store)).toEqual([
      { label: 'Actor', value: 'Guest' },
      { label: 'Reads', value: 'AvailableRoomsView' },
    ]);
  });

  it('shows an em-dash placeholder for actor and reads the node does not state', () => {
    const store = createStore();

    UI.showDetailPanel(store, triggerNode());

    expect(shownRows(store)).toEqual([
      { label: 'Actor', value: '—' },
      { label: 'Reads', value: '—' },
    ]);
  });

  it('renders a stale kind the same as a missing kind, with no orphaned Kind row', () => {
    const withoutKind = createStore();
    UI.showDetailPanel(withoutKind, triggerNode());
    const withKind = createStore();
    UI.showDetailPanel(withKind, triggerNode({ kind: 'UI' }));

    expect(shownRows(withoutKind)).toEqual(shownRows(withKind));
    expect(shownRows(withKind)).toEqual([
      { label: 'Actor', value: '—' },
      { label: 'Reads', value: '—' },
    ]);
  });

  it.each([
    { label: 'Actor', key: 'actor', value: 'Guest<b>&</b>Actor' },
    { label: 'Reads', key: 'reads', value: 'Available<b>&</b>RoomsView' },
  ])('shows the $label value containing markup as text', ({ label, key, value }) => {
    const store = createStore();

    UI.showDetailPanel(store, triggerNode({ [key]: value }));

    expect(shownRows(store)).toContainEqual({ label, value });
    expect(store.dom.dpContent.querySelector('b')).toBeNull();
  });
});

describe('detail panel description', () => {
  it('shows a command node description in full, ahead of its fields', () => {
    const store = createStore();
    const description = 'Places a customer order, reserving stock for every line item until the payment clears or the reservation expires.';

    UI.showDetailPanel(store, commandNode({ description }));

    expect(sectionTitles(store)).toEqual(['Description', 'Fields', 'Source']);
    expect(sectionText(store, 'Description')).toBe(description);
  });

  it.each([
    { stated: 'no description', attrs: {} },
    { stated: 'an empty description', attrs: { description: '' } },
  ])('titles its sections without a Description for a node stating $stated', ({ attrs }) => {
    const store = createStore();

    UI.showDetailPanel(store, commandNode(attrs));

    expect(sectionTitles(store)).toEqual(['Fields', 'Source']);
  });

  it.each([
    { type: 'trigger', description: 'A guest submits the reservation form' },
    { type: 'view', description: 'Rooms still free on the requested dates' },
    { type: 'automation', description: 'Chases an invoice once it falls overdue' },
    { type: 'translation', description: 'Turns a provider webhook into a payment command' },
    { type: 'context', description: 'Everything the front desk owns' },
    { type: 'aggregate', description: 'A reservation and its lifecycle' },
    { type: 'slice', description: 'Booking a room from enquiry to confirmation' },
  ])('shows the description a $type states even with no fields section', ({ type, description }) => {
    const store = createStore();

    UI.showDetailPanel(store, { id: type + '1', type, label: 'Front Desk', description });

    expect(sectionTitles(store)).not.toContain('Fields');
    expect(sectionText(store, 'Description')).toBe(description);
  });

  it('shows a description containing markup as text', () => {
    const store = createStore();
    const description = 'Places an <b>&</b> order';

    UI.showDetailPanel(store, commandNode({ description }));

    expect(sectionText(store, 'Description')).toBe(description);
    expect(store.dom.dpContent.querySelector('b')).toBeNull();
  });
});

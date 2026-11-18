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

function editableControls(store) {
  return [...store.dom.dpContent.querySelectorAll('input, textarea, button, [contenteditable]')]
    .map((el) => ({ control: el.tagName.toLowerCase(), reads: el.value || el.textContent }));
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
        { label: 'After', value: '—' },
      ],
    },
    {
      activation: 'a schedule',
      attrs: { every: '0 9 * * 1-5' },
      activationRows: [
        { label: 'On Event', value: '—' },
        { label: 'Every', value: '0 9 * * 1-5' },
        { label: 'After', value: '—' },
      ],
    },
    {
      activation: 'an event after a delay',
      attrs: { on_event: 'InvoiceOverdue', after: '72h' },
      activationRows: [
        { label: 'On Event', value: 'InvoiceOverdue' },
        { label: 'Every', value: '—' },
        { label: 'After', value: '72h' },
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
      { label: 'After', value: '—' },
      { label: 'Reads', value: '—' },
      { label: 'Command', value: '—' },
      { label: 'Target Context', value: '—' },
    ]);
  });

  it.each([
    { label: 'On Event', key: 'on_event', value: 'Invoice<b>&</b>Overdue' },
    { label: 'Reads', key: 'reads', value: 'Invoices<b>&</b>Payments' },
    { label: 'Every', key: 'every', value: '0 9 * * <b>&</b>' },
    { label: 'After', key: 'after', value: '72<b>&</b>h' },
  ])('shows the $label value containing markup as text', ({ label, key, value }) => {
    const store = createStore();

    UI.showDetailPanel(store, automationNode({ [key]: value }));

    expect(shownRows(store)).toContainEqual({ label, value });
    expect(store.dom.dpContent.querySelector('b')).toBeNull();
  });

  it.each([
    { label: 'On Event', key: 'trigger_event', value: 'InvoiceOverdue' },
    { label: 'Every', key: 'schedule', value: '0 9 * * 1-5' },
    { label: 'After', key: 'delay', value: '72h' },
  ])('shows no $label for a node stating it under a key the exporter never writes', ({ label, key, value }) => {
    const store = createStore();

    UI.showDetailPanel(store, automationNode({ [key]: value }));

    expect(shownRows(store)).toContainEqual({ label, value: '—' });
  });
});

describe('detail panel for a shows block', () => {
  function showsNode(attrs = {}) {
    return Object.assign({ id: 'shows1', type: 'shows', label: 'CheckoutPage' }, attrs);
  }

  it('lists the views and the commands of the shows block in the order of their lists, under the screen', () => {
    const store = createStore();

    UI.showDetailPanel(store, showsNode({ reads: ['DiscountView', 'CartView'], command: ['ApplyDiscount', 'ClearCart'] }));

    expect(shownSections(store).map((section) => section.title)).toContain('Screen');
    expect(shownRows(store)).toEqual([
      { label: 'Reads', value: 'DiscountView, CartView' },
      { label: 'Commands', value: 'ApplyDiscount, ClearCart' },
    ]);
  });

  it('shows a list that a node gives as one name', () => {
    const store = createStore();

    UI.showDetailPanel(store, showsNode({ reads: 'CartView', command: 'ApplyDiscount' }));

    expect(shownRows(store)).toEqual([
      { label: 'Reads', value: 'CartView' },
      { label: 'Commands', value: 'ApplyDiscount' },
    ]);
  });

  it.each([
    { label: 'Reads', key: 'reads', value: 'Available<b>&</b>RoomsView' },
    { label: 'Commands', key: 'command', value: 'Apply<b>&</b>Discount' },
  ])('shows a $label value containing markup as text', ({ label, key, value }) => {
    const store = createStore();

    UI.showDetailPanel(store, showsNode({ [key]: [value] }));

    expect(shownRows(store)).toContainEqual({ label, value });
    expect(store.dom.dpContent.querySelector('b')).toBeNull();
  });

  it('shows an em-dash placeholder for an empty list and for a list the node does not state', () => {
    const store = createStore();

    UI.showDetailPanel(store, showsNode({ reads: [] }));

    expect(shownRows(store)).toEqual([
      { label: 'Reads', value: '—' },
      { label: 'Commands', value: '—' },
    ]);
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
    { type: 'shows', description: 'The booking form on the public site' },
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

describe('detail panel comments', () => {
  it('offers nothing to write a comment with, listing the same sections and controls for a commented node as for a bare one', () => {
    const commented = createStore();
    const bare = createStore();

    UI.showDetailPanel(commented, commandNode({
      comments: [{ text: '# Stock is reserved by the basket, not here' }],
    }));
    UI.showDetailPanel(bare, commandNode());

    expect(sectionTitles(commented)).toEqual(['Fields', 'Source']);
    expect(editableControls(commented)).toEqual(editableControls(bare));
  });
});

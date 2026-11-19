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
    nodes: [],
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
      { label: 'Image', value: '—' },
    ]);
  });

  it('shows a list that a node gives as one name', () => {
    const store = createStore();

    UI.showDetailPanel(store, showsNode({ reads: 'CartView', command: 'ApplyDiscount' }));

    expect(shownRows(store)).toEqual([
      { label: 'Reads', value: 'CartView' },
      { label: 'Commands', value: 'ApplyDiscount' },
      { label: 'Image', value: '—' },
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
      { label: 'Image', value: '—' },
    ]);
  });

  it('gives the image of the screen after the views and the commands of the appearance', () => {
    const store = createStore();

    UI.showDetailPanel(store, showsNode({ reads: ['CartView'], command: ['ApplyDiscount'], image: './wireframes/checkout.png' }));

    expect(shownRows(store)).toEqual([
      { label: 'Reads', value: 'CartView' },
      { label: 'Commands', value: 'ApplyDiscount' },
      { label: 'Image', value: './wireframes/checkout.png' },
    ]);
  });

  it.each([
    { name: 'an empty image', attrs: { image: '' } },
    { name: 'a path under a key that the exporter never writes', attrs: { wireframe: './wireframes/checkout.png' } },
  ])('shows an em-dash for $name', ({ attrs }) => {
    const store = createStore();

    UI.showDetailPanel(store, showsNode(attrs));

    expect(shownRows(store)).toContainEqual({ label: 'Image', value: '—' });
  });

  it('gives a node of another type no Image row', () => {
    const store = createStore();

    UI.showDetailPanel(store, automationNode({ reads: 'CartView', image: './wireframes/checkout.png' }));

    expect(shownRows(store).map((row) => row.label)).not.toContain('Image');
    expect(shownRows(store).map((row) => row.label)).toContain('Reads');
  });

  it('shows an image path containing markup as text', () => {
    const store = createStore();

    UI.showDetailPanel(store, showsNode({ image: './<b>checkout</b>.png' }));

    expect(shownRows(store)).toContainEqual({ label: 'Image', value: './<b>checkout</b>.png' });
    expect(store.dom.dpContent.querySelector('b')).toBeNull();
  });

  describe('the slices that show the screen', () => {
    function storeWith(nodes) {
      const store = createStore();
      store.nodes = nodes;
      store.nodeById = new Map(nodes.map((n) => [n.id, n]));
      return store;
    }

    function shownIn(store) {
      const section = [...store.dom.dpContent.querySelectorAll('.dp-section')]
        .find((el) => el.firstElementChild.textContent === 'Shown in');
      return section ? [...section.querySelectorAll('li')].map((item) => item.textContent) : null;
    }

    it('lists the slice of each appearance of the screen, its own included, in the order of the slices', () => {
      const store = storeWith([
        { id: 'slice-1', type: 'slice', label: 'Apply Discount', parentId: 'agg-1' },
        { id: 'slice-2', type: 'slice', label: 'Open Cart', parentId: 'agg-1' },
        { id: 'slice-3', type: 'slice', label: 'Reapply Discount', parentId: 'agg-1' },
        { id: 'shows-3', type: 'shows', label: 'CheckoutPage', parentId: 'slice-3' },
        { id: 'shows-2', type: 'shows', label: 'CartDrawer', parentId: 'slice-2' },
        { id: 'shows-1', type: 'shows', label: 'CheckoutPage', parentId: 'slice-1' },
      ]);

      UI.showDetailPanel(store, store.nodeById.get('shows-3'));

      expect(shownIn(store)).toEqual(['Apply Discount', 'Reapply Discount']);
    });

    it('lists one slice for a screen that one slice shows', () => {
      const store = storeWith([
        { id: 'slice-1', type: 'slice', label: 'Open Cart', parentId: 'agg-1' },
        { id: 'shows-1', type: 'shows', label: 'CartDrawer', parentId: 'slice-1' },
      ]);

      UI.showDetailPanel(store, store.nodeById.get('shows-1'));

      expect(shownIn(store)).toEqual(['Open Cart']);
    });

    it('shows a slice name containing markup as text', () => {
      const store = storeWith([
        { id: 'slice-1', type: 'slice', label: 'Apply <b>&</b> Discount', parentId: 'agg-1' },
        { id: 'shows-1', type: 'shows', label: 'CheckoutPage', parentId: 'slice-1' },
      ]);

      UI.showDetailPanel(store, store.nodeById.get('shows-1'));

      expect(shownIn(store)).toEqual(['Apply <b>&</b> Discount']);
      expect(store.dom.dpContent.querySelector('b')).toBeNull();
    });

    it('gives a node that is not an appearance no such list', () => {
      const store = storeWith([
        { id: 'slice-1', type: 'slice', label: 'Apply Discount', parentId: 'agg-1' },
        { id: 'shows-1', type: 'shows', label: 'ApplyDiscount', parentId: 'slice-1' },
        { id: 'cmd-1', type: 'command', label: 'ApplyDiscount', parentId: 'slice-1' },
      ]);

      UI.showDetailPanel(store, store.nodeById.get('cmd-1'));

      expect(shownIn(store)).toBeNull();
    });
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

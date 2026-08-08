import { describe, it, expect, beforeEach } from 'vitest';
import { readFileSync } from 'fs';
import { resolve } from 'path';
import { installSVGGeometry } from './svg-env.js';

installSVGGeometry();

const { Renderer } = await import('../static/renderer.js');
const { Layout } = await import('../static/layout.js');
const { nodePalette } = await import('../static/config.js');

const viewerHtml = readFileSync(resolve(__dirname, '../static/viewer.html'), 'utf-8');

const classForType = {
  trigger: 'trg',
  command: 'cmd',
  event: 'evt',
  view: 'view',
  automation: 'auto',
  translation: 'trans',
};

// Extracts the fill declaration for a selector from the embedded CSS.
function cssFill(selector) {
  const styleMatch = viewerHtml.match(/<style[^>]*>([\s\S]*?)<\/style>/i);
  expect(styleMatch).not.toBeNull();
  const css = styleMatch[1];
  const re = new RegExp(
    selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') +
    '\\s*\\{[^}]*\\bfill\\s*:\\s*(#[0-9a-fA-F]{6})',
    'i'
  );
  const m = css.match(re);
  expect(m).not.toBeNull();
  return m[1].toLowerCase();
}

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

const ellipsis = '…';
const contextDescription = 'Everything after an invoice falls due';
const overlongContextDescription =
  'Everything that happens after an invoice falls due, from the first reminder ' +
  'through negotiated instalment plans and on to the point where the balance is ' +
  'either settled in full or written off by finance';
const aggregateDescription = 'Payment terms';
const overlongAggregateDescription = 'Instalment plans negotiated with the customer';
const neighbourAggregateDescription = 'Card capture';
const crowdingAggregateName = 'ArrangementNegotiationSchedule';
const neighbourCrowdingName = 'SettlementReconciliationLedger';

function describedAs(nodes, textById) {
  return nodes.map((n) => (textById[n.id] ? { ...n, description: textById[n.id] } : n));
}

// Comments arrive from the exporter as the model's own notes, each still
// carrying the leading # the source spells it with.
function commentedAs(nodes, commentsById) {
  return nodes.map((n) => (commentsById[n.id] ? { ...n, comments: commentsById[n.id] } : n));
}

function labelledAs(nodes, labelById) {
  return nodes.map((n) => (labelById[n.id] ? { ...n, label: labelById[n.id] } : n));
}

// A described context beside an undescribed one, so a single render puts a
// header carrying prose next to a header carrying none.
function describedAndPlainContexts() {
  return describedAs(narrowAndWideContexts(), { ctxWide: contextDescription });
}

// A description far wider than any swimlane beside one of a few words, so the
// header that has to give way and the header that does not are drawn together.
function crowdedAndRoomyContexts() {
  return describedAs(narrowAndWideContexts(), {
    ctxWide: overlongContextDescription,
    ctxNarrow: contextDescription,
  });
}

function describedAndPlainAggregates() {
  return describedAs(twoAggregates(), { agg1: aggregateDescription });
}

// A description far wider than its aggregate's row beside one that fits its own,
// so the row that has to give way and the row that does not are drawn together.
function crowdedAndRoomyDescriptions() {
  return describedAs(twoAggregates(), {
    agg1: overlongAggregateDescription,
    agg2: neighbourAggregateDescription,
  });
}

// A documented aggregate whose name alone reaches past its row, leaving its
// description nowhere to go.
function aggregateNameFillingItsRow() {
  return labelledAs(describedAndPlainAggregates(), { agg1: crowdingAggregateName });
}

const contextComments = [{ text: '# Split out of Billing when the team divided' }];
const aggregateComments = [{ text: '# The plan, not the invoice that settles it' }];

// A container carrying notes beside one whose prose is already on show, so a
// single render draws the header that has something left to read next to the
// header that has not.
function commentedAndDescribedContexts() {
  return commentedAs(
    describedAs(narrowAndWideContexts(), {
      ctxWide: contextDescription,
      ctxNarrow: contextDescription,
    }),
    { ctxWide: contextComments },
  );
}

function commentedAndDescribedAggregates() {
  return commentedAs(
    describedAs(twoAggregates(), {
      agg1: aggregateDescription,
      agg2: neighbourAggregateDescription,
    }),
    { agg1: aggregateComments },
  );
}

// The crowded headers of the description fixtures, with notes on the crowded one
// only: its description now has the mark to give way to as well as the edge.
function crowdedCommentedContexts() {
  return commentedAs(crowdedAndRoomyContexts(), { ctxWide: contextComments });
}

function crowdedCommentedAggregates() {
  return commentedAs(crowdedAndRoomyDescriptions(), { agg1: aggregateComments });
}

// Two aggregate rows whose names alone reach past their own edge, notes on the
// first only, so one overflowing name has a mark standing where it runs and the
// other has nothing in its way.
function crowdingNamesOneCommented() {
  const threeRows = twoAggregates().concat([
    { id: 'agg3', type: 'aggregate', label: 'Refund', parentId: 'ctx1' },
    { id: 'sl3', type: 'slice', label: 'Refund overpayment', parentId: 'agg3' },
    { id: 'cmd3', type: 'command', label: 'RefundOverpayment', parentId: 'sl3' },
  ]);
  return commentedAs(
    labelledAs(threeRows, { agg1: crowdingAggregateName, agg2: neighbourCrowdingName }),
    { agg1: aggregateComments },
  );
}

const describedThroughoutText = {
  ctx1: contextDescription,
  agg1: aggregateDescription,
  agg2: 'Card and bank capture',
  sl1: 'Offers the customer a plan',
};

// The two-aggregate swimlane documented at every level that carries prose, the
// slice included — the slice description is what proves the slice header leaves
// it undrawn — so it can be compared against the undocumented twin
// twoAggregates() returns.
function describedThroughout() {
  return describedAs(twoAggregates(), describedThroughoutText);
}

const clockMarking = '⏱';
const cronExpression = '0 9 * * 1-5';
const cadenceBadge = clockMarking + ' ' + cronExpression;
const automationFill = '#e1d5e7';

// An automation woken by a clock beside one woken by an event and a command, all
// in one slice, so a single render draws the marked case, the unmarked case and
// a neighbouring box the cadence has to leave alone.
function pairedAutomations(schedule) {
  const scheduled = { id: 'autoScheduled', type: 'automation', label: 'SweepArrears', parentId: 'sl1' };
  if (schedule) scheduled.every = schedule;

  return [
    { id: 'ctx1', type: 'context', label: 'Collections' },
    { id: 'agg1', type: 'aggregate', label: 'Arrangement', parentId: 'ctx1' },
    { id: 'sl1', type: 'slice', label: 'Chase arrears', parentId: 'agg1' },
    scheduled,
    { id: 'autoOnEvent', type: 'automation', label: 'ChaseOverdue', parentId: 'sl1', on_event: 'InvoiceOverdue' },
    { id: 'cmd1', type: 'command', label: 'ChaseInvoice', parentId: 'sl1' },
  ];
}

const commandDescription = 'Offers the customer an instalment plan';
const sliceDescription = 'Everything it takes to agree a plan';
const automationDescription = 'Sweeps every arrears account each weekday';
const translationDescription = 'Hands the arrears file to the dialler';

const commandComments = [
  { text: '# Only the plan the customer accepted' },
  { text: '# Finance signs off anything over twelve months' },
];
const sliceComments = [{ text: '# Rewritten after the 2024 affordability rules' }];
const automationComments = [{ text: '# Skips accounts already in a payment holiday' }];
const translationComments = [{ text: '# The dialler drops anything sent after 8pm' }];

// Two commands in one slice, so a render can put a marked box beside an
// unmarked one whatever the mark is for.
function twoCommandsInOneSlice() {
  return [
    { id: 'ctx1', type: 'context', label: 'Collections' },
    { id: 'agg1', type: 'aggregate', label: 'Arrangement', parentId: 'ctx1' },
    { id: 'sl1', type: 'slice', label: 'Propose plan', parentId: 'agg1' },
    { id: 'cmd1', type: 'command', label: 'ProposePlan', parentId: 'sl1' },
    { id: 'cmd2', type: 'command', label: 'TakePayment', parentId: 'sl1' },
  ];
}

function describedAndPlainCommands() {
  return describedAs(twoCommandsInOneSlice(), { cmd1: commandDescription });
}

function commentedAndPlainCommands() {
  return commentedAs(twoCommandsInOneSlice(), { cmd1: commandComments });
}

const eventDescription = 'The money reached the trust account';
const eventComments = [{ text: '# A payment made after 6pm lands the next morning' }];

// Three short-named events share one row of a slice, so each is placed a
// fraction of the width the command stacked above them gets — the narrowest
// block the layout draws, and the least room a corner mark ever has.
function documentedEventInARow() {
  return [
    { id: 'ctx1', type: 'context', label: 'Collections' },
    { id: 'agg1', type: 'aggregate', label: 'Arrangement', parentId: 'ctx1' },
    { id: 'sl1', type: 'slice', label: 'Take payment', parentId: 'agg1' },
    { id: 'cmd1', type: 'command', label: 'TakePayment', parentId: 'sl1' },
    { id: 'evt1', type: 'event', label: 'Made', parentId: 'sl1', description: eventDescription, comments: eventComments },
    { id: 'evt2', type: 'event', label: 'Paid', parentId: 'sl1' },
    { id: 'evt3', type: 'event', label: 'Sent', parentId: 'sl1' },
  ];
}

// An automation stating its cadence and a translation naming its external
// system, both carrying prose of either kind: each box already draws a second
// row of text, which is what a mark in the corner can end up painted over.
function documentedProcessors() {
  return [
    { id: 'ctx1', type: 'context', label: 'Collections' },
    { id: 'agg1', type: 'aggregate', label: 'Arrangement', parentId: 'ctx1' },
    { id: 'sl1', type: 'slice', label: 'Chase arrears', parentId: 'agg1' },
    { id: 'auto1', type: 'automation', label: 'SweepArrears', parentId: 'sl1', every: cronExpression, description: automationDescription, comments: automationComments },
    { id: 'sl2', type: 'slice', label: 'Push to dialler', parentId: 'agg1' },
    { id: 'trans1', type: 'translation', label: 'PushToDialler', parentId: 'sl2', external_system: 'Genesys', description: translationDescription, comments: translationComments },
  ];
}

const triggerDescription = 'The screen an agent works the arrears list from';
const triggerComments = [{ text: '# Replaces the spreadsheet the team used to work from' }];

// A documented trigger: its box is drawn as a screen, with a bar painted across
// the top — the one thing already standing where a corner mark lands.
function documentedTrigger() {
  return [
    { id: 'ctx1', type: 'context', label: 'Collections' },
    { id: 'agg1', type: 'aggregate', label: 'Arrangement', parentId: 'ctx1' },
    { id: 'sl1', type: 'slice', label: 'Chase arrears', parentId: 'agg1' },
    { id: 'trg1', type: 'trigger', label: 'CollectorScreen', parentId: 'sl1', description: triggerDescription, comments: triggerComments },
  ];
}

const viewLabel = 'ArrearsView';

// One view read by a trigger, an automation and a translation in the same
// slice. Each reader carries the read on itself as well, so a render can tell
// an arrow drawn from the edge list apart from one drawn off block metadata.
function viewReadThreeWays() {
  return [
    { id: 'ctx1', type: 'context', label: 'Collections' },
    { id: 'agg1', type: 'aggregate', label: 'Arrangement', parentId: 'ctx1' },
    { id: 'sl1', type: 'slice', label: 'Chase arrears', parentId: 'agg1' },
    { id: 'trg1', type: 'trigger', label: 'NightlySweep', parentId: 'sl1', reads: viewLabel },
    { id: 'auto1', type: 'automation', label: 'ChaseOverdue', parentId: 'sl1', reads: viewLabel },
    { id: 'trans1', type: 'translation', label: 'PushToDialler', parentId: 'sl1', reads: viewLabel },
    { id: 'view1', type: 'view', label: viewLabel, parentId: 'sl1' },
  ];
}

// One of each node type, so a single render can show the trigger's screen framing
// beside the other five shapes.
function allElementTypes() {
  return [
    { id: 'ctx1', type: 'context', label: 'Palette' },
    { id: 'agg1', type: 'aggregate', label: 'Agg', parentId: 'ctx1' },
    { id: 'sl1', type: 'slice', label: 'S', parentId: 'agg1' },
    { id: 'trg1', type: 'trigger', label: 'Form', parentId: 'sl1' },
    { id: 'cmd1', type: 'command', label: 'Cmd', parentId: 'sl1' },
    { id: 'evt1', type: 'event', label: 'Evt', parentId: 'sl1' },
    { id: 'view1', type: 'view', label: 'Rmo', parentId: 'sl1' },
    { id: 'auto1', type: 'automation', label: 'Auto', parentId: 'sl1' },
    { id: 'trans1', type: 'translation', label: 'Trans', parentId: 'sl1', external_system: 'Stripe' },
  ];
}

function readsEdgesOutOfView() {
  return [
    { source: 'view1', target: 'trg1', type: 'reads' },
    { source: 'view1', target: 'auto1', type: 'reads' },
    { source: 'view1', target: 'trans1', type: 'reads' },
  ];
}

function arrowBetween(svg, sourceId, targetId) {
  return svg.querySelector('.arrow[data-source="' + sourceId + '"][data-target="' + targetId + '"]');
}

function appearanceOf(arrow) {
  return {
    cls: arrow.getAttribute('class'),
    stroke: arrow.getAttribute('stroke'),
    marker: arrow.getAttribute('marker-end'),
    dash: arrow.getAttribute('stroke-dasharray'),
  };
}

function drawnArrowEnds(svg) {
  return [...svg.querySelectorAll('.arrow')]
    .map((el) => el.getAttribute('data-source') + '>' + el.getAttribute('data-target'));
}

// The sides a block can be connected from, each described independently of the
// renderer: the midpoint of the side, and the unit vector pointing away from
// the block. Ports are asserted against these rather than against the layout
// constants they are built from.
const SIDES = {
  top:    (box) => ({ anchor: { x: box.x + box.w / 2, y: box.y },         out: { x: 0,  y: -1 } }),
  right:  (box) => ({ anchor: { x: box.x + box.w, y: box.y + box.h / 2 }, out: { x: 1,  y: 0  } }),
  bottom: (box) => ({ anchor: { x: box.x + box.w / 2, y: box.y + box.h }, out: { x: 0,  y: 1  } }),
  left:   (box) => ({ anchor: { x: box.x, y: box.y + box.h / 2 },         out: { x: -1, y: 0  } }),
};

// How far a point sits beyond the edge the port is on. Negative is inside the
// block, positive is out in the gap around it.
function reachBeyondEdge(point, side) {
  return (point.x - side.anchor.x) * side.out.x + (point.y - side.anchor.y) * side.out.y;
}

// How far a point sits from the middle of the edge, measured along the edge.
function offsetAlongEdge(point, side) {
  return Math.abs((point.x - side.anchor.x) * side.out.y - (point.y - side.anchor.y) * side.out.x);
}

function portOf(group, direction) {
  return group.querySelector('[data-port="' + direction + '"]');
}

function arrowheadCorners(port) {
  return port.querySelector('.port-head').getAttribute('d')
    .match(/-?[\d.]+,-?[\d.]+/g)
    .map((pair) => pair.split(','))
    .map(([x, y]) => ({ x: Number(x), y: Number(y) }));
}

function hitAreaOf(port) {
  const circle = port.querySelector('.port-hit');
  return {
    centre: { x: numeric(circle, 'cx'), y: numeric(circle, 'cy') },
    radius: numeric(circle, 'r'),
  };
}

function createStore(nodes, { edges = [], hiddenNodes = {} } = {}) {
  return {
    nodes: nodes,
    edges: edges,
    hiddenNodes: hiddenNodes,
    nodeOffsets: {},
    layoutPositions: {},
  };
}

function render(nodes, edges) {
  const store = createStore(nodes, { edges });
  store.layoutPositions = Layout.computeLayout(store).positions;
  return { svg: Renderer.buildSVG(store), positions: store.layoutPositions };
}

function labelFor(svg, aggId) {
  return svg.querySelector('.agg-label[data-agg-id="' + aggId + '"]');
}

function rowFor(svg, aggId) {
  return svg.querySelector('.agg-row[data-agg-id="' + aggId + '"]');
}

function headerFor(svg, ctxId) {
  return svg.querySelector('rect.ctx-header[data-ctx-id="' + ctxId + '"]');
}

function sliceHeaderFor(svg, sliceId) {
  return svg.querySelector('rect.slice-header[data-slice-id="' + sliceId + '"]');
}

function rightEdgeOfBand(band) {
  return numeric(band, 'x') + numeric(band, 'width');
}

function bottomEdgeOfBand(band) {
  return numeric(band, 'y') + numeric(band, 'height');
}

// A string is drawn from a single point, so the band it sits in is the band that
// point falls in.
function insideBand(band, el) {
  return numeric(el, 'x') > numeric(band, 'x') && numeric(el, 'x') < rightEdgeOfBand(band) &&
    numeric(el, 'y') > numeric(band, 'y') && numeric(el, 'y') < bottomEdgeOfBand(band);
}

function textsOwnedBy(svg, attr, id) {
  return [...svg.querySelectorAll('text[' + attr + '="' + id + '"]')];
}

const ctxHeaderTexts = (svg, ctxId) => textsOwnedBy(svg, 'data-ctx-id', ctxId);
const aggRowTexts = (svg, aggId) => textsOwnedBy(svg, 'data-agg-id', aggId);
const sliceHeaderTexts = (svg, sliceId) => textsOwnedBy(svg, 'data-slice-id', sliceId);

const withoutMarks = (els) => els.filter((el) => !el.hasAttribute('data-marker'));
const sliceHeaderNames = (svg, sliceId) => withoutMarks(sliceHeaderTexts(svg, sliceId));

const textOf = (els) => els.map((el) => el.textContent);

// Layout.labelWidth pads its measurement for layout breathing room, which the
// drawn glyphs never reach into.
const LABEL_PADDING = 16;
const drawnWidth = (el) => Layout.labelWidth(el.textContent) - LABEL_PADDING;

function rightEdgeOf(el) {
  return numeric(el, 'x') + drawnWidth(el);
}

function nodeGroup(svg, nodeId) {
  return svg.querySelector('.diagram-node[data-node-id="' + nodeId + '"]');
}

// Reading the <text> elements keeps the badge drawn on the box apart from the
// tooltip: the group's textContent folds its <title> in with the drawn labels.
function drawnLabels(group) {
  return [...group.querySelectorAll('text')];
}

function drawnText(group) {
  return drawnLabels(group).map((el) => el.textContent);
}

function tooltipOf(group) {
  const title = group.querySelector('title');
  return title ? title.textContent : null;
}

function badgeIn(group) {
  return drawnLabels(group).find((el) => el.textContent.includes(clockMarking));
}

function marksIn(root, kind) {
  return [...root.querySelectorAll(kind ? '[data-marker="' + kind + '"]' : '[data-marker]')];
}

const markedNodeIds = (marks) => marks.map((el) => el.getAttribute('data-node-id'));

const marksWithOwners = (root) =>
  marksIn(root).map((el) => el.getAttribute('data-marker') + ' ' + el.getAttribute('data-node-id'));

// The drawn strings without the marks: a mark is a glyph standing for prose held
// elsewhere, so the leaves comparing two renders account for it on its own.
function proseIn(root) {
  return withoutMarks(drawnLabels(root));
}

const placements = (els) => els.map((el) => [el.textContent, numeric(el, 'x'), numeric(el, 'y')]);

// Everything inside a block is centred on its own x, so it runs half its
// measured width either side.
const leftEdgeOfCentred = (el) => numeric(el, 'x') - Layout.labelWidth(el.textContent) / 2;
const rightEdgeOfCentred = (el) => numeric(el, 'x') + Layout.labelWidth(el.textContent) / 2;

// What a centred string actually paints: its font size vertically and its drawn
// width horizontally, half of each either side of the point it is centred on.
function inkOfCentred(el) {
  const half = numeric(el, 'font-size') / 2;
  const reach = drawnWidth(el) / 2;
  return {
    left: numeric(el, 'x') - reach,
    right: numeric(el, 'x') + reach,
    top: numeric(el, 'y') - half,
    bottom: numeric(el, 'y') + half,
  };
}

// A stroked rect paints half its stroke outside the edge it is drawn on.
function inkOfRect(el) {
  const bleed = numeric(el, 'stroke-width') / 2;
  return {
    left: numeric(el, 'x') - bleed,
    right: rightEdgeOfBand(el) + bleed,
    top: numeric(el, 'y') - bleed,
    bottom: bottomEdgeOfBand(el) + bleed,
  };
}

function overlapArea(a, b) {
  const width = Math.min(a.right, b.right) - Math.max(a.left, b.left);
  const height = Math.min(a.bottom, b.bottom) - Math.max(a.top, b.top);
  return Math.max(0, width) * Math.max(0, height);
}

function cornerOf(box, mark) {
  const x = numeric(mark, 'x');
  const y = numeric(mark, 'y');
  if (x <= box.x || x >= box.x + box.w || y <= box.y || y >= box.y + box.h) return 'outside';
  const vertical = y < box.y + box.h / 2 ? 'top' : 'bottom';
  const horizontal = x > box.x + box.w / 2 ? 'right' : 'left';
  return vertical + '-' + horizontal;
}

function drawnBoxes(svg) {
  const entries = [...svg.querySelectorAll('.diagram-node')].map((group) => {
    const box = group.querySelector('rect');
    return [group.getAttribute('data-node-id'), {
      x: box.getAttribute('x'),
      y: box.getAttribute('y'),
      width: box.getAttribute('width'),
      height: box.getAttribute('height'),
      fill: box.getAttribute('fill'),
    }];
  });
  return Object.fromEntries(entries);
}

// Every rect the swimlane and its slices are built from, so a change that
// reflowed the diagram shows up as a moved or resized frame.
function drawnFrames(svg) {
  const selector = 'rect.ctx-header, rect.agg-row, rect.agg-area, rect.slice-box, rect.slice-header';
  return [...svg.querySelectorAll(selector)].map((el) => [
    el.getAttribute('class'),
    numeric(el, 'x'), numeric(el, 'y'), numeric(el, 'width'), numeric(el, 'height'),
  ]);
}

function namePlacements(svg) {
  return placements([...svg.querySelectorAll('.ctx-label, .agg-label')]);
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
      const { svg } = render(describedThroughout());

      const painted = [...svg.querySelectorAll('.agg-row, .agg-label, .agg-desc')]
        .map((el) => el.getAttribute('class'));
      expect(painted).toEqual([
        'agg-row', 'agg-row',
        'agg-label', 'agg-desc',
        'agg-label', 'agg-desc',
      ]);
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
      const store = createStore(twoAggregates(), { hiddenNodes: { agg1: true } });
      store.layoutPositions = Layout.computeLayout(store).positions;

      const svg = Renderer.buildSVG(store);

      expect(rowFor(svg, 'agg1')).toBeNull();
      expect(labelFor(svg, 'agg1')).toBeNull();
      expect(rowFor(svg, 'agg2')).not.toBeNull();
    });
  });

  describe('context header descriptions', () => {
    it('reads a description after the context name, leaving an undescribed header to its name alone', () => {
      const { svg } = render(describedAndPlainContexts());

      const described = ctxHeaderTexts(svg, 'ctxWide');
      expect(textOf(described)).toEqual(['Wide', contextDescription]);
      expect(numeric(described[1], 'x')).toBeGreaterThan(numeric(described[0], 'x'));

      expect(textOf(ctxHeaderTexts(svg, 'ctxNarrow'))).toEqual(['Narrow']);
    });

    it('keeps that description on the name\'s line inside the header band', () => {
      const { svg } = render(describedAndPlainContexts());

      const [name, description] = ctxHeaderTexts(svg, 'ctxWide');
      const header = headerFor(svg, 'ctxWide');
      expect(numeric(description, 'y')).toBe(numeric(name, 'y'));
      expect(numeric(description, 'y')).toBeGreaterThan(numeric(header, 'y'));
      expect(numeric(description, 'y')).toBeLessThanOrEqual(bottomEdgeOfBand(header));
    });

    it('cuts an over-long context description short of the swimlane edge while one that fits is drawn whole', () => {
      const { svg } = render(crowdedAndRoomyContexts());

      const [, crowded] = ctxHeaderTexts(svg, 'ctxWide');
      expect(crowded.textContent.length).toBeLessThan(overlongContextDescription.length);
      expect(crowded.textContent.endsWith(ellipsis)).toBe(true);
      expect(rightEdgeOf(crowded)).toBeLessThanOrEqual(rightEdgeOfBand(headerFor(svg, 'ctxWide')));

      const [, roomy] = ctxHeaderTexts(svg, 'ctxNarrow');
      expect(roomy.textContent).toBe(contextDescription);
      expect(roomy.textContent).not.toContain(ellipsis);
    });

    it('draws that description smaller and in another colour than the name it follows', () => {
      const { svg } = render(describedAndPlainContexts());

      const [name, description] = ctxHeaderTexts(svg, 'ctxWide');
      expect(numeric(description, 'font-size')).toBeLessThan(numeric(name, 'font-size'));
      expect(description.getAttribute('fill')).not.toBe(name.getAttribute('fill'));
    });
  });

  describe('aggregate row descriptions', () => {
    it('reads a description after the aggregate name inside its own row, leaving an undescribed row to its name alone', () => {
      const { svg } = render(describedAndPlainAggregates());

      const described = aggRowTexts(svg, 'agg1');
      expect(textOf(described)).toEqual(['Arrangement', aggregateDescription]);
      expect(numeric(described[1], 'x')).toBeGreaterThan(numeric(described[0], 'x'));

      const row = rowFor(svg, 'agg1');
      expect(numeric(described[1], 'x')).toBeGreaterThan(numeric(row, 'x'));
      expect(rightEdgeOf(described[1])).toBeLessThanOrEqual(rightEdgeOfBand(row));

      expect(textOf(aggRowTexts(svg, 'agg2'))).toEqual(['Payment']);
    });

    it('keeps that description on the name\'s line inside the row it belongs to', () => {
      const { svg } = render(describedAndPlainAggregates());

      const [name, description] = aggRowTexts(svg, 'agg1');
      const row = rowFor(svg, 'agg1');
      expect(numeric(description, 'y')).toBe(numeric(name, 'y'));
      expect(numeric(description, 'y')).toBeGreaterThan(numeric(row, 'y'));
      expect(numeric(description, 'y')).toBeLessThanOrEqual(bottomEdgeOfBand(row));
    });

    it('cuts an over-long description short of the next row while a description that fits is drawn whole', () => {
      const { svg } = render(crowdedAndRoomyDescriptions());

      const [, crowded] = aggRowTexts(svg, 'agg1');
      expect(crowded.textContent.length).toBeLessThan(overlongAggregateDescription.length);
      expect(crowded.textContent.endsWith(ellipsis)).toBe(true);
      expect(rightEdgeOf(crowded)).toBeLessThanOrEqual(numeric(rowFor(svg, 'agg2'), 'x'));

      const [, roomy] = aggRowTexts(svg, 'agg2');
      expect(roomy.textContent).toBe(neighbourAggregateDescription);
      expect(roomy.textContent).not.toContain(ellipsis);
    });

    it('drops the description entirely from a row its aggregate name already fills', () => {
      const { svg } = render(aggregateNameFillingItsRow());

      expect(rightEdgeOf(labelFor(svg, 'agg1')))
        .toBeGreaterThan(numeric(rowFor(svg, 'agg2'), 'x'));
      expect(textOf(aggRowTexts(svg, 'agg1'))).toEqual([crowdingAggregateName]);
    });
  });

  describe('headers documented and undocumented', () => {
    it('adds the prose without moving a single frame, name or block', () => {
      const plain = render(twoAggregates());
      const described = render(describedThroughout());

      expect(drawnFrames(described.svg)).toEqual(drawnFrames(plain.svg));
      expect(namePlacements(described.svg)).toEqual(namePlacements(plain.svg));
      expect(drawnBoxes(described.svg)).toEqual(drawnBoxes(plain.svg));

      const plainText = drawnText(plain.svg);
      const added = proseIn(described.svg)
        .map((el) => el.textContent)
        .filter((t) => !plainText.includes(t));
      expect(added).toEqual([
        describedThroughoutText.ctx1,
        describedThroughoutText.agg1,
        describedThroughoutText.agg2,
      ]);
    });

    it('lets the pointer through the prose to the header and row it is painted over', () => {
      const { svg } = render(describedThroughout());

      const prose = [...svg.querySelectorAll('.ctx-desc, .agg-desc')];
      expect(textOf(prose)).toEqual([
        describedThroughoutText.ctx1,
        describedThroughoutText.agg1,
        describedThroughoutText.agg2,
      ]);
      prose.forEach((el) => expect(el.getAttribute('pointer-events')).toBe('none'));

      // Each string really is drawn inside the band it is painted over. Without
      // this the rules below would be guarding prose that covers nothing, and
      // would go on passing if the prose stopped being drawn at all.
      expect(insideBand(headerFor(svg, 'ctx1'), prose[0])).toBe(true);
      expect(insideBand(rowFor(svg, 'agg1'), prose[1])).toBe(true);

      // The bands underneath answer the right-click and the highlighting click,
      // and the name is what a double-click renames — none of them may go deaf.
      const answering = [headerFor(svg, 'ctx1'), rowFor(svg, 'agg1'), labelFor(svg, 'agg1')];
      answering.forEach((el) => expect(el.getAttribute('pointer-events')).toBeNull());
    });

    it('keeps the slice name centred where it is and draws no prose in the slice header', () => {
      const plain = render(twoAggregates());
      const described = render(describedThroughout());

      expect(drawnText(described.svg)).toContain(describedThroughoutText.agg1);

      const [plainName] = sliceHeaderNames(plain.svg, 'sl1');
      const names = sliceHeaderNames(described.svg, 'sl1');
      expect(textOf(names)).toEqual(['Propose plan']);
      expect(names[0].getAttribute('text-anchor')).toBe('middle');
      expect(numeric(names[0], 'x')).toBe(numeric(plainName, 'x'));
      expect(numeric(names[0], 'y')).toBe(numeric(plainName, 'y'));
    });
  });

  describe('trigger screen framing', () => {
    it('draws a second rect inside the trigger box and no other node type', () => {
      const { svg } = render(allElementTypes());

      const trigger = nodeGroup(svg, 'trg1');
      const triggerRects = trigger.querySelectorAll('rect');
      expect(triggerRects.length).toBe(2);

      const main = triggerRects[0];
      const framing = triggerRects[1];
      const mainX = Number(main.getAttribute('x'));
      const mainY = Number(main.getAttribute('y'));
      const mainW = Number(main.getAttribute('width'));
      const mainH = Number(main.getAttribute('height'));

      expect(Number(framing.getAttribute('x'))).toBeGreaterThan(mainX);
      expect(Number(framing.getAttribute('y'))).toBeGreaterThan(mainY);
      expect(Number(framing.getAttribute('x')) + Number(framing.getAttribute('width')))
        .toBeLessThan(mainX + mainW);
      expect(Number(framing.getAttribute('y')) + Number(framing.getAttribute('height')))
        .toBeLessThan(mainY + mainH);

      for (const id of ['cmd1', 'evt1', 'view1', 'auto1', 'trans1']) {
        const group = nodeGroup(svg, id);
        expect(group.querySelectorAll('rect').length).toBe(1);
      }
    });

    it('keeps the main rect as the first rect so drawnBoxes still reports the box', () => {
      const { svg } = render(allElementTypes());

      const triggerGroup = nodeGroup(svg, 'trg1');
      const first = triggerGroup.querySelector('rect');
      expect(first.getAttribute('width')).not.toBe('0');
      expect(first.getAttribute('height')).not.toBe('0');

      const boxes = drawnBoxes(svg);
      expect(boxes.trg1.width).toBe(first.getAttribute('width'));
      expect(boxes.trg1.height).toBe(first.getAttribute('height'));
    });

    it('leaves the trigger label as the only text in the group', () => {
      const { svg } = render(allElementTypes());

      const trigger = nodeGroup(svg, 'trg1');
      expect(drawnText(trigger)).toEqual(['Form']);
    });

    it('is still the only node with a second rect after every fill is normalised', () => {
      const { svg } = render(allElementTypes());
      const svgEl = svg;
      svgEl.querySelectorAll('rect').forEach((rect) => {
        rect.setAttribute('fill', '#888888');
      });

      const trigger = nodeGroup(svgEl, 'trg1');
      expect(trigger.querySelectorAll('rect').length).toBe(2);

      for (const id of ['cmd1', 'evt1', 'view1', 'auto1', 'trans1']) {
        const group = nodeGroup(svgEl, id);
        expect(group.querySelectorAll('rect').length).toBe(1);
      }
    });
  });

  describe('automation cadence', () => {
    it('draws the clock badge and its tooltip on the scheduled automation only', () => {
      const { svg } = render(pairedAutomations(cronExpression));

      const scheduled = nodeGroup(svg, 'autoScheduled');
      expect(drawnText(scheduled)).toContain('SweepArrears');
      expect(drawnText(scheduled)).toContain(cadenceBadge);
      expect(tooltipOf(scheduled)).toBe(cronExpression);

      const eventActivated = nodeGroup(svg, 'autoOnEvent');
      expect(drawnText(eventActivated)).toEqual(['ChaseOverdue']);
      expect(tooltipOf(eventActivated)).toBeNull();
    });

    it('fits the cadence inside a box it neither moves, resizes nor repaints', () => {
      const eventActivated = render(pairedAutomations());
      const scheduled = render(pairedAutomations(cronExpression));

      const boxes = drawnBoxes(scheduled.svg);
      const box = boxes.autoScheduled;
      const badge = badgeIn(nodeGroup(scheduled.svg, 'autoScheduled'));

      expect(badge.textContent).toBe(cadenceBadge);
      expect(numeric(badge, 'y')).toBeGreaterThan(Number(box.y));
      expect(numeric(badge, 'y')).toBeLessThan(Number(box.y) + Number(box.height));

      expect(box.fill).toBe(automationFill);
      expect(box.height).toBe(boxes.cmd1.height);
      expect(boxes).toEqual(drawnBoxes(eventActivated.svg));
    });
  });

  describe('marks on documented constructs', () => {
    it.each([
      ['description', describedAndPlainCommands],
      ['comments', commentedAndPlainCommands],
    ])('marks the command carrying a %s and leaves the bare one beside it unmarked', (kind, documented) => {
      const { svg, positions } = render(documented());

      const marks = marksIn(svg, kind);
      expect(markedNodeIds(marks)).toEqual(['cmd1']);
      expect(cornerOf(positions.cmd1, marks[0])).toBe('top-right');
      expect(marksIn(nodeGroup(svg, 'cmd2'))).toEqual([]);
    });

    it('sets a comments mark beside the description mark, clear of it and of the label between them', () => {
      const { svg, positions } = render(commentedAs(describedAndPlainCommands(), { cmd1: commandComments }));

      const group = nodeGroup(svg, 'cmd1');
      const [description] = marksIn(group, 'description');
      const [comments] = marksIn(group, 'comments');
      const rows = proseIn(group);

      expect(comments.textContent).not.toBe(description.textContent);
      expect([description, comments].map((mark) => cornerOf(positions.cmd1, mark)))
        .toEqual(['top-right', 'top-right']);
      expect(rightEdgeOfCentred(comments)).toBeLessThan(leftEdgeOfCentred(description));

      expect(textOf(rows)).toEqual(['ProposePlan']);
      expect(inkOfCentred(comments).left).toBeGreaterThan(inkOfCentred(rows[0]).right);
    });

    it('closes both marks into the corner of a block a row has left a fraction of a stacked one', () => {
      const { svg, positions } = render(documentedEventInARow());

      const box = positions.evt1;
      expect(box.w).toBeLessThan(positions.cmd1.w / 2);

      const group = nodeGroup(svg, 'evt1');
      const marks = marksIn(group);
      const [label] = proseIn(group);
      expect(marks.length).toBe(2);

      marks.forEach((mark) => {
        expect(cornerOf(box, mark)).toBe('top-right');
        expect(inkOfCentred(mark).left).toBeGreaterThan(box.x + box.w / 2);
        expect(inkOfCentred(mark).right).toBeLessThan(box.x + box.w);
        expect(overlapArea(inkOfCentred(mark), inkOfCentred(label))).toBe(0);
      });
      expect(inkOfCentred(marks[1]).right).toBeLessThan(inkOfCentred(marks[0]).left);
    });

    it.each([
      { box: 'an automation stating a cadence', id: 'auto1' },
      { box: 'a translation naming an external system', id: 'trans1' },
    ])('keeps every mark on $box off the rows that box already draws', ({ id }) => {
      const { svg } = render(documentedProcessors());

      const group = nodeGroup(svg, id);
      const marks = marksIn(group);
      const rows = proseIn(group);

      expect(marks.length).toBe(2);
      expect(rows.length).toBe(2);
      marks.forEach((mark) => rows.forEach((row) => {
        expect(inkOfCentred(mark).left).toBeGreaterThan(inkOfCentred(row).right);
      }));
    });

    it('keeps both marks on a trigger in the corner and off the screen bar drawn across its top', () => {
      const { svg, positions } = render(documentedTrigger());

      const group = nodeGroup(svg, 'trg1');
      const marks = marksIn(group);
      const screenBar = group.querySelectorAll('rect')[1];

      expect(marks.length).toBe(2);
      marks.forEach((mark) => {
        expect(overlapArea(inkOfCentred(mark), inkOfRect(screenBar))).toBe(0);
        expect(cornerOf(positions.trg1, mark)).toBe('top-right');
      });
    });

    it.each([
      ['description', () => describedAs(twoAggregates(), { sl1: sliceDescription })],
      ['comments', () => commentedAs(twoAggregates(), { sl1: sliceComments })],
    ])('marks the slice carrying a %s inside its header, leaving both slice names centred where they were', (kind, documenting) => {
      const plain = render(twoAggregates());
      const documented = render(documenting());

      const marks = marksIn(documented.svg, kind);
      expect(markedNodeIds(marks)).toEqual(['sl1']);
      expect(marksIn(documented.svg).length).toBe(1);

      const header = sliceHeaderFor(documented.svg, 'sl1');
      expect(numeric(marks[0], 'x')).toBeGreaterThan(numeric(header, 'x'));
      expect(numeric(marks[0], 'x')).toBeLessThan(rightEdgeOfBand(header));
      expect(numeric(marks[0], 'y')).toBeGreaterThan(numeric(header, 'y'));
      expect(numeric(marks[0], 'y')).toBeLessThan(bottomEdgeOfBand(header));

      ['sl1', 'sl2'].forEach((sliceId) => {
        const names = sliceHeaderNames(documented.svg, sliceId);
        const [plainName] = sliceHeaderNames(plain.svg, sliceId);
        expect(textOf(names)).toEqual([plainName.textContent]);
        expect(names[0].getAttribute('text-anchor')).toBe('middle');
        expect(numeric(names[0], 'x')).toBe(numeric(plainName, 'x'));
        expect(numeric(names[0], 'y')).toBe(numeric(plainName, 'y'));
      });
    });

    it('adds the marks without moving a single frame, box or label', () => {
      const plain = render(twoAggregates());
      const documented = render(commentedAs(
        describedAs(twoAggregates(), { cmd1: commandDescription, sl1: sliceDescription }),
        { cmd1: commandComments, sl1: sliceComments },
      ));

      expect(drawnFrames(documented.svg)).toEqual(drawnFrames(plain.svg));
      expect(drawnBoxes(documented.svg)).toEqual(drawnBoxes(plain.svg));
      expect(placements(proseIn(documented.svg))).toEqual(placements(proseIn(plain.svg)));

      expect(marksWithOwners(documented.svg).sort()).toEqual([
        'comments cmd1', 'comments sl1', 'description cmd1', 'description sl1',
      ]);
      expect(marksIn(plain.svg)).toEqual([]);
    });

    it('gives no mark a native tooltip of its own, and leaves the cadence with the one it had', () => {
      const { svg } = render(commentedAs(
        describedAs(pairedAutomations(cronExpression), {
          autoScheduled: automationDescription,
          sl1: sliceDescription,
        }),
        { autoScheduled: automationComments, sl1: sliceComments },
      ));

      const marks = marksIn(svg);
      expect(marksWithOwners(svg).sort()).toEqual([
        'comments autoScheduled', 'comments sl1', 'description autoScheduled', 'description sl1',
      ]);
      marks.forEach((mark) => expect(mark.querySelector('title')).toBeNull());

      expect(tooltipOf(nodeGroup(svg, 'autoScheduled'))).toBe(cronExpression);
    });
  });

  describe('marks on documented containers', () => {
    it.each([
      {
        band: 'a swimlane header',
        fixture: commentedAndDescribedContexts,
        commented: 'ctxWide',
        bandFor: headerFor,
        textsOf: ctxHeaderTexts,
        unmarked: { id: 'ctxNarrow', reads: ['Narrow', contextDescription] },
      },
      {
        band: 'an aggregate label row',
        fixture: commentedAndDescribedAggregates,
        commented: 'agg1',
        bandFor: rowFor,
        textsOf: aggRowTexts,
        unmarked: { id: 'agg2', reads: ['Payment', neighbourAggregateDescription] },
      },
    ])(
      'marks $band whose container carries comments and leaves the described one beside it unmarked',
      ({ fixture, commented, bandFor, textsOf, unmarked }) => {
        const { svg } = render(fixture());

        expect(marksWithOwners(svg)).toEqual(['comments ' + commented]);
        expect(insideBand(bandFor(svg, commented), marksIn(svg)[0])).toBe(true);

        expect(textOf(withoutMarks(textsOf(svg, unmarked.id)))).toEqual(unmarked.reads);
        expect(marksIn(svg).filter((mark) => insideBand(bandFor(svg, unmarked.id), mark))).toEqual([]);
      });

    it('sets the aggregate mark clear of its own row\'s text and short of the row next door', () => {
      const { svg } = render(commentedAndDescribedAggregates());

      const [name, description] = withoutMarks(aggRowTexts(svg, 'agg1'));
      const [mark] = marksIn(svg, 'comments');

      expect(textOf([name, description])).toEqual(['Arrangement', aggregateDescription]);
      expect(rightEdgeOf(name)).toBeLessThan(inkOfCentred(mark).left);
      expect(rightEdgeOf(description)).toBeLessThanOrEqual(inkOfCentred(mark).left);
      expect(inkOfCentred(mark).right).toBeLessThanOrEqual(numeric(rowFor(svg, 'agg2'), 'x'));
    });

    it.each([
      {
        header: 'a swimlane header',
        fixture: crowdedCommentedContexts,
        commented: 'ctxWide',
        textsOf: ctxHeaderTexts,
        overlong: overlongContextDescription,
        roomy: { id: 'ctxNarrow', description: contextDescription },
      },
      {
        header: 'an aggregate label row',
        fixture: crowdedCommentedAggregates,
        commented: 'agg1',
        textsOf: aggRowTexts,
        overlong: overlongAggregateDescription,
        roomy: { id: 'agg2', description: neighbourAggregateDescription },
      },
    ])('cuts a description too wide for $header short of the mark, drawing an uncommented roomy one whole',
      ({ fixture, commented, textsOf, overlong, roomy }) => {
        const { svg } = render(fixture());

        const [, crowded] = withoutMarks(textsOf(svg, commented));
        const [mark] = marksIn(svg, 'comments');
        expect(crowded.textContent.length).toBeLessThan(overlong.length);
        expect(crowded.textContent.endsWith(ellipsis)).toBe(true);
        expect(rightEdgeOf(crowded)).toBeLessThanOrEqual(inkOfCentred(mark).left);

        const [, whole] = withoutMarks(textsOf(svg, roomy.id));
        expect(whole.textContent).toBe(roomy.description);
        expect(whole.textContent).not.toContain(ellipsis);
      });

    it('cuts a name short of the mark standing in its row and leaves the unmarked name beside it whole', () => {
      const { svg } = render(crowdingNamesOneCommented());

      const marked = labelFor(svg, 'agg1');
      const [mark] = marksIn(svg, 'comments');
      expect(marked.textContent.length).toBeLessThan(crowdingAggregateName.length);
      expect(marked.textContent.endsWith(ellipsis)).toBe(true);
      expect(rightEdgeOf(marked)).toBeLessThanOrEqual(inkOfCentred(mark).left);

      // The mark is the only thing that cuts a name: the row beside it carries
      // none, and its name is left to reach past its own edge.
      const unmarked = labelFor(svg, 'agg2');
      expect(unmarked.textContent).toBe(neighbourCrowdingName);
      expect(rightEdgeOf(unmarked)).toBeGreaterThan(numeric(rowFor(svg, 'agg3'), 'x'));
    });

    it('adds the header marks without moving a single frame, name, description or block', () => {
      const plain = render(describedThroughout());
      const commented = render(commentedAs(describedThroughout(), {
        ctx1: contextComments,
        agg1: aggregateComments,
        agg2: aggregateComments,
      }));

      expect(drawnFrames(commented.svg)).toEqual(drawnFrames(plain.svg));
      expect(namePlacements(commented.svg)).toEqual(namePlacements(plain.svg));
      expect(drawnBoxes(commented.svg)).toEqual(drawnBoxes(plain.svg));
      expect(placements(proseIn(commented.svg))).toEqual(placements(proseIn(plain.svg)));

      expect(marksWithOwners(commented.svg).sort()).toEqual([
        'comments agg1', 'comments agg2', 'comments ctx1', 'description sl1',
      ]);
      expect(marksWithOwners(plain.svg)).toEqual(['description sl1']);
    });
  });

  describe('read arrows', () => {
    it.each([
      { reader: 'trigger', id: 'trg1' },
      { reader: 'automation', id: 'auto1' },
    ])('draws the arrow onto the $reader the way it draws the one onto the translation beside it', ({ id }) => {
      const { svg } = render(viewReadThreeWays(), readsEdgesOutOfView());

      const translation = arrowBetween(svg, 'view1', 'trans1');
      const reader = arrowBetween(svg, 'view1', id);

      const readerAppearance = appearanceOf(reader);
      expect(readerAppearance).toEqual(appearanceOf(translation));
      expect(readerAppearance.dash).toBeNull();
      expect(reader.getAttribute('d')).not.toBe(translation.getAttribute('d'));
    });

    it('draws an arrow only where an edge joins the two blocks, not where a block names the view it reads', () => {
      const nodes = viewReadThreeWays();

      const connected = render(nodes, readsEdgesOutOfView());
      const unconnected = render(nodes, []);

      expect(drawnArrowEnds(connected.svg)).toEqual(['view1>trg1', 'view1>auto1', 'view1>trans1']);
      expect(drawnArrowEnds(unconnected.svg)).toEqual([]);
    });
  });

  describe('connection ports', () => {
    it('offers a port on every side of every block, each naming the block it starts from', () => {
      const { svg } = render(twoAggregates());

      [...svg.querySelectorAll('.diagram-node')].forEach((group) => {
        const ports = [...group.querySelectorAll('[data-port]')];
        expect(ports.map((p) => p.getAttribute('data-port')))
          .toEqual(['top', 'right', 'bottom', 'left']);
        ports.forEach((port) => {
          expect(port.getAttribute('data-node-id')).toBe(group.getAttribute('data-node-id'));
        });
      });
    });

    it.each(['top', 'right', 'bottom', 'left'])(
      'draws the %s port as an arrowhead aimed away from the side it sits on', (direction) => {
        const { svg, positions } = render(twoAggregates());
        const side = SIDES[direction](positions.cmd1);

        const corners = arrowheadCorners(portOf(nodeGroup(svg, 'cmd1'), direction));
        const reaches = corners.map((c) => reachBeyondEdge(c, side));

        // The tip is the corner furthest out, and the two behind it sit either
        // side of the centre line — an arrowhead rather than a blunt tab.
        const [left, tip, right] = corners;
        expect(reaches[1]).toBeGreaterThan(Math.max(reaches[0], reaches[2]));
        expect(offsetAlongEdge(tip, side)).toBeCloseTo(0);
        expect(offsetAlongEdge(left, side)).toBeCloseTo(offsetAlongEdge(right, side));
        expect(offsetAlongEdge(left, side)).toBeGreaterThan(0);
        reaches.forEach((reach) => expect(reach).toBeGreaterThan(0));
      });

    it.each(['top', 'right', 'bottom', 'left'])(
      'gives the %s port a hit area around the arrowhead, not over the block it belongs to', (direction) => {
        const { svg, positions } = render(twoAggregates());
        const side = SIDES[direction](positions.cmd1);

        const port = portOf(nodeGroup(svg, 'cmd1'), direction);
        const hit = hitAreaOf(port);
        const tip = arrowheadCorners(port)[1];

        // Wider than the 5px dot it replaces, so the port can be grabbed
        // without landing on the arrowhead exactly.
        expect(hit.radius).toBeGreaterThan(5);
        expect(offsetAlongEdge(hit.centre, side)).toBeCloseTo(0);
        expect(reachBeyondEdge(hit.centre, side)).toBeGreaterThan(0);

        // Dragging a block starts anywhere on it, so the hit area has to stay
        // out in the gap: it may graze the edge but never reach the label.
        expect(reachBeyondEdge(hit.centre, side) - hit.radius).toBeGreaterThan(-2);
        expect(reachBeyondEdge(hit.centre, side) + hit.radius)
          .toBeGreaterThan(reachBeyondEdge(tip, side));
      });
  });

  describe('palette table', () => {
    // Per type, not as a set: comparing the collected fills against the palette's
    // fills says nothing about which type wore which, so the six could be
    // permuted and still match. Stroke is read here for the first time — a block
    // drawn in a neighbour's outline was invisible to every other leaf.
    it('paints each node type in its own palette fill and stroke', () => {
      const { svg } = render(allElementTypes());

      const painted = {};
      svg.querySelectorAll('.diagram-node').forEach((group) => {
        const type = group.getAttribute('class').split(' ')[0].replace(/-block$/, '');
        const rect = group.querySelector('rect');
        painted[type] = {
          fill: rect.getAttribute('fill'),
          stroke: rect.getAttribute('stroke'),
        };
      });

      const expected = {};
      for (const [type, palette] of Object.entries(nodePalette)) {
        expected[type] = { fill: palette.fill, stroke: palette.stroke };
      }
      expect(painted).toEqual(expected);
    });

    it('uses no node fill or stroke literals outside the palette table in renderer.js', () => {
      const rendererSource = readFileSync(resolve(__dirname, '../static/renderer.js'), 'utf-8');
      // buildBlock is the only place that picks a fill or stroke by node type;
      // it must read from nodePalette instead of spelling a hex. The window
      // stops at the next top-level declaration because container chrome
      // carries its own colours, which are not palette values.
      const fromBuildBlock = rendererSource.slice(rendererSource.indexOf('function buildBlock('));
      const buildBlockBody = fromBuildBlock.slice(0, fromBuildBlock.indexOf('\nfunction '));
      // A window that missed the palette lookup would pass however many hexes
      // buildBlock spelled, so prove the slice caught the code being guarded.
      expect(buildBlockBody).toContain('palette.fill');

      const hex = /#([0-9a-fA-F]{6})/g;
      const literals = [...buildBlockBody.matchAll(hex)].map((m) => m[0].toLowerCase());
      const paletteValues = new Set(Object.values(nodePalette).flatMap((p) => [
        p.fill, p.stroke, p.hoverFill, p.highlightFill,
      ]));
      for (const literal of literals) {
        expect(paletteValues).toContain(literal);
      }
    });

    it('declares hover fills in the stylesheet that match the palette', () => {
      for (const [type, palette] of Object.entries(nodePalette)) {
        expect(cssFill(`#diagram-canvas .${classForType[type]}-block:hover rect`)).toBe(palette.hoverFill);
      }
    });

    it('declares highlight fills in the stylesheet that match the palette', () => {
      for (const [type, palette] of Object.entries(nodePalette)) {
        expect(cssFill(`#diagram-canvas .hl.${classForType[type]}-block rect`)).toBe(palette.highlightFill);
      }
    });

    it('keeps hover and highlight fills pairwise distinct', () => {
      const hoverFills = Object.values(nodePalette).map((p) => p.hoverFill);
      const highlightFills = Object.values(nodePalette).map((p) => p.highlightFill);
      expect(new Set(hoverFills).size).toBe(hoverFills.length);
      expect(new Set(highlightFills).size).toBe(highlightFills.length);
    });
  });
});

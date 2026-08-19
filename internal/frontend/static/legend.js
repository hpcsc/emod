import { nodePalette, edgeConfig } from './config.js';

const NS = "http://www.w3.org/2000/svg";

// The vocabulary a reader knows the diagram by, keyed on the type the renderer
// draws. A type missing here falls back to its own key, so a new element or
// connection still reaches the legend — unnamed rather than undrawn.
const ELEMENT_NAMES = {
  trigger: 'Trigger',
  command: 'Command',
  event: 'Event',
  view: 'View',
  automation: 'Automation',
  translation: 'Translation',
};

const CONNECTION_NAMES = {
  flow: 'Command produces event',
  subscription: 'Event updates view',
  automation_trigger: 'Event wakes automation',
  automation_command: 'Automation issues command',
  trigger_command: 'Trigger sends command',
  reads: 'View read by a trigger or automation',
  translation_command: 'Translation issues command',
};

function createEl(tag, cls) {
  const node = document.createElement(tag);
  if (cls) node.className = cls;
  return node;
}

function createSvgEl(tag, attrs) {
  const node = document.createElementNS(NS, tag);
  Object.keys(attrs).forEach(function(name) {
    node.setAttribute(name, attrs[name]);
  });
  return node;
}

function buildElementSwatch(palette) {
  const swatch = createSvgEl('svg', { class: 'lg-swatch', width: 40, height: 18 });
  swatch.appendChild(createSvgEl('rect', {
    x: 1, y: 1, width: 28, height: 16, rx: 3,
    fill: palette.fill, stroke: palette.stroke, 'stroke-width': 1.5,
  }));
  return swatch;
}

// The arrowhead is the canvas's own marker, referenced by the same id: the defs
// live once in the page, so a sample cannot drift from the arrow it stands for.
function buildConnectionSwatch(config) {
  const swatch = createSvgEl('svg', { class: 'lg-swatch', width: 40, height: 18 });
  const attrs = {
    x1: 2, y1: 9, x2: 28, y2: 9,
    stroke: config.stroke, 'stroke-width': 1.5, 'marker-end': config.marker,
  };
  if (config.dash) attrs['stroke-dasharray'] = config.dash;
  swatch.appendChild(createSvgEl('line', attrs));
  return swatch;
}

function buildRow(swatch, name) {
  const row = createEl('div', 'lg-row');
  row.appendChild(swatch);
  const label = createEl('span', 'lg-name');
  label.textContent = name;
  row.appendChild(label);
  return row;
}

function buildSection(title, rows) {
  const section = createEl('div', 'lg-section');
  const heading = createEl('div', 'lg-section-title');
  heading.textContent = title;
  section.appendChild(heading);
  rows.forEach(function(row) { section.appendChild(row); });
  return section;
}

function renderLegend(container) {
  if (!container) return;
  container.textContent = '';

  const elements = Object.keys(nodePalette).map(function(type) {
    return buildRow(buildElementSwatch(nodePalette[type]), ELEMENT_NAMES[type] || type);
  });
  const connections = Object.keys(edgeConfig).map(function(type) {
    return buildRow(buildConnectionSwatch(edgeConfig[type]), CONNECTION_NAMES[type] || type);
  });

  container.appendChild(buildSection('Elements', elements));
  container.appendChild(buildSection('Connections', connections));
}

function toggleLegendPanel(store, show) {
  const panelEl = store.dom.legendPanel;
  if (!panelEl) return;

  if (show === undefined) {
    panelEl.classList.toggle("hidden");
  } else if (show) {
    panelEl.classList.remove("hidden");
  } else {
    panelEl.classList.add("hidden");
  }

  const isHidden = panelEl.classList.contains("hidden");
  if (store.dom.legendToggle) {
    store.dom.legendToggle.classList.toggle("active", !isHidden);
  }
  if (!isHidden) renderLegend(store.dom.legendContent);
}

export const Legend = {
  renderLegend,
  toggleLegendPanel,
};

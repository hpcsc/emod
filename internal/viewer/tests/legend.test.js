import { describe, it, expect, beforeEach } from 'vitest';
import { readFileSync } from 'fs';
import { resolve } from 'path';

const { Legend } = await import('../static/legend.js');
const { nodePalette, edgeConfig } = await import('../static/config.js');

const viewerHtml = readFileSync(resolve(__dirname, '../static/viewer.html'), 'utf-8');

function createStore() {
  document.body.innerHTML =
    '<button id="legend-toggle">Legend</button>' +
    '<div id="legend-panel" class="hidden"><div id="legend-content"></div></div>';
  return {
    dom: {
      legendPanel: document.getElementById('legend-panel'),
      legendToggle: document.getElementById('legend-toggle'),
      legendContent: document.getElementById('legend-content'),
    },
  };
}

function sectionTitles(root) {
  return [...root.querySelectorAll('.lg-section-title')].map((el) => el.textContent);
}

function rowsUnder(root, title) {
  const section = [...root.querySelectorAll('.lg-section')]
    .find((s) => s.querySelector('.lg-section-title').textContent === title);
  return [...section.querySelectorAll('.lg-row')];
}

function namesUnder(root, title) {
  return rowsUnder(root, title).map((row) => row.querySelector('.lg-name').textContent);
}

function swatchesUnder(root, title) {
  return rowsUnder(root, title).map((row) => row.querySelector('.lg-swatch'));
}

describe('Legend', () => {
  let store;

  beforeEach(() => {
    store = createStore();
  });

  describe('opening and closing', () => {
    it('starts hidden and opens on the first toggle', () => {
      expect(store.dom.legendPanel.classList.contains('hidden')).toBe(true);

      Legend.toggleLegendPanel(store);

      expect(store.dom.legendPanel.classList.contains('hidden')).toBe(false);
      expect(store.dom.legendToggle.classList.contains('active')).toBe(true);
    });

    it('closes again on the next toggle, and on an explicit close', () => {
      Legend.toggleLegendPanel(store);
      Legend.toggleLegendPanel(store);
      expect(store.dom.legendPanel.classList.contains('hidden')).toBe(true);
      expect(store.dom.legendToggle.classList.contains('active')).toBe(false);

      Legend.toggleLegendPanel(store, true);
      Legend.toggleLegendPanel(store, false);
      expect(store.dom.legendPanel.classList.contains('hidden')).toBe(true);
    });

    it('fills the panel only once it is open', () => {
      expect(store.dom.legendContent.children.length).toBe(0);

      Legend.toggleLegendPanel(store);

      expect(sectionTitles(store.dom.legendContent)).toEqual(['Elements', 'Connections']);
    });

    it('leaves one set of sections behind however many times it is reopened', () => {
      Legend.toggleLegendPanel(store, true);
      Legend.toggleLegendPanel(store, false);
      Legend.toggleLegendPanel(store, true);

      expect(sectionTitles(store.dom.legendContent)).toEqual(['Elements', 'Connections']);
    });
  });

  describe('elements', () => {
    it('names every element type the renderer can paint', () => {
      Legend.renderLegend(store.dom.legendContent);

      expect(namesUnder(store.dom.legendContent, 'Elements')).toEqual([
        'Trigger', 'Command', 'Event', 'View', 'Automation', 'Translation',
      ]);
    });

    it('draws each swatch in that type\'s own palette fill and stroke', () => {
      Legend.renderLegend(store.dom.legendContent);

      const drawn = swatchesUnder(store.dom.legendContent, 'Elements').map((swatch) => {
        const rect = swatch.querySelector('rect');
        return [rect.getAttribute('fill'), rect.getAttribute('stroke')];
      });

      expect(drawn).toEqual(
        Object.values(nodePalette).map((p) => [p.fill, p.stroke]),
      );
    });

    it('gives the six element types six distinct fills, as the canvas does', () => {
      Legend.renderLegend(store.dom.legendContent);

      const fills = swatchesUnder(store.dom.legendContent, 'Elements')
        .map((swatch) => swatch.querySelector('rect').getAttribute('fill'));

      expect(new Set(fills).size).toBe(fills.length);
    });
  });

  describe('connections', () => {
    it('describes every connection type as a relationship rather than its key', () => {
      Legend.renderLegend(store.dom.legendContent);

      const names = namesUnder(store.dom.legendContent, 'Connections');
      expect(names).toEqual([
        'Command produces event',
        'Event updates view',
        'Event wakes automation',
        'Automation issues command',
        'Trigger sends command',
        'View read by a trigger or automation',
        'Translation issues command',
      ]);
      Object.keys(edgeConfig).forEach((key) => expect(names).not.toContain(key));
    });

    it('draws each sample with that type\'s own stroke, dash and arrowhead', () => {
      Legend.renderLegend(store.dom.legendContent);

      const drawn = swatchesUnder(store.dom.legendContent, 'Connections').map((swatch) => {
        const line = swatch.querySelector('line');
        return {
          stroke: line.getAttribute('stroke'),
          dash: line.getAttribute('stroke-dasharray') || '',
          marker: line.getAttribute('marker-end'),
        };
      });

      expect(drawn).toEqual(
        Object.values(edgeConfig).map((c) => ({ stroke: c.stroke, dash: c.dash, marker: c.marker })),
      );
    });

    // Subscription and automation-trigger are both dashed and differ only in
    // colour, which is the pair a reader cannot tell apart without the legend.
    it('separates the two dashed connections by colour', () => {
      Legend.renderLegend(store.dom.legendContent);

      const rows = rowsUnder(store.dom.legendContent, 'Connections');
      const byName = {};
      rows.forEach((row) => {
        byName[row.querySelector('.lg-name').textContent] = row.querySelector('line');
      });

      const subscription = byName['Event updates view'];
      const automation = byName['Event wakes automation'];
      expect(subscription.getAttribute('stroke-dasharray'))
        .toBe(automation.getAttribute('stroke-dasharray'));
      expect(subscription.getAttribute('stroke'))
        .not.toBe(automation.getAttribute('stroke'));
    });
  });

  describe('staying in step with the palette', () => {
    // The swatches must be read from the palette at render time. Restating a
    // colour in markup or CSS would pass every leaf above while drifting the
    // first time an element type is added or recoloured.
    it('spells no palette colour in legend.js or the stylesheet', () => {
      const legendSource = readFileSync(resolve(__dirname, '../static/legend.js'), 'utf-8');
      const styleMatch = viewerHtml.match(/<style[^>]*>([\s\S]*?)<\/style>/i);
      expect(styleMatch).not.toBeNull();
      const legendCss = styleMatch[1].match(/#legend-[\s\S]*?(?=\n#visibility-panel-header)/);
      expect(legendCss).not.toBeNull();

      const palette = new Set(
        Object.values(nodePalette)
          .flatMap((p) => [p.fill, p.stroke, p.hoverFill, p.highlightFill])
          .concat(Object.values(edgeConfig).map((c) => c.stroke))
          .map((c) => c.toLowerCase()),
      );

      const hexes = (text) => [...text.matchAll(/#([0-9a-fA-F]{6})/g)].map((m) => m[0].toLowerCase());
      // Proves the two regions really were read, so a miss cannot pass as clean.
      expect(legendSource.length).toBeGreaterThan(0);
      expect(legendCss[0]).toContain('#legend-panel');

      hexes(legendSource).forEach((hex) => expect(palette).not.toContain(hex));
      hexes(legendCss[0]).forEach((hex) => expect(palette).not.toContain(hex));
    });

    it('carries one row per palette entry, so a new type reaches the legend unedited', () => {
      Legend.renderLegend(store.dom.legendContent);

      expect(rowsUnder(store.dom.legendContent, 'Elements'))
        .toHaveLength(Object.keys(nodePalette).length);
      expect(rowsUnder(store.dom.legendContent, 'Connections'))
        .toHaveLength(Object.keys(edgeConfig).length);
    });
  });
});

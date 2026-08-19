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

// The declaration block for a selector, or null when the stylesheet has no such
// rule. The trailing `{` keeps `#legend-toggle` from matching `:hover`.
function cssRule(selector) {
  const styleMatch = viewerHtml.match(/<style[^>]*>([\s\S]*?)<\/style>/i);
  expect(styleMatch).not.toBeNull();
  const escaped = selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const found = styleMatch[1].match(new RegExp(escaped + '\\s*\\{([^}]*)\\}'));
  return found ? found[1] : null;
}

function declarationsOf(block) {
  return block.split(';').map((d) => d.trim()).filter(Boolean).sort();
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

  describe('looking like the rest of the viewer', () => {
    // Every toolbar button is styled by its own id rule — there is no shared
    // selector to inherit from — so a missing rule leaves a default browser
    // button sitting among the others.
    it('styles the toolbar button exactly as the sibling panel toggle', () => {
      const legend = cssRule('#legend-toggle');
      const visibility = cssRule('#visibility-toggle');
      expect(legend).not.toBeNull();
      expect(visibility).not.toBeNull();
      expect(declarationsOf(legend)).toEqual(declarationsOf(visibility));
    });

    it('marks the button active while the panel is open, as the sibling does', () => {
      expect(declarationsOf(cssRule('#legend-toggle.active') || ''))
        .toEqual(declarationsOf(cssRule('#visibility-toggle.active')));
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
      // Gathered by selector rather than by slicing between two landmarks: the
      // legend's rules sit in more than one place in the sheet, and a slice
      // would swallow whichever rules happened to fall between them.
      const legendCss = [...styleMatch[1].matchAll(/#legend[^{}]*\{[^}]*\}/g)].map((m) => m[0]);
      expect(legendCss.length).toBeGreaterThan(0);

      const palette = new Set(
        Object.values(nodePalette)
          .flatMap((p) => [p.fill, p.stroke, p.hoverFill, p.highlightFill])
          .concat(Object.values(edgeConfig).map((c) => c.stroke))
          .map((c) => c.toLowerCase()),
      );

      const hexes = (text) => [...text.matchAll(/#([0-9a-fA-F]{6})/g)].map((m) => m[0].toLowerCase());
      // Proves both regions really were read, so a miss cannot pass as clean.
      expect(legendSource).toContain('buildElementSwatch');
      expect(legendCss.join('\n')).toContain('#legend-panel');

      hexes(legendSource).forEach((hex) => expect(palette).not.toContain(hex));
      hexes(legendCss.join('\n')).forEach((hex) => expect(palette).not.toContain(hex));
    });

    it('sizes each swatch itself, because the canvas stretches every svg inside it', () => {
      // Stating the hazard first: without this the swatch rule below would be
      // guarding a stretch that no longer happens, and would pass unmaintained.
      expect(cssRule('#canvas-container svg')).toMatch(/width:\s*100%/);

      const swatch = cssRule('#legend-content .lg-swatch');
      expect(swatch).not.toBeNull();
      expect(swatch).toMatch(/width:\s*\d+px/);
      expect(swatch).toMatch(/height:\s*\d+px/);
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

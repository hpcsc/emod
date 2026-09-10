import { test, expect } from '@playwright/test';
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { open, render, SAMPLE, SAMPLE_WITH_VIEW } from './helpers.js';
import { REVALIDATE_PAUSE_MS } from '../../internal/frontend/static/config.js';

// A context left open is the smallest edit that stops the source parsing, and
// the parser recovers only what comes before it.
const UNPARSEABLE = SAMPLE + 'context "Refunds" {\n';

// landsOn asks the browser what a click at an element's centre would reach,
// which is what a user reading or pressing it depends on: a jsdom assertion
// cannot see an overlay drawn over it.
async function landsOn(page, selector) {
  return page.evaluate((sel) => {
    const target = document.querySelector(sel);
    const box = target.getBoundingClientRect();
    const hit = document.elementFromPoint(box.left + box.width / 2, box.top + box.height / 2);
    return box.width > 0 && box.height > 0 && target.contains(hit);
  }, selector);
}

test.describe('a diagram the source panel no longer parses into', () => {
  test('stays on screen with every node, marked out of date by the real pipeline', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const nodes = await page.locator('.diagram-node').count();
    await page.locator('#data-panel-header').click();

    await page.locator('#source-input').fill(UNPARSEABLE);
    await page.locator('#render-btn').click();

    await expect(page.locator('#stale-notice')).toHaveText('Diagram out of date — the source does not parse');
    await expect(page.locator('.diagram-node')).toHaveCount(nodes);
    await expect(page.locator('.diagram-node', { hasText: 'TakePayment' })).toBeVisible();
  });

  test('can be read with the data panel open or collapsed, and with every panel the header opens', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    await page.locator('#data-panel-header').click();
    await page.locator('#source-input').fill(UNPARSEABLE);
    await page.locator('#render-btn').click();
    await expect(page.locator('#stale-notice')).toBeVisible();

    await expect(page.locator('#data-panel')).not.toHaveClass(/collapsed/);
    expect(await landsOn(page, '#stale-notice')).toBe(true);

    await page.locator('#data-panel-header').click();
    await expect(page.locator('#data-panel')).toHaveClass(/collapsed/);
    expect(await landsOn(page, '#stale-notice')).toBe(true);

    await page.locator('#visibility-toggle').click();
    await page.locator('#legend-toggle').click();
    await expect(page.locator('#visibility-panel')).toBeVisible();
    await expect(page.locator('#legend-panel')).toBeVisible();
    await expect(page.locator('#minimap')).toBeVisible();
    await expect(page.locator('#diagnostics-panel')).toBeVisible();
    expect(await landsOn(page, '#stale-notice')).toBe(true);
  });
});

function largestExample() {
  const dir = new URL('../../examples/', import.meta.url);
  const [largest] = readdirSync(dir)
    .filter((name) => name.endsWith('.emod') && !name.endsWith('_test.emod'))
    .map((name) => ({ name, size: statSync(new URL(name, dir)).size }))
    .sort((a, b) => b.size - a.size);
  return readFileSync(new URL(largest.name, dir), 'utf8');
}

const PARSES_WITH_UNDECLARED_EVENT = SAMPLE.replace('TakePayment -> PaymentTaken', 'TakePayment -> PaymentRefunded');

test.describe('the source panel revalidating as it is edited', () => {
  test('redraws what was typed once typing pauses, with no Render click', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    await page.locator('#data-panel-header').click();

    await page.locator('#source-input').fill(SAMPLE_WITH_VIEW);

    await expect(page.locator('.diagram-node', { hasText: 'PaymentsView' })).toBeVisible();
    await expect(page.locator('#data-panel')).not.toHaveClass(/collapsed/);
  });

  test('keeps every main-thread task under 50 ms while the largest example is typed into', async ({ page }) => {
    await open(page);
    await page.locator('#source-input').fill(largestExample());
    await page.locator('#render-btn').click();
    await expect(page.locator('#render-status')).toHaveText('✓ Rendered');
    await page.locator('#data-panel-header').click();

    await page.evaluate(() => {
      window.longTasks = [];
      new PerformanceObserver((list) => {
        list.getEntries().forEach((entry) => window.longTasks.push(Math.round(entry.duration)));
      }).observe({ type: 'longtask' });
      const panel = document.getElementById('source-input');
      panel.focus();
      panel.setSelectionRange(panel.value.length, panel.value.length);
    });

    const typePausing = async (text) => {
      await page.keyboard.type(text, { delay: 20 });
      await page.waitForTimeout(REVALIDATE_PAUSE_MS + 200);
    };
    await typePausing('\n# typed while the model revalidates');
    await typePausing('\nactor "Clerk');
    await expect(page.locator('#stale-notice')).toBeVisible();
    await typePausing('"');
    await expect(page.locator('#stale-notice')).toBeHidden();
    await typePausing('\n# and once more');

    expect(await page.evaluate(() => window.longTasks)).toEqual([]);
  });
});

test.describe('the panels along the bottom of the canvas', () => {
  test('keeps the source, the Render control and its status in reach while diagnostics are listed', async ({ page }) => {
    await open(page);
    await page.locator('#source-input').fill(PARSES_WITH_UNDECLARED_EVENT);
    await page.locator('#render-btn').click();
    await expect(page.locator('#render-status')).toHaveText('✓ Rendered');
    await page.locator('#data-panel-header').click();
    await expect(page.locator('#diagnostics-panel')).toBeVisible();
    await expect(page.locator('#data-panel')).not.toHaveClass(/collapsed/);

    expect({
      source: await landsOn(page, '#source-input'),
      render: await landsOn(page, '#render-btn'),
      status: await landsOn(page, '#render-status'),
    }).toEqual({ source: true, render: true, status: true });
  });

  test("keeps the diagnostics panel's close control clickable over the minimap when both panels fill a short window", async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 620 });
    await open(page);
    await page.locator('#source-input').fill(readFileSync(new URL('../../examples/error_diagnostics_test.emod', import.meta.url), 'utf8'));
    await page.locator('#render-btn').click();
    await expect(page.locator('#diagnostics-panel')).toBeVisible();
    await page.locator('#data-panel-header').click();
    await expect(page.locator('#data-panel')).not.toHaveClass(/collapsed/);
    await expect(page.locator('#minimap')).toBeVisible();

    const overlapsMinimap = await page.evaluate(() => {
      const close = document.getElementById('diagnostics-close').getBoundingClientRect();
      const minimap = document.getElementById('minimap').getBoundingClientRect();
      return close.top < minimap.bottom && close.bottom > minimap.top && close.left < minimap.right && close.right > minimap.left;
    });
    expect(overlapsMinimap).toBe(true);
    expect(await landsOn(page, '#diagnostics-close')).toBe(true);
  });
});

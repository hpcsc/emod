import { test, expect } from '@playwright/test';
import { open, render, SAMPLE } from './helpers.js';

// A context left open is the smallest edit that stops the source parsing, and
// the parser recovers only what comes before it.
const UNPARSEABLE = SAMPLE + 'context "Refunds" {\n';

// landsOnStaleNotice asks the browser what a click at the notice's centre would
// reach, which is what a user reading it depends on: a jsdom assertion on its
// text cannot see an overlay drawn over it.
async function landsOnStaleNotice(page) {
  return page.evaluate(() => {
    const notice = document.getElementById('stale-notice');
    const box = notice.getBoundingClientRect();
    const hit = document.elementFromPoint(box.left + box.width / 2, box.top + box.height / 2);
    return box.width > 0 && notice.contains(hit);
  });
}

test.describe('a diagram the source panel no longer parses into', () => {
  test('stays on screen with every node, marked out of date by the real pipeline', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const nodes = await page.locator('.diagram-node').count();

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
    expect(await landsOnStaleNotice(page)).toBe(true);

    await page.locator('#data-panel-header').click();
    await expect(page.locator('#data-panel')).toHaveClass(/collapsed/);
    expect(await landsOnStaleNotice(page)).toBe(true);

    await page.locator('#visibility-toggle').click();
    await page.locator('#legend-toggle').click();
    await expect(page.locator('#visibility-panel')).toBeVisible();
    await expect(page.locator('#legend-panel')).toBeVisible();
    await expect(page.locator('#minimap')).toBeVisible();
    await expect(page.locator('#diagnostics-panel')).toBeVisible();
    expect(await landsOnStaleNotice(page)).toBe(true);
  });
});

import { test, expect } from '@playwright/test';
import { open, render, SAMPLE } from './helpers.js';

test.describe('viewer bootstrap', () => {
  test('loads the WASM parser and reports it is ready', async ({ page }) => {
    await open(page);

    await expect(page.locator('#render-status')).not.toHaveText(/✗/);
  });

  test('renders a diagram from pasted .emod source', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);

    await expect(page.locator('#model-name-display')).toHaveText('Billing');
    await expect(page.locator('.diagram-node[data-node-id]').first()).toBeVisible();
  });

  test('reports a parse failure instead of rendering', async ({ page }) => {
    await open(page);
    await page.locator('#source-input').fill('model "Broken"\n\ncontext {\n');
    await page.locator('#render-btn').click();

    await expect(page.locator('#diagnostics-badge')).toBeVisible();
  });
});

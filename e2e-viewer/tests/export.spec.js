import { test, expect } from '@playwright/test';
import { open, render, exportEmod, centreOf, SAMPLE } from './helpers.js';

// These assert the full chain the viewer runs on export: on-screen edits land
// in the store, the store is serialised to diagram JSON, the WASM importer
// rebuilds an AST from it, and the Go formatter writes the text back out.
test.describe('export round trip', () => {
  test('reproduces the rendered source exactly', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);

    expect(await exportEmod(page)).toBe(SAMPLE);
  });

  test('names the download after the model', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);

    const downloadPromise = page.waitForEvent('download');
    await page.locator('#export-emod').click();

    expect((await downloadPromise).suggestedFilename()).toBe('Billing.emod');
  });

  test('is unchanged by panning and zooming, which carry no model meaning', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);

    await page.mouse.move(900, 640);
    await page.mouse.down();
    await page.mouse.move(700, 500, { steps: 4 });
    await page.mouse.up();
    await page.mouse.wheel(0, -200);

    expect(await exportEmod(page)).toBe(SAMPLE);
  });

  test('is unchanged by dragging a node to a new position', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);

    const node = page.locator('.diagram-node[data-node-id="command-1"]');
    const from = await centreOf(node);
    await node.hover();
    await page.mouse.down();
    await page.mouse.move(from.x + 90, from.y + 60, { steps: 6 });
    await page.mouse.up();

    expect(await exportEmod(page)).toBe(SAMPLE);
  });
});

test.describe('editing then exporting', () => {
  // The slice header is drawn as a rect with its label text on top, so the
  // text is what a real right-click lands on.
  const sliceHeader = (page) => page.locator('text.slice-header').first();

  test('includes a command added from the slice context menu', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);

    await sliceHeader(page).click({ button: 'right' });
    await page.locator('.ctx-menu-item[data-action="add-command"]').click();

    const exported = await exportEmod(page);
    expect(exported).toContain('command new-command-2 {');
    expect(exported).toContain('command TakePayment {');
  });

  test('includes a command, event and flow added by Add Flow', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);

    await sliceHeader(page).click({ button: 'right' });
    await page.locator('.ctx-menu-item[data-action="add-flow"]').click();

    const exported = await exportEmod(page);
    expect(exported).toContain('event new-event-2 {');
    expect(exported).toContain('command -> event: TakePayment -> new-event-2');
  });

  test('includes a slice added from the context header menu', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);

    await page.locator('.ctx-header').first().click({ button: 'right', force: true });
    await page.locator('.ctx-menu-item[data-action="add-slice"]').click();

    // Numbered -2 because the aggregate already holds the "Take Payment" slice.
    expect(await exportEmod(page)).toContain('slice "new-slice-2"');
  });

  test('drops an arrow deleted from its context menu', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    expect(await exportEmod(page)).toContain('command -> event: TakePayment -> PaymentTaken');

    // A straight arrow has a zero-width bounding box, which fails Playwright's
    // visibility check, so the click is forced onto its computed centre.
    await page.locator('.flow-arrow').first().click({ button: 'right', force: true });
    await page.locator('.ctx-menu-item[data-action="delete-arrow"]').click();

    expect(await exportEmod(page)).not.toContain('command -> event:');
  });

  test('round-trips a re-rendered export back to the same text', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const first = await exportEmod(page);

    await page.locator('#data-panel-header').click();
    await render(page, first);

    expect(await exportEmod(page)).toBe(first);
  });
});

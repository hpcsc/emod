import { test, expect } from '@playwright/test';
import { open, render, viewport, emptyCanvasPoint, centreOf, SAMPLE, WIDE } from './helpers.js';

test.describe('panning', () => {
  // Asserted on screen rather than on the viewport offset: the offset is in the
  // SVG's own units, so comparing it to a pixel delta passes even when the
  // diagram visibly lags the cursor.
  test('moves the diagram on screen by exactly the pointer delta', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const node = page.locator('.diagram-node').first();
    const before = await node.boundingBox();
    const bg = await emptyCanvasPoint(page);

    await page.mouse.move(bg.x, bg.y);
    await page.mouse.down();
    await page.mouse.move(bg.x + 120, bg.y + 45, { steps: 5 });
    await page.mouse.up();

    const after = await node.boundingBox();
    expect(after.x - before.x).toBeCloseTo(120, 0);
    expect(after.y - before.y).toBeCloseTo(45, 0);
  });

  test('keeps pace with the cursor on a diagram far wider than the canvas', async ({ page }) => {
    await open(page);
    await render(page, WIDE);
    const node = page.locator('.diagram-node').first();
    const before = await node.boundingBox();
    const bg = await emptyCanvasPoint(page);

    await page.mouse.move(bg.x, bg.y);
    await page.mouse.down();
    await page.mouse.move(bg.x + 200, bg.y, { steps: 5 });
    await page.mouse.up();

    const after = await node.boundingBox();
    expect(after.x - before.x).toBeCloseTo(200, 0);
  });

  test('leaves the zoom alone', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const before = await viewport(page);
    const bg = await emptyCanvasPoint(page);

    await page.mouse.move(bg.x, bg.y);
    await page.mouse.down();
    await page.mouse.move(bg.x + 120, bg.y + 45, { steps: 5 });
    await page.mouse.up();

    expect((await viewport(page)).scale).toBeCloseTo(before.scale, 5);
  });

  test('leaves the diagram where the pan ended', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const bg = await emptyCanvasPoint(page);

    await page.mouse.move(bg.x, bg.y);
    await page.mouse.down();
    await page.mouse.move(bg.x - 60, bg.y, { steps: 3 });
    await page.mouse.up();
    const afterRelease = await viewport(page);

    await page.mouse.move(bg.x + 200, bg.y + 200);

    expect(await viewport(page)).toEqual(afterRelease);
  });
});

test.describe('zooming', () => {
  test('scales up on a scroll toward the user', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const before = await viewport(page);
    const bg = await emptyCanvasPoint(page);

    await page.mouse.move(bg.x, bg.y);
    await page.mouse.wheel(0, -100);

    const after = await viewport(page);
    expect(after.scale).toBeCloseTo(before.scale * Math.pow(1.001, 100), 3);
  });

  test('scales down on a scroll away from the user', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const before = await viewport(page);
    const bg = await emptyCanvasPoint(page);

    await page.mouse.move(bg.x, bg.y);
    await page.mouse.wheel(0, 100);

    const after = await viewport(page);
    expect(after.scale).toBeCloseTo(before.scale * Math.pow(1.001, -100), 3);
  });

  // Zoom is anchored to the cursor, so whatever sits under the pointer must
  // stay under it. Asserted on the node's own on-screen box rather than by
  // redoing the viewer's screen-to-diagram maths, which would only prove that
  // maths agrees with itself.
  test('keeps the node under the cursor in place while zooming in', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);

    const node = page.locator('.diagram-node[data-node-id="command-1"]');
    const before = await centreOf(node);

    await page.mouse.move(before.x, before.y);
    await page.mouse.wheel(0, -240);

    const after = await centreOf(node);
    expect(after.x).toBeCloseTo(before.x, 0);
    expect(after.y).toBeCloseTo(before.y, 0);
  });

  test('keeps the node under the cursor in place while zooming out', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);

    const node = page.locator('.diagram-node[data-node-id="command-1"]');
    const before = await centreOf(node);

    await page.mouse.move(before.x, before.y);
    await page.mouse.wheel(0, 240);

    const after = await centreOf(node);
    expect(after.x).toBeCloseTo(before.x, 0);
    expect(after.y).toBeCloseTo(before.y, 0);
  });

  test('actually resizes the node it is anchored on', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);

    const node = page.locator('.diagram-node[data-node-id="command-1"]');
    const before = await node.boundingBox();

    await page.mouse.move(before.x + before.width / 2, before.y + before.height / 2);
    await page.mouse.wheel(0, -240);

    const after = await node.boundingBox();
    expect(after.width).toBeGreaterThan(before.width * 1.2);
  });

  test('refuses to zoom out past the minimum scale', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const bg = await emptyCanvasPoint(page);

    await page.mouse.move(bg.x, bg.y);
    for (let i = 0; i < 12; i++) await page.mouse.wheel(0, 1000);

    expect((await viewport(page)).scale).toBeCloseTo(0.1, 5);
  });

  test('refuses to zoom in past the maximum scale', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const bg = await emptyCanvasPoint(page);

    await page.mouse.move(bg.x, bg.y);
    for (let i = 0; i < 12; i++) await page.mouse.wheel(0, -1000);

    expect((await viewport(page)).scale).toBeCloseTo(5.0, 5);
  });
});

test.describe('fit to view', () => {
  test('brings the whole diagram back into frame after panning away', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const bg = await emptyCanvasPoint(page);

    await page.mouse.move(bg.x, bg.y);
    await page.mouse.down();
    await page.mouse.move(bg.x + 4000, bg.y + 3000, { steps: 5 });
    await page.mouse.up();

    await page.locator('#fit-view').click();

    const canvas = await page.locator('#diagram-canvas').boundingBox();
    const group = await page.locator('#viewport-group').boundingBox();
    expect(group.x).toBeGreaterThanOrEqual(canvas.x - 1);
    expect(group.y).toBeGreaterThanOrEqual(canvas.y - 1);
    expect(group.x + group.width).toBeLessThanOrEqual(canvas.x + canvas.width + 1);
    expect(group.y + group.height).toBeLessThanOrEqual(canvas.y + canvas.height + 1);
  });
});

test.describe('node selection', () => {
  test('opens the field editor from the node context menu', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);

    await page.locator('.diagram-node[data-node-id="command-1"]').click({ button: 'right' });
    await page.locator('.ctx-menu-item[data-action="open-field-editor"]').click();

    await expect(page.locator('#detail-panel')).toBeVisible();
    await expect(page.locator('#dp-content')).toContainText('TakePayment');
  });

  test('leaves the field editor closed when the node is clicked', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);

    await page.locator('.diagram-node[data-node-id="command-1"]').click();

    await expect(page.locator('#detail-panel')).toBeHidden();
  });
});

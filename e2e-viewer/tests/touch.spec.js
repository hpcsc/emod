import { test, expect } from '@playwright/test';
import { open, render, viewport, emptyCanvasPoint, centreOf, exportEmod, SAMPLE } from './helpers.js';
import { touchInput, spread } from './touch-input.js';

test.describe('single-finger pan', () => {
  test('drags the diagram by exactly the finger delta', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const before = await viewport(page);
    const bg = await emptyCanvasPoint(page);
    const touch = await touchInput(page);

    await touch.drag(bg, { x: bg.x + 110, y: bg.y + 70 });

    const after = await viewport(page);
    expect(after.x).toBeCloseTo(before.x + 110, 1);
    expect(after.y).toBeCloseTo(before.y + 70, 1);
    expect(after.scale).toBeCloseTo(before.scale, 5);
  });

  test('leaves the diagram alone once the finger lifts', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const bg = await emptyCanvasPoint(page);
    const touch = await touchInput(page);

    await touch.drag(bg, { x: bg.x - 80, y: bg.y - 40 });
    const afterLift = await viewport(page);

    await touch.press([{ x: bg.x + 150, y: bg.y + 150 }]);
    await touch.release();

    expect(await viewport(page)).toEqual(afterLift);
  });

  test('does not alter the model', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const bg = await emptyCanvasPoint(page);
    const touch = await touchInput(page);

    await touch.drag(bg, { x: bg.x + 100, y: bg.y + 100 });

    expect(await exportEmod(page)).toBe(SAMPLE);
  });
});

test.describe('two-finger pinch', () => {
  test('zooms in by the ratio the fingers spread', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const before = await viewport(page);
    const bg = await emptyCanvasPoint(page);
    const touch = await touchInput(page);

    await touch.pinch(bg, 100, 200);

    expect((await viewport(page)).scale).toBeCloseTo(before.scale * 2, 2);
  });

  test('zooms out by the ratio the fingers close', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const before = await viewport(page);
    const bg = await emptyCanvasPoint(page);
    const touch = await touchInput(page);

    await touch.pinch(bg, 240, 60);

    expect((await viewport(page)).scale).toBeCloseTo(before.scale * 0.25, 2);
  });

  // Anchored on background, not on the node: a finger landing on a node starts
  // dragging that node instead, which is a different gesture entirely.
  test('scales the diagram about the midpoint between the fingers', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);

    const pivot = await emptyCanvasPoint(page);
    const node = page.locator('.diagram-node[data-node-id="command-1"]');
    const before = await centreOf(node);
    const touch = await touchInput(page);

    await touch.pinch(pivot, 100, 200);

    // Scaling about a pivot moves every other point away from it by the same
    // factor, which is arithmetic the viewer plays no part in.
    const after = await centreOf(node);
    expect(after.x).toBeCloseTo(pivot.x + (before.x - pivot.x) * 2, 0);
    expect(after.y).toBeCloseTo(pivot.y + (before.y - pivot.y) * 2, 0);
  });

  test('refuses to pinch past the maximum scale', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const bg = await emptyCanvasPoint(page);
    const touch = await touchInput(page);

    await touch.pinch(bg, 20, 900);

    expect((await viewport(page)).scale).toBeCloseTo(5.0, 5);
  });

  test('refuses to pinch past the minimum scale', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const bg = await emptyCanvasPoint(page);
    const touch = await touchInput(page);

    await touch.pinch(bg, 900, 4);

    expect((await viewport(page)).scale).toBeCloseTo(0.1, 5);
  });

  test('does not alter the model', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const bg = await emptyCanvasPoint(page);
    const touch = await touchInput(page);

    await touch.pinch(bg, 100, 220);

    expect(await exportEmod(page)).toBe(SAMPLE);
  });
});

// A real two-finger gesture rarely starts with both fingers landing together,
// and rarely ends with both lifting together, so the handler switches mode
// mid-gesture. These are the transitions that are easiest to get wrong.
test.describe('switching between pan and pinch mid-gesture', () => {
  test('pans on one finger, then zooms when a second joins', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const before = await viewport(page);
    const bg = await emptyCanvasPoint(page);
    const anchor = { x: bg.x + 30, y: bg.y };
    const touch = await touchInput(page);

    await touch.press([bg]);
    await touch.move([anchor]);
    const panned = await viewport(page);

    await touch.press([{ x: anchor.x + 100, y: anchor.y }]);
    await touch.move(spread(anchor, 100));
    await touch.move(spread(anchor, 300));
    await touch.release();

    const after = await viewport(page);
    expect(panned.x).toBeCloseTo(before.x + 30, 1);
    expect(panned.scale).toBeCloseTo(before.scale, 5);
    expect(after.scale).toBeCloseTo(before.scale * 3, 2);
  });

  test('resumes panning when one finger lifts from a pinch', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const bg = await emptyCanvasPoint(page);
    const touch = await touchInput(page);

    await touch.press(spread(bg, 90));
    await touch.move(spread(bg, 100));
    await touch.move(spread(bg, 200));
    const zoomed = await viewport(page);

    // Lift one finger; the other takes over as a pan.
    await touch.lift(1);
    const [survivor] = touch.down();
    await touch.move([{ x: survivor.x + 60, y: survivor.y + 25 }]);
    await touch.release();

    const after = await viewport(page);
    expect(after.scale).toBeCloseTo(zoomed.scale, 5);
    expect(after.x).toBeCloseTo(zoomed.x + 60, 1);
    expect(after.y).toBeCloseTo(zoomed.y + 25, 1);
  });
});

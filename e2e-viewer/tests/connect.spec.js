import { test, expect } from '@playwright/test';
import {
  open, render, exportEmod, centreOf, dropPointIn, portOf, handleOf, dragBetween,
  SAMPLE, SAMPLE_WITH_VIEW,
} from './helpers.js';

// Adds a second event to the slice so there is somewhere to point an arrow
// other than where it already points.
async function addSecondEvent(page) {
  await page.locator('text.slice-header').first().click({ button: 'right' });
  await page.locator('.ctx-menu-item[data-action="add-event"]').click();
  const added = page.locator('.diagram-node').filter({ hasText: 'new-event-2' });
  await expect(added).toBeVisible();
  return added;
}

test.describe('drawing an edge from a port', () => {
  test('connects a command to an event and records the flow', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const added = await addSecondEvent(page);

    await dragBetween(page, await portOf(page, 'command-1', 'out'), await dropPointIn(added));

    expect(await exportEmod(page)).toContain('command -> event: TakePayment -> new-event-2');
  });

  test('keeps the flow the model already had', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const added = await addSecondEvent(page);

    await dragBetween(page, await portOf(page, 'command-1', 'out'), await dropPointIn(added));

    expect(await exportEmod(page)).toContain('command -> event: TakePayment -> PaymentTaken');
  });

  test('does nothing when the drag ends on empty canvas', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const before = await exportEmod(page);
    const port = await portOf(page, 'command-1', 'out');

    await dragBetween(page, port, { x: port.x + 40, y: port.y + 260 });

    expect(await exportEmod(page)).toBe(before);
  });

  test('does nothing when the port is clicked without dragging', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const before = await exportEmod(page);
    const port = await portOf(page, 'command-1', 'out');

    await page.mouse.move(port.x, port.y);
    await page.mouse.down();
    await page.mouse.up();

    expect(await exportEmod(page)).toBe(before);
  });

  test('refuses to connect a node to itself', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const before = await exportEmod(page);

    await dragBetween(
      page,
      await portOf(page, 'command-1', 'out'),
      await dropPointIn(page.locator('.diagram-node[data-node-id="command-1"]')));

    expect(await exportEmod(page)).toBe(before);
  });

  test('does not duplicate an edge that already exists', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const before = await exportEmod(page);

    await dragBetween(
      page,
      await portOf(page, 'command-1', 'out'),
      await dropPointIn(page.locator('.diagram-node[data-node-id="event-1"]')));

    expect(await exportEmod(page)).toBe(before);
  });

  test('subscribes a view when an event is connected to it', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE_WITH_VIEW);

    await dragBetween(
      page,
      await portOf(page, 'event-1', 'out'),
      await dropPointIn(page.locator('.diagram-node[data-node-id="view-1"]')));

    expect(await exportEmod(page)).toContain('subscribes [PaymentTaken]');
  });

  test('leaves no preview line behind after the drag', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const added = await addSecondEvent(page);

    await dragBetween(page, await portOf(page, 'command-1', 'out'), await dropPointIn(added));

    await expect(page.locator('.connect-preview')).toHaveCount(0);
  });
});

test.describe('repointing an arrow', () => {
  test('moves the target end onto another event', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const added = await addSecondEvent(page);

    const handle = handleOf(page, 'command-1', 'event-1', 'target');
    await dragBetween(page, await centreOf(handle), await dropPointIn(added));

    const exported = await exportEmod(page);
    expect(exported).toContain('command -> event: TakePayment -> new-event-2');
    expect(exported).not.toContain('TakePayment -> PaymentTaken');
  });

  async function addSecondCommand(page) {
    await page.locator('text.slice-header').first().click({ button: 'right' });
    await page.locator('.ctx-menu-item[data-action="add-command"]').click();
    const added = page.locator('.diagram-node').filter({ hasText: 'new-command-2' });
    await expect(added).toBeVisible();
    return added;
  }

  test('moves the source end onto another command', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const added = await addSecondCommand(page);

    const handle = handleOf(page, 'command-1', 'event-1', 'source');
    await dragBetween(page, await centreOf(handle), await dropPointIn(added));

    const exported = await exportEmod(page);
    expect(exported).toContain('command -> event: new-command-2 -> PaymentTaken');
    expect(exported).not.toContain('TakePayment -> PaymentTaken');
  });

  // The added command sits directly below the original, so the command-to-event
  // arrow runs through its centre. Arrows paint over the nodes, so aiming at the
  // obvious spot used to drop onto the arrow and be discarded.
  test('accepts a drop on the target centre, where an arrow crosses it', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const added = await addSecondCommand(page);

    const handle = handleOf(page, 'command-1', 'event-1', 'source');
    await dragBetween(page, await centreOf(handle), await centreOf(added));

    expect(await exportEmod(page)).toContain('command -> event: new-command-2 -> PaymentTaken');
  });

  test('refuses to collapse an arrow onto its own other end', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const before = await exportEmod(page);

    const handle = handleOf(page, 'command-1', 'event-1', 'target');
    await dragBetween(
      page,
      await centreOf(handle),
      await dropPointIn(page.locator('.diagram-node[data-node-id="command-1"]')));

    expect(await exportEmod(page)).toBe(before);
  });

  test('does nothing when the handle is dropped on empty canvas', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const before = await exportEmod(page);

    const handle = handleOf(page, 'command-1', 'event-1', 'target');
    const from = await centreOf(handle);
    await dragBetween(page, from, { x: from.x + 60, y: from.y + 240 });

    expect(await exportEmod(page)).toBe(before);
  });
});

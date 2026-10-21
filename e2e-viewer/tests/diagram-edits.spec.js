import { test, expect } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { open, render } from './helpers.js';

const specsHotel = readFileSync(new URL('../../examples/specs_hotel.emod', import.meta.url), 'utf-8');

// landsOn asks the browser what a press at an element's centre reaches, which
// is what a user reading or pressing it depends on.
async function landsOn(page, selector) {
  return page.evaluate((sel) => {
    const target = document.querySelector(sel);
    const box = target.getBoundingClientRect();
    const hit = document.elementFromPoint(box.left + box.width / 2, box.top + box.height / 2);
    return box.width > 0 && box.height > 0 && target.contains(hit);
  }, selector);
}

// Types into the panel and adds a command from a slice menu in one task, so the
// diagram edit lands before the pause that would redraw from the typed text.
async function typeThenAddCommand(page, text, slice) {
  await page.evaluate(([typed, sliceName]) => {
    const panel = document.getElementById('source-input');
    panel.value = typed;
    panel.dispatchEvent(new Event('input', { bubbles: true }));
    const header = [...document.querySelectorAll('text.slice-header')].find((el) => el.textContent === sliceName);
    header.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true }));
    document.querySelector('.ctx-menu-item[data-action="add-command"]').click();
  }, [text, slice]);
}

const withCommandInReleaseHold = (text) => text.replace(`      command "ReleaseHold" {
        fields {
          holdId = required(string)
        }
      }
`, `      command "ReleaseHold" {
        fields {
          holdId = required(string)
        }
      }

      command "new-command-2" {
      }
`);

function replacedOnce(text, old, replacement) {
  expect(text.split(old)).toHaveLength(2);
  return text.replace(old, replacement);
}

test.describe('a diagram edit reaching the source panel through the real pipeline', () => {
  test('writes an added command and a deleted arrow into the file, keeping everything the diagram does not show', async ({ page }) => {
    await open(page);
    await render(page, specsHotel);
    const panel = page.locator('#source-input');

    await page.locator('text.slice-header', { hasText: 'Release Hold' }).click({ button: 'right' });
    await page.locator('.ctx-menu-item[data-action="add-command"]').click();

    const withCommand = withCommandInReleaseHold(specsHotel);
    expect(withCommand).not.toBe(specsHotel);
    await expect(panel).toHaveValue(withCommand);
    await expect(page.locator('#stale-notice')).toBeHidden();

    // A straight arrow has a zero-width bounding box, which fails Playwright's
    // visibility check, so the click is forced onto its computed centre.
    const holdFlow = page.locator('.flow-arrow[data-source="command-2"][data-target="event-2"]');
    await holdFlow.click({ button: 'right', force: true });
    await page.locator('.ctx-menu-item[data-action="delete-arrow"]').click();

    await expect(panel).toHaveValue(replacedOnce(withCommand,
      '      flow = <<-FLOW\n        command -> event:    HoldRoom -> RoomHeld\n      FLOW\n\n', ''));
  });
});

test.describe('a conflict between the diagram and the source panel through the real pipeline', () => {
  const typed = specsHotel + '# typed\n';

  test('shows the notice in place of the out-of-date one, readable and pressable with the data panel open and collapsed', async ({ page }) => {
    await open(page);
    await render(page, specsHotel);
    await page.locator('#data-panel-header').click();
    await page.locator('#source-input').fill(specsHotel + 'context "Refunds" {\n');
    await page.locator('#render-btn').click();
    await expect(page.locator('#stale-notice')).toBeVisible();

    await page.locator('text.slice-header', { hasText: 'Release Hold' }).click({ button: 'right' });
    await page.locator('.ctx-menu-item[data-action="add-command"]').click();

    await expect(page.locator('#conflict-notice')).toBeVisible();
    await expect(page.locator('#stale-notice')).toBeHidden();
    expect(await page.locator('#conflict-message').evaluate((el) => el.scrollWidth <= el.clientWidth),
      'the message fits the notice at this width rather than being cut short').toBe(true);
    await expect(page.locator('#data-panel')).not.toHaveClass(/collapsed/);
    for (const selector of ['#conflict-message', '#keep-diagram-edits', '#keep-source-edits']) {
      expect(await landsOn(page, selector), selector).toBe(true);
    }

    await page.locator('#data-panel-header').click();
    await expect(page.locator('#data-panel')).toHaveClass(/collapsed/);
    for (const selector of ['#conflict-message', '#keep-diagram-edits', '#keep-source-edits']) {
      expect(await landsOn(page, selector), selector).toBe(true);
    }
  });

  test('Keep diagram edits leaves the panel holding the command and not the typed text', async ({ page }) => {
    await open(page);
    await render(page, specsHotel);

    await typeThenAddCommand(page, typed, 'Release Hold');
    await expect(page.locator('#conflict-notice')).toBeVisible();
    await page.locator('#keep-diagram-edits').click();

    await expect(page.locator('#source-input')).toHaveValue(withCommandInReleaseHold(specsHotel));
    await expect(page.locator('#conflict-notice')).toBeHidden();
  });

  test('Keep source edits leaves the panel holding the typed text and the diagram drawn from it', async ({ page }) => {
    await open(page);
    await render(page, specsHotel);

    await typeThenAddCommand(page, typed, 'Release Hold');
    await expect(page.locator('#conflict-notice')).toBeVisible();
    await expect(page.locator('.diagram-node', { hasText: 'new-command-2' })).toHaveCount(1);
    await page.locator('#keep-source-edits').click();

    await expect(page.locator('#conflict-notice')).toBeHidden();
    await expect(page.locator('.diagram-node', { hasText: 'new-command-2' })).toHaveCount(0);
    await expect(page.locator('.diagram-node', { hasText: 'ReleaseHold' })).toHaveCount(1);
    await expect(page.locator('#source-input')).toHaveValue(typed);
  });
});

import { test, expect } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { open, render } from './helpers.js';

const specsHotel = readFileSync(new URL('../../examples/specs_hotel.emod', import.meta.url), 'utf-8');

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

    const withCommand = replacedOnce(specsHotel, `      command "ReleaseHold" {
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

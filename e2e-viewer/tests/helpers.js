import { expect } from '@playwright/test';

export const SAMPLE = `model "Billing"

context "Payments" {
  aggregate "Payment" {
    slice "Take Payment" {
      trigger UI "Checkout Form" {
        actor Customer
      }

      command TakePayment {
        fields {
          amount int required
        }
      }

      event PaymentTaken {
        fields {
          amount int required
        }
      }

      flow {
        command -> event: TakePayment -> PaymentTaken
      }
    }
  }
}
`;

// open loads the viewer and waits for the WASM parser to report ready, so a
// test never races the module fetch.
export async function open(page) {
  await page.goto('/');
  await expect(page.locator('#render-btn')).toBeEnabled();
  await page.waitForFunction(() => typeof globalThis.parseEmod === 'function');
}

// render pastes .emod source and renders it, leaving the diagram on screen.
export async function render(page, source) {
  await page.locator('#source-input').fill(source);
  await page.locator('#render-btn').click();
  await expect(page.locator('#render-status')).toHaveText('✓ Rendered');
  await expect(page.locator('#viewport-group')).toBeAttached();
}

// viewport reads back the transform the viewer applies to the diagram group.
export async function viewport(page) {
  return page.evaluate(() => {
    const vg = document.querySelector('#viewport-group');
    const m = /translate\(([-\d.]+), ([-\d.]+)\) scale\(([-\d.]+)\)/.exec(vg.getAttribute('transform') || '');
    return m ? { x: Number(m[1]), y: Number(m[2]), scale: Number(m[3]) } : null;
  });
}

// exportEmod clicks Export .emod and returns the downloaded file's contents.
export async function exportEmod(page) {
  const downloadPromise = page.waitForEvent('download');
  await page.locator('#export-emod').click();
  const download = await downloadPromise;
  const stream = await download.createReadStream();
  const chunks = [];
  for await (const chunk of stream) chunks.push(chunk);
  return Buffer.concat(chunks).toString('utf8');
}

// centreOf returns viewport coordinates of an element's centre, for driving
// the mouse at real on-screen positions.
export async function centreOf(page, selector) {
  const box = await page.locator(selector).boundingBox();
  return { x: box.x + box.width / 2, y: box.y + box.height / 2 };
}

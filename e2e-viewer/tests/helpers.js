import { expect } from '@playwright/test';

// Canonical formatting, verified with `emod fmt --check`. Export tests assert
// the viewer reproduces this byte for byte, so it must stay canonical.
export const SAMPLE = `model "Billing"

actor "Customer"

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
export async function centreOf(locator) {
  const box = await locator.boundingBox();
  return { x: box.x + box.width / 2, y: box.y + box.height / 2 };
}

// Anything the mousedown handler treats specially before it falls through to
// starting a pan.
const NOT_BACKGROUND = '.diagram-node, .slice-header, [data-port], [data-arrow-handle], ' +
  '.flow-arrow, .sub-arrow, .auto-trg-arrow, .auto-cmd-arrow, .trg-cmd-arrow, ' +
  '.reads-arrow, .trans-cmd-arrow, .ctx-label, .agg-label';

// emptyCanvasPoint finds a spot that really does pan the diagram: inside the
// canvas, not on a node or arrow, and not under an overlay such as the data
// panel or minimap. Scanning beats hardcoded coordinates, which silently start
// hitting something else the moment the fixture or layout changes.
export async function emptyCanvasPoint(page) {
  const point = await page.evaluate((notBackground) => {
    const svg = document.getElementById('diagram-canvas');
    const box = svg.getBoundingClientRect();
    for (let ny = 0.85; ny > 0.05; ny -= 0.05) {
      for (let nx = 0.95; nx > 0.05; nx -= 0.05) {
        const x = box.left + box.width * nx;
        const y = box.top + box.height * ny;
        const el = document.elementFromPoint(x, y);
        if (!el || !svg.contains(el)) continue;
        if (el.closest(notBackground)) continue;
        return { x: Math.round(x), y: Math.round(y) };
      }
    }
    return null;
  }, NOT_BACKGROUND);

  if (!point) throw new Error('no empty canvas point found — the diagram fills the viewport');
  return point;
}

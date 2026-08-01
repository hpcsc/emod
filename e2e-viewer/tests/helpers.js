import { expect } from '@playwright/test';

// Canonical formatting, verified with `emod fmt --check`. Export tests assert
// the viewer reproduces this byte for byte, so it must stay canonical.
export const SAMPLE = `emod 1
model "Billing"

actor "Customer"

context "Payments" {
  aggregate "Payment" {
    slice "Take Payment" {
      trigger "Checkout Form" {
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

// Far wider than the canvas, so the viewBox squeezes several diagram units into
// each screen pixel. That is the case where a pan measured in raw pointer
// pixels visibly lags the cursor, and it only shows up at this width.
export const WIDE = (function () {
  const slices = [];
  for (let i = 0; i < 12; i++) {
    slices.push(
      `    slice "Step ${i}" {\n` +
      `      command Cmd${i} {\n` +
      '        fields {\n' +
      '          amount int required\n' +
      '        }\n' +
      '      }\n' +
      '    }');
  }
  return 'emod 1\nmodel "Wide"\n\ncontext "Ctx" {\n  aggregate "Agg" {\n' +
    slices.join('\n\n') + '\n  }\n}\n';
})();

// A second slice holding a view, for the edge types that need one. Also
// canonical — `emod fmt --check` passes on it.
export const SAMPLE_WITH_VIEW = `emod 1
model "Billing"

actor "Customer"

context "Payments" {
  aggregate "Payment" {
    slice "Take Payment" {
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

    slice "Payment History" {
      view PaymentsView {
        fields {
          amount int required
        }
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

// diagramScreenPos reports where the diagram actually sits on screen, read off
// a node's box. Panning is asserted through this rather than through the
// viewport offset: that offset is in the SVG's own units, so comparing it to a
// pixel delta passes even when the diagram visibly lags the pointer.
export async function diagramScreenPos(page) {
  const box = await page.locator('.diagram-node').first().boundingBox();
  return { x: box.x, y: box.y };
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

// dropPointIn returns a spot inside a node that is off its centre line.
//
// Arrows are appended after the nodes, so they are painted on top, and their
// stroke swallows a drop aimed at a node the arrow happens to cross — which is
// exactly what a vertical command-to-event arrow does to the node's centre. A
// test releasing there proves nothing about the drop, only about the arrow.
export async function dropPointIn(locator) {
  const box = await locator.boundingBox();
  return { x: box.x + box.width * 0.8, y: box.y + box.height * 0.5 };
}

// portOf locates the connection port on one side of a node — 'top', 'right',
// 'bottom' or 'left'. The ports render at opacity 0 until the node is hovered,
// which leaves them hit-testable throughout — only display and visibility take
// an element out of the running.
export async function portOf(page, nodeId, which) {
  const node = page.locator(`.diagram-node[data-node-id="${nodeId}"]`);
  await node.hover();
  return centreOf(page.locator(`[data-port="${which}"][data-node-id="${nodeId}"]`));
}

// handleOf locates one end of an arrow's repoint handles.
export function handleOf(page, source, target, end) {
  return page.locator(
    `[data-arrow-handle="${end}"][data-edge-source="${source}"][data-edge-target="${target}"]`);
}

// grabPointFor reveals an arrow's repoint handles by hovering the arrow, then
// returns where to press to take hold of one end. The hover is what a user does
// to see the handles, and what makes them take the pointer: an arrow ends on a
// block's connection port, and a handle that is not showing leaves the port to
// take the press.
export async function grabPointFor(page, source, target, end) {
  // Walk the arrow for a spot where it is the thing under the pointer. Its own
  // hit path cannot be hovered through Playwright — a straight arrow is zero
  // pixels wide, which reads as an invisible element — and the obvious middle
  // is sometimes covered by a block the arrow passes behind.
  const onArrow = await page.evaluate((edgeId) => {
    const hit = document.querySelector(`.arrow-hit[data-edge-id="${edgeId}"]`);
    const ctm = hit.getScreenCTM();
    const length = hit.getTotalLength();
    for (let along = 0.05; along < 1; along += 0.05) {
      const p = hit.getPointAtLength(length * along);
      const point = { x: ctm.a * p.x + ctm.c * p.y + ctm.e, y: ctm.b * p.x + ctm.d * p.y + ctm.f };
      if (document.elementFromPoint(point.x, point.y) === hit) return point;
    }
    return null;
  }, `${source}--${target}`);

  if (!onArrow) throw new Error(`the ${source} → ${target} arrow is covered along its whole length`);

  await page.mouse.move(onArrow.x, onArrow.y);
  return centreOf(handleOf(page, source, target, end));
}

// dragBetween presses at one point, moves in steps so the drag threshold is
// crossed and the preview line updates, then releases at the other.
export async function dragBetween(page, from, to, steps = 8) {
  await page.mouse.move(from.x, from.y);
  await page.mouse.down();
  await page.mouse.move(to.x, to.y, { steps });
  await page.mouse.up();
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

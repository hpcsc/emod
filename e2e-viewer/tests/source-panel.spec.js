import { test, expect } from '@playwright/test';
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { open, render, SAMPLE, SAMPLE_WITH_VIEW } from './helpers.js';
import { REVALIDATE_PAUSE_MS } from '../../internal/frontend/static/config.js';

// A context left open is the smallest edit that stops the source parsing, and
// the parser recovers only what comes before it.
const UNPARSEABLE = SAMPLE + 'context "Refunds" {\n';

// landsOn asks the browser what a click at an element's centre would reach,
// which is what a user reading or pressing it depends on: a jsdom assertion
// cannot see an overlay drawn over it.
async function landsOn(page, selector) {
  return page.evaluate((sel) => {
    const target = document.querySelector(sel);
    const box = target.getBoundingClientRect();
    const hit = document.elementFromPoint(box.left + box.width / 2, box.top + box.height / 2);
    return box.width > 0 && box.height > 0 && target.contains(hit);
  }, selector);
}

test.describe('a diagram the source panel no longer parses into', () => {
  test('stays on screen with every node, marked out of date by the real pipeline', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    const nodes = await page.locator('.diagram-node').count();
    await page.locator('#data-panel-header').click();

    await page.locator('#source-input').fill(UNPARSEABLE);
    await page.locator('#render-btn').click();

    await expect(page.locator('#stale-notice')).toHaveText('Diagram out of date — the source does not parse');
    await expect(page.locator('.diagram-node')).toHaveCount(nodes);
    await expect(page.locator('.diagram-node', { hasText: 'TakePayment' })).toBeVisible();
  });

  test('can be read with the data panel open or collapsed, and with every panel the header opens', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    await page.locator('#data-panel-header').click();
    await page.locator('#source-input').fill(UNPARSEABLE);
    await page.locator('#render-btn').click();
    await expect(page.locator('#stale-notice')).toBeVisible();

    await expect(page.locator('#data-panel')).not.toHaveClass(/collapsed/);
    expect(await landsOn(page, '#stale-notice')).toBe(true);

    await page.locator('#data-panel-header').click();
    await expect(page.locator('#data-panel')).toHaveClass(/collapsed/);
    expect(await landsOn(page, '#stale-notice')).toBe(true);

    await page.locator('#visibility-toggle').click();
    await page.locator('#legend-toggle').click();
    await expect(page.locator('#visibility-panel')).toBeVisible();
    await expect(page.locator('#legend-panel')).toBeVisible();
    await expect(page.locator('#minimap')).toBeVisible();
    await expect(page.locator('#diagnostics-panel')).toBeVisible();
    expect(await landsOn(page, '#stale-notice')).toBe(true);
  });
});

function largestExample() {
  const dir = new URL('../../examples/', import.meta.url);
  const [largest] = readdirSync(dir)
    .filter((name) => name.endsWith('.emod') && !name.endsWith('_test.emod'))
    .map((name) => ({ name, size: statSync(new URL(name, dir)).size }))
    .sort((a, b) => b.size - a.size);
  return readFileSync(new URL(largest.name, dir), 'utf8');
}

const PARSES_WITH_UNDECLARED_EVENT = SAMPLE.replace('TakePayment -> PaymentTaken', 'TakePayment -> PaymentRefunded');

// Enough findings to fill the diagnostics panel: every slice names an event no
// slice declares, and every command it names is orphaned along with it.
const MANY_DIAGNOSTICS = (function () {
  const slices = [];
  for (let i = 0; i < 8; i++) {
    slices.push(
      `    slice "Step ${i}" {\n` +
      `      command "Cmd${i}" {\n` +
      '        fields {\n' +
      '          amount = required(int)\n' +
      '        }\n' +
      '      }\n' +
      '\n' +
      '      flow = <<-FLOW\n' +
      `        command -> event:    Cmd${i} -> Missing${i}\n` +
      '      FLOW\n' +
      '    }');
  }
  return 'emod = 1\n\nmodel "Faulty" {\n}\n\ncontext "Ctx" {\n  aggregate "Agg" {\n' +
    slices.join('\n\n') + '\n  }\n}\n';
})();

test.describe('the source panel revalidating as it is edited', () => {
  test('redraws what was typed once typing pauses, with no Render click', async ({ page }) => {
    await open(page);
    await render(page, SAMPLE);
    await page.locator('#data-panel-header').click();

    await page.locator('#source-input').fill(SAMPLE_WITH_VIEW);

    await expect(page.locator('.diagram-node', { hasText: 'PaymentsView' })).toBeVisible();
    await expect(page.locator('#data-panel')).not.toHaveClass(/collapsed/);
  });

  test('keeps every main-thread task under 50 ms while the largest example is typed into', async ({ page }) => {
    await open(page);
    await page.locator('#source-input').fill(largestExample());
    await page.locator('#render-btn').click();
    await expect(page.locator('#render-status')).toHaveText('✓ Rendered');
    await page.locator('#data-panel-header').click();

    await page.evaluate(() => {
      window.longTasks = [];
      new PerformanceObserver((list) => {
        list.getEntries().forEach((entry) => window.longTasks.push(Math.round(entry.duration)));
      }).observe({ type: 'longtask' });
      const panel = document.getElementById('source-input');
      panel.focus();
      panel.setSelectionRange(panel.value.length, panel.value.length);
    });

    const typePausing = async (text) => {
      await page.keyboard.type(text, { delay: 20 });
      await page.waitForTimeout(REVALIDATE_PAUSE_MS + 200);
    };
    await typePausing('\n# typed while the model revalidates');
    await typePausing('\nactor "Clerk');
    await expect(page.locator('#stale-notice')).toBeVisible();
    await typePausing('" {}');
    await expect(page.locator('#stale-notice')).toBeHidden();
    await typePausing('\n# and once more');

    expect(await page.evaluate(() => window.longTasks)).toEqual([]);
  });
});

test.describe('the panels along the bottom of the canvas', () => {
  test('keeps the source, the Render control and its status in reach while diagnostics are listed', async ({ page }) => {
    await open(page);
    await page.locator('#source-input').fill(PARSES_WITH_UNDECLARED_EVENT);
    await page.locator('#render-btn').click();
    await expect(page.locator('#render-status')).toHaveText('✓ Rendered');
    await page.locator('#data-panel-header').click();
    await expect(page.locator('#diagnostics-panel')).toBeVisible();
    await expect(page.locator('#data-panel')).not.toHaveClass(/collapsed/);

    expect({
      source: await landsOn(page, '#source-input'),
      render: await landsOn(page, '#render-btn'),
      status: await landsOn(page, '#render-status'),
    }).toEqual({ source: true, render: true, status: true });
  });

  test("keeps the diagnostics panel's close control clickable over the minimap when both panels fill a short window", async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 620 });
    await open(page);
    await page.locator('#source-input').fill(MANY_DIAGNOSTICS);
    await page.locator('#render-btn').click();
    await expect(page.locator('#diagnostics-panel')).toBeVisible();
    await page.locator('#data-panel-header').click();
    await expect(page.locator('#data-panel')).not.toHaveClass(/collapsed/);
    await expect(page.locator('#minimap')).toBeVisible();

    const overlapsMinimap = await page.evaluate(() => {
      const close = document.getElementById('diagnostics-close').getBoundingClientRect();
      const minimap = document.getElementById('minimap').getBoundingClientRect();
      return close.top < minimap.bottom && close.bottom > minimap.top && close.left < minimap.right && close.right > minimap.left;
    });
    expect(overlapsMinimap).toBe(true);
    expect(await landsOn(page, '#diagnostics-close')).toBe(true);
  });
});

const LARGEST_WITH_FINDINGS_NEAR_ITS_END = largestExample()
  .replace('then  = rejected(OneConfirmationPerBooking)', 'then  = rejected(NoSuchRule)');

function lineOf(source, needle) {
  return source.split('\n').findIndex((line) => line.includes(needle)) + 1;
}

// How far a gutter entry's centre sits from the centre of the line it names,
// where the line is placed from the source panel's own computed style and
// scroll rather than from anything the gutter is styled with.
async function gutterOffsetFrom(page, line) {
  return page.evaluate((n) => {
    const panel = document.getElementById('source-input');
    const style = getComputedStyle(panel);
    const lineHeight = parseFloat(style.lineHeight);
    const lineTop = panel.getBoundingClientRect().top + parseFloat(style.borderTopWidth) +
      parseFloat(style.paddingTop) + (n - 1) * lineHeight - panel.scrollTop;
    const entry = document.querySelector(`#source-gutter [title^="Line ${n}:"]`).getBoundingClientRect();
    return Math.abs((entry.top + entry.height / 2) - (lineTop + lineHeight / 2));
  }, line);
}

test.describe('the gutter beside the source panel', () => {
  test('marks each line the real pipeline reports, level with that line before and after the panel scrolls', async ({ page }) => {
    const source = LARGEST_WITH_FINDINGS_NEAR_ITS_END;
    expect(source).not.toBe(largestExample());
    const errorLine = lineOf(source, 'rejected(NoSuchRule)');
    const warningLine = lineOf(source, 'OneConfirmationPerBooking = ');
    expect(errorLine).toBeGreaterThan(300);
    expect(warningLine).toBeGreaterThan(300);

    await open(page);
    await page.locator('#source-input').fill(source);
    await page.locator('#render-btn').click();
    await expect(page.locator('#diagnostics-badge')).toHaveText('1 error, 1 warning');
    await page.locator('#data-panel-header').click();

    await expect(page.locator('#source-gutter [title]')).toHaveCount(2);
    await expect(page.locator(`#source-gutter [title="Line ${errorLine}: 1 error"]`)).toHaveAttribute('data-severity', 'error');
    await expect(page.locator(`#source-gutter [title="Line ${warningLine}: 1 warning"]`)).toHaveAttribute('data-severity', 'warning');
    const colours = await page.evaluate(() => {
      const colourOf = (selector, property) => getComputedStyle(document.querySelector(selector))[property];
      return {
        errorEntry: colourOf('#source-gutter [data-severity="error"]', 'backgroundColor'),
        errorPill: colourOf('#diagnostics-list .diag-severity.error', 'color'),
        warningEntry: colourOf('#source-gutter [data-severity="warning"]', 'backgroundColor'),
        warningPill: colourOf('#diagnostics-list .diag-severity.warning', 'color'),
      };
    });
    expect(colours.errorEntry).toBe(colours.errorPill);
    expect(colours.warningEntry).toBe(colours.warningPill);
    expect(colours.errorEntry).not.toBe(colours.warningEntry);
    expect(await gutterOffsetFrom(page, errorLine)).toBeLessThan(1);

    await page.evaluate((n) => {
      const panel = document.getElementById('source-input');
      panel.scrollTop = (n - 3) * parseFloat(getComputedStyle(panel).lineHeight);
    }, errorLine);

    await expect.poll(() => gutterOffsetFrom(page, errorLine)).toBeLessThan(1);
    expect(await gutterOffsetFrom(page, warningLine)).toBeLessThan(1);
    expect(await landsOn(page, `#source-gutter [title^="Line ${errorLine}:"]`)).toBe(true);
  });

  test('keeps each entry level with its line when the source is revalidated while the data panel is collapsed', async ({ page }) => {
    const source = LARGEST_WITH_FINDINGS_NEAR_ITS_END;
    const errorLine = lineOf(source, 'rejected(NoSuchRule)');
    await open(page);
    await page.locator('#source-input').fill(source);
    await page.locator('#render-btn').click();
    await expect(page.locator('#diagnostics-badge')).toHaveText('1 error, 1 warning');
    await page.locator('#data-panel-header').click();
    await page.evaluate((n) => {
      const panel = document.getElementById('source-input');
      panel.scrollTop = (n - 3) * parseFloat(getComputedStyle(panel).lineHeight);
    }, errorLine);
    await expect.poll(() => gutterOffsetFrom(page, errorLine)).toBeLessThan(1);

    await page.locator('#data-panel-header').click();
    await expect(page.locator('#data-panel')).toHaveClass(/collapsed/);
    await page.evaluate(() => {
      const panel = document.getElementById('source-input');
      panel.value = panel.value + '\n# revalidated while collapsed\n';
      panel.dispatchEvent(new Event('input', { bubbles: true }));
    });
    await page.waitForTimeout(REVALIDATE_PAUSE_MS + 400);
    await page.locator('#data-panel-header').click();

    await expect.poll(() => gutterOffsetFrom(page, errorLine)).toBeLessThan(1);
  });

  test('scrolls a line longer than the panel sideways rather than wrapping it', async ({ page }) => {
    await page.setViewportSize({ width: 900, height: 800 });
    await open(page);
    await page.locator('#source-input').fill(largestExample());

    const overflowsSideways = await page.evaluate(() => {
      const panel = document.getElementById('source-input');
      return panel.scrollWidth > panel.clientWidth;
    });

    expect(overflowsSideways).toBe(true);
  });
});

// Where the source panel's caret is, and whether the browser shows it: its line
// and column are counted from the text, and its place on screen from the
// panel's own computed style and a run of characters drawn in the panel's font.
async function caretInView(page) {
  return page.evaluate(() => {
    const panel = document.getElementById('source-input');
    const style = getComputedStyle(panel);
    const before = panel.value.slice(0, panel.selectionStart);
    const line = before.split('\n').length;
    const column = before.length - before.lastIndexOf('\n');

    const probe = document.createElement('span');
    probe.style.fontFamily = style.fontFamily;
    probe.style.fontSize = style.fontSize;
    probe.style.whiteSpace = 'pre';
    probe.style.position = 'absolute';
    probe.textContent = 'x'.repeat(100);
    document.body.appendChild(probe);
    const charWidth = probe.getBoundingClientRect().width / 100;
    probe.remove();

    const tabSize = Number(style.tabSize);
    let drawnColumns = 0;
    for (const char of before.slice(before.lastIndexOf('\n') + 1)) {
      drawnColumns = char === '\t' ? (Math.floor(drawnColumns / tabSize) + 1) * tabSize : drawnColumns + 1;
    }

    const lineHeight = parseFloat(style.lineHeight);
    const top = parseFloat(style.paddingTop) + (line - 1) * lineHeight - panel.scrollTop;
    const left = parseFloat(style.paddingLeft) + drawnColumns * charWidth - panel.scrollLeft;
    return {
      line,
      column,
      shown: top >= 0 && top + lineHeight <= panel.clientHeight && left >= 0 && left <= panel.clientWidth,
    };
  });
}

async function locationOf(item) {
  const [, line, column] = /:(\d+):(\d+)/.exec(await item.locator('.diag-location').textContent());
  return { line: Number(line), column: Number(column) };
}

test.describe('clicking a diagnostic', () => {
  test('opens the source panel with the caret at the location the real pipeline reports, scrolled into view', async ({ page }) => {
    const source = LARGEST_WITH_FINDINGS_NEAR_ITS_END;
    await open(page);
    await page.locator('#source-input').fill(source);
    await page.locator('#render-btn').click();
    const item = page.locator('#diagnostics-list .diag-item', { hasText: 'NoSuchRule' });
    const reported = await locationOf(item);
    expect(reported.line).toBe(lineOf(source, 'rejected(NoSuchRule)'));
    await page.locator('#data-panel-header').click();
    await page.locator('#source-input').evaluate((panel) => {
      panel.setSelectionRange(0, 0);
      panel.scrollTop = 0;
    });
    expect(await caretInView(page)).toEqual({ line: 1, column: 1, shown: true });
    await page.locator('#data-panel-header').click();
    await expect(page.locator('#data-panel')).toHaveClass(/collapsed/);

    await item.click();

    await expect(page.locator('#source-input')).toBeFocused();
    expect(await caretInView(page)).toEqual({ ...reported, shown: true });
  });

  test('scrolls the source panel back up to a location above the rows it shows', async ({ page }) => {
    const source = largestExample().replace('emod = 1\n', 'emod = 1\noops = 1\n');
    expect(source).not.toBe(largestExample());
    await open(page);
    await page.locator('#source-input').fill(source);
    await page.locator('#render-btn').click();
    const item = page.locator('#diagnostics-list .diag-item', { hasText: 'oops' });
    const reported = await locationOf(item);
    expect(reported.line).toBe(2);
    await page.locator('#data-panel-header').click();
    await page.locator('#source-input').evaluate((panel) => {
      panel.setSelectionRange(panel.value.length, panel.value.length);
      panel.scrollTop = panel.scrollHeight;
    });
    expect((await caretInView(page)).shown).toBe(true);

    await item.click();

    expect(await caretInView(page)).toEqual({ ...reported, shown: true });
  });

  test('scrolls the source panel sideways to a location past its width', async ({ page }) => {
    await page.setViewportSize({ width: 900, height: 800 });
    const source = SAMPLE.replace(
      'command -> event:    TakePayment -> PaymentTaken',
      'command -> event:' + ' '.repeat(140) + 'TakePayment -> PaymentRefunded');
    expect(source).not.toBe(SAMPLE);
    await open(page);
    await page.locator('#source-input').fill(source);
    await page.locator('#render-btn').click();
    const item = page.locator('#diagnostics-list .diag-item', { hasText: 'PaymentRefunded' });
    const reported = await locationOf(item);
    expect(reported.column).toBeGreaterThan(150);

    await item.click();

    expect(await caretInView(page)).toEqual({ ...reported, shown: true });
    expect(await page.locator('#source-input').evaluate((panel) => panel.scrollLeft)).toBeGreaterThan(0);

    const nearTheLeft = page.locator('#diagnostics-list .diag-item', { hasText: 'PaymentTaken' });
    const nearTheLeftReported = await locationOf(nearTheLeft);
    await nearTheLeft.click();

    expect(await caretInView(page)).toEqual({ ...nearTheLeftReported, shown: true });
  });

  test('scrolls the source panel sideways to a location that tabs push past its width', async ({ page }) => {
    await page.setViewportSize({ width: 900, height: 800 });
    const source = SAMPLE.replace(
      'command -> event:    TakePayment -> PaymentTaken',
      'command -> event:' + '\t'.repeat(20) + 'TakePayment -> PaymentRefunded');
    expect(source).not.toBe(SAMPLE);
    await open(page);
    await page.locator('#source-input').fill(source);
    await page.locator('#render-btn').click();
    const item = page.locator('#diagnostics-list .diag-item', { hasText: 'PaymentRefunded' });
    const reported = await locationOf(item);

    await item.click();

    expect(await caretInView(page)).toEqual({ ...reported, shown: true });
  });
});

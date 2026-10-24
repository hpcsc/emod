import { test, expect } from '@playwright/test';
import { open, render } from './helpers.js';

// PaymentsView subscribes to four events that four other slices declare, which
// is past the threshold, so the exporter draws a copy of each in its slice.
const VIEW_READING_FAR = `emod = 1

model "Billing" {
}

context "Payments" {
  aggregate "Payment" {
    slice "Take Payment" {
      command "TakePayment" {
        fields {
          amount = required(int)
        }
      }

      event "PaymentTaken" {
        fields {
          amount = required(int)
        }
      }

      flow = <<-FLOW
        command -> event:    TakePayment -> PaymentTaken
      FLOW
    }

    slice "Refund Payment" {
      event "PaymentRefunded" {
        fields {
          amount = required(int)
        }
      }
    }

    slice "Dispute Payment" {
      event "PaymentDisputed" {
        fields {
          amount = required(int)
        }
      }
    }

    slice "Settle Payment" {
      event "PaymentSettled" {
        fields {
          amount = required(int)
        }
      }
    }

    slice "Payment History" {
      view "PaymentsView" {
        subscribes = [PaymentTaken, PaymentRefunded, PaymentDisputed, PaymentSettled]

        fields {
          amount = required(int)
        }
      }
    }
  }
}
`;

const block = (page, nodeId) => page.locator('.diagram-node[data-node-id="' + nodeId + '"]');

function paintOf(locator) {
  return locator.locator('rect').first().evaluate((rect) => {
    const style = getComputedStyle(rect);
    return { fill: style.fill, stroke: style.stroke, dash: style.strokeDasharray };
  });
}

test.describe('a copy of an event', () => {
  test('is drawn for each event a view reads from other slices', async ({ page }) => {
    await open(page);
    await render(page, VIEW_READING_FAR);

    await expect(page.locator('.event-copy-block')).toHaveCount(4);
    await expect(block(page, 'event_copy-1')).toContainText('from Take Payment');
  });

  test("takes the event's stroke, a white fill and a dashed outline", async ({ page }) => {
    await open(page);
    await render(page, VIEW_READING_FAR);

    const copy = await paintOf(block(page, 'event_copy-1'));
    const event = await paintOf(block(page, 'event-1'));

    expect(copy.stroke).toBe(event.stroke);
    expect(copy.fill).toBe('rgb(255, 255, 255)');
    expect(copy.dash).toBe('5px, 3px');
    expect(event.dash).toBe('none');
  });

  test('outlines the event it copies in blue while the pointer is over it', async ({ page }) => {
    await open(page);
    await render(page, VIEW_READING_FAR);

    await block(page, 'event_copy-1').hover();

    expect((await paintOf(block(page, 'event-1'))).stroke).toBe('rgb(52, 152, 219)');
  });

  test('brings the event it copies to the middle of the canvas when clicked', async ({ page }) => {
    await open(page);
    await render(page, VIEW_READING_FAR);

    // The fourth copy stands for PaymentSettled, the fourth event declared.
    await block(page, 'event_copy-4').click();

    const canvas = await page.locator('#diagram-canvas').boundingBox();
    const event = await block(page, 'event-4').locator('rect').first().boundingBox();
    expect(event.x + event.width / 2).toBeCloseTo(canvas.x + canvas.width / 2, -1);
    expect(event.y + event.height / 2).toBeCloseTo(canvas.y + canvas.height / 2, -1);
  });
});

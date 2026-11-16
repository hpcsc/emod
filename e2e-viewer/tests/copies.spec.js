import { test, expect } from '@playwright/test';
import { open, render } from './helpers.js';

// PaymentsView subscribes to four events that the four slices before it
// declare. Three of them are two or more slices away, so the exporter draws a
// copy of each in its slice. PaymentSettled is in the slice next to it.
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

// ChaseLoans reads LoansView three slices away and issues RemindMember two slices
// away, so the exporter draws a copy of each in its slice.
const AUTOMATION_READING_AND_ISSUING_FAR = `emod = 1

model "Library" {
}

context "Lending" {
  aggregate "Loan" {
    slice "Review Loans" {
      view "LoansView" {
        subscribes = [MemberReminded]

        fields {
          memberId = required(string)
        }
      }
    }

    slice "Remind Member" {
      command "RemindMember" {
        fields {
          memberId = required(string)
        }
      }

      event "MemberReminded" {
        fields {
          memberId = required(string)
        }
      }

      flow = <<-FLOW
        command -> event:    RemindMember -> MemberReminded
      FLOW
    }

    slice "Close Loan" {
      command "CloseLoan" {
        fields {
          memberId = required(string)
        }
      }

      event "LoanClosed" {
        fields {
          memberId = required(string)
        }
      }

      flow = <<-FLOW
        command -> event:    CloseLoan -> LoanClosed
      FLOW
    }

    slice "Chase Loans" {
      automation "ChaseLoans" {
        every   = "0 9 * * *"
        reads   = LoansView
        command = RemindMember
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

function opacityOf(page, edgeId) {
  return page.locator('path.arrow[data-edge-id="' + edgeId + '"]').evaluate((arrow) => getComputedStyle(arrow).opacity);
}

test.describe('a copy of an event', () => {
  test('is drawn for each event a view reads from two or more slices away', async ({ page }) => {
    await open(page);
    await render(page, VIEW_READING_FAR);

    await expect(page.locator('.copy-block')).toHaveCount(3);
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

    // The third copy stands for PaymentDisputed, the third event declared.
    await block(page, 'event_copy-3').click();

    const canvas = await page.locator('#diagram-canvas').boundingBox();
    const event = await block(page, 'event-3').locator('rect').first().boundingBox();
    expect(event.x + event.width / 2).toBeCloseTo(canvas.x + canvas.width / 2, -1);
    expect(event.y + event.height / 2).toBeCloseTo(canvas.y + canvas.height / 2, -1);
  });
});

test.describe('a copy of a view and a copy of a command', () => {
  test('are drawn in the slice of the automation that reads and issues them from far away', async ({ page }) => {
    await open(page);
    await render(page, AUTOMATION_READING_AND_ISSUING_FAR);

    await expect(block(page, 'view_copy-1')).toContainText('from Review Loans');
    await expect(block(page, 'command_copy-1')).toContainText('in Remind Member');
  });

  test('take the stroke of a view and of a command, a white fill and a dashed outline', async ({ page }) => {
    await open(page);
    await render(page, AUTOMATION_READING_AND_ISSUING_FAR);

    const viewCopy = await paintOf(block(page, 'view_copy-1'));
    const commandCopy = await paintOf(block(page, 'command_copy-1'));

    expect(viewCopy.stroke).toBe((await paintOf(block(page, 'view-1'))).stroke);
    expect(commandCopy.stroke).toBe((await paintOf(block(page, 'command-1'))).stroke);
    expect([viewCopy.fill, commandCopy.fill]).toEqual(['rgb(255, 255, 255)', 'rgb(255, 255, 255)']);
    expect([viewCopy.dash, commandCopy.dash]).toEqual(['5px, 3px', '5px, 3px']);
  });

  test('join the automation with its reads arrow and its command arrow', async ({ page }) => {
    await open(page);
    await render(page, AUTOMATION_READING_AND_ISSUING_FAR);

    await expect(page.locator('path.arrow[data-edge-id="view_copy-1--auto-1"]')).toHaveCount(1);
    await expect(page.locator('path.arrow[data-edge-id="auto-1--command_copy-1"]')).toHaveCount(1);
  });
});

test.describe('the arrows around a view', () => {
  test('dim where they do not touch the view under the pointer', async ({ page }) => {
    await open(page);
    await render(page, VIEW_READING_FAR);

    await block(page, 'view-1').hover();

    expect(await opacityOf(page, 'command-1--event-1')).toBe('0.15');
    expect(await opacityOf(page, 'event_copy-1--view-1')).toBe('1');
  });

  test('stay at full strength while no view is under the pointer', async ({ page }) => {
    await open(page);
    await render(page, VIEW_READING_FAR);

    await block(page, 'command-1').hover();

    expect(await opacityOf(page, 'command-1--event-1')).toBe('1');
  });
});

//go:build unit

package query_test

import (
	"errors"
	"testing"

	"github.com/hpcsc/emod/internal/query"
	"github.com/stretchr/testify/require"
)

const hotelModel = `context "Hotel" {
  aggregate "Stay" {
    invariants {
      RoomFree = "A room holds one booking at a time"
    }

    slice "Book Room" {
      trigger "Booking Form" {
        reads = RoomsView
      }

      command "BookRoom" {
      }

      event "RoomBooked" {
      }

      flow = <<-FLOW
        command -> event:    BookRoom -> RoomBooked
        command -> rejected: BookRoom -> RoomFree
      FLOW
    }

    slice "Show Rooms" {
      view "RoomsView" {
        subscribes = [RoomBooked]
      }
    }

    slice "Confirm Booking" {
      automation "Confirmer" {
        on      = RoomBooked
        after   = "15m"
        command = SendConfirmation

        target {
          context = Mail
        }
      }
    }

    slice "Sweep Bookings" {
      automation "Sweeper" {
        every   = "0 2 * * *"
        reads   = RoomsView
        command = ReleaseRoom
      }

      command "ReleaseRoom" {
      }

      event "RoomReleased" {
      }

      flow = <<-FLOW
        command -> event: ReleaseRoom -> RoomReleased
      FLOW
    }

    slice "Import Booking" {
      translation "PartnerImport" {
        external_system = "Partner API"
        command         = BookRoom

        event "BookingImported" {
        }
      }
    }
  }
}

context "Mail" {
  aggregate "Message" {
    slice "Send Confirmation" {
      command "SendConfirmation" {
      }

      event "ConfirmationSent" {
      }

      flow = <<-FLOW
        command -> event: SendConfirmation -> ConfirmationSent
      FLOW
    }
  }
}
`

type walked struct {
	Label string
	After string
	Name  string
	From  string
	Depth int
}

func walkOf(tree query.Tree) []walked {
	steps := make([]walked, len(tree.Steps))
	for i, step := range tree.Steps {
		steps[i] = walked{Label: step.Label, After: step.After, Name: step.Name, From: step.From.Name, Depth: step.Depth}
	}
	return steps
}

func TestTrace(t *testing.T) {
	t.Run("what follows", func(t *testing.T) {
		t.Run("lists each element one time, breadth-first, under the first element that reaches it", func(t *testing.T) {
			tree, err := query.Trace(parse(t, hotelModel), "RoomBooked", query.TraceOptions{})

			require.NoError(t, err)
			require.Equal(t, []walked{
				{Label: "subscribes", Name: "RoomsView", From: "RoomBooked", Depth: 1},
				{Label: "on", After: "15m", Name: "Confirmer", From: "RoomBooked", Depth: 1},
				{Label: "reads", Name: "Booking Form", From: "RoomsView", Depth: 2},
				{Label: "reads", Name: "Sweeper", From: "RoomsView", Depth: 2},
				{Label: "command", Name: "SendConfirmation", From: "Confirmer", Depth: 2},
				{Label: "issues", Name: "BookRoom", From: "Booking Form", Depth: 3},
				{Label: "command", Name: "ReleaseRoom", From: "Sweeper", Depth: 3},
				{Label: "flow", Name: "ConfirmationSent", From: "SendConfirmation", Depth: 3},
				{Label: "rejected", Name: "RoomFree", From: "BookRoom", Depth: 4},
				{Label: "flow", Name: "BookingImported", From: "BookRoom", Depth: 4},
				{Label: "flow", Name: "RoomReleased", From: "ReleaseRoom", Depth: 4},
			}, walkOf(tree))
		})

		t.Run("a step carries the kind, the place and the line of the element, the element that reaches it, and a step into another context", func(t *testing.T) {
			tree, err := query.Trace(parse(t, hotelModel), "Confirmer", query.TraceOptions{})

			require.NoError(t, err)
			require.Equal(t, query.Tree{
				Root: query.Node{
					Element:   query.Element{Kind: query.KindAutomation, Name: "Confirmer"},
					Context:   "Hotel",
					Aggregate: "Stay",
					Slice:     "Confirm Booking",
					Line:      31,
				},
				Steps: []query.Step{
					{
						Label: "command",
						Node: query.Node{
							Element:   query.Element{Kind: query.KindCommand, Name: "SendConfirmation"},
							Context:   "Mail",
							Aggregate: "Message",
							Slice:     "Send Confirmation",
							Line:      75,
						},
						From:           query.Element{Kind: query.KindAutomation, Name: "Confirmer"},
						CrossesContext: true,
						Depth:          1,
					},
					{
						Label: "flow",
						Node: query.Node{
							Element:   query.Element{Kind: query.KindEvent, Name: "ConfirmationSent"},
							Context:   "Mail",
							Aggregate: "Message",
							Slice:     "Send Confirmation",
							Line:      78,
						},
						From:  query.Element{Kind: query.KindCommand, Name: "SendConfirmation"},
						Depth: 2,
					},
				},
			}, tree)
		})

		t.Run("the walk reaches an invariant through rejected, and starts at an external system through from", func(t *testing.T) {
			tree, err := query.Trace(parse(t, hotelModel), "Partner API", query.TraceOptions{NoViews: true})

			require.NoError(t, err)
			require.Equal(t, []walked{
				{Label: "from", Name: "PartnerImport", From: "Partner API", Depth: 1},
				{Label: "command", Name: "BookRoom", From: "PartnerImport", Depth: 2},
				{Label: "flow", Name: "RoomBooked", From: "BookRoom", Depth: 3},
				{Label: "rejected", Name: "RoomFree", From: "BookRoom", Depth: 3},
				{Label: "flow", Name: "BookingImported", From: "BookRoom", Depth: 3},
				{Label: "on", After: "15m", Name: "Confirmer", From: "RoomBooked", Depth: 4},
				{Label: "command", Name: "SendConfirmation", From: "Confirmer", Depth: 5},
				{Label: "flow", Name: "ConfirmationSent", From: "SendConfirmation", Depth: 6},
			}, walkOf(tree))
		})

		t.Run("an element that nothing follows gives no step", func(t *testing.T) {
			tree, err := query.Trace(parse(t, hotelModel), "ConfirmationSent", query.TraceOptions{})

			require.NoError(t, err)
			require.Equal(t, []query.Step{}, tree.Steps)
		})
	})

	t.Run("upstream", func(t *testing.T) {
		t.Run("lists what leads to the element, through the edges backward", func(t *testing.T) {
			tree, err := query.Trace(parse(t, hotelModel), "ConfirmationSent", query.TraceOptions{Upstream: true})

			require.NoError(t, err)
			require.Equal(t, []walked{
				{Label: "flow", Name: "SendConfirmation", From: "ConfirmationSent", Depth: 1},
				{Label: "command", Name: "Confirmer", From: "SendConfirmation", Depth: 2},
				{Label: "on", After: "15m", Name: "RoomBooked", From: "Confirmer", Depth: 3},
				{Label: "flow", Name: "BookRoom", From: "RoomBooked", Depth: 4},
				{Label: "issues", Name: "Booking Form", From: "BookRoom", Depth: 5},
				{Label: "command", Name: "PartnerImport", From: "BookRoom", Depth: 5},
				{Label: "reads", Name: "RoomsView", From: "Booking Form", Depth: 6},
				{Label: "from", Name: "Partner API", From: "PartnerImport", Depth: 6},
			}, walkOf(tree))
		})
	})

	t.Run("views", func(t *testing.T) {
		t.Run("the walk follows the reads of each entry of a trigger's list", func(t *testing.T) {
			const checkoutModel = `context "Shop" {
  slice "Show Cart" {
    event "CartUpdated" {
    }

    event "DiscountApplied" {
    }

    view "CartView" {
      subscribes = [CartUpdated]
    }

    view "DiscountView" {
      subscribes = [DiscountApplied]
    }
  }

  slice "Apply Discount" {
    trigger "Checkout Page" {
      reads = [CartView, DiscountView]
    }
  }
}
`
			tree, err := query.Trace(parse(t, checkoutModel), "DiscountApplied", query.TraceOptions{})

			require.NoError(t, err)
			require.Equal(t, []walked{
				{Label: "subscribes", Name: "DiscountView", From: "DiscountApplied", Depth: 1},
				{Label: "reads", Name: "Checkout Page", From: "DiscountView", Depth: 2},
			}, walkOf(tree))
		})

		t.Run("no views leaves out the subscribes and reads edges, so the walk misses the scheduled automation", func(t *testing.T) {
			tree, err := query.Trace(parse(t, hotelModel), "RoomBooked", query.TraceOptions{NoViews: true})

			require.NoError(t, err)
			require.Equal(t, []walked{
				{Label: "on", After: "15m", Name: "Confirmer", From: "RoomBooked", Depth: 1},
				{Label: "command", Name: "SendConfirmation", From: "Confirmer", Depth: 2},
				{Label: "flow", Name: "ConfirmationSent", From: "SendConfirmation", Depth: 3},
			}, walkOf(tree))
		})
	})

	t.Run("lookup", func(t *testing.T) {
		const twoKindsModel = `context "Desk" {
  slice "Check In" {
    command "CheckIn" {
    }

    event "CheckedIn" {
    }

    flow = <<-FLOW
      command -> event: CheckIn -> CheckedIn
    FLOW
  }

  slice "Log Check In" {
    event "CheckIn" {
    }
  }
}
`

		t.Run("a name that two kinds declare stops with each match", func(t *testing.T) {
			_, err := query.Trace(parse(t, twoKindsModel), "CheckIn", query.TraceOptions{})

			var ambiguous *query.AmbiguousNameError
			require.True(t, errors.As(err, &ambiguous))
			require.Equal(t, []query.Kind{query.KindCommand, query.KindEvent}, ambiguous.Kinds())
			require.EqualError(t, err, `the model declares "CheckIn" more than one time: as command in Desk, as event in Desk`)
		})

		t.Run("a kind picks the element of that kind", func(t *testing.T) {
			tree, err := query.Trace(parse(t, twoKindsModel), "CheckIn", query.TraceOptions{Kind: query.KindCommand})

			require.NoError(t, err)
			require.Equal(t, []walked{
				{Label: "flow", Name: "CheckedIn", From: "CheckIn", Depth: 1},
			}, walkOf(tree))
		})

		t.Run("an invariant that two scopes declare stops with each scope, also with its kind", func(t *testing.T) {
			model := parse(t, `context "Stays" {
  invariants {
    Open = "The hotel takes stays only while it is open"
  }

  aggregate "Room" {
    invariants {
      Open = "A room takes stays only while it is open"
    }
  }
}
`)

			_, err := query.Trace(model, "Open", query.TraceOptions{Kind: query.KindInvariant})

			require.EqualError(t, err, `the model declares "Open" more than one time: as invariant in Stays, as invariant in Stays > Room`)
		})

		t.Run("a name that the model does not declare lists the closest element names", func(t *testing.T) {
			_, err := query.Trace(parse(t, hotelModel), "Sweepr", query.TraceOptions{})

			require.EqualError(t, err, `the model declares no element named "Sweepr"; the closest names are Sweeper`)
		})

		t.Run("a name of another kind than the one asked for names the kind that declares it", func(t *testing.T) {
			_, err := query.Trace(parse(t, hotelModel), "Sweeper", query.TraceOptions{Kind: query.KindCommand})

			require.EqualError(t, err, `the model declares "Sweeper" as automation, not as command`)
		})
	})
}

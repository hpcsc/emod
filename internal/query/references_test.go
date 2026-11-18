//go:build unit

package query_test

import (
	"testing"

	"github.com/hpcsc/emod/examples"
	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/lsp"
	"github.com/hpcsc/emod/internal/oracle"
	"github.com/hpcsc/emod/internal/query"
	"github.com/stretchr/testify/require"
)

func parse(t *testing.T, source string) *ast.Model {
	t.Helper()
	model, diagnostics := oracle.Parse(source, "model.emod")
	require.Empty(t, diagnostics)
	return model
}

func element(kind query.Kind, name string) *query.Element {
	return &query.Element{Kind: kind, Name: name}
}

const bookingModel = `context "Booking" {
  aggregate "Room" {
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

      spec "books a free room" {
        when = BookRoom
        then = [RoomBooked]
      }

      spec "refuses a booked room" {
        given = [RoomBooked]
        when  = BookRoom
        then  = rejected(RoomFree)
      }
    }

    slice "Show Rooms" {
      view "RoomsView" {
        subscribes = [RoomBooked]
      }

      spec "shows a booked room" {
        given = [RoomBooked]
        then  = view(RoomsView)
      }
    }

    slice "Confirm Booking" {
      automation "Confirmer" {
        on      = RoomBooked
        reads   = RoomsView
        command = BookRoom
      }

      spec "books again on a booking" {
        when = RoomBooked
        then = command(BookRoom)
      }
    }

    slice "Import Booking" {
      translation "PartnerImport" {
        external_system = "Partner"
        reads           = RoomsView
        command         = BookRoom

        event "BookingImported" {
        }
      }
    }
  }
}
`

func TestReferences(t *testing.T) {
	t.Run("event", func(t *testing.T) {
		t.Run("groups the flow that produces it, the view and the automation that read it, and the specs that name it", func(t *testing.T) {
			found, err := query.References(parse(t, bookingModel), "RoomBooked", "")

			require.NoError(t, err)
			require.Equal(t, []query.NameReferences{{
				Kind:      query.KindEvent,
				Name:      "RoomBooked",
				Context:   "Booking",
				Aggregate: "Room",
				Slice:     "Book Room",
				Line:      15,
				References: []query.Reference{
					{Group: query.GroupProducedBy, Keyword: "flow", Line: 19, Element: element(query.KindCommand, "BookRoom"), Slice: "Book Room"},
					{Group: query.GroupNamedInSpecs, Keyword: "then", Line: 25, Slice: "Book Room", Spec: "books a free room"},
					{Group: query.GroupNamedInSpecs, Keyword: "given", Line: 29, Slice: "Book Room", Spec: "refuses a booked room"},
					{Group: query.GroupReadBy, Keyword: "subscribes", Line: 37, Element: element(query.KindView, "RoomsView"), Slice: "Show Rooms"},
					{Group: query.GroupNamedInSpecs, Keyword: "given", Line: 41, Slice: "Show Rooms", Spec: "shows a booked room"},
					{Group: query.GroupReadBy, Keyword: "on", Line: 48, Element: element(query.KindAutomation, "Confirmer"), Slice: "Confirm Booking"},
					{Group: query.GroupNamedInSpecs, Keyword: "when", Line: 54, Slice: "Confirm Booking", Spec: "books again on a booking"},
				},
			}}, found)
		})

		t.Run("a translation produces the event that it declares, through the command that it names", func(t *testing.T) {
			found, err := query.References(parse(t, bookingModel), "BookingImported", "")

			require.NoError(t, err)
			require.Equal(t, []query.NameReferences{{
				Kind:      query.KindEvent,
				Name:      "BookingImported",
				Context:   "Booking",
				Aggregate: "Room",
				Slice:     "Import Booking",
				Line:      65,
				References: []query.Reference{
					{Group: query.GroupProducedBy, Keyword: "translation", Line: 63, Element: element(query.KindCommand, "BookRoom"), Slice: "Import Booking"},
				},
			}}, found)
		})
	})

	t.Run("command", func(t *testing.T) {
		t.Run("groups who issues it, what it produces, the invariant that refuses it, and the specs that name it", func(t *testing.T) {
			found, err := query.References(parse(t, bookingModel), "BookRoom", "")

			require.NoError(t, err)
			require.Equal(t, []query.NameReferences{{
				Kind:      query.KindCommand,
				Name:      "BookRoom",
				Context:   "Booking",
				Aggregate: "Room",
				Slice:     "Book Room",
				Line:      12,
				References: []query.Reference{
					{Group: query.GroupIssuedBy, Keyword: "issues", Line: 8, Element: element(query.KindScreen, "Booking Form"), Slice: "Book Room"},
					{Group: query.GroupProduces, Keyword: "flow", Line: 19, Element: element(query.KindEvent, "RoomBooked"), Slice: "Book Room"},
					{Group: query.GroupRefusedBy, Keyword: "rejected", Line: 20, Element: element(query.KindInvariant, "RoomFree"), Slice: "Book Room"},
					{Group: query.GroupNamedInSpecs, Keyword: "when", Line: 24, Slice: "Book Room", Spec: "books a free room"},
					{Group: query.GroupNamedInSpecs, Keyword: "when", Line: 30, Slice: "Book Room", Spec: "refuses a booked room"},
					{Group: query.GroupIssuedBy, Keyword: "command", Line: 50, Element: element(query.KindAutomation, "Confirmer"), Slice: "Confirm Booking"},
					{Group: query.GroupNamedInSpecs, Keyword: "then", Line: 55, Slice: "Confirm Booking", Spec: "books again on a booking"},
					{Group: query.GroupIssuedBy, Keyword: "command", Line: 63, Element: element(query.KindTranslation, "PartnerImport"), Slice: "Import Booking"},
					{Group: query.GroupProduces, Keyword: "translation", Line: 63, Element: element(query.KindEvent, "BookingImported"), Slice: "Import Booking"},
				},
			}}, found)
		})
	})

	t.Run("view", func(t *testing.T) {
		t.Run("groups the events that build it, the trigger, the automation and the translation that read it, and the specs that name it", func(t *testing.T) {
			found, err := query.References(parse(t, bookingModel), "RoomsView", "")

			require.NoError(t, err)
			require.Equal(t, []query.NameReferences{{
				Kind:      query.KindView,
				Name:      "RoomsView",
				Context:   "Booking",
				Aggregate: "Room",
				Slice:     "Show Rooms",
				Line:      36,
				References: []query.Reference{
					{Group: query.GroupReadBy, Keyword: "reads", Line: 9, Element: element(query.KindScreen, "Booking Form"), Slice: "Book Room"},
					{Group: query.GroupBuiltFrom, Keyword: "subscribes", Line: 37, Element: element(query.KindEvent, "RoomBooked"), Slice: "Show Rooms"},
					{Group: query.GroupNamedInSpecs, Keyword: "then", Line: 42, Slice: "Show Rooms", Spec: "shows a booked room"},
					{Group: query.GroupReadBy, Keyword: "reads", Line: 49, Element: element(query.KindAutomation, "Confirmer"), Slice: "Confirm Booking"},
					{Group: query.GroupReadBy, Keyword: "reads", Line: 62, Element: element(query.KindTranslation, "PartnerImport"), Slice: "Import Booking"},
				},
			}}, found)
		})
	})

	t.Run("shows", func(t *testing.T) {
		const checkoutModel = `screen "CheckoutPage" {
}

screen "CartDrawer" {
}

context "Shop" {
  slice "Show Cart" {
    view "CartView" {
    }
  }

  slice "Apply Discount" {
    shows {
      screen  = CheckoutPage
      reads   = CartView
      command = ApplyDiscount
    }

    command "ApplyDiscount" {
    }

    command "ClearCart" {
    }
  }

  slice "Reapply Discount" {
    shows {
      screen = CartDrawer
    }

    shows {
      screen  = CheckoutPage
      reads   = [OrderView, CartView]
      command = [RemoveDiscount, ApplyDiscount]
    }
  }
}
`

		t.Run("a command lists each shows block that names it in its command as an issuer, at the line of that entry", func(t *testing.T) {
			found, err := query.References(parse(t, checkoutModel), "ApplyDiscount", "")

			require.NoError(t, err)
			require.Equal(t, []query.NameReferences{{
				Kind:    query.KindCommand,
				Name:    "ApplyDiscount",
				Context: "Shop",
				Slice:   "Apply Discount",
				Line:    20,
				References: []query.Reference{
					{Group: query.GroupIssuedBy, Keyword: "command", Line: 17, Element: element(query.KindScreen, "CheckoutPage"), Slice: "Apply Discount"},
					{Group: query.GroupIssuedBy, Keyword: "command", Line: 35, Element: element(query.KindScreen, "CheckoutPage"), Slice: "Reapply Discount"},
				},
			}}, found)
		})

		t.Run("a command that no shows block names lists no screen, also when its slice shows one", func(t *testing.T) {
			found, err := query.References(parse(t, checkoutModel), "ClearCart", "")

			require.NoError(t, err)
			require.Equal(t, []query.NameReferences{{
				Kind:       query.KindCommand,
				Name:       "ClearCart",
				Context:    "Shop",
				Slice:      "Apply Discount",
				Line:       23,
				References: []query.Reference{},
			}}, found)
		})

		t.Run("a view lists each shows block that names it in its reads as a reader, at the line of that entry", func(t *testing.T) {
			found, err := query.References(parse(t, checkoutModel), "CartView", "")

			require.NoError(t, err)
			require.Equal(t, []query.NameReferences{{
				Kind:    query.KindView,
				Name:    "CartView",
				Context: "Shop",
				Slice:   "Show Cart",
				Line:    9,
				References: []query.Reference{
					{Group: query.GroupReadBy, Keyword: "reads", Line: 16, Element: element(query.KindScreen, "CheckoutPage"), Slice: "Apply Discount"},
					{Group: query.GroupReadBy, Keyword: "reads", Line: 34, Element: element(query.KindScreen, "CheckoutPage"), Slice: "Reapply Discount"},
				},
			}}, found)
		})
	})

	t.Run("trigger", func(t *testing.T) {
		const checkoutModel = `context "Shop" {
  slice "Show Cart" {
    view "CartView" {
    }

    view "DiscountView" {
    }
  }

  slice "Apply Discount" {
    trigger "Checkout Page" {
      reads = [
        CartView,
        DiscountView,
      ]
    }
  }
}
`
		for _, tc := range []struct {
			view      string
			line      int
			entryLine int
		}{
			{view: "CartView", line: 3, entryLine: 13},
			{view: "DiscountView", line: 6, entryLine: 14},
		} {
			t.Run("each view of a reads list lists the trigger among its readers: "+tc.view, func(t *testing.T) {
				found, err := query.References(parse(t, checkoutModel), tc.view, "")

				require.NoError(t, err)
				require.Equal(t, []query.NameReferences{{
					Kind:    query.KindView,
					Name:    tc.view,
					Context: "Shop",
					Slice:   "Show Cart",
					Line:    tc.line,
					References: []query.Reference{
						{Group: query.GroupReadBy, Keyword: "reads", Line: tc.entryLine, Element: element(query.KindScreen, "Checkout Page"), Slice: "Apply Discount"},
					},
				}}, found)
			})
		}
	})

	t.Run("invariant", func(t *testing.T) {
		t.Run("groups the commands that it refuses and the specs that expect the refusal", func(t *testing.T) {
			found, err := query.References(parse(t, bookingModel), "RoomFree", "")

			require.NoError(t, err)
			require.Equal(t, []query.NameReferences{{
				Kind:      query.KindInvariant,
				Name:      "RoomFree",
				Context:   "Booking",
				Aggregate: "Room",
				Line:      4,
				References: []query.Reference{
					{Group: query.GroupRefusedCommands, Keyword: "rejected", Line: 20, Element: element(query.KindCommand, "BookRoom"), Slice: "Book Room"},
					{Group: query.GroupNamedInSpecs, Keyword: "then", Line: 31, Slice: "Book Room", Spec: "refuses a booked room"},
				},
			}}, found)
		})

		t.Run("one name in a context and in its aggregate gives each scope with only the references that resolve to it", func(t *testing.T) {
			model := parse(t, `context "Stays" {
  invariants {
    Open = "The hotel takes stays only while it is open"
  }

  slice "Plan Stay" {
    command "PlanStay" {
    }

    flow = <<-FLOW
      command -> rejected: PlanStay -> Open
    FLOW
  }

  aggregate "Room" {
    invariants {
      Open = "A room takes stays only while it is open"
    }

    slice "Book Room" {
      command "BookRoom" {
      }

      flow = <<-FLOW
        command -> rejected: BookRoom -> Open
      FLOW
    }
  }
}
`)

			found, err := query.References(model, "Open", "")

			require.NoError(t, err)
			require.Equal(t, []query.NameReferences{
				{
					Kind:    query.KindInvariant,
					Name:    "Open",
					Context: "Stays",
					Line:    3,
					References: []query.Reference{
						{Group: query.GroupRefusedCommands, Keyword: "rejected", Line: 11, Element: element(query.KindCommand, "PlanStay"), Slice: "Plan Stay"},
					},
				},
				{
					Kind:      query.KindInvariant,
					Name:      "Open",
					Context:   "Stays",
					Aggregate: "Room",
					Line:      17,
					References: []query.Reference{
						{Group: query.GroupRefusedCommands, Keyword: "rejected", Line: 25, Element: element(query.KindCommand, "BookRoom"), Slice: "Book Room"},
					},
				},
			}, found)
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

		t.Run("a name that two kinds declare gives one result for each kind, in source order", func(t *testing.T) {
			found, err := query.References(parse(t, twoKindsModel), "CheckIn", "")

			require.NoError(t, err)
			require.Equal(t, []query.NameReferences{
				{
					Kind:    query.KindCommand,
					Name:    "CheckIn",
					Context: "Desk",
					Slice:   "Check In",
					Line:    3,
					References: []query.Reference{
						{Group: query.GroupProduces, Keyword: "flow", Line: 10, Element: element(query.KindEvent, "CheckedIn"), Slice: "Check In"},
					},
				},
				{
					Kind:       query.KindEvent,
					Name:       "CheckIn",
					Context:    "Desk",
					Slice:      "Log Check In",
					Line:       15,
					References: []query.Reference{},
				},
			}, found)
		})

		t.Run("a kind picks the declaration of that kind only", func(t *testing.T) {
			found, err := query.References(parse(t, twoKindsModel), "CheckIn", query.KindEvent)

			require.NoError(t, err)
			require.Equal(t, []query.NameReferences{{
				Kind:       query.KindEvent,
				Name:       "CheckIn",
				Context:    "Desk",
				Slice:      "Log Check In",
				Line:       15,
				References: []query.Reference{},
			}}, found)
		})

		t.Run("a name that the model does not declare lists the declared names closest to it", func(t *testing.T) {
			_, err := query.References(parse(t, bookingModel), "RoomBoked", "")

			require.EqualError(t, err, `the model declares no command, event, view or invariant named "RoomBoked"; the closest names are RoomBooked`)
		})

		t.Run("a part of a name lists the names that hold it, nearest first", func(t *testing.T) {
			_, err := query.References(parse(t, bookingModel), "room", "")

			require.EqualError(t, err, `the model declares no command, event, view or invariant named "room"; the closest names are BookRoom, RoomFree, RoomsView`)
		})

		t.Run("a name far from each declared name lists no closest name", func(t *testing.T) {
			_, err := query.References(parse(t, bookingModel), "SendInvoiceToAccounting", "")

			require.EqualError(t, err, `the model declares no command, event, view or invariant named "SendInvoiceToAccounting"`)
		})

		t.Run("a name of another kind than the one asked for names the kind that declares it", func(t *testing.T) {
			_, err := query.References(parse(t, bookingModel), "RoomBooked", query.KindView)

			require.EqualError(t, err, `the model declares "RoomBooked" as event, not as view`)
		})
	})

	t.Run("agreement with the editor", func(t *testing.T) {
		t.Run("each place that find-references returns for a name of an example is a reference of that name", func(t *testing.T) {
			for _, example := range examples.All() {
				uri := "file:///" + example.Name + ".emod"
				model := parse(t, example.Source)

				type cursorOn struct {
					name string
					pos  ast.Position
				}
				var names []cursorOn
				for _, decl := range query.Declarations(model) {
					if decl.Kind != query.KindContext {
						names = append(names, cursorOn{name: decl.Name, pos: decl.Pos})
					}
				}
				for _, scope := range query.InvariantScopes(model) {
					for _, inv := range scope.Declared {
						names = append(names, cursorOn{name: inv.Name, pos: inv.NamePos})
					}
				}

				for _, n := range names {
					found, err := query.References(model, n.name, "")
					require.NoError(t, err)

					var lines []int
					for _, f := range found {
						lines = append(lines, f.Line)
						for _, ref := range f.References {
							lines = append(lines, ref.Line)
						}
					}

					locations := lsp.GetReferences(example.Source, n.pos.Line-1, n.pos.Column-1, uri)
					require.NotEmpty(t, locations, "%s: find-references found nothing for %s", example.Name, n.name)
					for _, loc := range locations {
						require.Contains(t, lines, loc.Range.Start.Line+1,
							"%s: find-references returns line %d for %s, and emod references does not", example.Name, loc.Range.Start.Line+1, n.name)
					}
				}
			}
		})
	})
}

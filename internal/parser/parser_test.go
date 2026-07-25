//go:build unit

package parser_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/parser"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
)

var ignoreCommentPositions = cmpopts.IgnoreTypes(ast.Position{})

func TestParser(t *testing.T) {
	t.Run("model and actors", func(t *testing.T) {
		t.Run("model declaration", func(t *testing.T) {
			input := `model "Test Model"`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			require.Equal(t, "Test Model", model.Name)
		})

		t.Run("actor declaration", func(t *testing.T) {
			input := `model "Test"
actor "Guest"`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			require.Len(t, model.Actors, 1)
			require.Equal(t, "Guest", model.Actors[0].Name)
		})

		t.Run("multiple actors", func(t *testing.T) {
			input := `model "Test"
actor "Guest"
actor "FrontDesk"`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			require.Len(t, model.Actors, 2)
			require.Equal(t, "Guest", model.Actors[0].Name)
			require.Equal(t, "FrontDesk", model.Actors[1].Name)
		})
	})

	t.Run("contexts, aggregates and slices", func(t *testing.T) {
		t.Run("context with aggregate", func(t *testing.T) {
			input := `model "Test"
context "Reservations" {
  aggregate "Reservation" {
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			require.Len(t, model.Contexts, 1)
			require.Equal(t, "Reservations", model.Contexts[0].Name)
			require.Len(t, model.Contexts[0].Aggregates, 1)
			require.Equal(t, "Reservation", model.Contexts[0].Aggregates[0].Name)
		})

		t.Run("context without mode clause (backward compatible)", func(t *testing.T) {
			input := `model "Test"
context "Reservations" {
  aggregate "Reservation" {
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			require.Len(t, model.Contexts, 1)
			require.Equal(t, "Reservations", model.Contexts[0].Name)
			require.Empty(t, model.Contexts[0].Mode)
			require.Len(t, model.Contexts[0].Aggregates, 1)
		})

		t.Run("context with mode dcb", func(t *testing.T) {
			input := `model "Test"
context "Ctx" mode dcb {
  slice "Slice" {
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			require.Len(t, model.Contexts, 1)
			require.Equal(t, "Ctx", model.Contexts[0].Name)
			require.Equal(t, "dcb", model.Contexts[0].Mode)
			require.Len(t, model.Contexts[0].Slices, 1)
			require.Equal(t, "Slice", model.Contexts[0].Slices[0].Name)
		})

		t.Run("context with mode aggregate", func(t *testing.T) {
			input := `model "Test"
context "Ctx" mode aggregate {
  aggregate "Agg" {
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			require.Len(t, model.Contexts, 1)
			require.Equal(t, "Ctx", model.Contexts[0].Name)
			require.Equal(t, "aggregate", model.Contexts[0].Mode)
			require.Len(t, model.Contexts[0].Aggregates, 1)
			require.Equal(t, "Agg", model.Contexts[0].Aggregates[0].Name)
		})

		t.Run("context with mode mixed", func(t *testing.T) {
			input := `model "Test"
context "Ctx" mode mixed {
  aggregate "Agg" {
  }
  slice "Slice" {
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			require.Len(t, model.Contexts, 1)
			require.Equal(t, "Ctx", model.Contexts[0].Name)
			require.Equal(t, "mixed", model.Contexts[0].Mode)
			require.Len(t, model.Contexts[0].Aggregates, 1)
			require.Equal(t, "Agg", model.Contexts[0].Aggregates[0].Name)
			require.Len(t, model.Contexts[0].Slices, 1)
			require.Equal(t, "Slice", model.Contexts[0].Slices[0].Name)
		})

		t.Run("context with slice directly (no aggregate)", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  slice "Slice" {
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			require.Len(t, model.Contexts, 1)
			require.Equal(t, "Ctx", model.Contexts[0].Name)
			require.Empty(t, model.Contexts[0].Mode)
			require.Len(t, model.Contexts[0].Slices, 1)
			require.Equal(t, "Slice", model.Contexts[0].Slices[0].Name)
			require.Empty(t, model.Contexts[0].Aggregates)
		})

		t.Run("context with both aggregate and slice", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
  }
  slice "Slice" {
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			require.Len(t, model.Contexts, 1)
			require.Equal(t, "Ctx", model.Contexts[0].Name)
			require.Empty(t, model.Contexts[0].Mode)
			require.Len(t, model.Contexts[0].Aggregates, 1)
			require.Equal(t, "Agg", model.Contexts[0].Aggregates[0].Name)
			require.Len(t, model.Contexts[0].Slices, 1)
			require.Equal(t, "Slice", model.Contexts[0].Slices[0].Name)
		})

		t.Run("context with mode dcb and slice with content", func(t *testing.T) {
			input := `model "Test"
context "Ctx" mode dcb {
  slice "Slice" {
    command DoThing {
      fields {
        id string
      }
    }
    event ThingDone {
      fields {
        id string
      }
    }
    flow {
      command -> event: DoThing -> ThingDone
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			require.Equal(t, "dcb", model.Contexts[0].Mode)
			require.Len(t, model.Contexts[0].Slices, 1)
			slice := model.Contexts[0].Slices[0]
			require.Equal(t, "Slice", slice.Name)
			require.Len(t, slice.Commands, 1)
			require.Equal(t, "DoThing", slice.Commands[0].Name)
			require.Len(t, slice.Events, 1)
			require.Equal(t, "ThingDone", slice.Events[0].Name)
			require.Len(t, slice.Flows, 1)
		})

		t.Run("aggregate with slice", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			require.Len(t, model.Contexts[0].Aggregates[0].Slices, 1)
			require.Equal(t, "Slice", model.Contexts[0].Aggregates[0].Slices[0].Name)
		})
	})

	t.Run("commands, events and flows", func(t *testing.T) {
		t.Run("command in slice", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command TestCommand {
        fields {
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			slice := model.Contexts[0].Aggregates[0].Slices[0]
			require.Len(t, slice.Commands, 1)
			require.Equal(t, "TestCommand", slice.Commands[0].Name)
		})

		t.Run("event in slice", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      event TestEvent {
        fields {
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			slice := model.Contexts[0].Aggregates[0].Slices[0]
			require.Len(t, slice.Events, 1)
			require.Equal(t, "TestEvent", slice.Events[0].Name)
		})

		t.Run("fields in command", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command TestCommand {
        fields {
          fieldOne string required
          fieldTwo int optional
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			cmd := model.Contexts[0].Aggregates[0].Slices[0].Commands[0]
			require.Len(t, cmd.Fields, 2)
			require.Equal(t, "fieldOne", cmd.Fields[0].Name)
			require.Equal(t, "string", cmd.Fields[0].Type)
			require.Equal(t, "required", cmd.Fields[0].Modifier)
			require.Equal(t, "fieldTwo", cmd.Fields[1].Name)
			require.Equal(t, "int", cmd.Fields[1].Type)
			require.Equal(t, "optional", cmd.Fields[1].Modifier)
		})

		t.Run("flow in slice", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      flow {
        command -> event: TestCommand -> TestEvent
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			slice := model.Contexts[0].Aggregates[0].Slices[0]
			require.Len(t, slice.Flows, 1)
			require.Equal(t, "TestCommand", slice.Flows[0].CommandName)
			require.Equal(t, "TestEvent", slice.Flows[0].EventName)
		})

		t.Run("complete sample", func(t *testing.T) {
			input := `# Hotel Reservation System
model "Hotel Reservation"

actor "Guest"

context "Reservations" {
  aggregate "Reservation" {
    slice "Make Reservation" {
      command MakeReservation {
        fields {
          guestId     string required
          roomType    string required
          checkIn     date   required
          checkOut    date   required
        }
      }

      event ReservationMade {
        fields {
          reservationId string required
          guestId       string required
          roomType      string required
          checkIn       date   required
          checkOut      date   required
          status        string required
        }
      }

      flow {
        command -> event: MakeReservation -> ReservationMade
      }
    }
  }
}`

			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			require.Equal(t, "Hotel Reservation", model.Name)
			require.Len(t, model.Actors, 1)
			require.Equal(t, "Guest", model.Actors[0].Name)
			require.Len(t, model.Contexts, 1)

			ctx := model.Contexts[0]
			require.Equal(t, "Reservations", ctx.Name)
			require.Len(t, ctx.Aggregates, 1)

			agg := ctx.Aggregates[0]
			require.Equal(t, "Reservation", agg.Name)
			require.Len(t, agg.Slices, 1)

			slice := agg.Slices[0]
			require.Equal(t, "Make Reservation", slice.Name)
			require.Len(t, slice.Commands, 1)
			require.Len(t, slice.Events, 1)
			require.Len(t, slice.Flows, 1)

			cmd := slice.Commands[0]
			require.Equal(t, "MakeReservation", cmd.Name)
			require.Len(t, cmd.Fields, 4)

			evt := slice.Events[0]
			require.Equal(t, "ReservationMade", evt.Name)
			require.Len(t, evt.Fields, 6)

			flow := slice.Flows[0]
			require.Equal(t, "MakeReservation", flow.CommandName)
			require.Equal(t, "ReservationMade", flow.EventName)
		})
	})

	t.Run("triggers", func(t *testing.T) {
		t.Run("trigger with kind, name, actor, and reads", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      trigger UI "Reservation Form" {
        actor Guest
        reads AvailableRoomsView
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			slice := model.Contexts[0].Aggregates[0].Slices[0]
			require.NotNil(t, slice.Trigger)
			require.Equal(t, "UI", slice.Trigger.Kind)
			require.Equal(t, "Reservation Form", slice.Trigger.Name)
			require.Equal(t, "Guest", slice.Trigger.Actor)
			require.Equal(t, "AvailableRoomsView", slice.Trigger.Reads)
		})

		t.Run("trigger with only kind and name (empty body)", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      trigger UI "Reservation Form" {
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			slice := model.Contexts[0].Aggregates[0].Slices[0]
			require.NotNil(t, slice.Trigger)
			require.Equal(t, "UI", slice.Trigger.Kind)
			require.Equal(t, "Reservation Form", slice.Trigger.Name)
			require.Equal(t, "", slice.Trigger.Actor)
			require.Equal(t, "", slice.Trigger.Reads)
		})

		t.Run("trigger alongside command, event, and flow", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      trigger UI "Reservation Form" {
        actor Guest
      }
      command MakeReservation {
        fields {
        }
      }
      event ReservationMade {
        fields {
        }
      }
      flow {
        command -> event: MakeReservation -> ReservationMade
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			slice := model.Contexts[0].Aggregates[0].Slices[0]
			require.NotNil(t, slice.Trigger)
			require.Equal(t, "Reservation Form", slice.Trigger.Name)
			require.Len(t, slice.Commands, 1)
			require.Equal(t, "MakeReservation", slice.Commands[0].Name)
			require.Len(t, slice.Events, 1)
			require.Equal(t, "ReservationMade", slice.Events[0].Name)
			require.Len(t, slice.Flows, 1)
			require.Equal(t, "MakeReservation", slice.Flows[0].CommandName)
		})

		t.Run("trigger with only actor (no reads)", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      trigger UI "Name" {
        actor Guest
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			slice := model.Contexts[0].Aggregates[0].Slices[0]
			require.NotNil(t, slice.Trigger)
			require.Equal(t, "Guest", slice.Trigger.Actor)
			require.Equal(t, "", slice.Trigger.Reads)
		})

		t.Run("trigger with only reads (no actor)", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      trigger UI "Name" {
        reads SomeView
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			slice := model.Contexts[0].Aggregates[0].Slices[0]
			require.NotNil(t, slice.Trigger)
			require.Equal(t, "SomeView", slice.Trigger.Reads)
			require.Equal(t, "", slice.Trigger.Actor)
		})
	})

	t.Run("views", func(t *testing.T) {
		t.Run("view with fields and subscribes in slice", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      view AvailableRoomsView {
        fields {
          roomId RoomID
        }
        subscribes [RoomReserved, GuestCheckedOut]
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			slice := model.Contexts[0].Aggregates[0].Slices[0]
			require.Len(t, slice.Views, 1)
			view := slice.Views[0]
			require.Equal(t, "AvailableRoomsView", view.Name)
			require.Len(t, view.Fields, 1)
			require.Equal(t, "roomId", view.Fields[0].Name)
			require.Equal(t, "RoomID", view.Fields[0].Type)
			require.Equal(t, []string{"RoomReserved", "GuestCheckedOut"}, view.Subscribes)
		})

		t.Run("view with only fields (no subscribes)", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      view MyView {
        fields {
          id UUID
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			view := model.Contexts[0].Aggregates[0].Slices[0].Views[0]
			require.Equal(t, "MyView", view.Name)
			require.Len(t, view.Fields, 1)
			require.Equal(t, "id", view.Fields[0].Name)
			require.Equal(t, "UUID", view.Fields[0].Type)
			require.Empty(t, view.Subscribes)
		})

		t.Run("view with only subscribes (no fields)", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      view MyView {
        subscribes [SomeEvent]
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			view := model.Contexts[0].Aggregates[0].Slices[0].Views[0]
			require.Equal(t, "MyView", view.Name)
			require.Empty(t, view.Fields)
			require.Equal(t, []string{"SomeEvent"}, view.Subscribes)
		})

		t.Run("subscribes with multiple comma-separated identifiers", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      view MyView {
        subscribes [EventA, EventB, EventC]
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			view := model.Contexts[0].Aggregates[0].Slices[0].Views[0]
			require.Equal(t, []string{"EventA", "EventB", "EventC"}, view.Subscribes)
		})

		t.Run("subscribes with single identifier (no commas)", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      view MyView {
        subscribes [OnlyEvent]
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			view := model.Contexts[0].Aggregates[0].Slices[0].Views[0]
			require.Equal(t, []string{"OnlyEvent"}, view.Subscribes)
		})

		t.Run("subscribes positions recorded for each identifier", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      view MyView {
        subscribes [EventA, EventB, EventC]
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			view := model.Contexts[0].Aggregates[0].Slices[0].Views[0]
			require.Equal(t, []string{"EventA", "EventB", "EventC"}, view.Subscribes)
			require.Len(t, view.SubscribesPos, 3)
			require.Equal(t, "test.emod", view.SubscribesPos[0].Filename)
			require.Equal(t, 6, view.SubscribesPos[0].Line)
			require.Equal(t, 21, view.SubscribesPos[0].Column)
			require.Equal(t, "test.emod", view.SubscribesPos[1].Filename)
			require.Equal(t, 6, view.SubscribesPos[1].Line)
			require.Equal(t, 29, view.SubscribesPos[1].Column)
			require.Equal(t, "test.emod", view.SubscribesPos[2].Filename)
			require.Equal(t, 6, view.SubscribesPos[2].Line)
			require.Equal(t, 37, view.SubscribesPos[2].Column)
		})

		t.Run("subscribes positions for single identifier", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      view MyView {
        subscribes [SingleEvent]
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			view := model.Contexts[0].Aggregates[0].Slices[0].Views[0]
			require.Len(t, view.SubscribesPos, 1)
			require.Equal(t, "test.emod", view.SubscribesPos[0].Filename)
			require.Equal(t, 6, view.SubscribesPos[0].Line)
			require.Equal(t, 21, view.SubscribesPos[0].Column)
		})

		t.Run("subscribes positions is empty when no subscribes", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      view MyView {
        fields {
          id string
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			view := model.Contexts[0].Aggregates[0].Slices[0].Views[0]
			require.Empty(t, view.SubscribesPos)
		})

		t.Run("view alongside command, event, and flow", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command MakeReservation {
        fields {
        }
      }
      event ReservationMade {
        fields {
        }
      }
      flow {
        command -> event: MakeReservation -> ReservationMade
      }
      view AvailableRoomsView {
        fields {
          roomId RoomID
        }
        subscribes [ReservationMade]
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			slice := model.Contexts[0].Aggregates[0].Slices[0]
			require.Len(t, slice.Commands, 1)
			require.Equal(t, "MakeReservation", slice.Commands[0].Name)
			require.Len(t, slice.Events, 1)
			require.Equal(t, "ReservationMade", slice.Events[0].Name)
			require.Len(t, slice.Flows, 1)
			require.Len(t, slice.Views, 1)
			require.Equal(t, "AvailableRoomsView", slice.Views[0].Name)
		})

		t.Run("view missing opening brace produces diagnostic", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      view MyView subscribes [Evt]
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			require.NotEmpty(t, errs)
			found := false
			for _, e := range errs {
				if strings.Contains(e.Message, "{") {
					found = true
					break
				}
			}
			require.True(t, found, "expected a diagnostic mentioning '{', got: %v", errs)
		})
	})

	t.Run("automations", func(t *testing.T) {
		t.Run("automation with trigger event, command, and target context", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      automation ConfirmationEmailReactor {
        trigger RoomReserved
        command SendConfirmationEmail
        target context Notifications
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			slice := model.Contexts[0].Aggregates[0].Slices[0]
			require.Len(t, slice.Automations, 1)
			a := slice.Automations[0]
			require.Equal(t, "ConfirmationEmailReactor", a.Name)
			require.Equal(t, "RoomReserved", a.TriggerEvent)
			require.Equal(t, "SendConfirmationEmail", a.Command)
			require.Equal(t, "Notifications", a.TargetContext)
		})

		t.Run("automation without target context", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      automation Reactor {
        trigger SomeEvent
        command SomeCmd
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			a := model.Contexts[0].Aggregates[0].Slices[0].Automations[0]
			require.Equal(t, "Reactor", a.Name)
			require.Equal(t, "SomeEvent", a.TriggerEvent)
			require.Equal(t, "SomeCmd", a.Command)
			require.Equal(t, "", a.TargetContext)
		})

		t.Run("automation is stored in slice AST node", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      automation Reactor {
        trigger SomeEvent
        command SomeCmd
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			slice := model.Contexts[0].Aggregates[0].Slices[0]
			require.Len(t, slice.Automations, 1)
			require.Empty(t, slice.Commands)
			require.Empty(t, slice.Events)
			require.Empty(t, slice.Flows)
			require.Empty(t, slice.Views)
			require.Nil(t, slice.Trigger)
		})

		t.Run("trigger keyword inside automation is event name, not trigger block", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      automation Reactor {
        trigger SomeEvent
        command SomeCmd
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			slice := model.Contexts[0].Aggregates[0].Slices[0]
			require.Nil(t, slice.Trigger, "trigger keyword inside automation should not produce a slice-level Trigger")
			require.Len(t, slice.Automations, 1)
			require.Equal(t, "SomeEvent", slice.Automations[0].TriggerEvent)
		})

		t.Run("automation alongside other slice elements", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command MakeReservation {
        fields {
        }
      }
      event ReservationMade {
        fields {
        }
      }
      flow {
        command -> event: MakeReservation -> ReservationMade
      }
      automation Reactor {
        trigger ReservationMade
        command SendConfirmation
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			slice := model.Contexts[0].Aggregates[0].Slices[0]
			require.Len(t, slice.Commands, 1)
			require.Len(t, slice.Events, 1)
			require.Len(t, slice.Flows, 1)
			require.Len(t, slice.Automations, 1)
			require.Equal(t, "Reactor", slice.Automations[0].Name)
		})

		t.Run("multiple automations in the same slice", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      automation ReactorA {
        trigger EventA
        command CmdA
      }
      automation ReactorB {
        trigger EventB
        command CmdB
        target context OtherCtx
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			slice := model.Contexts[0].Aggregates[0].Slices[0]
			require.Len(t, slice.Automations, 2)
			require.Equal(t, "ReactorA", slice.Automations[0].Name)
			require.Equal(t, "EventA", slice.Automations[0].TriggerEvent)
			require.Equal(t, "CmdA", slice.Automations[0].Command)
			require.Equal(t, "", slice.Automations[0].TargetContext)
			require.Equal(t, "ReactorB", slice.Automations[1].Name)
			require.Equal(t, "EventB", slice.Automations[1].TriggerEvent)
			require.Equal(t, "CmdB", slice.Automations[1].Command)
			require.Equal(t, "OtherCtx", slice.Automations[1].TargetContext)
		})

		t.Run("automation missing opening brace produces diagnostic", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      automation Reactor trigger SomeEvent
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			require.NotEmpty(t, errs)
			found := false
			for _, e := range errs {
				if strings.Contains(e.Message, "{") {
					found = true
					break
				}
			}
			require.True(t, found, "expected a diagnostic mentioning '{', got: %v", errs)
		})

		t.Run("automation with unrecognized keyword in body produces diagnostic", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      automation Reactor {
        unknown_thing foo
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			require.NotEmpty(t, errs)
			found := false
			for _, e := range errs {
				if strings.Contains(e.Message, "trigger") && strings.Contains(e.Message, "command") {
					found = true
					break
				}
			}
			require.True(t, found, "expected a diagnostic mentioning expected keywords, got: %v", errs)
		})
	})

	t.Run("translations", func(t *testing.T) {
		t.Run("translation with all fields including inline event", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      translation BookingComImport {
        external_system "Booking.com API"
        reads BookingComWebhookView
        command ImportExternalReservation
        event ExternalReservationImported {
          fields {
            bookingRef string required
            guestName string required
          }
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			slice := model.Contexts[0].Aggregates[0].Slices[0]
			require.Len(t, slice.Translations, 1)
			tr := slice.Translations[0]
			require.Equal(t, "BookingComImport", tr.Name)
			require.Equal(t, "Booking.com API", tr.ExternalSystem)
			require.Equal(t, "BookingComWebhookView", tr.Reads)
			require.Equal(t, "ImportExternalReservation", tr.Command)
			require.NotNil(t, tr.Event)
			require.Equal(t, "ExternalReservationImported", tr.Event.Name)
			require.Len(t, tr.Event.Fields, 2)
			require.Equal(t, "bookingRef", tr.Event.Fields[0].Name)
			require.Equal(t, "string", tr.Event.Fields[0].Type)
			require.Equal(t, "required", tr.Event.Fields[0].Modifier)
			require.Equal(t, "guestName", tr.Event.Fields[1].Name)
			require.Equal(t, "string", tr.Event.Fields[1].Type)
			require.Equal(t, "required", tr.Event.Fields[1].Modifier)
		})

		t.Run("translation without inline event", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      translation SimpleImport {
        external_system "External API"
        reads WebhookView
        command DoImport
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			tr := model.Contexts[0].Aggregates[0].Slices[0].Translations[0]
			require.Equal(t, "SimpleImport", tr.Name)
			require.Equal(t, "External API", tr.ExternalSystem)
			require.Equal(t, "WebhookView", tr.Reads)
			require.Equal(t, "DoImport", tr.Command)
			require.Nil(t, tr.Event)
		})

		t.Run("translation is stored in slice Translations field", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      translation BookingImport {
        external_system "Booking API"
        reads WebhookView
        command ImportReservation
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			slice := model.Contexts[0].Aggregates[0].Slices[0]
			require.Len(t, slice.Translations, 1)
			require.Empty(t, slice.Commands)
			require.Empty(t, slice.Events)
			require.Empty(t, slice.Flows)
			require.Empty(t, slice.Views)
			require.Empty(t, slice.Automations)
			require.Nil(t, slice.Trigger)
		})

		t.Run("translation alongside other slice elements", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command MakeReservation {
        fields {
        }
      }
      event ReservationMade {
        fields {
        }
      }
      flow {
        command -> event: MakeReservation -> ReservationMade
      }
      translation BookingImport {
        external_system "Booking API"
        reads WebhookView
        command ImportReservation
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			slice := model.Contexts[0].Aggregates[0].Slices[0]
			require.Len(t, slice.Commands, 1)
			require.Len(t, slice.Events, 1)
			require.Len(t, slice.Flows, 1)
			require.Len(t, slice.Translations, 1)
			require.Equal(t, "BookingImport", slice.Translations[0].Name)
		})

		t.Run("multiple translations in the same slice", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      translation BookingImport {
        external_system "Booking API"
        reads BookingView
        command ImportBooking
      }
      translation ExpediaImport {
        external_system "Expedia API"
        reads ExpediaView
        command ImportExpedia
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			slice := model.Contexts[0].Aggregates[0].Slices[0]
			require.Len(t, slice.Translations, 2)
			require.Equal(t, "BookingImport", slice.Translations[0].Name)
			require.Equal(t, "Booking API", slice.Translations[0].ExternalSystem)
			require.Equal(t, "BookingView", slice.Translations[0].Reads)
			require.Equal(t, "ImportBooking", slice.Translations[0].Command)
			require.Equal(t, "ExpediaImport", slice.Translations[1].Name)
			require.Equal(t, "Expedia API", slice.Translations[1].ExternalSystem)
			require.Equal(t, "ExpediaView", slice.Translations[1].Reads)
			require.Equal(t, "ImportExpedia", slice.Translations[1].Command)
		})

		t.Run("translation missing opening brace produces diagnostic", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      translation BookingImport external_system "API"
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			require.NotEmpty(t, errs)
			found := false
			for _, e := range errs {
				if strings.Contains(e.Message, "{") {
					found = true
					break
				}
			}
			require.True(t, found, "expected a diagnostic mentioning '{', got: %v", errs)
		})

		t.Run("unrecognized keyword inside translation body produces diagnostic", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      translation BookingImport {
        unknown_thing foo
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			require.NotEmpty(t, errs)
			found := false
			for _, e := range errs {
				if strings.Contains(e.Message, "external_system") && strings.Contains(e.Message, "command") {
					found = true
					break
				}
			}
			require.True(t, found, "expected a diagnostic mentioning expected keywords, got: %v", errs)
		})
	})

	t.Run("error reporting", func(t *testing.T) {
		t.Run("multiple errors collected", func(t *testing.T) {
			input := `unknown_keyword "Test"
actor
context`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			require.Greater(t, len(errs), 0)
		})

		t.Run("unrecognized keyword includes the keyword and expected alternatives", func(t *testing.T) {
			input := `foobar { }`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			require.NotEmpty(t, errs)
			require.Equal(t, "test.emod", errs[0].Filename)
			require.Equal(t, 1, errs[0].Line)
			require.Contains(t, errs[0].Message, `"foobar"`)
			require.Contains(t, errs[0].Message, "model")
			require.Contains(t, errs[0].Message, "actor")
			require.Contains(t, errs[0].Message, "context")
		})

		t.Run("unclosed brace reports the block type and opening line", func(t *testing.T) {
			input := `model "Test"
context "Foo" {`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			require.GreaterOrEqual(t, len(errs), 1)
			lastErr := errs[len(errs)-1]
			require.Equal(t, "test.emod", lastErr.Filename)
			require.Contains(t, lastErr.Message, `"context"`)
			require.Contains(t, lastErr.Message, "unclosed brace")
			require.Contains(t, lastErr.Message, "line 2")
		})

		t.Run("unexpected token after model reports what was found", func(t *testing.T) {
			input := `model {`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			require.GreaterOrEqual(t, len(errs), 1)
			require.Equal(t, "test.emod", errs[0].Filename)
			require.Equal(t, 1, errs[0].Line)
			require.Contains(t, errs[0].Message, `"model"`)
			require.Contains(t, errs[0].Message, "expected quoted string")
		})

		t.Run("diagnostics include filename and line number", func(t *testing.T) {
			input := `model "OK"
foobar "bad"
actor
context "Missing" {
  unknown_inside`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "errors.emod")
			_, errs := p.Parse()

			for _, e := range errs {
				require.Equal(t, "errors.emod", e.Filename)
				require.Greater(t, e.Line, 0)
				require.NotEmpty(t, e.Message)
			}
		})

		t.Run("automation missing trigger produces error", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      automation Reactor {
        command SomeCmd
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			found := false
			for _, e := range errs {
				if e.Message == "automation block requires a trigger event" {
					found = true
					require.Equal(t, "test.emod", e.Filename)
					require.Equal(t, 5, e.Line)
					break
				}
			}
			require.True(t, found, "expected diagnostic 'automation block requires a trigger event', got: %v", errs)
		})

		t.Run("automation missing command produces error", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      automation Reactor {
        trigger SomeEvent
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			found := false
			for _, e := range errs {
				if e.Message == "automation block requires a command" {
					found = true
					require.Equal(t, "test.emod", e.Filename)
					require.Equal(t, 5, e.Line)
					break
				}
			}
			require.True(t, found, "expected diagnostic 'automation block requires a command', got: %v", errs)
		})

		t.Run("automation missing both trigger and command produces both errors", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      automation Reactor {
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			foundTrigger := false
			foundCommand := false
			for _, e := range errs {
				if e.Message == "automation block requires a trigger event" {
					foundTrigger = true
				}
				if e.Message == "automation block requires a command" {
					foundCommand = true
				}
			}
			require.True(t, foundTrigger, "expected diagnostic 'automation block requires a trigger event', got: %v", errs)
			require.True(t, foundCommand, "expected diagnostic 'automation block requires a command', got: %v", errs)
		})

		t.Run("translation missing external_system produces error", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      translation Foo {
        reads V
        command C
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			found := false
			for _, e := range errs {
				if e.Message == "translation block requires an external_system" {
					found = true
					require.Equal(t, "test.emod", e.Filename)
					require.Equal(t, 5, e.Line)
					break
				}
			}
			require.True(t, found, "expected diagnostic 'translation block requires an external_system', got: %v", errs)
		})

		t.Run("translation missing reads produces error", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      translation Foo {
        external_system "API"
        command C
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			found := false
			for _, e := range errs {
				if e.Message == "translation block requires a reads view" {
					found = true
					require.Equal(t, "test.emod", e.Filename)
					require.Equal(t, 5, e.Line)
					break
				}
			}
			require.True(t, found, "expected diagnostic 'translation block requires a reads view', got: %v", errs)
		})

		t.Run("translation missing command produces error", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      translation Foo {
        external_system "API"
        reads V
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			found := false
			for _, e := range errs {
				if e.Message == "translation block requires a command" {
					found = true
					require.Equal(t, "test.emod", e.Filename)
					require.Equal(t, 5, e.Line)
					break
				}
			}
			require.True(t, found, "expected diagnostic 'translation block requires a command', got: %v", errs)
		})

		t.Run("translation missing all three required sub-blocks produces all errors", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      translation Foo {
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			foundExtSys := false
			foundReads := false
			foundCommand := false
			for _, e := range errs {
				if e.Message == "translation block requires an external_system" {
					foundExtSys = true
				}
				if e.Message == "translation block requires a reads view" {
					foundReads = true
				}
				if e.Message == "translation block requires a command" {
					foundCommand = true
				}
			}
			require.True(t, foundExtSys, "expected diagnostic 'translation block requires an external_system', got: %v", errs)
			require.True(t, foundReads, "expected diagnostic 'translation block requires a reads view', got: %v", errs)
			require.True(t, foundCommand, "expected diagnostic 'translation block requires a command', got: %v", errs)
		})

		t.Run("view missing both fields and subscribes produces error", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      view MyView {
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			found := false
			for _, e := range errs {
				if e.Message == "view block requires fields or subscribes" {
					found = true
					require.Equal(t, "test.emod", e.Filename)
					require.Equal(t, 5, e.Line)
					break
				}
			}
			require.True(t, found, "expected diagnostic 'view block requires fields or subscribes', got: %v", errs)
		})

		t.Run("missing sub-block error references block opening position not closing brace", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      automation Reactor {
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			found := false
			for _, e := range errs {
				if e.Message == "automation block requires a trigger event" {
					found = true
					require.Equal(t, 5, e.Line, "error should reference the automation declaration line (5), not the closing brace line")
					break
				}
			}
			require.True(t, found, "expected diagnostic 'automation block requires a trigger event', got: %v", errs)
		})

		t.Run("event with source external and provider name", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      event TestEvent {
        source external "SendGrid Webhook"
        fields {
          id string
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			evt := model.Contexts[0].Aggregates[0].Slices[0].Events[0]
			require.Equal(t, "TestEvent", evt.Name)
			require.Equal(t, "external", evt.Source)
			require.Equal(t, "SendGrid Webhook", evt.ExternalName)
			require.Len(t, evt.Fields, 1)
		})
	})

	t.Run("event sources and tags", func(t *testing.T) {
		t.Run("event without source clause", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      event TestEvent {
        fields {
          id string
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			evt := model.Contexts[0].Aggregates[0].Slices[0].Events[0]
			require.Equal(t, "TestEvent", evt.Name)
			require.Equal(t, "", evt.Source)
			require.Equal(t, "", evt.ExternalName)
		})

		t.Run("event source before fields", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      event TestEvent {
        source external "X"
        fields {
          id string
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			evt := model.Contexts[0].Aggregates[0].Slices[0].Events[0]
			require.Equal(t, "external", evt.Source)
			require.Equal(t, "X", evt.ExternalName)
			require.Len(t, evt.Fields, 1)
		})

		t.Run("event source after fields", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      event TestEvent {
        fields {
          id string
        }
        source external "X"
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			evt := model.Contexts[0].Aggregates[0].Slices[0].Events[0]
			require.Equal(t, "external", evt.Source)
			require.Equal(t, "X", evt.ExternalName)
			require.Len(t, evt.Fields, 1)
		})

		t.Run("event with tags", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      event TestEvent {
        tags {
          priority: statusCode
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			evt := model.Contexts[0].Aggregates[0].Slices[0].Events[0]
			require.Equal(t, "TestEvent", evt.Name)
			require.Len(t, evt.Tags, 1)
			require.Equal(t, "priority", evt.Tags[0].Key)
			require.Equal(t, "statusCode", evt.Tags[0].FieldRef)
		})

		t.Run("event with multiple tags", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      event TestEvent {
        tags {
          priority: statusCode
          category: eventType
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			evt := model.Contexts[0].Aggregates[0].Slices[0].Events[0]
			require.Len(t, evt.Tags, 2)
			require.Equal(t, "priority", evt.Tags[0].Key)
			require.Equal(t, "statusCode", evt.Tags[0].FieldRef)
			require.Equal(t, "category", evt.Tags[1].Key)
			require.Equal(t, "eventType", evt.Tags[1].FieldRef)
		})

		t.Run("event with tags and fields", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      event TestEvent {
        tags {
          priority: statusCode
        }
        fields {
          statusCode string
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			evt := model.Contexts[0].Aggregates[0].Slices[0].Events[0]
			require.Len(t, evt.Tags, 1)
			require.Equal(t, "priority", evt.Tags[0].Key)
			require.Equal(t, "statusCode", evt.Tags[0].FieldRef)
			require.Len(t, evt.Fields, 1)
			require.Equal(t, "statusCode", evt.Fields[0].Name)
		})

		t.Run("event with tags and source external", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      event TestEvent {
        source external "SendGrid Webhook"
        tags {
          priority: statusCode
        }
        fields {
          statusCode string
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			evt := model.Contexts[0].Aggregates[0].Slices[0].Events[0]
			require.Equal(t, "external", evt.Source)
			require.Equal(t, "SendGrid Webhook", evt.ExternalName)
			require.Len(t, evt.Tags, 1)
			require.Equal(t, "priority", evt.Tags[0].Key)
			require.Equal(t, "statusCode", evt.Tags[0].FieldRef)
			require.Len(t, evt.Fields, 1)
		})

		t.Run("event without tags (backward compatible)", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      event TestEvent {
        fields {
          id string
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			evt := model.Contexts[0].Aggregates[0].Slices[0].Events[0]
			require.Equal(t, "TestEvent", evt.Name)
			require.Empty(t, evt.Tags)
			require.Len(t, evt.Fields, 1)
		})

		t.Run("tags block without opening brace produces diagnostic", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      event TestEvent {
        tags priority: statusCode
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			require.NotEmpty(t, errs)
			found := false
			for _, e := range errs {
				if strings.Contains(e.Message, "{") {
					found = true
					break
				}
			}
			require.True(t, found, "expected a diagnostic mentioning '{', got: %v", errs)
		})

		t.Run("tags block missing colon produces diagnostic", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      event TestEvent {
        tags {
          priority statusCode
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			require.NotEmpty(t, errs)
			found := false
			for _, e := range errs {
				if strings.Contains(e.Message, ":") {
					found = true
					break
				}
			}
			require.True(t, found, "expected a diagnostic mentioning ':', got: %v", errs)
		})

		t.Run("tags block missing field reference produces diagnostic", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      event TestEvent {
        tags {
          priority:
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			require.NotEmpty(t, errs)
			found := false
			for _, e := range errs {
				if strings.Contains(e.Message, "field reference") {
					found = true
					break
				}
			}
			require.True(t, found, "expected a diagnostic mentioning 'field reference', got: %v", errs)
		})

		t.Run("event source without external keyword", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      event TestEvent {
        source "SendGrid"
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			require.Len(t, errs, 1)
			require.Equal(t, "expected external after source in event", errs[0].Message)
		})

		t.Run("event source external without quoted string", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      event TestEvent {
        source external
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 1)
			require.Equal(t, "expected quoted string after source external in event", errs[0].Message)

			event := model.Contexts[0].Aggregates[0].Slices[0].Events[0]
			require.Equal(t, "external", event.Source)
			require.Empty(t, event.ExternalName)
		})
	})

	t.Run("comments", func(t *testing.T) {
		t.Run("comments before model are attached to Model node", func(t *testing.T) {
			input := `# Header comment
model "Test"`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Empty(t, errs)
			require.Equal(t, "Test", model.Name)
			test.RequireEqual(t, []*ast.Comment{{Text: "# Header comment"}}, model.Comments, ignoreCommentPositions)
		})

		t.Run("multiple consecutive comments before model are all attached", func(t *testing.T) {
			input := `# Line 1
# Line 2
model "Test"`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Empty(t, errs)
			test.RequireEqual(t, []*ast.Comment{
				{Text: "# Line 1"},
				{Text: "# Line 2"},
			}, model.Comments, ignoreCommentPositions)
		})

		t.Run("comments before actor are attached to Actor node", func(t *testing.T) {
			input := `model "Test"
# Actor comment
actor "Guest"`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Empty(t, errs)
			require.Equal(t, "Guest", model.Actors[0].Name)
			test.RequireEqual(t, []*ast.Comment{{Text: "# Actor comment"}}, model.Actors[0].Comments, ignoreCommentPositions)
		})

		t.Run("comments before context are attached to Context node", func(t *testing.T) {
			input := `model "Test"
# Context comment
context "Reservations" {
  aggregate "Reservation" {
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Empty(t, errs)
			require.Equal(t, "Reservations", model.Contexts[0].Name)
			test.RequireEqual(t, []*ast.Comment{{Text: "# Context comment"}}, model.Contexts[0].Comments, ignoreCommentPositions)
		})

		t.Run("comments before slice are attached to Slice node", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    # Slice comment
    slice "My Slice" {
      command DoThing {
        fields {
          id string
        }
      }
      event ThingDone {
        fields {
          id string
        }
      }
      flow {
        command -> event: DoThing -> ThingDone
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Empty(t, errs)
			slice := model.Contexts[0].Aggregates[0].Slices[0]
			require.Equal(t, "My Slice", slice.Name)
			test.RequireEqual(t, []*ast.Comment{{Text: "# Slice comment"}}, slice.Comments, ignoreCommentPositions)
		})

		t.Run("comments before command event view automation translation trigger are attached", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      # Command comment
      command DoThing {
        fields {
          id string
        }
      }
      # Event comment
      event ThingDone {
        fields {
          id string
        }
      }
      # Trigger comment
      trigger UI "Form" {
        actor Guest
      }
      # View comment
      view MyView {
        fields {
          id string
        }
      }
      # Automation comment
      automation Reactor {
        trigger ThingDone
        command DoOther
      }
      # Translation comment
      translation Import {
        external_system "API"
        reads WebhookView
        command DoImport
      }
      # Flow comment
      flow {
        command -> event: DoThing -> ThingDone
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Empty(t, errs)
			slice := model.Contexts[0].Aggregates[0].Slices[0]

			test.RequireEqual(t, []*ast.Comment{{Text: "# Command comment"}}, slice.Commands[0].Comments, ignoreCommentPositions)
			test.RequireEqual(t, []*ast.Comment{{Text: "# Event comment"}}, slice.Events[0].Comments, ignoreCommentPositions)
			test.RequireEqual(t, []*ast.Comment{{Text: "# Trigger comment"}}, slice.Trigger.Comments, ignoreCommentPositions)
			test.RequireEqual(t, []*ast.Comment{{Text: "# View comment"}}, slice.Views[0].Comments, ignoreCommentPositions)
			test.RequireEqual(t, []*ast.Comment{{Text: "# Automation comment"}}, slice.Automations[0].Comments, ignoreCommentPositions)
			test.RequireEqual(t, []*ast.Comment{{Text: "# Translation comment"}}, slice.Translations[0].Comments, ignoreCommentPositions)
			test.RequireEqual(t, []*ast.Comment{{Text: "# Flow comment"}}, slice.Flows[0].Comments, ignoreCommentPositions)
		})

		t.Run("comments before aggregate are attached to Aggregate node", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  # Aggregate comment
  aggregate "Agg" {
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Empty(t, errs)
			require.Equal(t, "Agg", model.Contexts[0].Aggregates[0].Name)
			test.RequireEqual(t, []*ast.Comment{{Text: "# Aggregate comment"}}, model.Contexts[0].Aggregates[0].Comments, ignoreCommentPositions)
		})

		t.Run("attached comment carries correct position", func(t *testing.T) {
			input := `# Header
model "Test"
  # Indented actor comment
actor "Guest"`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Empty(t, errs)
			test.RequireEqual(t, []*ast.Comment{{
				Text:     "# Header",
				Position: ast.Position{Filename: "test.emod", Line: 1, Column: 1},
			}}, model.Comments)

			test.RequireEqual(t, []*ast.Comment{{
				Text:     "# Indented actor comment",
				Position: ast.Position{Filename: "test.emod", Line: 3, Column: 3},
			}}, model.Actors[0].Comments)
		})
	})

	t.Run("decides_on", func(t *testing.T) {
		t.Run("command with decides_on and simple tag predicate", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command DoThing {
        fields {
          id string
        }
        decides_on {
          events [ThingDone]
          where tag(priority = high)
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Empty(t, errs)
			cmd := model.Contexts[0].Aggregates[0].Slices[0].Commands[0]
			require.Equal(t, "DoThing", cmd.Name)
			require.NotNil(t, cmd.DecidesOn)
			require.Equal(t, []string{"ThingDone"}, cmd.DecidesOn.Events)
			require.NotNil(t, cmd.DecidesOn.Predicate)

			pred, ok := cmd.DecidesOn.Predicate.(*ast.TagPredicate)
			require.True(t, ok, "expected *ast.TagPredicate, got %T", cmd.DecidesOn.Predicate)
			require.Equal(t, "priority", pred.Field)
			require.Equal(t, "=", pred.Operator)
			require.Equal(t, "high", pred.Value)
		})

		t.Run("command with decides_on and multiple events", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command DoThing {
        decides_on {
          events [EventA, EventB, EventC]
          where tag(priority = high)
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Empty(t, errs)
			cmd := model.Contexts[0].Aggregates[0].Slices[0].Commands[0]
			require.NotNil(t, cmd.DecidesOn)
			require.Equal(t, []string{"EventA", "EventB", "EventC"}, cmd.DecidesOn.Events)
		})

		t.Run("command with decides_on and single event (no commas)", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command DoThing {
        decides_on {
          events [OnlyEvent]
          where tag(priority = high)
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Empty(t, errs)
			cmd := model.Contexts[0].Aggregates[0].Slices[0].Commands[0]
			require.Equal(t, []string{"OnlyEvent"}, cmd.DecidesOn.Events)
		})

		t.Run("command with decides_on with both fields", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command DoThing {
        fields {
          id string
        }
        decides_on {
          events [ThingDone]
          where tag(priority = high)
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Empty(t, errs)
			cmd := model.Contexts[0].Aggregates[0].Slices[0].Commands[0]
			require.NotNil(t, cmd.DecidesOn)
			require.Len(t, cmd.Fields, 1)
			require.Equal(t, "id", cmd.Fields[0].Name)
			require.Equal(t, []string{"ThingDone"}, cmd.DecidesOn.Events)
		})

		t.Run("command with decides_on and compound predicate (and)", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command DoThing {
        decides_on {
          events [ThingDone]
          where tag(priority = high) and tag(region = us)
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Empty(t, errs)
			cmd := model.Contexts[0].Aggregates[0].Slices[0].Commands[0]
			require.NotNil(t, cmd.DecidesOn)

			logical, ok := cmd.DecidesOn.Predicate.(*ast.LogicalExpr)
			require.True(t, ok, "expected *ast.LogicalExpr, got %T", cmd.DecidesOn.Predicate)
			require.Equal(t, "and", logical.Operator)

			left, ok := logical.Left.(*ast.TagPredicate)
			require.True(t, ok)
			require.Equal(t, "priority", left.Field)
			require.Equal(t, "high", left.Value)

			right, ok := logical.Right.(*ast.TagPredicate)
			require.True(t, ok)
			require.Equal(t, "region", right.Field)
			require.Equal(t, "us", right.Value)
		})

		t.Run("command with decides_on and compound predicate (or)", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command DoThing {
        decides_on {
          events [ThingDone]
          where tag(priority = high) or tag(priority = low)
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Empty(t, errs)
			cmd := model.Contexts[0].Aggregates[0].Slices[0].Commands[0]

			logical, ok := cmd.DecidesOn.Predicate.(*ast.LogicalExpr)
			require.True(t, ok, "expected *ast.LogicalExpr, got %T", cmd.DecidesOn.Predicate)
			require.Equal(t, "or", logical.Operator)
		})

		t.Run("command with decides_on and not predicate", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command DoThing {
        decides_on {
          events [ThingDone]
          where not tag(priority = high)
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Empty(t, errs)
			cmd := model.Contexts[0].Aggregates[0].Slices[0].Commands[0]

			notExpr, ok := cmd.DecidesOn.Predicate.(*ast.NotExpr)
			require.True(t, ok, "expected *ast.NotExpr, got %T", cmd.DecidesOn.Predicate)

			_, ok = notExpr.Expr.(*ast.TagPredicate)
			require.True(t, ok, "expected *ast.TagPredicate inside NotExpr, got %T", notExpr.Expr)
		})

		t.Run("command with decides_on and double not", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command DoThing {
        decides_on {
          events [ThingDone]
          where not not tag(priority = high)
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Empty(t, errs)
			cmd := model.Contexts[0].Aggregates[0].Slices[0].Commands[0]

			outer, ok := cmd.DecidesOn.Predicate.(*ast.NotExpr)
			require.True(t, ok, "expected outer *ast.NotExpr")

			inner, ok := outer.Expr.(*ast.NotExpr)
			require.True(t, ok, "expected inner *ast.NotExpr")

			_, ok = inner.Expr.(*ast.TagPredicate)
			require.True(t, ok, "expected *ast.TagPredicate inside inner NotExpr")
		})

		t.Run("command with decides_on and parenthesised sub-expression", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command DoThing {
        decides_on {
          events [ThingDone]
          where tag(priority = high) and (tag(region = us) or tag(region = eu))
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Empty(t, errs)
			cmd := model.Contexts[0].Aggregates[0].Slices[0].Commands[0]

			outerLogical, ok := cmd.DecidesOn.Predicate.(*ast.LogicalExpr)
			require.True(t, ok, "expected outer *ast.LogicalExpr")
			require.Equal(t, "and", outerLogical.Operator)

			innerLogical, ok := outerLogical.Right.(*ast.LogicalExpr)
			require.True(t, ok, "expected inner *ast.LogicalExpr for parenthesised group")
			require.Equal(t, "or", innerLogical.Operator)
		})

		t.Run("command with decides_on and nested parentheses", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command DoThing {
        decides_on {
          events [ThingDone]
          where (tag(priority = high) and tag(region = us)) or tag(status = active)
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Empty(t, errs)
			cmd := model.Contexts[0].Aggregates[0].Slices[0].Commands[0]

			outerLogical, ok := cmd.DecidesOn.Predicate.(*ast.LogicalExpr)
			require.True(t, ok, "expected outer *ast.LogicalExpr")
			require.Equal(t, "or", outerLogical.Operator)

			innerLogical, ok := outerLogical.Left.(*ast.LogicalExpr)
			require.True(t, ok, "expected inner *ast.LogicalExpr inside parentheses")
			require.Equal(t, "and", innerLogical.Operator)
		})

		t.Run("command without decides_on remains valid (backward compatible)", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command DoThing {
        fields {
          id string
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Empty(t, errs)
			cmd := model.Contexts[0].Aggregates[0].Slices[0].Commands[0]
			require.Nil(t, cmd.DecidesOn)
			require.Len(t, cmd.Fields, 1)
		})

		t.Run("command with decides_on missing events produces error", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command DoThing {
        decides_on {
          where tag(priority = high)
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			found := false
			for _, e := range errs {
				if e.Message == "decides_on block requires an events clause" {
					found = true
					require.Equal(t, "test.emod", e.Filename)
					break
				}
			}
			require.True(t, found, "expected diagnostic 'decides_on block requires an events clause', got: %v", errs)
		})

		t.Run("command with decides_on missing where produces error", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command DoThing {
        decides_on {
          events [ThingDone]
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			found := false
			for _, e := range errs {
				if e.Message == "decides_on block requires a where clause" {
					found = true
					require.Equal(t, "test.emod", e.Filename)
					break
				}
			}
			require.True(t, found, "expected diagnostic 'decides_on block requires a where clause', got: %v", errs)
		})

		t.Run("command with decides_on missing both events and where produces both errors", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command DoThing {
        decides_on {
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			foundEvents := false
			foundWhere := false
			for _, e := range errs {
				if e.Message == "decides_on block requires an events clause" {
					foundEvents = true
				}
				if e.Message == "decides_on block requires a where clause" {
					foundWhere = true
				}
			}
			require.True(t, foundEvents, "expected 'decides_on block requires an events clause', got: %v", errs)
			require.True(t, foundWhere, "expected 'decides_on block requires a where clause', got: %v", errs)
		})

		t.Run("command with decides_on and bad predicate token produces error", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command DoThing {
        decides_on {
          events [ThingDone]
          where badtoken
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			require.NotEmpty(t, errs)
			found := false
			for _, e := range errs {
				if strings.Contains(e.Message, "tag()") || strings.Contains(e.Message, "(") {
					found = true
					break
				}
			}
			require.True(t, found, "expected a diagnostic mentioning 'tag()' or '(', got: %v", errs)
		})

		t.Run("decides_on error reports correct location", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command DoThing {
        decides_on {
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			found := false
			for _, e := range errs {
				if e.Message == "decides_on block requires an events clause" {
					found = true
					require.Equal(t, 6, e.Line, "error should reference the decides_on block opening line (6)")
					break
				}
			}
			require.True(t, found, "expected 'decides_on block requires an events clause' diagnostic, got: %v", errs)
		})

		t.Run("event name positions recorded in decides_on events list", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command DoThing {
        decides_on {
          events [EventA, EventB]
          where tag(priority = high)
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Empty(t, errs)
			cmd := model.Contexts[0].Aggregates[0].Slices[0].Commands[0]
			require.Len(t, cmd.DecidesOn.EventsPos, 2)
			require.Equal(t, "test.emod", cmd.DecidesOn.EventsPos[0].Filename)
			require.Equal(t, 7, cmd.DecidesOn.EventsPos[0].Line)
			require.Equal(t, "test.emod", cmd.DecidesOn.EventsPos[1].Filename)
			require.Equal(t, 7, cmd.DecidesOn.EventsPos[1].Line)
		})

		t.Run("decides_on events list missing opening bracket produces error", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command DoThing {
        decides_on {
          events ThingDone]
          where tag(priority = high)
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			found := false
			for _, e := range errs {
				if strings.Contains(e.Message, "[") {
					found = true
					break
				}
			}
			require.True(t, found, "expected a diagnostic mentioning '[', got: %v", errs)
		})

		t.Run("decides_on missing opening brace produces error", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command DoThing {
        decides_on events [ThingDone] where tag(priority = high)
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			found := false
			for _, e := range errs {
				if strings.Contains(e.Message, "{") {
					found = true
					break
				}
			}
			require.True(t, found, "expected a diagnostic mentioning '{', got: %v", errs)
		})

		t.Run("decides_on with unrecognized keyword in body produces diagnostic", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command DoThing {
        decides_on {
          unknown_directive
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			found := false
			for _, e := range errs {
				if strings.Contains(e.Message, "events") && strings.Contains(e.Message, "where") {
					found = true
					break
				}
			}
			require.True(t, found, "expected a diagnostic mentioning 'events' and 'where', got: %v", errs)
		})

		t.Run("command with decides_on alongside command, event, and flow in slice", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command DoThing {
        fields {
          id string required
        }
        decides_on {
          events [ThingDone]
          where tag(priority = high)
        }
      }
      event ThingDone {
        fields {
          id string required
          priority string required
        }
      }
      flow {
        command -> event: DoThing -> ThingDone
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Empty(t, errs)
			slice := model.Contexts[0].Aggregates[0].Slices[0]
			require.Len(t, slice.Commands, 1)
			require.NotNil(t, slice.Commands[0].DecidesOn)
			require.Len(t, slice.Events, 1)
			require.Len(t, slice.Flows, 1)
			require.Equal(t, "ThingDone", slice.Commands[0].DecidesOn.Events[0])
		})

		t.Run("command with fields before and decides_on after works", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command DoThing {
        fields {
          id string
        }
        decides_on {
          events [ThingDone]
          where tag(priority = high)
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Empty(t, errs)
			cmd := model.Contexts[0].Aggregates[0].Slices[0].Commands[0]
			require.Len(t, cmd.Fields, 1)
			require.NotNil(t, cmd.DecidesOn)
		})

		t.Run("decides_on events are parsed in order", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command DoThing {
        decides_on {
          events [Alpha, Beta, Gamma]
          where tag(priority = high)
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")
			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Empty(t, errs)
			cmd := model.Contexts[0].Aggregates[0].Slices[0].Commands[0]
			require.Equal(t, []string{"Alpha", "Beta", "Gamma"}, cmd.DecidesOn.Events)
		})
	})
}

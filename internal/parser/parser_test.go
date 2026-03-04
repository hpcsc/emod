//go:build unit

package parser_test

import (
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/parser"
	"github.com/stretchr/testify/require"
)

func TestParser(t *testing.T) {
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
}

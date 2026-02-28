package parser

import (
	"testing"

	"github.com/hpcsc/emod/internal/lexer"
	"github.com/stretchr/testify/require"
)

func TestParser(t *testing.T) {
	t.Run("model declaration", func(t *testing.T) {
		input := `model "Test Model"`
		lex := lexer.New(input)
		tokens := lex.Tokenize()

		parser := New(tokens, "test.emod")
		model, errs := parser.Parse()

		require.Len(t, errs, 0)
		require.Equal(t, "Test Model", model.Name)
	})

	t.Run("actor declaration", func(t *testing.T) {
		input := `model "Test"
actor "Guest"`
		lex := lexer.New(input)
		tokens := lex.Tokenize()

		parser := New(tokens, "test.emod")
		model, errs := parser.Parse()

		require.Len(t, errs, 0)
		require.Len(t, model.Actors, 1)
		require.Equal(t, "Guest", model.Actors[0].Name)
	})

	t.Run("multiple actors", func(t *testing.T) {
		input := `model "Test"
actor "Guest"
actor "FrontDesk"`
		lex := lexer.New(input)
		tokens := lex.Tokenize()

		parser := New(tokens, "test.emod")
		model, errs := parser.Parse()

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
		lex := lexer.New(input)
		tokens := lex.Tokenize()

		parser := New(tokens, "test.emod")
		model, errs := parser.Parse()

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
		lex := lexer.New(input)
		tokens := lex.Tokenize()

		parser := New(tokens, "test.emod")
		model, errs := parser.Parse()

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
		lex := lexer.New(input)
		tokens := lex.Tokenize()

		parser := New(tokens, "test.emod")
		model, errs := parser.Parse()

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
		lex := lexer.New(input)
		tokens := lex.Tokenize()

		parser := New(tokens, "test.emod")
		model, errs := parser.Parse()

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
		lex := lexer.New(input)
		tokens := lex.Tokenize()

		parser := New(tokens, "test.emod")
		model, errs := parser.Parse()

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
		lex := lexer.New(input)
		tokens := lex.Tokenize()

		parser := New(tokens, "test.emod")
		model, errs := parser.Parse()

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

		lex := lexer.New(input)
		tokens := lex.Tokenize()

		parser := New(tokens, "test.emod")
		model, errs := parser.Parse()

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

	t.Run("multiple errors collected", func(t *testing.T) {
		input := `unknown_keyword "Test"
actor
context`
		lex := lexer.New(input)
		tokens := lex.Tokenize()

		parser := New(tokens, "test.emod")
		_, errs := parser.Parse()

		require.Greater(t, len(errs), 0)
	})
}

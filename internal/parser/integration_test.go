//go:build integration

package parser_test

import (
	"os"
	"testing"

	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/parser"
	"github.com/stretchr/testify/require"
)

func TestIntegration(t *testing.T) {
	t.Run("parses minimal.emod fixture with zero errors and fully populated AST", func(t *testing.T) {
		source, err := os.ReadFile("testdata/minimal.emod")
		require.NoError(t, err)

		tokens, lexErrs := lexer.Scan(string(source))
		require.Empty(t, lexErrs)

		p := parser.New(tokens, "testdata/minimal.emod")
		model, diags := p.Parse()

		require.Empty(t, diags)
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
		cmd := slice.Commands[0]
		require.Equal(t, "MakeReservation", cmd.Name)
		require.Len(t, cmd.Fields, 4)
		require.Equal(t, "guestId", cmd.Fields[0].Name)
		require.Equal(t, "string", cmd.Fields[0].Type)
		require.Equal(t, "required", cmd.Fields[0].Modifier)

		require.Len(t, slice.Events, 1)
		evt := slice.Events[0]
		require.Equal(t, "ReservationMade", evt.Name)
		require.Len(t, evt.Fields, 6)

		require.Len(t, slice.Flows, 1)
		flow := slice.Flows[0]
		require.Equal(t, "MakeReservation", flow.CommandName)
		require.Equal(t, "ReservationMade", flow.EventName)
	})

	t.Run("parses invalid.emod fixture and returns errors with correct filenames and line numbers", func(t *testing.T) {
		source, err := os.ReadFile("testdata/invalid.emod")
		require.NoError(t, err)

		tokens, _ := lexer.Scan(string(source))

		p := parser.New(tokens, "testdata/invalid.emod")
		_, diags := p.Parse()

		require.NotEmpty(t, diags)
		for _, d := range diags {
			require.Equal(t, "testdata/invalid.emod", d.Filename)
			require.Greater(t, d.Line, 0)
			require.NotEmpty(t, d.Message)
		}

		// foobar on line 3 should produce an unrecognized keyword error
		require.Contains(t, diags[0].Message, `"foobar"`)
		require.Equal(t, 3, diags[0].Line)
	})
}

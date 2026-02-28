//go:build unit

package lexer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLexer(t *testing.T) {
	t.Run("keyword tokens", func(t *testing.T) {
		keywords := map[string]Kind{
			"model":     KeywordModel,
			"actor":     KeywordActor,
			"context":   KeywordContext,
			"aggregate": KeywordAggregate,
			"slice":     KeywordSlice,
			"command":   KeywordCommand,
			"event":     KeywordEvent,
			"fields":    KeywordFields,
			"flow":      KeywordFlow,
		}

		for keyword, expectedType := range keywords {
			t.Run(keyword, func(t *testing.T) {
				lexer := New(keyword)
				tokens := lexer.Scan()
				require.Greater(t, len(tokens), 1)
				require.Equal(t, expectedType, tokens[0].Type)
				require.Equal(t, keyword, tokens[0].Value)
			})
		}
	})

	t.Run("identifiers", func(t *testing.T) {
		tests := []string{"MakeReservation", "string", "required", "date", "foo_bar", "Foo123"}
		for _, id := range tests {
			t.Run(id, func(t *testing.T) {
				lexer := New(id)
				tokens := lexer.Scan()
				require.Equal(t, Identifier, tokens[0].Type)
				require.Equal(t, id, tokens[0].Value)
			})
		}
	})

	t.Run("quoted strings", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{`"Hotel Reservation"`, "Hotel Reservation"},
			{`"Make Reservation"`, "Make Reservation"},
			{`""`, ""},
			{`"test"`, "test"},
		}

		for _, test := range tests {
			t.Run(test.input, func(t *testing.T) {
				lexer := New(test.input)
				tokens := lexer.Scan()
				require.Equal(t, String, tokens[0].Type)
				require.Equal(t, test.expected, tokens[0].Value)
			})
		}
	})

	t.Run("braces", func(t *testing.T) {
		lexer := New("{ }")
		tokens := lexer.Scan()
		require.Equal(t, OpenBrace, tokens[0].Type)
		require.Equal(t, CloseBrace, tokens[1].Type)
	})

	t.Run("arrow operator", func(t *testing.T) {
		lexer := New("->")
		tokens := lexer.Scan()
		require.Equal(t, Arrow, tokens[0].Type)
		require.Equal(t, "->", tokens[0].Value)
	})

	t.Run("colon", func(t *testing.T) {
		lexer := New(":")
		tokens := lexer.Scan()
		require.Equal(t, Colon, tokens[0].Type)
		require.Equal(t, ":", tokens[0].Value)
	})

	t.Run("comments", func(t *testing.T) {
		input := `# This is a comment
model "Test"
  # Another comment
actor "Guest"`

		lexer := New(input)
		tokens := lexer.Scan()

		require.Len(t, tokens, 5)
		require.Equal(t, KeywordModel, tokens[0].Type)
		require.Equal(t, String, tokens[1].Type)
		require.Equal(t, "Test", tokens[1].Value)
		require.Equal(t, KeywordActor, tokens[2].Type)
		require.Equal(t, String, tokens[3].Type)
		require.Equal(t, "Guest", tokens[3].Value)
	})

	t.Run("position tracking", func(t *testing.T) {
		input := `model "Test"
actor "Guest"`

		lexer := New(input)
		tokens := lexer.Scan()

		require.Equal(t, 1, tokens[0].Line)
		require.Equal(t, 1, tokens[0].Column)

		require.Equal(t, 2, tokens[2].Line)
		require.Equal(t, 1, tokens[2].Column)
	})

	t.Run("whitespace handling", func(t *testing.T) {
		input := "model    \t  \t  \"Test\""
		lexer := New(input)
		tokens := lexer.Scan()

		require.Len(t, tokens, 3)
		require.Equal(t, KeywordModel, tokens[0].Type)
		require.Equal(t, String, tokens[1].Type)
	})

	t.Run("EOF token", func(t *testing.T) {
		lexer := New("model")
		tokens := lexer.Scan()

		require.Greater(t, len(tokens), 0)
		require.Equal(t, EOF, tokens[len(tokens)-1].Type)
	})

	t.Run("unterminated string", func(t *testing.T) {
		lexer := New(`"unterminated`)
		lexer.Scan()
		errs := lexer.Errors()

		require.Greater(t, len(errs), 0)
		require.Equal(t, Error, errs[0].Type)
		require.Equal(t, "unterminated string", errs[0].Value)
	})

	t.Run("unrecognized character", func(t *testing.T) {
		lexer := New("model @")
		lexer.Scan()
		errs := lexer.Errors()

		require.Greater(t, len(errs), 0)
		require.Equal(t, Error, errs[0].Type)
	})

	t.Run("empty input", func(t *testing.T) {
		lexer := New("")
		tokens := lexer.Scan()

		require.Len(t, tokens, 1)
		require.Equal(t, EOF, tokens[0].Type)
	})

	t.Run("comment only input", func(t *testing.T) {
		lexer := New(`# Just a comment
# Another line`)
		tokens := lexer.Scan()

		require.Len(t, tokens, 1)
		require.Equal(t, EOF, tokens[0].Type)
	})

	t.Run("complex input", func(t *testing.T) {
		input := `# Hotel Reservation System
model "Hotel Reservation"

actor "Guest"

context "Reservations" {
  aggregate "Reservation" {
    slice "Make Reservation" {
      command MakeReservation {
        fields {
          guestId string required
        }
      }
      event ReservationMade {
        fields {
          reservationId string required
        }
      }
      flow {
        command -> event: MakeReservation -> ReservationMade
      }
    }
  }
}`

		lexer := New(input)
		tokens := lexer.Scan()

		require.Len(t, lexer.Errors(), 0)

		hasModel := false
		hasActor := false
		hasContext := false
		hasCommand := false
		hasArrow := false

		for _, tok := range tokens {
			if tok.Type == KeywordModel {
				hasModel = true
			}
			if tok.Type == KeywordActor {
				hasActor = true
			}
			if tok.Type == KeywordContext {
				hasContext = true
			}
			if tok.Type == KeywordCommand {
				hasCommand = true
			}
			if tok.Type == Arrow {
				hasArrow = true
			}
		}

		require.True(t, hasModel)
		require.True(t, hasActor)
		require.True(t, hasContext)
		require.True(t, hasCommand)
		require.True(t, hasArrow)
		require.Equal(t, EOF, tokens[len(tokens)-1].Type)
	})
}

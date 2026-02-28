package lexer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLexer(t *testing.T) {
	t.Run("keyword tokens", func(t *testing.T) {
		keywords := map[string]TokenType{
			"model":     TokenKeywordModel,
			"actor":     TokenKeywordActor,
			"context":   TokenKeywordContext,
			"aggregate": TokenKeywordAggregate,
			"slice":     TokenKeywordSlice,
			"command":   TokenKeywordCommand,
			"event":     TokenKeywordEvent,
			"fields":    TokenKeywordFields,
			"flow":      TokenKeywordFlow,
		}

		for keyword, expectedType := range keywords {
			t.Run(keyword, func(t *testing.T) {
				lexer := New(keyword)
				tokens := lexer.Tokenize()
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
				tokens := lexer.Tokenize()
				require.Equal(t, TokenIdentifier, tokens[0].Type)
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
				tokens := lexer.Tokenize()
				require.Equal(t, TokenString, tokens[0].Type)
				require.Equal(t, test.expected, tokens[0].Value)
			})
		}
	})

	t.Run("braces", func(t *testing.T) {
		lexer := New("{ }")
		tokens := lexer.Tokenize()
		require.Equal(t, TokenOpenBrace, tokens[0].Type)
		require.Equal(t, TokenCloseBrace, tokens[1].Type)
	})

	t.Run("arrow operator", func(t *testing.T) {
		lexer := New("->")
		tokens := lexer.Tokenize()
		require.Equal(t, TokenArrow, tokens[0].Type)
		require.Equal(t, "->", tokens[0].Value)
	})

	t.Run("colon", func(t *testing.T) {
		lexer := New(":")
		tokens := lexer.Tokenize()
		require.Equal(t, TokenColon, tokens[0].Type)
		require.Equal(t, ":", tokens[0].Value)
	})

	t.Run("comments", func(t *testing.T) {
		input := `# This is a comment
model "Test"
  # Another comment
actor "Guest"`

		lexer := New(input)
		tokens := lexer.Tokenize()

		require.Len(t, tokens, 5)
		require.Equal(t, TokenKeywordModel, tokens[0].Type)
		require.Equal(t, TokenString, tokens[1].Type)
		require.Equal(t, "Test", tokens[1].Value)
		require.Equal(t, TokenKeywordActor, tokens[2].Type)
		require.Equal(t, TokenString, tokens[3].Type)
		require.Equal(t, "Guest", tokens[3].Value)
	})

	t.Run("position tracking", func(t *testing.T) {
		input := `model "Test"
actor "Guest"`

		lexer := New(input)
		tokens := lexer.Tokenize()

		require.Equal(t, 1, tokens[0].Line)
		require.Equal(t, 1, tokens[0].Column)

		require.Equal(t, 2, tokens[2].Line)
		require.Equal(t, 1, tokens[2].Column)
	})

	t.Run("whitespace handling", func(t *testing.T) {
		input := "model    \t  \t  \"Test\""
		lexer := New(input)
		tokens := lexer.Tokenize()

		require.Len(t, tokens, 3)
		require.Equal(t, TokenKeywordModel, tokens[0].Type)
		require.Equal(t, TokenString, tokens[1].Type)
	})

	t.Run("EOF token", func(t *testing.T) {
		lexer := New("model")
		tokens := lexer.Tokenize()

		require.Greater(t, len(tokens), 0)
		require.Equal(t, TokenEOF, tokens[len(tokens)-1].Type)
	})

	t.Run("unterminated string", func(t *testing.T) {
		lexer := New(`"unterminated`)
		lexer.Tokenize()
		errs := lexer.Errors()

		require.Greater(t, len(errs), 0)
		require.Equal(t, TokenError, errs[0].Type)
		require.Equal(t, "unterminated string", errs[0].Value)
	})

	t.Run("unrecognized character", func(t *testing.T) {
		lexer := New("model @")
		lexer.Tokenize()
		errs := lexer.Errors()

		require.Greater(t, len(errs), 0)
		require.Equal(t, TokenError, errs[0].Type)
	})

	t.Run("empty input", func(t *testing.T) {
		lexer := New("")
		tokens := lexer.Tokenize()

		require.Len(t, tokens, 1)
		require.Equal(t, TokenEOF, tokens[0].Type)
	})

	t.Run("comment only input", func(t *testing.T) {
		lexer := New(`# Just a comment
# Another line`)
		tokens := lexer.Tokenize()

		require.Len(t, tokens, 1)
		require.Equal(t, TokenEOF, tokens[0].Type)
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
		tokens := lexer.Tokenize()

		require.Len(t, lexer.Errors(), 0)

		hasModel := false
		hasActor := false
		hasContext := false
		hasCommand := false
		hasArrow := false

		for _, tok := range tokens {
			if tok.Type == TokenKeywordModel {
				hasModel = true
			}
			if tok.Type == TokenKeywordActor {
				hasActor = true
			}
			if tok.Type == TokenKeywordContext {
				hasContext = true
			}
			if tok.Type == TokenKeywordCommand {
				hasCommand = true
			}
			if tok.Type == TokenArrow {
				hasArrow = true
			}
		}

		require.True(t, hasModel)
		require.True(t, hasActor)
		require.True(t, hasContext)
		require.True(t, hasCommand)
		require.True(t, hasArrow)
		require.Equal(t, TokenEOF, tokens[len(tokens)-1].Type)
	})
}

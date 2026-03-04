//go:build unit

package lexer_test

import (
	"testing"

	"github.com/hpcsc/emod/internal/lexer"
	"github.com/stretchr/testify/require"
)

func TestLexer(t *testing.T) {
	t.Run("keyword tokens", func(t *testing.T) {
		keywords := map[string]lexer.Kind{
			"model":           lexer.KeywordModel,
			"actor":           lexer.KeywordActor,
			"context":         lexer.KeywordContext,
			"aggregate":       lexer.KeywordAggregate,
			"slice":           lexer.KeywordSlice,
			"command":         lexer.KeywordCommand,
			"event":           lexer.KeywordEvent,
			"fields":          lexer.KeywordFields,
			"flow":            lexer.KeywordFlow,
			"trigger":         lexer.KeywordTrigger,
			"view":            lexer.KeywordView,
			"automation":      lexer.KeywordAutomation,
			"translation":     lexer.KeywordTranslation,
			"subscribes":      lexer.KeywordSubscribes,
			"target":          lexer.KeywordTarget,
			"external_system": lexer.KeywordExternalSystem,
			"reads":           lexer.KeywordReads,
			"source":          lexer.KeywordSource,
			"external":        lexer.KeywordExternal,
		}

		for keyword, expectedType := range keywords {
			t.Run(keyword, func(t *testing.T) {
				tokens, diags := lexer.Scan(keyword, "test.emod")
				require.Empty(t, diags)
				require.Greater(t, len(tokens), 1)
				require.Equal(t, expectedType, tokens[0].Type)
				require.Equal(t, keyword, tokens[0].Value)
			})
		}
	})

	t.Run("identifiers", func(t *testing.T) {
		tests := []string{"MakeReservation", "string", "required", "date", "foo_bar", "Foo123", "sourced", "externally"}
		for _, id := range tests {
			t.Run(id, func(t *testing.T) {
				tokens, diags := lexer.Scan(id, "test.emod")
				require.Empty(t, diags)
				require.Equal(t, lexer.Identifier, tokens[0].Type)
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
				tokens, diags := lexer.Scan(test.input, "test.emod")
				require.Empty(t, diags)
				require.Equal(t, lexer.String, tokens[0].Type)
				require.Equal(t, test.expected, tokens[0].Value)
			})
		}
	})

	t.Run("braces", func(t *testing.T) {
		tokens, diags := lexer.Scan("{ }", "test.emod")
		require.Empty(t, diags)
		require.Equal(t, lexer.OpenBrace, tokens[0].Type)
		require.Equal(t, lexer.CloseBrace, tokens[1].Type)
	})

	t.Run("brackets", func(t *testing.T) {
		tokens, diags := lexer.Scan("[ ]", "test.emod")
		require.Empty(t, diags)
		require.Equal(t, lexer.OpenBracket, tokens[0].Type)
		require.Equal(t, "[", tokens[0].Value)
		require.Equal(t, lexer.CloseBracket, tokens[1].Type)
		require.Equal(t, "]", tokens[1].Value)
	})

	t.Run("comma", func(t *testing.T) {
		tokens, diags := lexer.Scan(",", "test.emod")
		require.Empty(t, diags)
		require.Equal(t, lexer.Comma, tokens[0].Type)
		require.Equal(t, ",", tokens[0].Value)
	})

	t.Run("arrow operator", func(t *testing.T) {
		tokens, diags := lexer.Scan("->", "test.emod")
		require.Empty(t, diags)
		require.Equal(t, lexer.Arrow, tokens[0].Type)
		require.Equal(t, "->", tokens[0].Value)
	})

	t.Run("colon", func(t *testing.T) {
		tokens, diags := lexer.Scan(":", "test.emod")
		require.Empty(t, diags)
		require.Equal(t, lexer.Colon, tokens[0].Type)
		require.Equal(t, ":", tokens[0].Value)
	})

	t.Run("comments", func(t *testing.T) {
		input := `# This is a comment
model "Test"
  # Another comment
actor "Guest"`

		tokens, diags := lexer.Scan(input, "test.emod")

		require.Empty(t, diags)
		require.Len(t, tokens, 5)
		require.Equal(t, lexer.KeywordModel, tokens[0].Type)
		require.Equal(t, lexer.String, tokens[1].Type)
		require.Equal(t, "Test", tokens[1].Value)
		require.Equal(t, lexer.KeywordActor, tokens[2].Type)
		require.Equal(t, lexer.String, tokens[3].Type)
		require.Equal(t, "Guest", tokens[3].Value)
	})

	t.Run("position tracking", func(t *testing.T) {
		input := `model "Test"
actor "Guest"`

		tokens, diags := lexer.Scan(input, "test.emod")

		require.Empty(t, diags)
		require.Equal(t, 1, tokens[0].Line)
		require.Equal(t, 1, tokens[0].Column)

		require.Equal(t, 2, tokens[2].Line)
		require.Equal(t, 1, tokens[2].Column)
	})

	t.Run("whitespace handling", func(t *testing.T) {
		input := "model    \t  \t  \"Test\""
		tokens, diags := lexer.Scan(input, "test.emod")

		require.Empty(t, diags)
		require.Len(t, tokens, 3)
		require.Equal(t, lexer.KeywordModel, tokens[0].Type)
		require.Equal(t, lexer.String, tokens[1].Type)
	})

	t.Run("EOF token", func(t *testing.T) {
		tokens, diags := lexer.Scan("model", "test.emod")

		require.Empty(t, diags)
		require.Greater(t, len(tokens), 0)
		require.Equal(t, lexer.EOF, tokens[len(tokens)-1].Type)
	})

	t.Run("unterminated string", func(t *testing.T) {
		_, diags := lexer.Scan(`"unterminated`, "test.emod")

		require.Len(t, diags, 1)
		require.Equal(t, "test.emod", diags[0].Filename)
		require.Equal(t, "unterminated string", diags[0].Message)
	})

	t.Run("unrecognized character", func(t *testing.T) {
		_, diags := lexer.Scan("model @", "test.emod")

		require.Len(t, diags, 1)
		require.Equal(t, "test.emod", diags[0].Filename)
		require.Contains(t, diags[0].Message, "unrecognized character")
	})

	t.Run("empty input", func(t *testing.T) {
		tokens, diags := lexer.Scan("", "test.emod")

		require.Empty(t, diags)
		require.Len(t, tokens, 1)
		require.Equal(t, lexer.EOF, tokens[0].Type)
	})

	t.Run("comment only input", func(t *testing.T) {
		tokens, diags := lexer.Scan(`# Just a comment
# Another line`, "test.emod")

		require.Empty(t, diags)
		require.Len(t, tokens, 1)
		require.Equal(t, lexer.EOF, tokens[0].Type)
	})

	t.Run("source external with provider string", func(t *testing.T) {
		input := `source external "Provider"`
		tokens, diags := lexer.Scan(input, "test.emod")
		require.Empty(t, diags)

		require.Len(t, tokens, 4)
		require.Equal(t, lexer.KeywordSource, tokens[0].Type)
		require.Equal(t, "source", tokens[0].Value)
		require.Equal(t, lexer.KeywordExternal, tokens[1].Type)
		require.Equal(t, "external", tokens[1].Value)
		require.Equal(t, lexer.String, tokens[2].Type)
		require.Equal(t, "Provider", tokens[2].Value)
		require.Equal(t, lexer.EOF, tokens[3].Type)
	})

	t.Run("mixed input with new tokens", func(t *testing.T) {
		input := `subscribes [EventA, EventB]`
		tokens, diags := lexer.Scan(input, "test.emod")
		require.Empty(t, diags)

		require.Len(t, tokens, 7)
		require.Equal(t, lexer.KeywordSubscribes, tokens[0].Type)
		require.Equal(t, "subscribes", tokens[0].Value)
		require.Equal(t, lexer.OpenBracket, tokens[1].Type)
		require.Equal(t, lexer.Identifier, tokens[2].Type)
		require.Equal(t, "EventA", tokens[2].Value)
		require.Equal(t, lexer.Comma, tokens[3].Type)
		require.Equal(t, lexer.Identifier, tokens[4].Type)
		require.Equal(t, "EventB", tokens[4].Value)
		require.Equal(t, lexer.CloseBracket, tokens[5].Type)
		require.Equal(t, lexer.EOF, tokens[6].Type)
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

		tokens, diags := lexer.Scan(input, "test.emod")

		require.Empty(t, diags)

		hasModel := false
		hasActor := false
		hasContext := false
		hasCommand := false
		hasArrow := false

		for _, tok := range tokens {
			if tok.Type == lexer.KeywordModel {
				hasModel = true
			}
			if tok.Type == lexer.KeywordActor {
				hasActor = true
			}
			if tok.Type == lexer.KeywordContext {
				hasContext = true
			}
			if tok.Type == lexer.KeywordCommand {
				hasCommand = true
			}
			if tok.Type == lexer.Arrow {
				hasArrow = true
			}
		}

		require.True(t, hasModel)
		require.True(t, hasActor)
		require.True(t, hasContext)
		require.True(t, hasCommand)
		require.True(t, hasArrow)
		require.Equal(t, lexer.EOF, tokens[len(tokens)-1].Type)
	})
}

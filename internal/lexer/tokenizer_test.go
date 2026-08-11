//go:build unit

package lexer_test

import (
	"testing"

	"github.com/hpcsc/emod/internal/lexer"
	"github.com/stretchr/testify/require"
)

func TestLexer(t *testing.T) {
	t.Run("keyword tokens", func(t *testing.T) {
		keywords := lexer.Keywords()
		require.NotEmpty(t, keywords)

		for _, keyword := range keywords {
			t.Run(keyword, func(t *testing.T) {
				tokens, diags := lexer.Scan(keyword, "test.emod")

				require.Empty(t, diags)
				require.Len(t, tokens, 2)
				require.NotEqual(t, lexer.Identifier, tokens[0].Type)
				require.Equal(t, keyword, tokens[0].Value)
				require.Equal(t, keyword, tokens[0].Type.String())
			})
		}
	})

	t.Run("identifiers", func(t *testing.T) {
		tests := []string{"MakeReservation", "string", "required", "date", "foo_bar", "Foo123", "sourced", "externally", "zebra", "Zone"}
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

		t.Run("a string spanning two lines keeps its newline and moves the line counter on", func(t *testing.T) {
			input := `model "Hotel
Reservation"
actor "Guest"`

			tokens, diags := lexer.Scan(input, "test.emod")

			require.Empty(t, diags)
			require.Equal(t, lexer.String, tokens[1].Type)
			require.Equal(t, "Hotel\nReservation", tokens[1].Value)
			require.Equal(t, 1, tokens[1].Line)
			require.Equal(t, lexer.KeywordActor, tokens[2].Type)
			require.Equal(t, 3, tokens[2].Line)
		})
	})

	t.Run("integers", func(t *testing.T) {
		t.Run("records the value and position of a trailing integer", func(t *testing.T) {
			tokens, diags := lexer.Scan("emod 123", "test.emod")

			require.Empty(t, diags)
			require.Len(t, tokens, 3)
			require.Equal(t, lexer.KeywordEmod, tokens[0].Type)
			require.Equal(t, lexer.Integer, tokens[1].Type)
			require.Equal(t, "123", tokens[1].Value)
			require.Equal(t, 1, tokens[1].Line)
			require.Equal(t, 6, tokens[1].Column)
			require.Equal(t, lexer.EOF, tokens[2].Type)
		})

		t.Run("an integer ends at the first non-digit", func(t *testing.T) {
			tokens, diags := lexer.Scan("emod 123\nmodel", "test.emod")

			require.Empty(t, diags)
			require.Len(t, tokens, 4)
			require.Equal(t, lexer.Integer, tokens[1].Type)
			require.Equal(t, "123", tokens[1].Value)
			require.Equal(t, lexer.KeywordModel, tokens[2].Type)
		})

		t.Run("digits at both ends of the range are accepted", func(t *testing.T) {
			tokens, diags := lexer.Scan("emod 90", "test.emod")

			require.Empty(t, diags)
			require.Equal(t, lexer.Integer, tokens[1].Type)
			require.Equal(t, "90", tokens[1].Value)
		})
	})

	t.Run("decimals", func(t *testing.T) {
		t.Run("a fractional part reads as one decimal token keeping its source text", func(t *testing.T) {
			tokens, diags := lexer.Scan("12.50", "test.emod")

			require.Empty(t, diags)
			require.Len(t, tokens, 2)
			require.Equal(t, lexer.Decimal, tokens[0].Type)
			require.Equal(t, "12.50", tokens[0].Value)
			require.Equal(t, 1, tokens[0].Line)
			require.Equal(t, 1, tokens[0].Column)
		})

		t.Run("a dot with no digit after it is not part of the number", func(t *testing.T) {
			tokens, diags := lexer.Scan("12.", "test.emod")

			require.Len(t, diags, 1)
			require.Equal(t, "unrecognized character: .", diags[0].Message)
			require.Equal(t, lexer.Integer, tokens[0].Type)
			require.Equal(t, "12", tokens[0].Value)
		})

		t.Run("names itself apart from the integer kind", func(t *testing.T) {
			require.Equal(t, "decimal", lexer.Decimal.String())
			require.NotEqual(t, lexer.Integer.String(), lexer.Decimal.String())
		})
	})

	t.Run("true and false scan as identifiers, so neither is a keyword", func(t *testing.T) {
		for _, literal := range []string{"true", "false"} {
			t.Run(literal, func(t *testing.T) {
				tokens, diags := lexer.Scan(literal, "test.emod")

				require.Empty(t, diags)
				require.Equal(t, lexer.Identifier, tokens[0].Type)
				require.Equal(t, literal, tokens[0].Value)
				require.NotContains(t, lexer.Keywords(), literal)
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

	t.Run("equals", func(t *testing.T) {
		tokens, diags := lexer.Scan("=", "test.emod")
		require.Empty(t, diags)
		require.Equal(t, lexer.Equals, tokens[0].Type)
		require.Equal(t, "=", tokens[0].Value)
	})

	t.Run("parentheses", func(t *testing.T) {
		tokens, diags := lexer.Scan("( )", "test.emod")
		require.Empty(t, diags)
		require.Equal(t, lexer.OpenParen, tokens[0].Type)
		require.Equal(t, "(", tokens[0].Value)
		require.Equal(t, lexer.CloseParen, tokens[1].Type)
		require.Equal(t, ")", tokens[1].Value)
	})

	t.Run("comments", func(t *testing.T) {
		input := `# This is a comment
model "Test"
  # Another comment
actor "Guest"`

		tokens, diags := lexer.Scan(input, "test.emod")

		require.Empty(t, diags)
		require.Len(t, tokens, 7)

		require.Equal(t, lexer.Comment, tokens[0].Type)
		require.Equal(t, "# This is a comment", tokens[0].Value)
		require.Equal(t, 1, tokens[0].Line)
		require.Equal(t, 1, tokens[0].Column)

		require.Equal(t, lexer.KeywordModel, tokens[1].Type)
		require.Equal(t, lexer.String, tokens[2].Type)
		require.Equal(t, "Test", tokens[2].Value)

		require.Equal(t, lexer.Comment, tokens[3].Type)
		require.Equal(t, "# Another comment", tokens[3].Value)
		require.Equal(t, 3, tokens[3].Line)
		require.Equal(t, 3, tokens[3].Column)

		require.Equal(t, lexer.KeywordActor, tokens[4].Type)
		require.Equal(t, lexer.String, tokens[5].Type)
		require.Equal(t, "Guest", tokens[5].Value)
		require.Equal(t, lexer.EOF, tokens[6].Type)
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
		t.Run("runs of spaces and tabs between tokens are skipped", func(t *testing.T) {
			tokens, diags := lexer.Scan("model    \t  \t  \"Test\"", "test.emod")

			require.Empty(t, diags)
			require.Len(t, tokens, 3)
			require.Equal(t, lexer.KeywordModel, tokens[0].Type)
			require.Equal(t, lexer.String, tokens[1].Type)
		})

		t.Run("input ending in whitespace scans cleanly to EOF", func(t *testing.T) {
			tokens, diags := lexer.Scan("model \t\n", "test.emod")

			require.Empty(t, diags)
			require.Len(t, tokens, 2)
			require.Equal(t, lexer.KeywordModel, tokens[0].Type)
			require.Equal(t, lexer.EOF, tokens[1].Type)
		})
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
		t.Run("a character outside the alphabet is reported", func(t *testing.T) {
			_, diags := lexer.Scan("model @", "test.emod")

			require.Len(t, diags, 1)
			require.Equal(t, "test.emod", diags[0].Filename)
			require.Contains(t, diags[0].Message, "unrecognized character")
		})

		t.Run("a dash that does not begin an arrow is reported, not fatal", func(t *testing.T) {
			_, diags := lexer.Scan("-", "test.emod")

			require.Len(t, diags, 1)
			require.Equal(t, "test.emod", diags[0].Filename)
			require.Contains(t, diags[0].Message, "unrecognized character")
		})
	})

	t.Run("empty input", func(t *testing.T) {
		tokens, diags := lexer.Scan("", "test.emod")

		require.Empty(t, diags)
		require.Len(t, tokens, 1)
		require.Equal(t, lexer.EOF, tokens[0].Type)
	})

	t.Run("comment only input", func(t *testing.T) {
		tokens, diags := lexer.Scan("# Just a comment\n# Another line", "test.emod")

		require.Empty(t, diags)
		require.Len(t, tokens, 3)

		require.Equal(t, lexer.Comment, tokens[0].Type)
		require.Equal(t, "# Just a comment", tokens[0].Value)
		require.Equal(t, 1, tokens[0].Line)
		require.Equal(t, 1, tokens[0].Column)

		require.Equal(t, lexer.Comment, tokens[1].Type)
		require.Equal(t, "# Another line", tokens[1].Value)
		require.Equal(t, 2, tokens[1].Line)
		require.Equal(t, 1, tokens[1].Column)

		require.Equal(t, lexer.EOF, tokens[2].Type)
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
		hasComment := false

		for _, tok := range tokens {
			switch tok.Type {
			case lexer.KeywordModel:
				hasModel = true
			case lexer.KeywordActor:
				hasActor = true
			case lexer.KeywordContext:
				hasContext = true
			case lexer.KeywordCommand:
				hasCommand = true
			case lexer.Arrow:
				hasArrow = true
			case lexer.Comment:
				hasComment = true
			}
		}

		require.True(t, hasModel)
		require.True(t, hasActor)
		require.True(t, hasContext)
		require.True(t, hasCommand)
		require.True(t, hasArrow)
		require.True(t, hasComment)
		require.Equal(t, lexer.Comment, tokens[0].Type)
		require.Equal(t, "# Hotel Reservation System", tokens[0].Value)
		require.Equal(t, lexer.EOF, tokens[len(tokens)-1].Type)
	})

	t.Run("comment token value includes hash prefix", func(t *testing.T) {
		tokens, diags := lexer.Scan("# This is a comment", "test.emod")

		require.Empty(t, diags)
		require.Len(t, tokens, 2)
		require.Equal(t, lexer.Comment, tokens[0].Type)
		require.Equal(t, "# This is a comment", tokens[0].Value)
		require.Equal(t, lexer.EOF, tokens[1].Type)
	})

	t.Run("comment token position tracking", func(t *testing.T) {
		input := "# first comment\n\n  # indented comment"

		tokens, diags := lexer.Scan(input, "test.emod")

		require.Empty(t, diags)
		require.Len(t, tokens, 3)

		require.Equal(t, lexer.Comment, tokens[0].Type)
		require.Equal(t, "# first comment", tokens[0].Value)
		require.Equal(t, 1, tokens[0].Line)
		require.Equal(t, 1, tokens[0].Column)

		require.Equal(t, lexer.Comment, tokens[1].Type)
		require.Equal(t, "# indented comment", tokens[1].Value)
		require.Equal(t, 3, tokens[1].Line)
		require.Equal(t, 3, tokens[1].Column)

		require.Equal(t, lexer.EOF, tokens[2].Type)
	})

	t.Run("Comment kind String returns comment", func(t *testing.T) {
		require.Equal(t, "comment", lexer.Comment.String())
	})
}

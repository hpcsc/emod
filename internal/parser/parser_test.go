//go:build unit

package parser_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/parser"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
)

var ignoreASTPositions = cmpopts.IgnoreTypes(ast.Position{})

func TestParser(t *testing.T) {
	t.Run("version header", func(t *testing.T) {
		t.Run("declaring version 1 leaves the rest of the model untouched", func(t *testing.T) {
			bareTokens, bareLexDiags := lexer.Scan(test.HotelReservation, "test.emod")
			pinnedTokens, pinnedLexDiags := lexer.Scan("emod 1\n"+test.HotelReservation, "test.emod")
			require.Empty(t, bareLexDiags)
			require.Empty(t, pinnedLexDiags)

			bare, bareDiags := parser.New(bareTokens, "test.emod").Parse()
			pinned, pinnedDiags := parser.New(pinnedTokens, "test.emod").Parse()

			require.Empty(t, bareDiags)
			require.Empty(t, pinnedDiags)
			require.Equal(t, 1, bare.Version)
			require.Equal(t, 1, pinned.Version)
			test.RequireEqual(t, bare, pinned, ignoreASTPositions, cmpopts.IgnoreFields(ast.Model{}, "VersionDeclared"))
		})

		t.Run("a declared version is recorded as declared", func(t *testing.T) {
			input := `emod 1
model "Test"`
			tokens, _ := lexer.Scan(input, "test.emod")

			model, diags := parser.New(tokens, "test.emod").Parse()

			require.Empty(t, diags)
			require.Equal(t, 1, model.Version)
			require.True(t, model.VersionDeclared)
			require.Equal(t, "Test", model.Name)
		})

		t.Run("a declared version the tool does not support is rejected", func(t *testing.T) {
			tests := []struct {
				name     string
				declared int
			}{
				{name: "a version below the supported one", declared: 0},
				{name: "a version above the supported one", declared: 2},
			}

			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					input := fmt.Sprintf("emod %d\n%s", tc.declared, test.HotelReservation)
					tokens, lexDiags := lexer.Scan(input, "versions.emod")
					require.Empty(t, lexDiags)

					_, diags := parser.New(tokens, "versions.emod").Parse()

					require.Len(t, diags, 1)
					require.Equal(t, "versions.emod", diags[0].Filename)
					require.Equal(t, 1, diags[0].Line)
					require.Equal(t, diagnostic.Error, diags[0].Severity)
					require.Empty(t, diags[0].RuleName)
					require.Contains(t, diags[0].Message, strconv.Itoa(tc.declared))
					require.Contains(t, diags[0].Message, strconv.Itoa(ast.SupportedVersion))
				})
			}
		})

		t.Run("a rejected version stops the parse before anything below the header is read", func(t *testing.T) {
			const brokenBelowHeader = `model "Test"
foobar {
}
`
			rejectedTokens, _ := lexer.Scan("emod 2\n"+brokenBelowHeader, "versions.emod")
			acceptedTokens, _ := lexer.Scan("emod 1\n"+brokenBelowHeader, "versions.emod")

			_, rejectedDiags := parser.New(rejectedTokens, "versions.emod").Parse()
			_, acceptedDiags := parser.New(acceptedTokens, "versions.emod").Parse()

			require.NotEmpty(t, acceptedDiags, "the same grammar under a supported version reports the breakage below the header")
			for _, d := range acceptedDiags {
				require.Greater(t, d.Line, 1)
			}
			require.Len(t, rejectedDiags, 1)
			require.Equal(t, 1, rejectedDiags[0].Line)
		})

		t.Run("a file without a header is version 1 but undeclared", func(t *testing.T) {
			input := `model "Test"`
			tokens, _ := lexer.Scan(input, "test.emod")

			model, diags := parser.New(tokens, "test.emod").Parse()

			require.Empty(t, diags)
			require.Equal(t, 1, model.Version)
			require.False(t, model.VersionDeclared)
		})

		t.Run("comments above the header stay attached to the model", func(t *testing.T) {
			input := `# Hotel Reservation System
# Second line of the preamble
emod 1
model "Test"`
			tokens, _ := lexer.Scan(input, "test.emod")

			model, diags := parser.New(tokens, "test.emod").Parse()

			require.Empty(t, diags)
			require.Equal(t, 1, model.Version)
			require.True(t, model.VersionDeclared)
			require.Equal(t, "Test", model.Name)
			test.RequireEqual(t, []*ast.Comment{
				{Text: "# Hotel Reservation System"},
				{Text: "# Second line of the preamble"},
			}, model.Comments, ignoreASTPositions)
		})

		t.Run("a malformed header is rejected and the declaration below it still parses", func(t *testing.T) {
			tests := []struct {
				name  string
				input string
			}{
				{
					name: "nothing follows the keyword",
					input: `emod
model "Test"`,
				},
				{
					name: "an identifier follows the keyword",
					input: `emod x
model "Test"`,
				},
				{
					name: "a quoted string follows the keyword",
					input: `emod "1"
model "Test"`,
				},
				{
					name: "a version number too large to represent",
					input: `emod 99999999999999999999
model "Test"`,
				},
				{
					name:  "the model declaration follows the keyword on the same line",
					input: `emod model "Test"`,
				},
			}

			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					tokens, _ := lexer.Scan(tc.input, "versions.emod")

					model, diags := parser.New(tokens, "versions.emod").Parse()

					require.Len(t, diags, 1)
					require.Equal(t, "versions.emod", diags[0].Filename)
					require.Equal(t, 1, diags[0].Line)
					require.Greater(t, diags[0].Column, 0)
					require.Contains(t, diags[0].Message, "version header")
					require.Equal(t, "Test", model.Name)
					require.False(t, model.VersionDeclared)
				})
			}
		})

		t.Run("a version number on the line below the keyword does not form a header", func(t *testing.T) {
			input := `emod
2
model "Test"`
			tokens, _ := lexer.Scan(input, "versions.emod")

			model, diags := parser.New(tokens, "versions.emod").Parse()

			require.Equal(t, 1, model.Version)
			require.False(t, model.VersionDeclared)
			require.Equal(t, "Test", model.Name)
			require.Len(t, diags, 2)
			require.Equal(t, 1, diags[0].Line)
			require.Contains(t, diags[0].Message, "version header")
			require.Equal(t, 2, diags[1].Line)
		})

		t.Run("header after the model declaration is reported as misplaced", func(t *testing.T) {
			input := `model "Test"
emod 2`
			tokens, _ := lexer.Scan(input, "test.emod")

			model, diags := parser.New(tokens, "test.emod").Parse()

			require.Len(t, diags, 1)
			require.Equal(t, 2, diags[0].Line)
			require.Contains(t, diags[0].Message, "version header")
			require.NotContains(t, diags[0].Message, "unrecognized keyword")
			require.Equal(t, "Test", model.Name)
		})

		t.Run("a version number below a misplaced header is reported in its own right", func(t *testing.T) {
			input := `model "Test"
emod
2`
			tokens, _ := lexer.Scan(input, "test.emod")

			model, diags := parser.New(tokens, "test.emod").Parse()

			require.Len(t, diags, 2)
			require.Equal(t, 2, diags[0].Line)
			require.Contains(t, diags[0].Message, "version header")
			require.Equal(t, 3, diags[1].Line)
			require.Equal(t, "Test", model.Name)
		})

		t.Run("keywords are usable as field names", func(t *testing.T) {
			keywords := lexer.Keywords()
			require.NotEmpty(t, keywords)

			for _, keyword := range keywords {
				t.Run(keyword, func(t *testing.T) {
					tokens, _ := lexer.Scan(modelWithField(keyword, "string", "required"), "test.emod")

					model, diags := parser.New(tokens, "test.emod").Parse()

					require.Empty(t, diags)
					test.RequireEqual(t, []*ast.Field{
						{Name: keyword, Type: "string", Modifier: "required"},
					}, model.Contexts[0].Aggregates[0].Slices[0].Commands[0].Fields, ignoreASTPositions)
				})
			}
		})

		t.Run("keywords are usable as field name, type and modifier at once", func(t *testing.T) {
			keywords := lexer.Keywords()
			require.NotEmpty(t, keywords)

			for _, keyword := range keywords {
				t.Run(keyword, func(t *testing.T) {
					tokens, _ := lexer.Scan(modelWithField(keyword, keyword, keyword), "test.emod")

					model, diags := parser.New(tokens, "test.emod").Parse()

					require.Empty(t, diags)
					test.RequireEqual(t, []*ast.Field{
						{Name: keyword, Type: keyword, Modifier: keyword},
					}, model.Contexts[0].Aggregates[0].Slices[0].Commands[0].Fields, ignoreASTPositions)
				})
			}
		})

		t.Run("a keyword in construct name position is rejected", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command source { }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			model, diags := parser.New(tokens, "test.emod").Parse()

			require.NotEmpty(t, diags)
			require.Equal(t, 5, diags[0].Line)
			require.Contains(t, diags[0].Message, "expected identifier after command")
			require.Empty(t, model.Contexts[0].Aggregates[0].Slices[0].Commands)
		})

		t.Run("an integer outside the header is rejected", func(t *testing.T) {
			tests := []struct {
				name  string
				input string
				line  int
			}{
				{
					name: "at top level",
					input: `model "Test"
2`,
					line: 2,
				},
				{
					name: "in field type position",
					input: `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command DoThing {
        fields {
          count 1
        }
      }
    }
  }
}`,
					line: 7,
				},
			}

			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					tokens, _ := lexer.Scan(tc.input, "test.emod")

					_, diags := parser.New(tokens, "test.emod").Parse()

					require.NotEmpty(t, diags)
					require.Equal(t, tc.line, diags[0].Line)
				})
			}
		})
	})

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

		t.Run("an empty block leaves the declaration's description empty", func(t *testing.T) {
			tests := []struct {
				construct string
				input     string
				want      *ast.Model
			}{
				{
					construct: "model",
					input: `model "Test Model" {
}`,
					want: &ast.Model{Version: 1, Name: "Test Model"},
				},
				{
					construct: "actor",
					input: `model "Test"
actor "Guest" {
}`,
					want: &ast.Model{Version: 1, Name: "Test", Actors: []*ast.Actor{{Name: "Guest"}}},
				},
			}

			for _, tc := range tests {
				t.Run(tc.construct, func(t *testing.T) {
					tokens, lexDiags := lexer.Scan(tc.input, "test.emod")
					require.Empty(t, lexDiags)

					model, diags := parser.New(tokens, "test.emod").Parse()

					require.Empty(t, diags)
					test.RequireEqual(t, tc.want, model, ignoreASTPositions)
				})
			}
		})

		t.Run("a block records where it opened and where it closed", func(t *testing.T) {
			tests := []struct {
				construct string
				input     string
				braces    func(*ast.Model) (ast.Position, ast.Position)
			}{
				{
					construct: "model",
					input: `model "Test" {
  description "How the hotel takes bookings"
}`,
					braces: func(m *ast.Model) (ast.Position, ast.Position) {
						return m.OpenPos, m.ClosePos
					},
				},
				{
					construct: "actor",
					input: `model "Test"
actor "Guest" {
  description "Someone who books a room"
}`,
					braces: func(m *ast.Model) (ast.Position, ast.Position) {
						return m.Actors[0].OpenPos, m.Actors[0].ClosePos
					},
				},
			}

			for _, tc := range tests {
				t.Run(tc.construct, func(t *testing.T) {
					tokens, lexDiags := lexer.Scan(tc.input, "test.emod")
					require.Empty(t, lexDiags)

					model, diags := parser.New(tokens, "test.emod").Parse()

					require.Empty(t, diags)
					openLine, openColumn := positionOf(t, tc.input, "{", "{")
					closeLine, closeColumn := positionOf(t, tc.input, "}", "}")
					openPos, closePos := tc.braces(model)
					require.Equal(t, ast.Position{Filename: "test.emod", Line: openLine, Column: openColumn}, openPos)
					require.Equal(t, ast.Position{Filename: "test.emod", Line: closeLine, Column: closeColumn}, closePos)
				})
			}
		})

		t.Run("an entry other than description is reported once and the block still closes", func(t *testing.T) {
			const offending = "descripton"
			tests := []struct {
				construct string
				input     string
				want      *ast.Model
			}{
				{
					construct: "model",
					input: `model "Test" {
  descripton
}
context "Reservations" {
}`,
					want: &ast.Model{
						Version:  1,
						Name:     "Test",
						Contexts: []*ast.Context{{Name: "Reservations"}},
					},
				},
				{
					construct: "actor",
					input: `model "Test"
actor "Guest" {
  descripton
}
context "Reservations" {
}`,
					want: &ast.Model{
						Version:  1,
						Name:     "Test",
						Actors:   []*ast.Actor{{Name: "Guest"}},
						Contexts: []*ast.Context{{Name: "Reservations"}},
					},
				},
			}

			for _, tc := range tests {
				t.Run(tc.construct, func(t *testing.T) {
					tokens, lexDiags := lexer.Scan(tc.input, "test.emod")
					require.Empty(t, lexDiags)

					model, diags := parser.New(tokens, "test.emod").Parse()

					require.Len(t, diags, 1)
					require.Contains(t, diags[0].Message, tc.construct)
					require.Contains(t, diags[0].Message, strconv.Quote(offending))
					test.RequireEqual(t, tc.want, model, ignoreASTPositions)
				})
			}
		})

		t.Run("a block whose brace is never closed names the construct and the line it opened on", func(t *testing.T) {
			tests := []struct {
				construct string
				input     string
			}{
				{
					construct: "model",
					input:     `model "Test" {`,
				},
				{
					construct: "actor",
					input: `model "Test"
actor "Guest" {`,
				},
			}

			for _, tc := range tests {
				t.Run(tc.construct, func(t *testing.T) {
					tokens, lexDiags := lexer.Scan(tc.input, "test.emod")
					require.Empty(t, lexDiags)

					_, diags := parser.New(tokens, "test.emod").Parse()

					require.Len(t, diags, 1)
					openLine, _ := positionOf(t, tc.input, tc.construct+" ", "{")
					require.Equal(t, fmt.Sprintf("unclosed brace for %q block opened at line %d", tc.construct, openLine), diags[0].Message)
				})
			}
		})

		t.Run("a single-line actor leaves the declaration below it to be parsed in its own right", func(t *testing.T) {
			tests := []struct {
				following string
				input     string
				want      *ast.Model
			}{
				{
					following: "another actor",
					input: `model "Test"
actor "Guest"
actor "FrontDesk"`,
					want: &ast.Model{
						Version: 1,
						Name:    "Test",
						Actors:  []*ast.Actor{{Name: "Guest"}, {Name: "FrontDesk"}},
					},
				},
				{
					following: "a context",
					input: `model "Test"
actor "Guest"
context "Reservations" {
  aggregate "Reservation" {
  }
}`,
					want: &ast.Model{
						Version: 1,
						Name:    "Test",
						Actors:  []*ast.Actor{{Name: "Guest"}},
						Contexts: []*ast.Context{{
							Name:       "Reservations",
							Aggregates: []*ast.Aggregate{{Name: "Reservation"}},
						}},
					},
				},
				{
					following: "a model",
					input: `actor "Guest"
model "Test"`,
					want: &ast.Model{
						Version: 1,
						Name:    "Test",
						Actors:  []*ast.Actor{{Name: "Guest"}},
					},
				},
			}

			for _, tc := range tests {
				t.Run(tc.following, func(t *testing.T) {
					tokens, lexDiags := lexer.Scan(tc.input, "test.emod")
					require.Empty(t, lexDiags)

					model, diags := parser.New(tokens, "test.emod").Parse()

					require.Empty(t, diags)
					test.RequireEqual(t, tc.want, model, ignoreASTPositions)
				})
			}
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

		t.Run("invariants are recorded in declaration order wherever they sit in the aggregate block", func(t *testing.T) {
			input := `model "Test"
context "Reservations" {
  aggregate "Reservation" {
    description "A booking of a room for a stay"
    invariant NoDoubleBooking "A room is held by at most one reservation per night"
    slice "Make Reservation" {
    }
    invariant WithinCapacity "A reservation never seats more guests than the room holds"
  }
}`
			tokens, lexDiags := lexer.Scan(input, "test.emod")
			require.Empty(t, lexDiags)

			model, diags := parser.New(tokens, "test.emod").Parse()

			require.Empty(t, diags)
			aggregate := model.Contexts[0].Aggregates[0]
			require.Equal(t, "A booking of a room for a stay", aggregate.Description)
			require.Len(t, aggregate.Slices, 1)
			require.Equal(t, "Make Reservation", aggregate.Slices[0].Name)
			test.RequireEqual(t, []*ast.Invariant{
				declaredInvariant(t, "test.emod", input, "NoDoubleBooking", "A room is held by at most one reservation per night"),
				declaredInvariant(t, "test.emod", input, "WithinCapacity", "A reservation never seats more guests than the room holds"),
			}, aggregate.Invariants)
		})

		t.Run("a malformed invariant is reported at its keyword and the entry below it still parses", func(t *testing.T) {
			tests := []struct {
				malformed    string
				entry        string
				wantMessages []string
			}{
				{
					malformed:    "an identifier with no statement",
					entry:        `    invariant NoDoubleBooking`,
					wantMessages: []string{"expected quoted statement after invariant name"},
				},
				{
					malformed:    "neither an identifier nor a statement",
					entry:        `    invariant`,
					wantMessages: []string{"expected identifier after invariant"},
				},
				{
					malformed:    "a statement written without quotes",
					entry:        `    invariant NoDoubleBooking A room is held by at most one reservation per night`,
					wantMessages: []string{"expected quoted statement after invariant name"},
				},
				{
					malformed:    "a quoted name where an identifier belongs",
					entry:        `    invariant "NoDoubleBooking" "A room is held by at most one reservation per night"`,
					wantMessages: []string{"expected identifier after invariant"},
				},
				{
					malformed: "a statement written on the line below the name",
					entry: `    invariant NoDoubleBooking
    "A room is held by at most one reservation per night"`,
					wantMessages: []string{
						"expected quoted statement after invariant name",
						"expected description, invariant or slice in aggregate",
					},
				},
			}

			for _, tc := range tests {
				t.Run(tc.malformed, func(t *testing.T) {
					input := fmt.Sprintf(`model "Test"
context "Reservations" {
  aggregate "Reservation" {
%s
    invariant WithinCapacity "A reservation never seats more guests than the room holds"
    slice "Make Reservation" {
    }
  }
}`, tc.entry)
					tokens, lexDiags := lexer.Scan(input, "test.emod")
					require.Empty(t, lexDiags)

					model, diags := parser.New(tokens, "test.emod").Parse()

					var messages []string
					for _, diag := range diags {
						messages = append(messages, diag.Message)
					}
					require.Equal(t, tc.wantMessages, messages)
					line, column := positionOf(t, input, "invariant", "invariant")
					require.Equal(t, line, diags[0].Line)
					require.Equal(t, column, diags[0].Column)
					require.Equal(t, "test.emod", diags[0].Filename)

					aggregate := model.Contexts[0].Aggregates[0]
					test.RequireEqual(t, []*ast.Invariant{{
						Name:      "WithinCapacity",
						Statement: "A reservation never seats more guests than the room holds",
					}}, aggregate.Invariants, ignoreASTPositions)
					require.Len(t, aggregate.Slices, 1)
					require.Equal(t, "Make Reservation", aggregate.Slices[0].Name)
				})
			}
		})

		t.Run("an invariant with no statement ahead of the closing brace leaves the aggregate closed", func(t *testing.T) {
			input := `model "Test"
context "Reservations" {
  aggregate "Reservation" {
    slice "Make Reservation" {
    }
    invariant NoDoubleBooking }
}`
			tokens, lexDiags := lexer.Scan(input, "test.emod")
			require.Empty(t, lexDiags)

			model, diags := parser.New(tokens, "test.emod").Parse()

			require.Len(t, diags, 1)
			require.Equal(t, "expected quoted statement after invariant name", diags[0].Message)
			aggregate := model.Contexts[0].Aggregates[0]
			require.Empty(t, aggregate.Invariants)
			require.Len(t, aggregate.Slices, 1)
			require.Equal(t, "Make Reservation", aggregate.Slices[0].Name)
			require.Equal(t, astPositionOf(t, "test.emod", input, "invariant NoDoubleBooking", "}"), aggregate.ClosePos)
			require.NotZero(t, model.Contexts[0].ClosePos.Line)
		})

		t.Run("keywords are usable as invariant names", func(t *testing.T) {
			const statement = "A room is held by at most one reservation per night"
			keywords := lexer.Keywords()
			require.NotEmpty(t, keywords)

			for _, keyword := range keywords {
				t.Run(keyword, func(t *testing.T) {
					tokens, lexDiags := lexer.Scan(modelWithInvariant(keyword, statement), "test.emod")
					require.Empty(t, lexDiags)

					model, diags := parser.New(tokens, "test.emod").Parse()

					require.Empty(t, diags)
					test.RequireEqual(t, []*ast.Invariant{{
						Name:      keyword,
						Statement: statement,
					}}, model.Contexts[0].Aggregates[0].Invariants, ignoreASTPositions)
				})
			}
		})

		t.Run("a context records its own invariants in declaration order whatever its mode", func(t *testing.T) {
			tests := []struct {
				mode   string
				clause string
			}{
				{mode: "mode dcb", clause: " mode dcb"},
				{mode: "mode aggregate", clause: " mode aggregate"},
				{mode: "no mode clause", clause: ""},
			}

			for _, tc := range tests {
				t.Run(tc.mode, func(t *testing.T) {
					input := fmt.Sprintf(`model "Test"
context "Reading Room"%s {
  invariant OneReaderPerDesk "A desk seats at most one reader at any moment"
  invariant OneDeskPerReader "A reader holds at most one desk for the length of a session"
  slice "Claim Desk" {
  }
}`, tc.clause)
					tokens, lexDiags := lexer.Scan(input, "test.emod")
					require.Empty(t, lexDiags)

					model, diags := parser.New(tokens, "test.emod").Parse()

					require.Empty(t, diags)
					context := model.Contexts[0]
					test.RequireEqual(t, []*ast.Invariant{
						declaredInvariant(t, "test.emod", input, "OneReaderPerDesk", "A desk seats at most one reader at any moment"),
						declaredInvariant(t, "test.emod", input, "OneDeskPerReader", "A reader holds at most one desk for the length of a session"),
					}, context.Invariants)
					require.Len(t, context.Slices, 1)
					require.Equal(t, "Claim Desk", context.Slices[0].Name)
				})
			}
		})

		t.Run("an aggregate's invariants and its enclosing context's invariants stay in their own homes", func(t *testing.T) {
			input := `model "Test"
context "Lending" {
  invariant BorrowingLimit "A member holds at most five copies at one time"
  aggregate "Loan" {
    invariant BorrowingLimit "A loan is refused once the member already holds five copies"
    slice "Borrow Copy" {
    }
  }
}`
			tokens, lexDiags := lexer.Scan(input, "test.emod")
			require.Empty(t, lexDiags)

			model, diags := parser.New(tokens, "test.emod").Parse()

			require.Empty(t, diags)
			context := model.Contexts[0]
			test.RequireEqual(t, []*ast.Invariant{{
				Name:      "BorrowingLimit",
				Statement: "A member holds at most five copies at one time",
			}}, context.Invariants, ignoreASTPositions)
			test.RequireEqual(t, []*ast.Invariant{{
				Name:      "BorrowingLimit",
				Statement: "A loan is refused once the member already holds five copies",
			}}, context.Aggregates[0].Invariants, ignoreASTPositions)
		})

		t.Run("a context-level invariant with no statement is reported once and the entry below it still parses", func(t *testing.T) {
			input := `model "Test"
context "Reading Room" mode dcb {
  invariant OneReaderPerDesk
  invariant OneDeskPerReader "A reader holds at most one desk for the length of a session"
  slice "Claim Desk" {
  }
}`
			tokens, lexDiags := lexer.Scan(input, "test.emod")
			require.Empty(t, lexDiags)

			model, diags := parser.New(tokens, "test.emod").Parse()

			require.Len(t, diags, 1)
			require.Equal(t, "expected quoted statement after invariant name", diags[0].Message)
			line, column := positionOf(t, input, "invariant OneReaderPerDesk", "invariant")
			require.Equal(t, line, diags[0].Line)
			require.Equal(t, column, diags[0].Column)

			context := model.Contexts[0]
			test.RequireEqual(t, []*ast.Invariant{{
				Name:      "OneDeskPerReader",
				Statement: "A reader holds at most one desk for the length of a session",
			}}, context.Invariants, ignoreASTPositions)
			require.Len(t, context.Slices, 1)
			require.Equal(t, "Claim Desk", context.Slices[0].Name)
		})

		t.Run("the shared invariant model declares invariants in both homes, each ahead of a later entry", func(t *testing.T) {
			tokens, lexDiags := lexer.Scan(test.InvariantLibraryLending, "invariants.emod")
			require.Empty(t, lexDiags)

			model, diags := parser.New(tokens, "invariants.emod").Parse()

			require.Empty(t, diags)
			var contextsDeclaring, aggregatesDeclaring int
			for _, context := range model.Contexts {
				if len(context.Invariants) > 0 {
					contextsDeclaring++
					requireAnInvariantAheadOfALaterSlice(t, "context "+context.Name, context.Invariants, context.Slices)
				}
				for _, aggregate := range context.Aggregates {
					if len(aggregate.Invariants) > 0 {
						aggregatesDeclaring++
						requireAnInvariantAheadOfALaterSlice(t, "aggregate "+aggregate.Name, aggregate.Invariants, aggregate.Slices)
					}
				}
			}

			require.NotZero(t, aggregatesDeclaring, "no aggregate declares an invariant")
			require.NotZero(t, contextsDeclaring, "no context declares an invariant of its own")
		})

		t.Run("a slice records every spec it declares, in order, with its given history, when command and then events", func(t *testing.T) {
			input := `model "Library Lending"
context "Lending" {
  aggregate "Loan" {
    slice "Borrow a Copy" {
      spec "borrows a free copy" {
        given []
        when BorrowCopy
        then [CopyBorrowed]
      }
      spec "renews an active loan" {
        given [CopyBorrowed, CopyReturned]
        when RenewLoan
        then [LoanRenewed, RenewalRecorded]
      }
    }
  }
}`
			tokens, lexDiags := lexer.Scan(input, "test.emod")
			require.Empty(t, lexDiags)

			model, diags := parser.New(tokens, "test.emod").Parse()

			require.Empty(t, diags)
			slice := model.Contexts[0].Aggregates[0].Slices[0]
			test.RequireEqual(t, []*ast.Spec{
				declaredSpec(t, "test.emod", input, "borrows a free copy", nil, "BorrowCopy", []string{"CopyBorrowed"}),
				declaredSpec(t, "test.emod", input, "renews an active loan", []string{"CopyBorrowed", "CopyReturned"}, "RenewLoan", []string{"LoanRenewed", "RenewalRecorded"}),
			}, slice.Specs, cmpopts.IgnoreFields(ast.Spec{}, "OpenPos", "ClosePos"))
			for _, spec := range slice.Specs {
				require.NotZero(t, spec.OpenPos.Line, spec.Name)
				require.NotZero(t, spec.ClosePos.Line, spec.Name)
			}
		})

		t.Run("differently arranged spellings of one spec parse to the same block", func(t *testing.T) {
			tests := []struct {
				name      string
				canonical string
				variant   string
				wantGiven []string
			}{
				{
					name: "an omitted given history reads as the empty one",
					canonical: `given []
        when BorrowCopy
        then [CopyBorrowed]`,
					variant: `when BorrowCopy
        then [CopyBorrowed]`,
				},
				{
					name: "when written ahead of given",
					canonical: `given [CopyReturned]
        when BorrowCopy
        then [CopyBorrowed]`,
					variant: `when BorrowCopy
        given [CopyReturned]
        then [CopyBorrowed]`,
					wantGiven: []string{"CopyReturned"},
				},
				{
					name: "an entry written twice keeps the value written last",
					canonical: `given [CopyReturned]
        when BorrowCopy
        then [CopyBorrowed]`,
					variant: `given [CopyBorrowed]
        given [CopyReturned]
        when RenewLoan
        when BorrowCopy
        then [CopyBorrowed]`,
					wantGiven: []string{"CopyReturned"},
				},
				{
					name: "an entry whose value sits on the line below its keyword",
					canonical: `given [CopyReturned]
        when BorrowCopy
        then [CopyBorrowed]`,
					variant: `given
          [CopyReturned]
        when
          BorrowCopy
        then
          [CopyBorrowed]`,
					wantGiven: []string{"CopyReturned"},
				},
				{
					name: "a list broken across lines between its brackets",
					canonical: `given [CopyBorrowed, CopyReturned]
        when BorrowCopy
        then [CopyBorrowed]`,
					variant: `given [
          CopyBorrowed,
          CopyReturned,
        ]
        when BorrowCopy
        then [CopyBorrowed]`,
					wantGiven: []string{"CopyBorrowed", "CopyReturned"},
				},
			}

			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					canonicalTokens, canonicalLexDiags := lexer.Scan(modelWithSpecEntries(tc.canonical), "test.emod")
					variantTokens, variantLexDiags := lexer.Scan(modelWithSpecEntries(tc.variant), "test.emod")
					require.Empty(t, canonicalLexDiags)
					require.Empty(t, variantLexDiags)

					canonical, canonicalDiags := parser.New(canonicalTokens, "test.emod").Parse()
					variant, variantDiags := parser.New(variantTokens, "test.emod").Parse()

					require.Empty(t, canonicalDiags)
					require.Empty(t, variantDiags)
					canonicalSpec := canonical.Contexts[0].Aggregates[0].Slices[0].Specs[0]
					variantSpec := variant.Contexts[0].Aggregates[0].Slices[0].Specs[0]
					require.Equal(t, tc.wantGiven, specElementNames(canonicalSpec.Given))
					require.Equal(t, tc.wantGiven, specElementNames(variantSpec.Given))
					test.RequireEqual(t, canonicalSpec, variantSpec, ignoreASTPositions)
				})
			}
		})

		t.Run("a spec declared among a slice's other entries leaves those entries intact", func(t *testing.T) {
			input := `model "Library Lending"
context "Lending" {
  aggregate "Loan" {
    slice "Borrow a Copy" {
      command BorrowCopy {
        fields {
          copyId string required
        }
      }
      spec "borrows a free copy" {
        given []
        when BorrowCopy
        then [CopyBorrowed]
      }
      event CopyBorrowed {
        fields {
          copyId string required
        }
      }
      flow {
        command -> event: BorrowCopy -> CopyBorrowed
      }
      spec "borrows a returned copy" {
        given [CopyBorrowed, CopyReturned]
        when BorrowCopy
        then [CopyBorrowed]
      }
    }
  }
}`
			tokens, lexDiags := lexer.Scan(input, "test.emod")
			require.Empty(t, lexDiags)

			model, diags := parser.New(tokens, "test.emod").Parse()

			require.Empty(t, diags)
			slice := model.Contexts[0].Aggregates[0].Slices[0]
			require.Len(t, slice.Commands, 1)
			require.Equal(t, "BorrowCopy", slice.Commands[0].Name)
			require.Len(t, slice.Events, 1)
			require.Equal(t, "CopyBorrowed", slice.Events[0].Name)
			test.RequireEqual(t, []*ast.Flow{{CommandName: "BorrowCopy", EventName: "CopyBorrowed"}}, slice.Flows, ignoreASTPositions)
			var specNames []string
			for _, spec := range slice.Specs {
				specNames = append(specNames, spec.Name)
			}
			require.Equal(t, []string{"borrows a free copy", "borrows a returned copy"}, specNames)
		})

		t.Run("a spec whose then names a rejected invariant states a different outcome from one listing events", func(t *testing.T) {
			input := `model "Library Lending"
context "Lending" {
  aggregate "Loan" {
    invariant OneCopyPerLoan "A loan covers exactly one copy of one title"
    slice "Borrow a Copy" {
      spec "borrows a free copy" {
        given []
        when BorrowCopy
        then [CopyBorrowed]
      }
      spec "refuses a copy already on loan" {
        given [CopyBorrowed]
        when BorrowCopy
        then rejected OneCopyPerLoan
      }
    }
  }
}`
			tokens, lexDiags := lexer.Scan(input, "test.emod")
			require.Empty(t, lexDiags)

			model, diags := parser.New(tokens, "test.emod").Parse()

			require.Empty(t, diags)
			specs := model.Contexts[0].Aggregates[0].Slices[0].Specs
			require.Len(t, specs, 2)
			require.Equal(t, []string{"events", "rejection"}, []string{outcomeKind(specs[0].Then), outcomeKind(specs[1].Then)})
			test.RequireEqual(t, &ast.ThenRejected{
				InvariantName: "OneCopyPerLoan",
				InvariantPos:  astPositionOf(t, "test.emod", input, "then rejected OneCopyPerLoan", "OneCopyPerLoan"),
			}, specs[1].Then)
		})

		t.Run("an invariant name spelling a keyword is accepted after rejected", func(t *testing.T) {
			keywords := lexer.Keywords()
			require.NotEmpty(t, keywords)

			for _, keyword := range keywords {
				t.Run(keyword, func(t *testing.T) {
					entries := `when BorrowCopy
        then rejected ` + keyword
					tokens, lexDiags := lexer.Scan(modelWithSpecEntries(entries), "test.emod")
					require.Empty(t, lexDiags)

					model, diags := parser.New(tokens, "test.emod").Parse()

					require.Empty(t, diags)
					spec := model.Contexts[0].Aggregates[0].Slices[0].Specs[0]
					test.RequireEqual(t, &ast.ThenRejected{InvariantName: keyword}, spec.Then, ignoreASTPositions)
				})
			}
		})

		t.Run("the shared spec model states both outcomes in both slice homes, each optional part exercised mid-block", func(t *testing.T) {
			model := test.SpecLibraryLendingModel(t)

			var aggregateRejections, dcbContextRejections int
			for _, context := range model.Contexts {
				rejections := requireRejectionsDeclaredBy(t, fmt.Sprintf("context %q", context.Name), context.Invariants, context.Slices)
				if context.Mode == "dcb" {
					dcbContextRejections += rejections
				}
				for _, aggregate := range context.Aggregates {
					aggregateRejections += requireRejectionsDeclaredBy(t, fmt.Sprintf("aggregate %q", aggregate.Name), aggregate.Invariants, aggregate.Slices)
				}
			}

			require.NotZero(t, aggregateRejections, "no slice nested in an aggregate states a rejection")
			require.NotZero(t, dcbContextRejections, "no slice declared on a dcb context states a rejection")
			requireSpecCoverage(t, test.SpecLibraryLending, declaredSpecs(model))
			requireASpecAheadOfALaterEntry(t, model)
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

		t.Run("a field without a modifier does not absorb the next line's field name", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command TestCommand {
        fields {
          roomType string
          guestId string required
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			model, diags := parser.New(tokens, "test.emod").Parse()

			require.Empty(t, diags)
			cmd := model.Contexts[0].Aggregates[0].Slices[0].Commands[0]
			test.RequireEqual(t, []*ast.Field{
				{Name: "roomType", Type: "string"},
				{Name: "guestId", Type: "string", Modifier: "required"},
			}, cmd.Fields, ignoreASTPositions)
		})

		t.Run("a field written entirely on one line keeps its modifier", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command TestCommand {
        fields { guestId string required }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			model, diags := parser.New(tokens, "test.emod").Parse()

			require.Empty(t, diags)
			cmd := model.Contexts[0].Aggregates[0].Slices[0].Commands[0]
			test.RequireEqual(t, []*ast.Field{
				{Name: "guestId", Type: "string", Modifier: "required"},
			}, cmd.Fields, ignoreASTPositions)
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
		t.Run("trigger alongside command, event, and flow", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      trigger "Reservation Form" {
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
      trigger "Name" {
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
      trigger "Name" {
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

		t.Run("trigger with name, actor, and reads records its position", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      trigger "Reservation Form" {
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
			require.Equal(t, "Reservation Form", slice.Trigger.Name)
			require.Equal(t, ast.Position{Filename: "test.emod", Line: 5, Column: 15}, slice.Trigger.NamePos)
			require.Equal(t, "Guest", slice.Trigger.Actor)
			require.Equal(t, "AvailableRoomsView", slice.Trigger.Reads)
		})

		t.Run("trigger with description", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      trigger "Reservation Form" {
        description "The booking form on the public site"
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
			require.Equal(t, "Reservation Form", slice.Trigger.Name)
			require.Equal(t, "The booking form on the public site", slice.Trigger.Description)
			require.Equal(t, "Guest", slice.Trigger.Actor)
			require.Equal(t, "AvailableRoomsView", slice.Trigger.Reads)
		})

		t.Run("trigger with empty body leaves actor and reads empty", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      trigger "Reservation Form" {
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
			require.Equal(t, "", slice.Trigger.Actor)
			require.Equal(t, "", slice.Trigger.Reads)
		})

		t.Run("trigger reports single diagnostic when name is missing", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      trigger { }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			require.Len(t, errs, 1)
			require.Contains(t, errs[0].Message, "quoted name after trigger")
			require.Equal(t, "test.emod", errs[0].Filename)
			require.Equal(t, 5, errs[0].Line)
			require.Equal(t, 15, errs[0].Column)
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
		t.Run("automation with activation event, command, and target context", func(t *testing.T) {
			input := modelWithAutomation("ConfirmationEmailReactor", `        on RoomReserved
        command SendConfirmationEmail
        target context Notifications`)
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			slice := model.Contexts[0].Aggregates[0].Slices[0]
			require.Len(t, slice.Automations, 1)
			a := slice.Automations[0]
			require.Equal(t, "ConfirmationEmailReactor", a.Name)
			require.Equal(t, "RoomReserved", a.OnEvent)
			require.Equal(t, "SendConfirmationEmail", a.Command)
			require.Equal(t, "Notifications", a.TargetContext)
		})

		t.Run("automation without target context", func(t *testing.T) {
			input := modelWithAutomation("Reactor", `        on SomeEvent
        command SomeCmd`)
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Len(t, errs, 0)
			a := model.Contexts[0].Aggregates[0].Slices[0].Automations[0]
			require.Equal(t, "Reactor", a.Name)
			require.Equal(t, "SomeEvent", a.OnEvent)
			require.Equal(t, "SomeCmd", a.Command)
			require.Equal(t, "", a.TargetContext)
		})

		t.Run("automation activating on an event records the event name and where it is written", func(t *testing.T) {
			tests := []struct {
				name    string
				entries string
			}{
				{
					name: "as the first entry of the block",
					entries: `        on RoomReserved
        command SendConfirmationEmail
        target context Notifications`,
				},
				{
					name: "between the command and the target context",
					entries: `        command SendConfirmationEmail
        on RoomReserved
        target context Notifications`,
				},
				{
					name: "after the target context",
					entries: `        command SendConfirmationEmail
        target context Notifications
        on RoomReserved`,
				},
			}

			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					input := modelWithAutomation("ConfirmationEmailReactor", tc.entries)
					tokens, lexDiags := lexer.Scan(input, "test.emod")
					require.Empty(t, lexDiags)

					model, diags := parser.New(tokens, "test.emod").Parse()

					require.Empty(t, diags)
					automation := model.Contexts[0].Aggregates[0].Slices[0].Automations[0]
					require.Equal(t, "RoomReserved", automation.OnEvent)
					require.Equal(t, astPositionOf(t, "test.emod", input, "on RoomReserved", "RoomReserved"), automation.OnEventPos)
				})
			}
		})

		t.Run("automation declaring on twice keeps the event named last", func(t *testing.T) {
			input := modelWithAutomation("ConfirmationEmailReactor", `        on RoomReserved
        on RoomReleased
        command SendConfirmationEmail`)
			tokens, lexDiags := lexer.Scan(input, "test.emod")
			require.Empty(t, lexDiags)

			model, diags := parser.New(tokens, "test.emod").Parse()

			require.Empty(t, diags)
			automation := model.Contexts[0].Aggregates[0].Slices[0].Automations[0]
			require.Equal(t, "RoomReleased", automation.OnEvent)
			require.Equal(t, astPositionOf(t, "test.emod", input, "on RoomReleased", "RoomReleased"), automation.OnEventPos)
		})

		t.Run("automation activating on a schedule records the expression and where it is written", func(t *testing.T) {
			tests := []struct {
				name    string
				entries string
			}{
				{
					name: "as the first entry of the block",
					entries: `        every "0 2 * * *"
        command SendConfirmationEmail
        target context Notifications`,
				},
				{
					name: "after the target context",
					entries: `        command SendConfirmationEmail
        target context Notifications
        every "0 2 * * *"`,
				},
			}

			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					input := modelWithAutomation("NightlyExpirySweep", tc.entries)
					tokens, lexDiags := lexer.Scan(input, "test.emod")
					require.Empty(t, lexDiags)

					model, diags := parser.New(tokens, "test.emod").Parse()

					require.Empty(t, diags)
					automation := model.Contexts[0].Aggregates[0].Slices[0].Automations[0]
					require.Equal(t, "0 2 * * *", automation.Schedule)
					require.Equal(t, astPositionOf(t, "test.emod", input, `every "0 2 * * *"`, `"0 2 * * *"`), automation.SchedulePos)
					require.Equal(t, "", automation.OnEvent)
				})
			}
		})

		t.Run("automation declaring every twice keeps the schedule written last", func(t *testing.T) {
			input := modelWithAutomation("NightlyExpirySweep", `        every "0 2 * * *"
        every "5m"
        command ExpireHold`)
			tokens, lexDiags := lexer.Scan(input, "test.emod")
			require.Empty(t, lexDiags)

			model, diags := parser.New(tokens, "test.emod").Parse()

			require.Empty(t, diags)
			automation := model.Contexts[0].Aggregates[0].Slices[0].Automations[0]
			require.Equal(t, "5m", automation.Schedule)
			require.Equal(t, astPositionOf(t, "test.emod", input, `every "5m"`, `"5m"`), automation.SchedulePos)
		})

		t.Run("an automation activating on a schedule sits beside one activating on an event", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      automation ConfirmationEmailReactor {
        on RoomReserved
        command SendConfirmationEmail
      }
      automation HoldSweeper {
        every "5m"
        command ExpireHold
      }
    }
  }
}`
			tokens, lexDiags := lexer.Scan(input, "test.emod")
			require.Empty(t, lexDiags)

			model, diags := parser.New(tokens, "test.emod").Parse()

			require.Empty(t, diags)
			type activation struct {
				onEvent  string
				schedule string
			}
			declared := map[string]activation{}
			for _, automation := range model.Contexts[0].Aggregates[0].Slices[0].Automations {
				declared[automation.Name] = activation{onEvent: automation.OnEvent, schedule: automation.Schedule}
			}
			require.Equal(t, map[string]activation{
				"ConfirmationEmailReactor": {onEvent: "RoomReserved"},
				"HoldSweeper":              {schedule: "5m"},
			}, declared)
		})

		t.Run("an event whose name differs from a keyword only in case is still an event name", func(t *testing.T) {
			for _, name := range []string{"On", "Every"} {
				t.Run(name, func(t *testing.T) {
					input := fmt.Sprintf(`model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      event %[1]s {
        fields {
          reservationId string required
        }
      }
      automation ConfirmationEmailReactor {
        on %[1]s
        command SendConfirmationEmail
      }
    }
  }
}`, name)
					tokens, lexDiags := lexer.Scan(input, "test.emod")
					require.Empty(t, lexDiags)

					model, diags := parser.New(tokens, "test.emod").Parse()

					require.Empty(t, diags)
					slice := model.Contexts[0].Aggregates[0].Slices[0]
					require.Equal(t, name, slice.Events[0].Name)
					require.Equal(t, name, slice.Automations[0].OnEvent)
					require.Equal(t, astPositionOf(t, "test.emod", input, "on "+name, name), slice.Automations[0].OnEventPos)
				})
			}
		})

		t.Run("automation reading a view records the view name and where it is written", func(t *testing.T) {
			tests := []struct {
				name    string
				entries string
			}{
				{
					name: "between the activation event and the command",
					entries: `        on RoomReserved
        reads PendingConfirmationsView
        command SendConfirmationEmail`,
				},
				{
					name: "as the first entry of the block",
					entries: `        reads PendingConfirmationsView
        on RoomReserved
        command SendConfirmationEmail`,
				},
				{
					name: "after the target context",
					entries: `        on RoomReserved
        command SendConfirmationEmail
        target context Notifications
        reads PendingConfirmationsView`,
				},
			}

			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					input := modelWithAutomation("ConfirmationEmailReactor", tc.entries)
					tokens, lexDiags := lexer.Scan(input, "test.emod")
					require.Empty(t, lexDiags)

					model, diags := parser.New(tokens, "test.emod").Parse()

					require.Empty(t, diags)
					automation := model.Contexts[0].Aggregates[0].Slices[0].Automations[0]
					require.Equal(t, "PendingConfirmationsView", automation.Reads)
					require.Equal(t, astPositionOf(t, "test.emod", input, "reads PendingConfirmationsView", "PendingConfirmationsView"), automation.ReadsPos)
				})
			}
		})

		t.Run("automation declaring reads twice keeps the view named last", func(t *testing.T) {
			input := modelWithAutomation("ConfirmationEmailReactor", `        on RoomReserved
        reads StaleConfirmationsView
        reads PendingConfirmationsView
        command SendConfirmationEmail`)
			tokens, lexDiags := lexer.Scan(input, "test.emod")
			require.Empty(t, lexDiags)

			model, diags := parser.New(tokens, "test.emod").Parse()

			require.Empty(t, diags)
			automation := model.Contexts[0].Aggregates[0].Slices[0].Automations[0]
			require.Equal(t, "PendingConfirmationsView", automation.Reads)
			require.Equal(t, astPositionOf(t, "test.emod", input, "reads PendingConfirmationsView", "PendingConfirmationsView"), automation.ReadsPos)
		})

		t.Run("a view read by one automation is not read by a sibling that omits reads", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      automation ConfirmationEmailReactor {
        on RoomReserved
        reads PendingConfirmationsView
        command SendConfirmationEmail
      }
      automation AuditTrailReactor {
        on RoomReserved
        command RecordReservationAudit
      }
    }
  }
}`
			tokens, lexDiags := lexer.Scan(input, "test.emod")
			require.Empty(t, lexDiags)

			model, diags := parser.New(tokens, "test.emod").Parse()

			require.Empty(t, diags)
			readsByAutomation := map[string]string{}
			for _, automation := range model.Contexts[0].Aggregates[0].Slices[0].Automations {
				readsByAutomation[automation.Name] = automation.Reads
			}
			require.Equal(t, map[string]string{
				"ConfirmationEmailReactor": "PendingConfirmationsView",
				"AuditTrailReactor":        "",
			}, readsByAutomation)
		})

		t.Run("automation is stored in slice AST node", func(t *testing.T) {
			input := modelWithAutomation("Reactor", `        on SomeEvent
        command SomeCmd`)
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

		t.Run("a slice-level trigger block and an automation activating on an event are kept apart", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      trigger "Reservation Form" {
        actor Guest
        reads AvailableRoomsView
      }
      automation ConfirmationEmailReactor {
        on RoomReserved
        command SendConfirmationEmail
      }
    }
  }
}`
			tokens, lexDiags := lexer.Scan(input, "test.emod")
			require.Empty(t, lexDiags)

			model, diags := parser.New(tokens, "test.emod").Parse()

			require.Empty(t, diags)
			slice := model.Contexts[0].Aggregates[0].Slices[0]
			require.NotNil(t, slice.Trigger)
			require.Equal(t, "Reservation Form", slice.Trigger.Name)
			require.Equal(t, "Guest", slice.Trigger.Actor)
			require.Equal(t, "AvailableRoomsView", slice.Trigger.Reads)
			require.Len(t, slice.Automations, 1)
			require.Equal(t, "RoomReserved", slice.Automations[0].OnEvent)
			require.Equal(t, "", slice.Automations[0].Reads)
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
        on ReservationMade
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
        on EventA
        command CmdA
      }
      automation ReactorB {
        on EventB
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
			require.Equal(t, "EventA", slice.Automations[0].OnEvent)
			require.Equal(t, "CmdA", slice.Automations[0].Command)
			require.Equal(t, "", slice.Automations[0].TargetContext)
			require.Equal(t, "ReactorB", slice.Automations[1].Name)
			require.Equal(t, "EventB", slice.Automations[1].OnEvent)
			require.Equal(t, "CmdB", slice.Automations[1].Command)
			require.Equal(t, "OtherCtx", slice.Automations[1].TargetContext)
		})

		t.Run("automation missing opening brace produces diagnostic", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      automation Reactor on SomeEvent
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
			input := modelWithAutomation("Reactor", `        unknown_thing foo`)
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			_, errs := p.Parse()

			require.NotEmpty(t, errs)
			found := false
			for _, e := range errs {
				if strings.Contains(e.Message, `"unknown_thing"`) {
					found = true
					break
				}
			}
			require.True(t, found, "expected a diagnostic naming the unrecognised entry, got: %v", errs)
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

		t.Run("a field name alone on its line is reported once and the block still closes", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command TestCommand {
        fields {
          guestId
          roomType string required
        }
      }

      event ReservationMade {
        fields {
          reservationId string required
        }
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			model, diags := parser.New(tokens, "test.emod").Parse()

			incompleteLine, _ := positionOf(t, input, "guestId", "guestId")
			require.Len(t, diags, 1)
			require.Equal(t, "test.emod", diags[0].Filename)
			require.Equal(t, incompleteLine, diags[0].Line)
			require.Contains(t, diags[0].Message, "field type")

			slice := model.Contexts[0].Aggregates[0].Slices[0]
			test.RequireEqual(t, []*ast.Field{
				{Name: "guestId"},
				{Name: "roomType", Type: "string", Modifier: "required"},
			}, slice.Commands[0].Fields, ignoreASTPositions)
			require.Len(t, slice.Events, 1)
			require.Equal(t, "ReservationMade", slice.Events[0].Name)
		})

		t.Run("a malformed spec entry is reported once and the entry below it still parses", func(t *testing.T) {
			tests := []struct {
				name      string
				entry     string
				keyword   string
				nextEntry string
				wantWhen  string
				wantThen  []string
			}{
				{
					name:      "a when entry naming no command",
					entry:     "when",
					keyword:   "when",
					nextEntry: "then [CopyBorrowed]",
					wantThen:  []string{"CopyBorrowed"},
				},
				{
					name:      "a given entry whose history is not a list",
					entry:     "given CopyReturned",
					keyword:   "given",
					nextEntry: "then [CopyBorrowed]",
					wantThen:  []string{"CopyBorrowed"},
				},
				{
					name:      "a rejection naming no invariant",
					entry:     "then rejected",
					keyword:   "then",
					nextEntry: "when BorrowCopy",
					wantWhen:  "BorrowCopy",
				},
			}

			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					input := fmt.Sprintf(`model "Test"
context "Lending" {
  aggregate "Loan" {
    slice "Borrow a Copy" {
      spec "borrows a free copy" {
        %s
        %s
      }
    }
  }
}`, tc.entry, tc.nextEntry)
					tokens, lexDiags := lexer.Scan(input, "test.emod")
					require.Empty(t, lexDiags)

					model, diags := parser.New(tokens, "test.emod").Parse()

					require.Len(t, diags, 1)
					require.Contains(t, diags[0].Message, "spec")
					line, column := positionOf(t, input, tc.entry, tc.keyword)
					require.Equal(t, line, diags[0].Line)
					require.Equal(t, column, diags[0].Column)

					specs := model.Contexts[0].Aggregates[0].Slices[0].Specs
					require.Len(t, specs, 1)
					require.Equal(t, tc.wantWhen, specElementName(specs[0].When))
					require.Empty(t, specs[0].Given)
					require.Equal(t, tc.wantThen, thenEventNames(t, specs[0].Then))
				})
			}
		})

		t.Run("a then stating neither outcome names both of them and the spec block still closes", func(t *testing.T) {
			input := `model "Test"
context "Lending" {
  aggregate "Loan" {
    slice "Borrow a Copy" {
      command BorrowCopy {
        fields {
          copyId string required
        }
      }
      spec "borrows a free copy" {
        given []
        when BorrowCopy
        then CopyBorrowed
      }
    }
  }
}`
			tokens, lexDiags := lexer.Scan(input, "test.emod")
			require.Empty(t, lexDiags)

			model, diags := parser.New(tokens, "test.emod").Parse()

			require.Len(t, diags, 1)
			line, column := positionOf(t, input, "then CopyBorrowed", "then")
			require.Equal(t, line, diags[0].Line)
			require.Equal(t, column, diags[0].Column)
			require.Contains(t, diags[0].Message, "spec")
			require.Contains(t, diags[0].Message, "event list")
			require.Contains(t, diags[0].Message, "rejected")

			slice := model.Contexts[0].Aggregates[0].Slices[0]
			require.Len(t, slice.Commands, 1)
			require.Equal(t, "BorrowCopy", slice.Commands[0].Name)
			require.Len(t, slice.Specs, 1)
			require.Nil(t, slice.Specs[0].Then)
			require.Equal(t, "BorrowCopy", specElementName(slice.Specs[0].When))
			require.NotZero(t, slice.Specs[0].ClosePos.Line)
			require.NotZero(t, slice.ClosePos.Line)
			require.NotZero(t, model.Contexts[0].ClosePos.Line)
		})

		t.Run("an unclosed given list is reported once and the enclosing blocks still close", func(t *testing.T) {
			tests := []struct {
				name     string
				entries  string
				wantWhen string
				wantThen []string
			}{
				{
					name:    "with the spec's closing brace on the next line",
					entries: `given [CopyReturned`,
				},
				{
					name: "with further entries below the truncated list",
					entries: `given [CopyReturned
        when BorrowCopy
        then [CopyBorrowed]`,
					wantWhen: "BorrowCopy",
					wantThen: []string{"CopyBorrowed"},
				},
			}

			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					input := fmt.Sprintf(`model "Test"
context "Lending" {
  aggregate "Loan" {
    slice "Borrow a Copy" {
      command BorrowCopy {
        fields {
          copyId string required
        }
      }
      spec "borrows a returned copy" {
        %s
      }
    }
  }
}`, tc.entries)
					tokens, lexDiags := lexer.Scan(input, "test.emod")
					require.Empty(t, lexDiags)

					model, diags := parser.New(tokens, "test.emod").Parse()

					require.Len(t, diags, 1)
					require.Contains(t, diags[0].Message, "given")
					require.Contains(t, diags[0].Message, "]")
					slice := model.Contexts[0].Aggregates[0].Slices[0]
					require.Len(t, slice.Commands, 1)
					require.Equal(t, "BorrowCopy", slice.Commands[0].Name)
					require.Len(t, slice.Specs, 1)
					spec := slice.Specs[0]
					require.Equal(t, "borrows a returned copy", spec.Name)
					require.Equal(t, tc.wantWhen, specElementName(spec.When))
					require.Equal(t, tc.wantThen, thenEventNames(t, spec.Then))
					require.NotZero(t, spec.ClosePos.Line)
					require.NotZero(t, slice.ClosePos.Line)
					require.NotZero(t, model.Contexts[0].ClosePos.Line)
				})
			}
		})

		t.Run("an unrecognised entry inside a spec names the entries a spec accepts", func(t *testing.T) {
			input := `model "Test"
context "Lending" {
  aggregate "Loan" {
    slice "Borrow a Copy" {
      spec "borrows a free copy" {
        gievn
      }
    }
  }
}`
			tokens, lexDiags := lexer.Scan(input, "test.emod")
			require.Empty(t, lexDiags)

			_, diags := parser.New(tokens, "test.emod").Parse()

			require.Len(t, diags, 1)
			require.Contains(t, diags[0].Message, "spec")
			for _, entry := range []string{"given", "when", "then"} {
				require.Contains(t, diags[0].Message, entry)
			}
		})

		t.Run("an automation entry naming nothing is reported once and the entry below it still parses", func(t *testing.T) {
			tests := []struct {
				name       string
				keyword    string
				entry      string
				reportedAt string
			}{
				{name: "reads with nothing after it", keyword: "reads", entry: "        reads", reportedAt: "reads"},
				{name: "on with nothing after it", keyword: "on", entry: "        on", reportedAt: "on"},
				{name: "on naming a quoted string", keyword: "on", entry: `        on "RoomReserved"`, reportedAt: "on"},
				{name: "every naming a bare identifier", keyword: "every", entry: "        every nightly", reportedAt: "nightly"},
			}

			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					input := modelWithAutomation("ConfirmationEmailReactor", tc.entry+`
        on RoomReserved
        command SendConfirmationEmail`)
					tokens, lexDiags := lexer.Scan(input, "test.emod")
					require.Empty(t, lexDiags)

					model, diags := parser.New(tokens, "test.emod").Parse()

					require.Len(t, diags, 1)
					require.Regexp(t, `\b`+tc.keyword+`\b`, diags[0].Message)
					require.Contains(t, diags[0].Message, "automation")
					line, column := positionOf(t, input, tc.entry, tc.reportedAt)
					require.Equal(t, line, diags[0].Line)
					require.Equal(t, column, diags[0].Column)

					automation := model.Contexts[0].Aggregates[0].Slices[0].Automations[0]
					require.Equal(t, "RoomReserved", automation.OnEvent)
					require.Equal(t, "", automation.Reads)
					require.Equal(t, "", automation.Schedule)
					require.Equal(t, "SendConfirmationEmail", automation.Command)
				})
			}
		})

		t.Run("an automation entry naming nothing as the last entry still closes the automation and its slice", func(t *testing.T) {
			for _, keyword := range []string{"reads", "on", "every"} {
				t.Run(keyword+" with nothing after it", func(t *testing.T) {
					input := fmt.Sprintf(`model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      automation ConfirmationEmailReactor {
        on RoomReserved
        command SendConfirmationEmail
        %s
      }
      automation ReminderReactor {
        on RoomReserved
        command SendReminder
      }
    }
  }
}`, keyword)
					tokens, lexDiags := lexer.Scan(input, "test.emod")
					require.Empty(t, lexDiags)

					model, diags := parser.New(tokens, "test.emod").Parse()

					require.Len(t, diags, 1)
					slice := model.Contexts[0].Aggregates[0].Slices[0]
					require.Len(t, slice.Automations, 2)
					require.NotZero(t, slice.Automations[0].ClosePos.Line)
					require.Equal(t, "ReminderReactor", slice.Automations[1].Name)
					require.NotZero(t, slice.ClosePos.Line)
					require.NotZero(t, model.Contexts[0].ClosePos.Line)
				})
			}
		})

		t.Run("a trigger entry inside an automation is rejected once and the entries below it still parse", func(t *testing.T) {
			input := modelWithAutomation("ConfirmationEmailReactor", `        on RoomReserved
        trigger RoomReleased
        reads PendingConfirmationsView
        command SendConfirmationEmail
        target context Notifications`)
			tokens, lexDiags := lexer.Scan(input, "test.emod")
			require.Empty(t, lexDiags)

			model, diags := parser.New(tokens, "test.emod").Parse()

			require.Len(t, diags, 1)
			line, column := positionOf(t, input, "trigger RoomReleased", "trigger")
			require.Equal(t,
				fmt.Sprintf("test.emod:%d: trigger is not an automation entry: name the activation event with on", line),
				diags[0].String())
			require.Equal(t, column, diags[0].Column)

			slice := model.Contexts[0].Aggregates[0].Slices[0]
			automation := slice.Automations[0]
			require.Equal(t, "RoomReserved", automation.OnEvent)
			require.Equal(t, "PendingConfirmationsView", automation.Reads)
			require.Equal(t, "SendConfirmationEmail", automation.Command)
			require.Equal(t, "Notifications", automation.TargetContext)
			require.NotZero(t, automation.ClosePos.Line)
			require.NotZero(t, slice.ClosePos.Line)
			require.NotZero(t, model.Contexts[0].ClosePos.Line)
		})

		t.Run("an automation whose only activation entry is a trigger is left without one and its sibling still parses", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      automation ConfirmationEmailReactor {
        command SendConfirmationEmail
        trigger RoomReserved
      }
      automation ReminderReactor {
        on RoomReserved
        command SendReminder
      }
    }
  }
}`
			tokens, lexDiags := lexer.Scan(input, "test.emod")
			require.Empty(t, lexDiags)

			model, diags := parser.New(tokens, "test.emod").Parse()

			require.Len(t, diags, 2)
			rejectedLine, _ := positionOf(t, input, "trigger RoomReserved", "trigger")
			require.Equal(t,
				fmt.Sprintf("test.emod:%d: trigger is not an automation entry: name the activation event with on", rejectedLine),
				diags[0].String())
			declaredLine, _ := positionOf(t, input, "automation ConfirmationEmailReactor", "ConfirmationEmailReactor")
			require.Equal(t,
				fmt.Sprintf("test.emod:%d: automation block requires either an on event or an every schedule", declaredLine),
				diags[1].String())

			slice := model.Contexts[0].Aggregates[0].Slices[0]
			require.Len(t, slice.Automations, 2)
			require.Equal(t, "", slice.Automations[0].OnEvent)
			require.Equal(t, "SendConfirmationEmail", slice.Automations[0].Command)
			require.NotZero(t, slice.Automations[0].ClosePos.Line)
			require.Equal(t, "ReminderReactor", slice.Automations[1].Name)
			require.Equal(t, "RoomReserved", slice.Automations[1].OnEvent)
			require.NotZero(t, slice.ClosePos.Line)
			require.NotZero(t, model.Contexts[0].ClosePos.Line)
		})

		t.Run("an unrecognised entry inside an automation names the entries an automation accepts", func(t *testing.T) {
			input := modelWithAutomation("ConfirmationEmailReactor", `        on RoomReserved
        command SendConfirmationEmail
        raeds`)
			tokens, lexDiags := lexer.Scan(input, "test.emod")
			require.Empty(t, lexDiags)

			_, diags := parser.New(tokens, "test.emod").Parse()

			require.Len(t, diags, 1)
			require.Contains(t, diags[0].Message, "automation")
			for _, entry := range []string{"description", "on", "every", "reads", "command", "target"} {
				require.Regexp(t, `\b`+entry+`\b`, diags[0].Message)
			}
			require.NotRegexp(t, `\btrigger\b`, diags[0].Message)
		})

		t.Run("an automation not declaring exactly one activation form is reported once at its name naming both", func(t *testing.T) {
			tests := []struct {
				name         string
				entries      string
				wantOnEvent  string
				wantSchedule string
			}{
				{
					name:    "declaring neither an activation event nor a schedule",
					entries: `        command SendConfirmationEmail`,
				},
				{
					name: "declaring both an activation event and a schedule",
					entries: `        on RoomReserved
        every "5m"
        command SendConfirmationEmail`,
					wantOnEvent:  "RoomReserved",
					wantSchedule: "5m",
				},
			}

			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					input := modelWithAutomation("ConfirmationEmailReactor", tc.entries)
					tokens, lexDiags := lexer.Scan(input, "test.emod")
					require.Empty(t, lexDiags)

					model, diags := parser.New(tokens, "test.emod").Parse()

					require.Len(t, diags, 1)
					require.Equal(t, "test.emod", diags[0].Filename)
					line, column := positionOf(t, input, "automation ConfirmationEmailReactor", "ConfirmationEmailReactor")
					require.Equal(t, line, diags[0].Line)
					require.Equal(t, column, diags[0].Column)
					require.Regexp(t, `\bon\b`, diags[0].Message)
					require.Regexp(t, `\bevery\b`, diags[0].Message)

					automation := model.Contexts[0].Aggregates[0].Slices[0].Automations[0]
					require.Equal(t, tc.wantOnEvent, automation.OnEvent)
					require.Equal(t, tc.wantSchedule, automation.Schedule)
					require.Equal(t, "SendConfirmationEmail", automation.Command)
				})
			}
		})

		t.Run("automation missing command produces error", func(t *testing.T) {
			input := modelWithAutomation("Reactor", `        on SomeEvent`)
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

		t.Run("automation missing both activation event and command produces both errors", func(t *testing.T) {
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

			foundActivation := false
			foundCommand := false
			for _, e := range errs {
				if e.Message == "automation block requires either an on event or an every schedule" {
					foundActivation = true
				}
				if e.Message == "automation block requires a command" {
					foundCommand = true
				}
			}
			require.True(t, foundActivation, "expected diagnostic 'automation block requires either an on event or an every schedule', got: %v", errs)
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
				if e.Message == "automation block requires either an on event or an every schedule" {
					found = true
					require.Equal(t, 5, e.Line, "error should reference the automation declaration line (5), not the closing brace line")
					break
				}
			}
			require.True(t, found, "expected diagnostic 'automation block requires either an on event or an every schedule', got: %v", errs)
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

		t.Run("retired trigger kind reports replacement message and recovers", func(t *testing.T) {
			t.Run("UI kind tells the author to drop the word", func(t *testing.T) {
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
				model, diags := parser.New(tokens, "test.emod").Parse()

				require.Len(t, diags, 1)
				line, column := positionOf(t, input, "trigger UI", "UI")
				require.Equal(t, line, diags[0].Line)
				require.Equal(t, column, diags[0].Column)
				require.Contains(t, diags[0].Message, "UI")
				require.Contains(t, diags[0].Message, "drop")

				trigger := model.Contexts[0].Aggregates[0].Slices[0].Trigger
				require.NotNil(t, trigger)
				require.Equal(t, "Reservation Form", trigger.Name)
				require.Equal(t, "Guest", trigger.Actor)
				require.Equal(t, "AvailableRoomsView", trigger.Reads)
				require.NotZero(t, trigger.ClosePos.Line)
				require.NotZero(t, model.Contexts[0].Aggregates[0].Slices[0].ClosePos.Line)
				require.NotZero(t, model.Contexts[0].ClosePos.Line)
			})

			t.Run("Schedule kind tells the author to use an automation with every", func(t *testing.T) {
				input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      trigger Schedule "Nightly Sweep" {
        reads PendingExpiries
      }
    }
  }
}`
				tokens, _ := lexer.Scan(input, "test.emod")
				model, diags := parser.New(tokens, "test.emod").Parse()

				require.Len(t, diags, 1)
				require.Contains(t, diags[0].Message, "Schedule")
				require.Contains(t, diags[0].Message, "automation")
				require.Regexp(t, `\bevery\b`, diags[0].Message)

				trigger := model.Contexts[0].Aggregates[0].Slices[0].Trigger
				require.NotNil(t, trigger)
				require.Equal(t, "Nightly Sweep", trigger.Name)
				require.Equal(t, "PendingExpiries", trigger.Reads)
				require.NotZero(t, trigger.ClosePos.Line)
				require.NotZero(t, model.Contexts[0].Aggregates[0].Slices[0].ClosePos.Line)
				require.NotZero(t, model.Contexts[0].ClosePos.Line)
			})

			t.Run("Processor kind tells the author to use an automation with every", func(t *testing.T) {
				input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      trigger Processor "Hold Sweeper" {
        reads PendingExpiries
      }
    }
  }
}`
				tokens, _ := lexer.Scan(input, "test.emod")
				model, diags := parser.New(tokens, "test.emod").Parse()

				require.Len(t, diags, 1)
				require.Contains(t, diags[0].Message, "Processor")
				require.Contains(t, diags[0].Message, "automation")
				require.Regexp(t, `\bevery\b`, diags[0].Message)

				trigger := model.Contexts[0].Aggregates[0].Slices[0].Trigger
				require.NotNil(t, trigger)
				require.Equal(t, "Hold Sweeper", trigger.Name)
				require.Equal(t, "PendingExpiries", trigger.Reads)
				require.NotZero(t, trigger.ClosePos.Line)
				require.NotZero(t, model.Contexts[0].Aggregates[0].Slices[0].ClosePos.Line)
				require.NotZero(t, model.Contexts[0].ClosePos.Line)
			})

			t.Run("arbitrary word tells the author to drop the word", func(t *testing.T) {
				input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      trigger OnOrder "ERP" {
      }
    }
  }
}`
				tokens, _ := lexer.Scan(input, "test.emod")
				model, diags := parser.New(tokens, "test.emod").Parse()

				require.Len(t, diags, 1)
				require.Contains(t, diags[0].Message, "OnOrder")
				require.Contains(t, diags[0].Message, "drop")

				trigger := model.Contexts[0].Aggregates[0].Slices[0].Trigger
				require.NotNil(t, trigger)
				require.Equal(t, "ERP", trigger.Name)
				require.NotZero(t, trigger.ClosePos.Line)
				require.NotZero(t, model.Contexts[0].Aggregates[0].Slices[0].ClosePos.Line)
				require.NotZero(t, model.Contexts[0].ClosePos.Line)
			})

			t.Run("kind with no quoted name reports both diagnostics and the slice keeps parsing", func(t *testing.T) {
				input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      trigger UI {
        actor Guest
      }
      command DoThing {
        fields {
        }
      }
    }
  }
}`
				tokens, _ := lexer.Scan(input, "test.emod")
				model, diags := parser.New(tokens, "test.emod").Parse()

				require.Len(t, diags, 2)
				line, column := positionOf(t, input, "trigger UI", "UI")
				require.Equal(t, line, diags[0].Line)
				require.Equal(t, column, diags[0].Column)
				require.Contains(t, diags[0].Message, "UI")

				require.Contains(t, diags[1].Message, "quoted name after trigger")

				slice := model.Contexts[0].Aggregates[0].Slices[0]
				require.Len(t, slice.Commands, 1)
				require.Equal(t, "DoThing", slice.Commands[0].Name)
				require.NotZero(t, slice.ClosePos.Line)
				require.NotZero(t, model.Contexts[0].ClosePos.Line)
			})

			t.Run("kindless trigger parses cleanly next to a rejected trigger", func(t *testing.T) {
				input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Bad" {
      trigger UI "Old Form" {
      }
    }
    slice "Good" {
      trigger "New Form" {
        actor Guest
      }
    }
  }
}`
				tokens, _ := lexer.Scan(input, "test.emod")
				model, diags := parser.New(tokens, "test.emod").Parse()

				require.Len(t, diags, 1)
				require.Contains(t, diags[0].Message, "UI")

				trigger := model.Contexts[0].Aggregates[0].Slices[1].Trigger
				require.NotNil(t, trigger)
				require.Equal(t, "New Form", trigger.Name)
				require.Equal(t, "Guest", trigger.Actor)
			})
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

	t.Run("descriptions", func(t *testing.T) {
		t.Run("a block construct carries the description written inside it", func(t *testing.T) {
			tests := []struct {
				construct string
				input     string
				want      string
				described func(*ast.Model) (string, ast.Position)
			}{
				{
					construct: "model",
					input: `model "Hotel Reservation" {
  description "How the hotel takes and keeps bookings"
}`,
					want: "How the hotel takes and keeps bookings",
					described: func(m *ast.Model) (string, ast.Position) {
						return m.Description, m.DescriptionPos
					},
				},
				{
					construct: "actor",
					input: `model "Test"
actor "Guest" {
  description "Someone who books a room and stays in it"
}`,
					want: "Someone who books a room and stays in it",
					described: func(m *ast.Model) (string, ast.Position) {
						return m.Actors[0].Description, m.Actors[0].DescriptionPos
					},
				},
				{
					construct: "context",
					input: `model "Test"
context "Reservations" {
  description "Everything the hotel knows about a stay"
  aggregate "Reservation" {
    slice "Make Reservation" {
    }
  }
}`,
					want: "Everything the hotel knows about a stay",
					described: func(m *ast.Model) (string, ast.Position) {
						return m.Contexts[0].Description, m.Contexts[0].DescriptionPos
					},
				},
				{
					construct: "aggregate",
					input: `model "Test"
context "Reservations" {
  aggregate "Reservation" {
    slice "Make Reservation" {
    }
    description "One guest holding one room"
  }
}`,
					want: "One guest holding one room",
					described: func(m *ast.Model) (string, ast.Position) {
						agg := m.Contexts[0].Aggregates[0]
						return agg.Description, agg.DescriptionPos
					},
				},
				{
					construct: "slice",
					input: `model "Test"
context "Reservations" {
  aggregate "Reservation" {
    slice "Make Reservation" {
      description "A guest books a room from the public site"
      command MakeReservation {
      }
    }
  }
}`,
					want: "A guest books a room from the public site",
					described: func(m *ast.Model) (string, ast.Position) {
						slice := m.Contexts[0].Aggregates[0].Slices[0]
						return slice.Description, slice.DescriptionPos
					},
				},
				{
					construct: "trigger",
					input: `model "Test"
context "Reservations" {
  aggregate "Reservation" {
    slice "Make Reservation" {
      trigger "Reservation Form" {
        actor Guest
        description "The booking form on the public site"
      }
    }
  }
}`,
					want: "The booking form on the public site",
					described: func(m *ast.Model) (string, ast.Position) {
						trigger := m.Contexts[0].Aggregates[0].Slices[0].Trigger
						return trigger.Description, trigger.DescriptionPos
					},
				},
				{
					construct: "command",
					input: `model "Test"
context "Reservations" {
  aggregate "Reservation" {
    slice "Make Reservation" {
      command MakeReservation {
        description "Ask the hotel to hold a room"
        fields {
          guestId string required
        }
      }
    }
  }
}`,
					want: "Ask the hotel to hold a room",
					described: func(m *ast.Model) (string, ast.Position) {
						cmd := m.Contexts[0].Aggregates[0].Slices[0].Commands[0]
						return cmd.Description, cmd.DescriptionPos
					},
				},
				{
					construct: "event",
					input: `model "Test"
context "Reservations" {
  aggregate "Reservation" {
    slice "Make Reservation" {
      event ReservationMade {
        fields {
          reservationId string required
        }
        description "A room is held for a guest"
      }
    }
  }
}`,
					want: "A room is held for a guest",
					described: func(m *ast.Model) (string, ast.Position) {
						evt := m.Contexts[0].Aggregates[0].Slices[0].Events[0]
						return evt.Description, evt.DescriptionPos
					},
				},
				{
					construct: "view",
					input: `model "Test"
context "Reservations" {
  aggregate "Reservation" {
    slice "View Reservations" {
      view ReservationsView {
        description "Every reservation with the stage it reached"
        subscribes [ReservationMade]
      }
    }
  }
}`,
					want: "Every reservation with the stage it reached",
					described: func(m *ast.Model) (string, ast.Position) {
						view := m.Contexts[0].Aggregates[0].Slices[0].Views[0]
						return view.Description, view.DescriptionPos
					},
				},
				{
					construct: "automation",
					input: `model "Test"
context "Reservations" {
  aggregate "Reservation" {
    slice "Auto Confirm" {
      automation AutoConfirm {
        on ReservationMade
        command ConfirmReservation
        description "Confirms a reservation the moment it is made"
      }
    }
  }
}`,
					want: "Confirms a reservation the moment it is made",
					described: func(m *ast.Model) (string, ast.Position) {
						automation := m.Contexts[0].Aggregates[0].Slices[0].Automations[0]
						return automation.Description, automation.DescriptionPos
					},
				},
				{
					construct: "translation",
					input: `model "Test"
context "Reservations" {
  aggregate "Reservation" {
    slice "Import Booking" {
      translation BookingImport {
        description "Restates a partner webhook in our own language"
        external_system "Booking.com API"
        reads BookingWebhookView
        command ImportBooking
        event BookingImported {
        }
      }
    }
  }
}`,
					want: "Restates a partner webhook in our own language",
					described: func(m *ast.Model) (string, ast.Position) {
						translation := m.Contexts[0].Aggregates[0].Slices[0].Translations[0]
						return translation.Description, translation.DescriptionPos
					},
				},
				{
					construct: "event nested in a translation",
					input: `model "Test"
context "Reservations" {
  aggregate "Reservation" {
    slice "Import Booking" {
      translation BookingImport {
        external_system "Booking.com API"
        reads BookingWebhookView
        command ImportBooking
        event BookingImported {
          fields {
            bookingId string required
          }
          description "A partner site reported a booking"
        }
      }
    }
  }
}`,
					want: "A partner site reported a booking",
					described: func(m *ast.Model) (string, ast.Position) {
						evt := m.Contexts[0].Aggregates[0].Slices[0].Translations[0].Event
						return evt.Description, evt.DescriptionPos
					},
				},
			}

			for _, tc := range tests {
				t.Run(tc.construct, func(t *testing.T) {
					entry := fmt.Sprintf("description %q", tc.want)
					tokens, lexDiags := lexer.Scan(tc.input, "test.emod")
					require.Empty(t, lexDiags)

					model, diags := parser.New(tokens, "test.emod").Parse()

					require.Empty(t, diags)
					description, pos := tc.described(model)
					require.Equal(t, tc.want, description)
					line, column := positionOf(t, tc.input, entry, strconv.Quote(tc.want))
					require.Equal(t, ast.Position{Filename: "test.emod", Line: line, Column: column}, pos)
				})
			}
		})

		t.Run("a description that is not a quoted string is reported once and the block still parses", func(t *testing.T) {
			tests := []struct {
				construct string
				offending string
				input     string
				remaining func(*testing.T, *ast.Model)
			}{
				{
					construct: "context",
					offending: "Reservations",
					input: `model "Test"
context "Reservations" {
  description Reservations
  aggregate "Reservation" {
    slice "Make Reservation" {
    }
  }
}`,
					remaining: func(t *testing.T, m *ast.Model) {
						require.Len(t, m.Contexts[0].Aggregates, 1)
						require.NotZero(t, m.Contexts[0].ClosePos.Line)
					},
				},
				{
					construct: "aggregate",
					offending: "42",
					input: `model "Test"
context "Reservations" {
  aggregate "Reservation" {
    description 42
    slice "Make Reservation" {
    }
  }
}`,
					remaining: func(t *testing.T, m *ast.Model) {
						agg := m.Contexts[0].Aggregates[0]
						require.Len(t, agg.Slices, 1)
						require.NotZero(t, agg.ClosePos.Line)
					},
				},
				{
					construct: "slice",
					offending: "command",
					input: `model "Test"
context "Reservations" {
  aggregate "Reservation" {
    slice "Make Reservation" {
      description command
      command MakeReservation {
      }
    }
  }
}`,
					remaining: func(t *testing.T, m *ast.Model) {
						slice := m.Contexts[0].Aggregates[0].Slices[0]
						require.Len(t, slice.Commands, 1)
						require.NotZero(t, slice.ClosePos.Line)
					},
				},
				{
					construct: "trigger",
					offending: "Form",
					input: `model "Test"
context "Reservations" {
  aggregate "Reservation" {
    slice "Make Reservation" {
      trigger "Reservation Form" {
        description Form
        actor Guest
        reads AvailableRoomsView
      }
    }
  }
}`,
					remaining: func(t *testing.T, m *ast.Model) {
						trigger := m.Contexts[0].Aggregates[0].Slices[0].Trigger
						require.Equal(t, "Guest", trigger.Actor)
						require.Equal(t, "AvailableRoomsView", trigger.Reads)
						require.NotZero(t, trigger.ClosePos.Line)
					},
				},
				{
					construct: "command",
					offending: "7",
					input: `model "Test"
context "Reservations" {
  aggregate "Reservation" {
    slice "Make Reservation" {
      command MakeReservation {
        description 7
        fields {
          guestId string required
        }
      }
    }
  }
}`,
					remaining: func(t *testing.T, m *ast.Model) {
						cmd := m.Contexts[0].Aggregates[0].Slices[0].Commands[0]
						test.RequireEqual(t, []*ast.Field{
							{Name: "guestId", Type: "string", Modifier: "required"},
						}, cmd.Fields, ignoreASTPositions)
						require.NotZero(t, cmd.ClosePos.Line)
					},
				},
				{
					construct: "event",
					offending: "source",
					input: `model "Test"
context "Reservations" {
  aggregate "Reservation" {
    slice "Make Reservation" {
      event ReservationMade {
        description source
        fields {
          reservationId string required
        }
      }
    }
  }
}`,
					remaining: func(t *testing.T, m *ast.Model) {
						evt := m.Contexts[0].Aggregates[0].Slices[0].Events[0]
						test.RequireEqual(t, []*ast.Field{
							{Name: "reservationId", Type: "string", Modifier: "required"},
						}, evt.Fields, ignoreASTPositions)
						require.NotZero(t, evt.ClosePos.Line)
					},
				},
				{
					construct: "view",
					offending: "ReservationsView",
					input: `model "Test"
context "Reservations" {
  aggregate "Reservation" {
    slice "View Reservations" {
      view ReservationsView {
        description ReservationsView
        subscribes [ReservationMade]
      }
    }
  }
}`,
					remaining: func(t *testing.T, m *ast.Model) {
						view := m.Contexts[0].Aggregates[0].Slices[0].Views[0]
						require.Equal(t, []string{"ReservationMade"}, view.Subscribes)
						require.NotZero(t, view.ClosePos.Line)
					},
				},
				{
					construct: "automation",
					offending: "0",
					input: `model "Test"
context "Reservations" {
  aggregate "Reservation" {
    slice "Auto Confirm" {
      automation AutoConfirm {
        description 0
        on ReservationMade
        command ConfirmReservation
      }
    }
  }
}`,
					remaining: func(t *testing.T, m *ast.Model) {
						automation := m.Contexts[0].Aggregates[0].Slices[0].Automations[0]
						require.Equal(t, "ReservationMade", automation.OnEvent)
						require.Equal(t, "ConfirmReservation", automation.Command)
						require.NotZero(t, automation.ClosePos.Line)
					},
				},
				{
					construct: "translation",
					offending: "external",
					input: `model "Test"
context "Reservations" {
  aggregate "Reservation" {
    slice "Import Booking" {
      translation BookingImport {
        description external
        external_system "Booking.com API"
        reads BookingWebhookView
        command ImportBooking
      }
    }
  }
}`,
					remaining: func(t *testing.T, m *ast.Model) {
						translation := m.Contexts[0].Aggregates[0].Slices[0].Translations[0]
						require.Equal(t, "Booking.com API", translation.ExternalSystem)
						require.Equal(t, "BookingWebhookView", translation.Reads)
						require.Equal(t, "ImportBooking", translation.Command)
						require.NotZero(t, translation.ClosePos.Line)
					},
				},
			}

			for _, tc := range tests {
				t.Run(tc.construct, func(t *testing.T) {
					entry := "description " + tc.offending
					tokens, lexDiags := lexer.Scan(tc.input, "test.emod")
					require.Empty(t, lexDiags)

					model, diags := parser.New(tokens, "test.emod").Parse()

					require.Len(t, diags, 1)
					require.Contains(t, diags[0].Message, tc.construct)
					require.Contains(t, diags[0].Message, strconv.Quote(tc.offending))
					line, column := positionOf(t, tc.input, entry, tc.offending)
					require.Equal(t, line, diags[0].Line)
					require.Equal(t, column, diags[0].Column)
					require.Equal(t, "test.emod", diags[0].Filename)
					tc.remaining(t, model)
				})
			}
		})

		t.Run("an unquoted multi-word description is reported once and the block still parses", func(t *testing.T) {
			tests := []struct {
				construct string
				prose     string
				input     string
				remaining func(*testing.T, *ast.Model)
			}{
				{
					construct: "slice",
					prose:     "A guest books a room",
					input: `model "Test"
context "Reservations" {
  aggregate "Reservation" {
    slice "Make Reservation" {
      description A guest books a room
      command MakeReservation {
      }
    }
  }
}`,
					remaining: func(t *testing.T, m *ast.Model) {
						slice := m.Contexts[0].Aggregates[0].Slices[0]
						require.Len(t, slice.Commands, 1)
						require.Equal(t, "MakeReservation", slice.Commands[0].Name)
						require.NotZero(t, slice.ClosePos.Line)
					},
				},
				{
					construct: "context",
					prose:     "A guest books a room",
					input: `model "Test"
context "Reservations" {
  description A guest books a room
  aggregate "Reservation" {
    slice "Make Reservation" {
    }
  }
}`,
					remaining: func(t *testing.T, m *ast.Model) {
						context := m.Contexts[0]
						require.Len(t, context.Aggregates, 1)
						require.Equal(t, "Reservation", context.Aggregates[0].Name)
						require.NotZero(t, context.ClosePos.Line)
					},
				},
			}

			for _, tc := range tests {
				t.Run(tc.construct, func(t *testing.T) {
					firstWord := strings.Fields(tc.prose)[0]
					tokens, lexDiags := lexer.Scan(tc.input, "test.emod")
					require.Empty(t, lexDiags)

					model, diags := parser.New(tokens, "test.emod").Parse()

					require.Len(t, diags, 1)
					require.Contains(t, diags[0].Message, tc.construct)
					require.Contains(t, diags[0].Message, strconv.Quote(firstWord))
					line, column := positionOf(t, tc.input, "description "+tc.prose, firstWord)
					require.Equal(t, line, diags[0].Line)
					require.Equal(t, column, diags[0].Column)
					tc.remaining(t, model)
				})
			}
		})

		t.Run("an unrecognised entry offers description among the block's valid entries", func(t *testing.T) {
			tests := []struct {
				construct  string
				alsoOffers []string
				input      string
			}{
				{
					construct:  "context",
					alsoOffers: []string{"invariant"},
					input: `model "Test"
context "Reservations" {
  descripton
}`,
				},
				{
					construct:  "aggregate",
					alsoOffers: []string{"invariant"},
					input: `model "Test"
context "Reservations" {
  aggregate "Reservation" {
    descripton
  }
}`,
				},
				{
					construct:  "slice",
					alsoOffers: []string{"spec"},
					input: `model "Test"
context "Reservations" {
  aggregate "Reservation" {
    slice "Make Reservation" {
      descripton
    }
  }
}`,
				},
				{
					construct: "trigger",
					input: `model "Test"
context "Reservations" {
  aggregate "Reservation" {
    slice "Make Reservation" {
      trigger "Reservation Form" {
        descripton
      }
    }
  }
}`,
				},
				{
					construct: "command",
					input: `model "Test"
context "Reservations" {
  aggregate "Reservation" {
    slice "Make Reservation" {
      command MakeReservation {
        descripton
      }
    }
  }
}`,
				},
				{
					construct: "event",
					input: `model "Test"
context "Reservations" {
  aggregate "Reservation" {
    slice "Make Reservation" {
      event ReservationMade {
        descripton
      }
    }
  }
}`,
				},
				{
					construct: "view",
					input: `model "Test"
context "Reservations" {
  aggregate "Reservation" {
    slice "View Reservations" {
      view ReservationsView {
        descripton
      }
    }
  }
}`,
				},
				{
					construct: "automation",
					input: `model "Test"
context "Reservations" {
  aggregate "Reservation" {
    slice "Auto Confirm" {
      automation AutoConfirm {
        descripton
      }
    }
  }
}`,
				},
				{
					construct: "translation",
					input: `model "Test"
context "Reservations" {
  aggregate "Reservation" {
    slice "Import Booking" {
      translation BookingImport {
        descripton
      }
    }
  }
}`,
				},
			}

			for _, tc := range tests {
				t.Run(tc.construct, func(t *testing.T) {
					tokens, lexDiags := lexer.Scan(tc.input, "test.emod")
					require.Empty(t, lexDiags)

					_, diags := parser.New(tokens, "test.emod").Parse()

					require.NotEmpty(t, diags)
					require.Contains(t, diags[0].Message, "description")
					require.Contains(t, diags[0].Message, tc.construct)
					for _, entry := range tc.alsoOffers {
						require.Contains(t, diags[0].Message, entry)
					}
				})
			}
		})

		t.Run("a description with no value at the end of a block leaves the block closed", func(t *testing.T) {
			input := `model "Test"
context "Reservations" {
  aggregate "Reservation" {
    slice "View Reservations" {
      view ReservationsView {
        subscribes [ReservationMade]
        description
      }
      view CancellationsView {
        subscribes [ReservationCancelled]
      }
    }
  }
}`
			tokens, lexDiags := lexer.Scan(input, "test.emod")
			require.Empty(t, lexDiags)

			model, diags := parser.New(tokens, "test.emod").Parse()

			require.Len(t, diags, 1)
			require.Contains(t, diags[0].Message, "view")
			views := model.Contexts[0].Aggregates[0].Slices[0].Views
			require.Len(t, views, 2)
			require.Equal(t, "ReservationsView", views[0].Name)
			require.NotZero(t, views[0].ClosePos.Line)
			require.Equal(t, "CancellationsView", views[1].Name)
			require.Equal(t, []string{"ReservationCancelled"}, views[1].Subscribes)
		})

		t.Run("the shared described model describes every construct that accepts one", func(t *testing.T) {
			tokens, lexDiags := lexer.Scan(test.DescribedHotelReservation, "described.emod")
			require.Empty(t, lexDiags)

			model, diags := parser.New(tokens, "described.emod").Parse()

			require.Empty(t, diags)
			var kinds, undescribed []string
			for _, construct := range describableConstructs(model) {
				kinds = append(kinds, construct.kind)
				if construct.description == "" {
					undescribed = append(undescribed, fmt.Sprintf("%s %q", construct.kind, construct.name))
				}
			}
			require.Empty(t, undescribed)
			require.Subset(t, kinds, []string{
				"model", "actor",
				"context", "aggregate", "slice", "trigger",
				"command", "event", "view", "automation", "translation",
			})
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
			test.RequireEqual(t, []*ast.Comment{{Text: "# Header comment"}}, model.Comments, ignoreASTPositions)
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
			}, model.Comments, ignoreASTPositions)
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
			test.RequireEqual(t, []*ast.Comment{{Text: "# Actor comment"}}, model.Actors[0].Comments, ignoreASTPositions)
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
			test.RequireEqual(t, []*ast.Comment{{Text: "# Context comment"}}, model.Contexts[0].Comments, ignoreASTPositions)
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
			test.RequireEqual(t, []*ast.Comment{{Text: "# Slice comment"}}, slice.Comments, ignoreASTPositions)
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
      trigger "Form" {
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
        on ThingDone
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
      # Spec comment
      spec "does the thing" {
        given []
        when DoThing
        then [ThingDone]
      }
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Empty(t, errs)
			slice := model.Contexts[0].Aggregates[0].Slices[0]

			test.RequireEqual(t, []*ast.Comment{{Text: "# Command comment"}}, slice.Commands[0].Comments, ignoreASTPositions)
			test.RequireEqual(t, []*ast.Comment{{Text: "# Event comment"}}, slice.Events[0].Comments, ignoreASTPositions)
			test.RequireEqual(t, []*ast.Comment{{Text: "# Trigger comment"}}, slice.Trigger.Comments, ignoreASTPositions)
			test.RequireEqual(t, []*ast.Comment{{Text: "# View comment"}}, slice.Views[0].Comments, ignoreASTPositions)
			test.RequireEqual(t, []*ast.Comment{{Text: "# Automation comment"}}, slice.Automations[0].Comments, ignoreASTPositions)
			test.RequireEqual(t, []*ast.Comment{{Text: "# Translation comment"}}, slice.Translations[0].Comments, ignoreASTPositions)
			test.RequireEqual(t, []*ast.Comment{{Text: "# Flow comment"}}, slice.Flows[0].Comments, ignoreASTPositions)
			test.RequireEqual(t, []*ast.Comment{{Text: "# Spec comment"}}, slice.Specs[0].Comments, ignoreASTPositions)
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
			test.RequireEqual(t, []*ast.Comment{{Text: "# Aggregate comment"}}, model.Contexts[0].Aggregates[0].Comments, ignoreASTPositions)
		})

		t.Run("comments before an invariant are attached to the Invariant node", func(t *testing.T) {
			input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    invariant NoDoubleBooking "A room is held by at most one reservation per night"
    # Capacity comment
    invariant WithinCapacity "A reservation never seats more guests than the room holds"
    slice "Slice" {
    }
  }
}`
			tokens, _ := lexer.Scan(input, "test.emod")

			p := parser.New(tokens, "test.emod")
			model, errs := p.Parse()

			require.Empty(t, errs)
			aggregate := model.Contexts[0].Aggregates[0]
			require.Empty(t, aggregate.Comments)
			require.Len(t, aggregate.Invariants, 2)
			require.Empty(t, aggregate.Invariants[0].Comments)
			test.RequireEqual(t, []*ast.Comment{{Text: "# Capacity comment"}}, aggregate.Invariants[1].Comments, ignoreASTPositions)
			require.Empty(t, aggregate.Slices[0].Comments)
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

func modelWithField(name, fieldType, modifier string) string {
	return fmt.Sprintf(`model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command DoThing {
        fields {
          %s %s %s
        }
      }
    }
  }
}`, name, fieldType, modifier)
}

func modelWithAutomation(name, entries string) string {
	return fmt.Sprintf(`model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      automation %s {
%s
      }
    }
  }
}`, name, entries)
}

func modelWithInvariant(name, statement string) string {
	return fmt.Sprintf(`model "Test"
context "Reservations" {
  aggregate "Reservation" {
    invariant %s "%s"
  }
}`, name, statement)
}

func modelWithSpecEntries(entries string) string {
	return fmt.Sprintf(`model "Library Lending"
context "Lending" {
  aggregate "Loan" {
    slice "Borrow a Copy" {
      spec "borrows a copy" {
        %s
      }
    }
  }
}`, entries)
}

func declaredSpec(t *testing.T, filename, source, name string, given []string, when string, then []string) *ast.Spec {
	t.Helper()
	spec := &ast.Spec{
		Name:    name,
		NamePos: astPositionOf(t, filename, source, `spec "`+name+`"`, `"`+name+`"`),
		Given:   declaredSpecElements(t, filename, source, "given", given),
		Then:    &ast.ThenEvents{Events: declaredSpecElements(t, filename, source, "then", then)},
	}
	if when != "" {
		spec.When = &ast.SpecElement{Name: when, NamePos: astPositionOf(t, filename, source, "when "+when, when)}
	}
	return spec
}

func declaredSpecElements(t *testing.T, filename, source, keyword string, names []string) []*ast.SpecElement {
	t.Helper()
	entry := fmt.Sprintf("%s [%s]", keyword, strings.Join(names, ", "))
	var elements []*ast.SpecElement
	for _, name := range names {
		elements = append(elements, &ast.SpecElement{Name: name, NamePos: astPositionOf(t, filename, source, entry, name)})
	}
	return elements
}

func specElementName(element *ast.SpecElement) string {
	if element == nil {
		return ""
	}
	return element.Name
}

func specElementNames(elements []*ast.SpecElement) []string {
	var names []string
	for _, element := range elements {
		names = append(names, element.Name)
	}
	return names
}

func thenEventNames(t *testing.T, outcome ast.ThenClause) []string {
	t.Helper()
	if outcome == nil {
		return nil
	}
	events, ok := outcome.(*ast.ThenEvents)
	require.True(t, ok, "outcome %T is not an event list", outcome)
	return specElementNames(events.Events)
}

func invariantNames(invariants []*ast.Invariant) []string {
	var names []string
	for _, invariant := range invariants {
		names = append(names, invariant.Name)
	}
	return names
}

func rejectedInvariantNames(specs []*ast.Spec) []string {
	var names []string
	for _, spec := range specs {
		if rejection, ok := spec.Then.(*ast.ThenRejected); ok {
			names = append(names, rejection.InvariantName)
		}
	}
	return names
}

func requireRejectionsDeclaredBy(t *testing.T, owner string, invariants []*ast.Invariant, slices []*ast.Slice) int {
	t.Helper()
	declared := invariantNames(invariants)
	rejections := 0
	for _, slice := range slices {
		for _, name := range rejectedInvariantNames(slice.Specs) {
			require.Contains(t, declared, name, "%s declares no invariant %s", owner, name)
			rejections++
		}
	}
	return rejections
}

func outcomeKind(outcome ast.ThenClause) string {
	switch outcome.(type) {
	case *ast.ThenEvents:
		return "events"
	case *ast.ThenRejected:
		return "rejection"
	default:
		return fmt.Sprintf("%T", outcome)
	}
}

func outcomeLine(outcome ast.ThenClause) int {
	switch then := outcome.(type) {
	case *ast.ThenEvents:
		if len(then.Events) == 0 {
			return 0
		}
		return then.Events[0].NamePos.Line
	case *ast.ThenRejected:
		return then.InvariantPos.Line
	default:
		return 0
	}
}

// requireSpecCoverage reads the given history back off the fixture's own source
// as well as off the parsed model, because a spec written with an empty list and
// one written with no given entry at all leave the same parsed spec behind.
func requireSpecCoverage(t *testing.T, source string, specs []*ast.Spec) {
	t.Helper()
	var multiEventGiven, emptyGivenList, omittedGiven, eventListOutcome, rejectionOutcome, thenAboveWhen bool
	for _, spec := range specs {
		multiEventGiven = multiEventGiven || len(spec.Given) > 1
		eventListOutcome = eventListOutcome || outcomeKind(spec.Then) == "events"
		rejectionOutcome = rejectionOutcome || outcomeKind(spec.Then) == "rejection"
		if spec.When != nil && outcomeLine(spec.Then) > 0 {
			thenAboveWhen = thenAboveWhen || outcomeLine(spec.Then) < spec.When.NamePos.Line
		}

		var givenEntries int
		for _, line := range specSourceLines(t, source, spec) {
			entry := strings.TrimSpace(line)
			if !strings.HasPrefix(entry, "given") {
				continue
			}
			givenEntries++
			emptyGivenList = emptyGivenList || entry == "given []"
		}
		omittedGiven = omittedGiven || givenEntries == 0
	}

	require.True(t, multiEventGiven, "no spec gives a history of more than one event")
	require.True(t, emptyGivenList, "no spec writes its empty given history as []")
	require.True(t, omittedGiven, "no spec omits its given history")
	require.True(t, eventListOutcome, "no spec states the events its command appends")
	require.True(t, rejectionOutcome, "no spec states a rejection")
	require.True(t, thenAboveWhen, "no spec writes its then above its when")
}

func specSourceLines(t *testing.T, source string, spec *ast.Spec) []string {
	t.Helper()
	lines := strings.Split(source, "\n")
	require.Less(t, spec.OpenPos.Line, spec.ClosePos.Line, spec.Name)
	require.LessOrEqual(t, spec.ClosePos.Line, len(lines), spec.Name)
	return lines[spec.OpenPos.Line : spec.ClosePos.Line-1]
}

func requireASpecAheadOfALaterEntry(t *testing.T, model *ast.Model) {
	t.Helper()
	for _, slice := range declaredSlices(model) {
		lastEntryLine := 0
		for _, evt := range slice.Events {
			lastEntryLine = max(lastEntryLine, evt.NamePos.Line)
		}
		for _, flow := range slice.Flows {
			lastEntryLine = max(lastEntryLine, flow.CommandPos.Line)
		}
		for _, spec := range slice.Specs {
			if spec.ClosePos.Line < lastEntryLine {
				return
			}
		}
	}
	require.Fail(t, "no spec runs into a later entry", "every spec is written after the last entry of its slice")
}

func declaredSpecs(model *ast.Model) []*ast.Spec {
	var specs []*ast.Spec
	for _, slice := range declaredSlices(model) {
		specs = append(specs, slice.Specs...)
	}
	return specs
}

func declaredSlices(model *ast.Model) []*ast.Slice {
	var slices []*ast.Slice
	for _, context := range model.Contexts {
		slices = append(slices, context.Slices...)
		for _, aggregate := range context.Aggregates {
			slices = append(slices, aggregate.Slices...)
		}
	}
	return slices
}

func declaredInvariant(t *testing.T, filename, source, name, statement string) *ast.Invariant {
	t.Helper()
	entry := "invariant " + name
	return &ast.Invariant{
		Name:         name,
		NamePos:      astPositionOf(t, filename, source, entry, name),
		Statement:    statement,
		StatementPos: astPositionOf(t, filename, source, entry, `"`+statement+`"`),
	}
}

func requireAnInvariantAheadOfALaterSlice(t *testing.T, block string, invariants []*ast.Invariant, slices []*ast.Slice) {
	t.Helper()
	lastSliceLine := 0
	for _, slice := range slices {
		lastSliceLine = max(lastSliceLine, slice.NamePos.Line)
	}

	for _, invariant := range invariants {
		if invariant.NamePos.Line < lastSliceLine {
			return
		}
	}
	require.Fail(t, "no invariant runs into a later entry", "every invariant of %s is written after its last slice", block)
}

func positionOf(t *testing.T, source, entry, token string) (line, column int) {
	t.Helper()
	for index, text := range strings.Split(source, "\n") {
		if strings.Contains(text, entry) {
			return index + 1, strings.Index(text, token) + 1
		}
	}
	require.FailNowf(t, "entry not found in source", "%q", entry)
	return 0, 0
}

func astPositionOf(t *testing.T, filename, source, entry, token string) ast.Position {
	t.Helper()
	line, column := positionOf(t, source, entry, token)
	return ast.Position{Filename: filename, Line: line, Column: column}
}

type describableConstruct struct {
	kind        string
	name        string
	description string
}

func describableConstructs(model *ast.Model) []describableConstruct {
	var found []describableConstruct
	add := func(kind, name, description string) {
		found = append(found, describableConstruct{kind: kind, name: name, description: description})
	}
	addEvent := func(evt *ast.Event) {
		add("event", evt.Name, evt.Description)
	}
	addSlice := func(slice *ast.Slice) {
		add("slice", slice.Name, slice.Description)
		if slice.Trigger != nil {
			add("trigger", slice.Trigger.Name, slice.Trigger.Description)
		}
		for _, cmd := range slice.Commands {
			add("command", cmd.Name, cmd.Description)
		}
		for _, evt := range slice.Events {
			addEvent(evt)
		}
		for _, view := range slice.Views {
			add("view", view.Name, view.Description)
		}
		for _, automation := range slice.Automations {
			add("automation", automation.Name, automation.Description)
		}
		for _, translation := range slice.Translations {
			add("translation", translation.Name, translation.Description)
			if translation.Event != nil {
				addEvent(translation.Event)
			}
		}
	}

	add("model", model.Name, model.Description)
	for _, actor := range model.Actors {
		add("actor", actor.Name, actor.Description)
	}

	for _, context := range model.Contexts {
		add("context", context.Name, context.Description)
		for _, slice := range context.Slices {
			addSlice(slice)
		}
		for _, aggregate := range context.Aggregates {
			add("aggregate", aggregate.Name, aggregate.Description)
			for _, slice := range aggregate.Slices {
				addSlice(slice)
			}
		}
	}

	return found
}

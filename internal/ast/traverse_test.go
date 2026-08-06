//go:build unit

package ast_test

import (
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/parser"
	"github.com/stretchr/testify/require"
)

// The fixtures go through the real lexer and parser so the positions the
// traversal sorts by are the ones production models carry.
func parseModel(t *testing.T, source string) *ast.Model {
	t.Helper()
	tokens, diags := lexer.Scan(source, "traverse_test.emod")
	require.Empty(t, diags)
	model, parserDiags := parser.New(tokens, "traverse_test.emod").Parse()
	require.Empty(t, parserDiags)
	return model
}

func sliceNames(refs []ast.SliceRef) []string {
	names := make([]string, len(refs))
	for i, ref := range refs {
		names[i] = ref.Slice.Name
	}
	return names
}

func TestModel(t *testing.T) {
	t.Run("slice traversal", func(t *testing.T) {
		t.Run("returns aggregate and direct context slices in source order", func(t *testing.T) {
			model := parseModel(t, `model "Mixed"
context "Lending" mode mixed {
  slice "First Direct" {
  }
  aggregate "Loan" {
    slice "From Aggregate" {
    }
  }
  slice "Last Direct" {
  }
}
context "Billing" {
  aggregate "Invoice" {
    slice "Billing Slice" {
    }
  }
}
`)

			refs := model.SliceRefs()

			require.Equal(t, []string{"First Direct", "From Aggregate", "Last Direct", "Billing Slice"}, sliceNames(refs))
		})

		t.Run("pairs each slice with its declaring context and aggregate", func(t *testing.T) {
			model := parseModel(t, `model "Mixed"
context "Lending" mode mixed {
  slice "Direct" {
  }
  aggregate "Loan" {
    slice "Nested" {
    }
  }
}
`)

			refs := model.SliceRefs()

			require.Len(t, refs, 2)
			require.Equal(t, "Lending", refs[0].Context.Name)
			require.Nil(t, refs[0].Aggregate, "a direct DCB-style slice hangs off no aggregate")
			require.Equal(t, "Lending", refs[1].Context.Name)
			require.Equal(t, "Loan", refs[1].Aggregate.Name)
		})

		t.Run("nil model yields no slices", func(t *testing.T) {
			var model *ast.Model

			require.Empty(t, model.SliceRefs())
			require.Empty(t, model.AllSlices())
		})
	})
}

func TestContext(t *testing.T) {
	t.Run("slice traversal", func(t *testing.T) {
		t.Run("interleaves direct and aggregate slices by position", func(t *testing.T) {
			model := parseModel(t, `model "Mixed"
context "Lending" mode mixed {
  aggregate "Loan" {
    slice "Borrow" {
    }
  }
  slice "Between" {
  }
  aggregate "Copy" {
    slice "Acquire" {
    }
  }
}
`)

			slices := model.Contexts[0].AllSlices()

			names := make([]string, len(slices))
			for i, s := range slices {
				names[i] = s.Name
			}
			require.Equal(t, []string{"Borrow", "Between", "Acquire"}, names)
		})

		t.Run("does not alias the context's own slice collection", func(t *testing.T) {
			model := parseModel(t, `model "Mixed"
context "Lending" mode mixed {
  slice "Direct" {
  }
  aggregate "Loan" {
    slice "Nested" {
    }
  }
}
`)
			ctx := model.Contexts[0]

			_ = ctx.AllSlices()

			require.Len(t, ctx.Slices, 1, "traversal must not grow the AST's own collection")
			require.Equal(t, "Direct", ctx.Slices[0].Name)
		})
	})
}

//go:build unit

package query_test

import (
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/query"
	"github.com/stretchr/testify/require"
)

func TestNameSites(t *testing.T) {
	t.Run("trigger", func(t *testing.T) {
		t.Run("each entry of a trigger's reads list is a view name at the position of that entry", func(t *testing.T) {
			model := parse(t, `context "Shop" {
  slice "Show Cart" {
    view "CartView" {
    }
  }

  slice "Apply Discount" {
    trigger "Checkout Page" {
      reads = [CartView, DiscountView]
    }
  }
}
`)

			require.Equal(t, []query.NameSite{
				{Kind: query.KindView, Name: "CartView", Pos: ast.Position{Filename: "model.emod", Line: 9, Column: 16}},
				{Kind: query.KindView, Name: "DiscountView", Pos: ast.Position{Filename: "model.emod", Line: 9, Column: 26}},
			}, query.NameSites(model))
		})
	})
}

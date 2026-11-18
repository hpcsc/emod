//go:build unit

package query_test

import (
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/query"
	"github.com/stretchr/testify/require"
)

func TestNameSites(t *testing.T) {
	t.Run("shows", func(t *testing.T) {
		t.Run("each entry of the reads and the command of a shows block is a name at the position of that entry", func(t *testing.T) {
			model := parse(t, `screen "CheckoutPage" {
}

screen "CartDrawer" {
}

context "Shop" {
  slice "Apply Discount" {
    shows {
      screen = CartDrawer
      reads  = CartView
    }

    shows {
      screen  = CheckoutPage
      reads   = [CartView, DiscountView]
      command = [ApplyDiscount, ClearCart]
    }
  }
}
`)

			require.Equal(t, []query.NameSite{
				{Kind: query.KindView, Name: "CartView", Pos: ast.Position{Filename: "model.emod", Line: 11, Column: 16}},
				{Kind: query.KindView, Name: "CartView", Pos: ast.Position{Filename: "model.emod", Line: 16, Column: 18}},
				{Kind: query.KindView, Name: "DiscountView", Pos: ast.Position{Filename: "model.emod", Line: 16, Column: 28}},
				{Kind: query.KindCommand, Name: "ApplyDiscount", Pos: ast.Position{Filename: "model.emod", Line: 17, Column: 18}},
				{Kind: query.KindCommand, Name: "ClearCart", Pos: ast.Position{Filename: "model.emod", Line: 17, Column: 33}},
			}, query.NameSites(model))
		})
	})

}

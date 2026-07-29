//go:build unit

package glossary_test

import (
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/glossary"
	"github.com/stretchr/testify/require"
)

func TestGlossary(t *testing.T) {
	t.Run("markdown rendering", func(t *testing.T) {
		t.Run("orders contexts as declared and nests each context's aggregates beneath it", func(t *testing.T) {
			model := &ast.Model{
				Name:        "Hotel Operations",
				Description: "How the hotel runs a stay end to end",
				Contexts: []*ast.Context{
					{
						Name:        "Reservations",
						Description: "Holding a room before the guest arrives",
						Aggregates: []*ast.Aggregate{
							{Name: "Booking", Description: "One room held over one date range"},
							{Name: "Allotment", Description: "Rooms set aside for a partner site"},
						},
					},
					{
						Name:        "Billing",
						Description: "Turning a completed stay into money",
						Aggregates: []*ast.Aggregate{
							{Name: "Invoice", Description: "What a guest owes for one stay"},
						},
					},
				},
			}

			rendered, err := glossary.RenderMarkdown(model)
			require.NoError(t, err)

			require.Equal(t, `# Hotel Operations

How the hotel runs a stay end to end

## Reservations

Holding a room before the guest arrives

### Booking

One room held over one date range

### Allotment

Rooms set aside for a partner site

## Billing

Turning a completed stay into money

### Invoice

What a guest owes for one stay
`, string(rendered))
		})
	})
}

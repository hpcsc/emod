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

		t.Run("lists under each context the commands, events and views its own slices declare, in declaration order", func(t *testing.T) {
			model := &ast.Model{
				Name:        "Hotel Operations",
				Description: "How the hotel runs a stay end to end",
				Contexts: []*ast.Context{
					{
						Name:        "Reservations",
						Description: "Holding a room before the guest arrives",
						Aggregates: []*ast.Aggregate{{
							Name:        "Booking",
							Description: "One room held over one date range",
							Slices: []*ast.Slice{
								{
									Name:     "Hold a room",
									Commands: []*ast.Command{{Name: "HoldRoom", Description: "Ask the hotel to hold a room for a date range"}},
									Events:   []*ast.Event{{Name: "RoomHeld", Description: "A room is held for a guest"}},
								},
								{
									Name:     "Cancel a hold",
									Commands: []*ast.Command{{Name: "CancelHold", Description: "Release a room the guest no longer wants"}},
									Events:   []*ast.Event{{Name: "HoldCancelled", Description: "A held room is back on sale"}},
									Views:    []*ast.View{{Name: "HeldRoomsView", Description: "Rooms held but not yet paid for"}},
								},
							},
						}},
					},
					{
						Name:        "Billing",
						Description: "Turning a completed stay into money",
						Aggregates: []*ast.Aggregate{{
							Name:        "Invoice",
							Description: "What a guest owes for one stay",
							Slices: []*ast.Slice{{
								Name:     "Issue an invoice",
								Commands: []*ast.Command{{Name: "IssueInvoice", Description: "Bill the guest for a finished stay"}},
								Events:   []*ast.Event{{Name: "InvoiceIssued", Description: "A guest has been billed"}},
								Views:    []*ast.View{{Name: "UnpaidInvoicesView", Description: "Invoices still owed"}},
							}},
						}},
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

### Commands

#### HoldRoom

Ask the hotel to hold a room for a date range

#### CancelHold

Release a room the guest no longer wants

### Events

#### RoomHeld

A room is held for a guest

#### HoldCancelled

A held room is back on sale

### Views

#### HeldRoomsView

Rooms held but not yet paid for

## Billing

Turning a completed stay into money

### Invoice

What a guest owes for one stay

### Commands

#### IssueInvoice

Bill the guest for a finished stay

### Events

#### InvoiceIssued

A guest has been billed

### Views

#### UnpaidInvoicesView

Invoices still owed
`, string(rendered))

			again, err := glossary.RenderMarkdown(model)
			require.NoError(t, err)
			require.Equal(t, string(rendered), string(again))
		})

		t.Run("names an actor once in each context whose triggers reference it, however many of them do", func(t *testing.T) {
			model := &ast.Model{
				Name:   "Hotel Operations",
				Actors: []*ast.Actor{{Name: "Guest", Description: "A person booking a room, not necessarily the one staying in it"}},
				Contexts: []*ast.Context{
					{
						Name: "Reservations",
						Aggregates: []*ast.Aggregate{{
							Name: "Booking",
							Slices: []*ast.Slice{
								{Name: "Hold a room", Trigger: &ast.Trigger{Kind: "UI", Name: "Booking Form", Actor: "Guest"}},
								{Name: "Cancel a hold", Trigger: &ast.Trigger{Kind: "UI", Name: "Cancellation Form", Actor: "Guest"}},
							},
						}},
					},
					{
						Name: "Billing",
						Aggregates: []*ast.Aggregate{{
							Name:   "Invoice",
							Slices: []*ast.Slice{{Name: "Pay an invoice", Trigger: &ast.Trigger{Kind: "UI", Name: "Payment Form", Actor: "Guest"}}},
						}},
					},
				},
			}

			rendered, err := glossary.RenderMarkdown(model)
			require.NoError(t, err)

			require.Equal(t, `# Hotel Operations

## Reservations

### Booking

### Actors

#### Guest

A person booking a room, not necessarily the one staying in it

## Billing

### Invoice

### Actors

#### Guest

A person booking a room, not necessarily the one staying in it
`, string(rendered))
		})

		t.Run("keeps a declared actor no trigger references, with the description declared for it", func(t *testing.T) {
			model := &ast.Model{
				Name: "Hotel Operations",
				Actors: []*ast.Actor{
					{Name: "Guest", Description: "A person booking a room"},
					{Name: "Auditor", Description: "Reviews the books once a quarter and never touches a screen"},
				},
				Contexts: []*ast.Context{{
					Name: "Reservations",
					Aggregates: []*ast.Aggregate{{
						Name:   "Booking",
						Slices: []*ast.Slice{{Name: "Hold a room", Trigger: &ast.Trigger{Kind: "UI", Name: "Booking Form", Actor: "Guest"}}},
					}},
				}},
			}

			rendered, err := glossary.RenderMarkdown(model)
			require.NoError(t, err)

			require.Equal(t, `# Hotel Operations

## Actors

### Auditor

Reviews the books once a quarter and never touches a screen

## Reservations

### Booking

### Actors

#### Guest

A person booking a room
`, string(rendered))
		})

		t.Run("keeps an actor a trigger names though the model never declares it, with an empty definition", func(t *testing.T) {
			model := &ast.Model{
				Name: "Hotel Operations",
				Contexts: []*ast.Context{{
					Name: "Reservations",
					Aggregates: []*ast.Aggregate{{
						Name:   "Booking",
						Slices: []*ast.Slice{{Name: "Hold a room", Trigger: &ast.Trigger{Kind: "UI", Name: "Booking Form", Actor: "Concierge"}}},
					}},
				}},
			}

			rendered, err := glossary.RenderMarkdown(model)
			require.NoError(t, err)

			require.Equal(t, `# Hotel Operations

## Reservations

### Booking

### Actors

#### Concierge
`, string(rendered))
		})

		t.Run("takes no actor from a trigger that names none", func(t *testing.T) {
			model := &ast.Model{
				Name: "Hotel Operations",
				Contexts: []*ast.Context{{
					Name: "Reservations",
					Aggregates: []*ast.Aggregate{{
						Name: "Booking",
						Slices: []*ast.Slice{{
							Name:    "Browse availability",
							Trigger: &ast.Trigger{Kind: "UI", Name: "Availability Screen", Reads: "AvailableRoomsView"},
							Views:   []*ast.View{{Name: "AvailableRoomsView", Description: "Rooms free over a date range"}},
						}},
					}},
				}},
			}

			rendered, err := glossary.RenderMarkdown(model)
			require.NoError(t, err)

			require.Equal(t, `# Hotel Operations

## Reservations

### Booking

### Views

#### AvailableRoomsView

Rooms free over a date range
`, string(rendered))
		})
	})
}

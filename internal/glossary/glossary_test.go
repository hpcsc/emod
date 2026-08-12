//go:build unit

package glossary_test

import (
	"encoding/json"
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/glossary"
	"github.com/hpcsc/emod/internal/test"
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
`, renderedMarkdown(t, model))
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
`, renderedMarkdown(t, model))
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
								{Name: "Hold a room", Trigger: &ast.Trigger{Name: "Booking Form", Actor: "Guest"}},
								{Name: "Cancel a hold", Trigger: &ast.Trigger{Name: "Cancellation Form", Actor: "Guest"}},
							},
						}},
					},
					{
						Name: "Billing",
						Aggregates: []*ast.Aggregate{{
							Name:   "Invoice",
							Slices: []*ast.Slice{{Name: "Pay an invoice", Trigger: &ast.Trigger{Name: "Payment Form", Actor: "Guest"}}},
						}},
					},
				},
			}

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
`, renderedMarkdown(t, model))
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
						Slices: []*ast.Slice{{Name: "Hold a room", Trigger: &ast.Trigger{Name: "Booking Form", Actor: "Guest"}}},
					}},
				}},
			}

			require.Equal(t, `# Hotel Operations

## Actors

### Auditor

Reviews the books once a quarter and never touches a screen

## Reservations

### Booking

### Actors

#### Guest

A person booking a room
`, renderedMarkdown(t, model))
		})

		t.Run("keeps an actor a trigger names though the model never declares it, with an empty definition", func(t *testing.T) {
			model := &ast.Model{
				Name: "Hotel Operations",
				Contexts: []*ast.Context{{
					Name: "Reservations",
					Aggregates: []*ast.Aggregate{{
						Name:   "Booking",
						Slices: []*ast.Slice{{Name: "Hold a room", Trigger: &ast.Trigger{Name: "Booking Form", Actor: "Concierge"}}},
					}},
				}},
			}

			require.Equal(t, `# Hotel Operations

## Reservations

### Booking

### Actors

#### Concierge
`, renderedMarkdown(t, model))
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
							Trigger: &ast.Trigger{Name: "Availability Screen", Reads: "AvailableRoomsView"},
							Views:   []*ast.View{{Name: "AvailableRoomsView", Description: "Rooms free over a date range"}},
						}},
					}},
				}},
			}

			require.Equal(t, `# Hotel Operations

## Reservations

### Booking

### Views

#### AvailableRoomsView

Rooms free over a date range
`, renderedMarkdown(t, model))
		})

		t.Run("lists an aggregate's invariants beneath it, in declaration order, each defined by its statement", func(t *testing.T) {
			model := &ast.Model{
				Name: "Library Lending",
				Contexts: []*ast.Context{{
					Name: "Lending",
					Aggregates: []*ast.Aggregate{{
						Name:        "Loan",
						Description: "One member holding one copy until it is due back",
						Invariants: []*ast.Invariant{
							{Name: "OneCopyPerLoan", Statement: "A loan covers exactly one copy of one title"},
							{Name: "FiveCopiesPerMember", Statement: "A member holds at most five copies at one time"},
						},
						Slices: []*ast.Slice{{
							Name:     "Borrow a copy",
							Commands: []*ast.Command{{Name: "BorrowCopy", Description: "Lend a copy to a member until a due date"}},
						}},
					}},
				}},
			}

			require.Equal(t, `# Library Lending

## Lending

### Loan

One member holding one copy until it is due back

#### Invariants

##### OneCopyPerLoan

A loan covers exactly one copy of one title

##### FiveCopiesPerMember

A member holds at most five copies at one time

### Commands

#### BorrowCopy

Lend a copy to a member until a due date
`, renderedMarkdown(t, model))
		})

		t.Run("lists a context's own invariants beneath the context and under none of its aggregates", func(t *testing.T) {
			model := &ast.Model{
				Name: "Library Lending",
				Contexts: []*ast.Context{{
					Name: "Reading Room",
					Mode: "dcb",
					Invariants: []*ast.Invariant{
						{Name: "OneReaderPerDesk", Statement: "A desk seats at most one reader at any moment"},
						{Name: "OneDeskPerReader", Statement: "A reader holds at most one desk for the length of a session"},
					},
					Aggregates: []*ast.Aggregate{{
						Name:       "Desk",
						Invariants: []*ast.Invariant{{Name: "DeskFreeAtClosing", Statement: "No desk stays claimed past the closing hour"}},
					}},
				}},
			}

			require.Equal(t, `# Library Lending

## Reading Room

### Invariants

#### OneReaderPerDesk

A desk seats at most one reader at any moment

#### OneDeskPerReader

A reader holds at most one desk for the length of a session

### Desk

#### Invariants

##### DeskFreeAtClosing

No desk stays claimed past the closing hour
`, renderedMarkdown(t, model))
		})

		t.Run("lists one invariant under each of two aggregates declaring the same identifier", func(t *testing.T) {
			model := &ast.Model{
				Name: "Library Lending",
				Contexts: []*ast.Context{{
					Name: "Lending",
					Aggregates: []*ast.Aggregate{
						{
							Name:       "Loan",
							Invariants: []*ast.Invariant{{Name: "OneAtATime", Statement: "A copy is out on at most one loan at any moment"}},
						},
						{
							Name:       "Hold",
							Invariants: []*ast.Invariant{{Name: "OneAtATime", Statement: "A member holds at most one reservation on a title"}},
						},
					},
				}},
			}

			require.Equal(t, `# Library Lending

## Lending

### Loan

#### Invariants

##### OneAtATime

A copy is out on at most one loan at any moment

### Hold

#### Invariants

##### OneAtATime

A member holds at most one reservation on a title
`, renderedMarkdown(t, model))
		})

		t.Run("keeps an invariant declared without a statement, with no definition beneath its name", func(t *testing.T) {
			model := &ast.Model{
				Name: "Library Lending",
				Contexts: []*ast.Context{{
					Name: "Lending",
					Aggregates: []*ast.Aggregate{{
						Name:       "Loan",
						Invariants: []*ast.Invariant{{Name: "OneCopyPerLoan"}},
					}},
				}},
			}

			require.Equal(t, `# Library Lending

## Lending

### Loan

#### Invariants

##### OneCopyPerLoan
`, renderedMarkdown(t, model))
		})
	})

	t.Run("json rendering", func(t *testing.T) {
		t.Run("carries an actor no trigger references at model level and the referenced one under its context", func(t *testing.T) {
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
						Slices: []*ast.Slice{{Name: "Hold a room", Trigger: &ast.Trigger{Name: "Booking Form", Actor: "Guest"}}},
					}},
				}},
			}

			require.Equal(t, map[string]any{
				"name":        "Hotel Operations",
				"description": "",
				"actors": []any{
					map[string]any{"name": "Auditor", "description": "Reviews the books once a quarter and never touches a screen"},
				},
				"contexts": []any{map[string]any{
					"name":        "Reservations",
					"description": "",
					"aggregates": []any{
						map[string]any{"name": "Booking", "description": ""},
					},
					"actors": []any{
						map[string]any{"name": "Guest", "description": "A person booking a room"},
					},
				}},
			}, renderedJSONDoc(t, model))
		})

		t.Run("carries each invariant under the aggregate or the context that declares it, in declaration order", func(t *testing.T) {
			model := &ast.Model{
				Name: "Library Lending",
				Contexts: []*ast.Context{
					{
						Name: "Lending",
						Aggregates: []*ast.Aggregate{{
							Name:        "Loan",
							Description: "One member holding one copy until it is due back",
							Invariants: []*ast.Invariant{
								{Name: "OneCopyPerLoan", Statement: "A loan covers exactly one copy of one title"},
								{Name: "FiveCopiesPerMember", Statement: "A member holds at most five copies at one time"},
							},
						}},
					},
					{
						Name: "Reading Room",
						Mode: "dcb",
						Invariants: []*ast.Invariant{
							{Name: "OneReaderPerDesk", Statement: "A desk seats at most one reader at any moment"},
							{Name: "DeskFreeAtClosing", Statement: "No desk stays claimed past the closing hour"},
						},
					},
				},
			}

			require.Equal(t, map[string]any{
				"name":        "Library Lending",
				"description": "",
				"contexts": []any{
					map[string]any{
						"name":        "Lending",
						"description": "",
						"aggregates": []any{map[string]any{
							"name":        "Loan",
							"description": "One member holding one copy until it is due back",
							"invariants": []any{
								map[string]any{"name": "OneCopyPerLoan", "description": "A loan covers exactly one copy of one title"},
								map[string]any{"name": "FiveCopiesPerMember", "description": "A member holds at most five copies at one time"},
							},
						}},
					},
					map[string]any{
						"name":        "Reading Room",
						"description": "",
						"invariants": []any{
							map[string]any{"name": "OneReaderPerDesk", "description": "A desk seats at most one reader at any moment"},
							map[string]any{"name": "DeskFreeAtClosing", "description": "No desk stays claimed past the closing hour"},
						},
					},
				},
			}, renderedJSONDoc(t, model))
		})

		t.Run("keeps a description key on an invariant declared without a statement", func(t *testing.T) {
			model := &ast.Model{
				Name: "Library Lending",
				Contexts: []*ast.Context{{
					Name: "Lending",
					Aggregates: []*ast.Aggregate{{
						Name:       "Loan",
						Invariants: []*ast.Invariant{{Name: "OneCopyPerLoan"}},
					}},
				}},
			}

			require.Equal(t, map[string]any{
				"name":        "Library Lending",
				"description": "",
				"contexts": []any{map[string]any{
					"name":        "Lending",
					"description": "",
					"aggregates": []any{map[string]any{
						"name":        "Loan",
						"description": "",
						"invariants": []any{
							map[string]any{"name": "OneCopyPerLoan", "description": ""},
						},
					}},
				}},
			}, renderedJSONDoc(t, model))
		})
	})

	t.Run("stating specs leaves both renderings untouched", func(t *testing.T) {
		stated := test.SpecLibraryLendingModel(t)
		unstated := test.WithoutSpecs(stated)

		require.NotEqual(t, stated, unstated,
			"the twin has to state no spec, or the comparisons below say nothing")
		require.Equal(t, test.SpecLibraryLendingSpecNames, test.DeclaredSpecNames(stated))
		require.Empty(t, test.DeclaredSpecNames(unstated),
			"the twin has to lose the specs of both slice homes, or the comparisons below are answered by whichever home it kept")

		require.Equal(t, renderedMarkdown(t, unstated), renderedMarkdown(t, stated),
			"a glossary defines the terms of a ubiquitous language, and a scenario is not one of them")
		require.Equal(t, renderedJSON(t, unstated), renderedJSON(t, stated),
			"a glossary defines the terms of a ubiquitous language, and a scenario is not one of them")
	})

	t.Run("binding wire types leaves both renderings untouched", func(t *testing.T) {
		bound := test.WireTypeLibraryLendingModel(t)
		unbound := test.WithoutWireTypes(bound)

		require.NotEqual(t, bound, unbound,
			"the twin has to bind no wire type, or the comparisons below say nothing")
		require.Equal(t, test.WireTypeLibraryLendingWireTypes, test.DeclaredWireTypes(bound))
		require.Empty(t, test.DeclaredWireTypes(unbound),
			"the twin has to lose the wire types of both slice homes and of the nested event, or the comparisons below are answered by whichever it kept")

		require.Equal(t, renderedMarkdown(t, unbound), renderedMarkdown(t, bound),
			"a glossary defines the terms of a ubiquitous language, and a deployment identifier is not one of them")
		require.Equal(t, renderedJSON(t, unbound), renderedJSON(t, bound),
			"a glossary defines the terms of a ubiquitous language, and a deployment identifier is not one of them")
	})

	t.Run("stating example payloads leaves both renderings untouched", func(t *testing.T) {
		stated := test.PayloadLibraryLendingModel(t)
		unstated := test.WithoutSpecPayloads(stated)

		require.NotEqual(t, stated, unstated,
			"the twin has to state no payload, or the comparisons below say nothing")
		require.Equal(t, test.PayloadLibraryLendingPayloads, test.DeclaredSpecPayloads(stated))
		require.Empty(t, test.DeclaredSpecPayloads(unstated),
			"the twin has to lose the payloads of both slice homes, or the comparisons below are answered by whichever home it kept")
		require.Equal(t, test.DeclaredSpecNames(stated), test.DeclaredSpecNames(unstated),
			"the twin has to keep every spec, or it is a spec differential rather than a payload one")

		require.Equal(t, renderedMarkdown(t, unstated), renderedMarkdown(t, stated),
			"a glossary defines the terms of a ubiquitous language, and an example value is not one of them")
		require.Equal(t, renderedJSON(t, unstated), renderedJSON(t, stated),
			"a glossary defines the terms of a ubiquitous language, and an example value is not one of them")
	})

	t.Run("stating specs with all four outcomes leaves both renderings untouched", func(t *testing.T) {
		stated := test.SlicePatternLibraryLendingModel(t)
		unstated := test.WithoutSpecs(stated)

		require.NotEqual(t, stated, unstated,
			"the twin has to state no spec, or the comparisons below say nothing")
		require.Equal(t, test.SlicePatternLibraryLendingSpecNames, test.DeclaredSpecNames(stated))
		require.Empty(t, test.DeclaredSpecNames(unstated),
			"the twin has to lose the specs of both slice homes, or the comparisons below are answered by whichever home it kept")

		require.Equal(t, renderedMarkdown(t, unstated), renderedMarkdown(t, stated),
			"a glossary defines the terms of a ubiquitous language, and a scenario is not one of them")
		require.Equal(t, renderedJSON(t, unstated), renderedJSON(t, stated),
			"a glossary defines the terms of a ubiquitous language, and a scenario is not one of them")
	})
}

func renderedMarkdown(t *testing.T, model *ast.Model) string {
	t.Helper()

	rendered, err := glossary.RenderMarkdown(model)
	require.NoError(t, err)

	return string(rendered)
}

func renderedJSON(t *testing.T, model *ast.Model) string {
	t.Helper()

	rendered, err := glossary.RenderJSON(model)
	require.NoError(t, err)

	return string(rendered)
}

func renderedJSONDoc(t *testing.T, model *ast.Model) map[string]any {
	t.Helper()

	var doc map[string]any
	require.NoError(t, json.Unmarshal([]byte(renderedJSON(t, model)), &doc))

	return doc
}

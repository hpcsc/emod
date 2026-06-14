//go:build unit

package diagram_test

import (
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagram"
	"github.com/stretchr/testify/require"
)

func TestExportMermaid(t *testing.T) {
	t.Run("nil model returns empty bytes with no error", func(t *testing.T) {
		raw, err := diagram.ExportMermaid(nil, diagram.StyleAuto)
		require.NoError(t, err)
		require.Empty(t, raw)
	})

	t.Run("empty model (no contexts) returns output starting with eventmodeling and no timeframe entries", func(t *testing.T) {
		model := &ast.Model{Name: "Empty"}
		raw, err := diagram.ExportMermaid(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "eventmodeling")
		require.NotContains(t, output, "tf ")
	})

	t.Run("single slice with UI trigger renders as tf NN ui TriggerName", func(t *testing.T) {
		model := singleSliceModel("Test", "S1",
			&ast.Trigger{Kind: "UI", Name: "SubmitForm"})
		raw, err := diagram.ExportMermaid(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "tf 01 ui Ctx.SubmitForm")
	})

	t.Run("single slice with schedule trigger renders as tf NN pcr TriggerName", func(t *testing.T) {
		model := singleSliceModel("Test", "S1",
			&ast.Trigger{Kind: "Schedule", Name: "NightlyBatch"})
		raw, err := diagram.ExportMermaid(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "tf 01 pcr Ctx.NightlyBatch")
	})

	t.Run("single slice with processor trigger renders as tf NN pcr TriggerName", func(t *testing.T) {
		model := singleSliceModel("Test", "S1",
			&ast.Trigger{Kind: "Processor", Name: "FileWatcher"})
		raw, err := diagram.ExportMermaid(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "tf 01 pcr Ctx.FileWatcher")
	})

	t.Run("commands render as tf NN cmd CommandName", func(t *testing.T) {
		model := singleSliceModel("Test", "S1", command("CreateOrder"))
		raw, err := diagram.ExportMermaid(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "tf 01 cmd Ctx.CreateOrder")
	})

	t.Run("events render as tf NN evt EventName", func(t *testing.T) {
		model := singleSliceModel("Test", "S1", event("OrderCreated"))
		raw, err := diagram.ExportMermaid(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "tf 01 evt Ctx.OrderCreated")
	})

	t.Run("views render as tf NN rmo ViewName", func(t *testing.T) {
		model := singleSliceModel("Test", "S1", view("OrderSummary"))
		raw, err := diagram.ExportMermaid(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "tf 01 rmo Ctx.OrderSummary")
	})

	t.Run("automations render as tf NN pcr AutomationName", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{{
						Name: "S1",
						Automations: []*ast.Automation{{
							Name: "AutoNotifier",
						}},
					}},
				}},
			}},
		}
		raw, err := diagram.ExportMermaid(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "tf 01 pcr Ctx.AutoNotifier")
	})

	t.Run("context names are prepended with dot notation", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Orders",
				Aggregates: []*ast.Aggregate{{
					Name: "Order",
					Slices: []*ast.Slice{{
						Name:     "CreateOrder",
						Commands: []*ast.Command{{Name: "CreateOrder"}},
					}},
				}},
			}},
		}
		raw, err := diagram.ExportMermaid(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "tf 01 cmd Orders.CreateOrder")
	})

	t.Run("automation with TargetContext uses the target context as the namespace prefix", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Orders",
				Aggregates: []*ast.Aggregate{{
					Name: "Order",
					Slices: []*ast.Slice{{
						Name: "S1",
						Automations: []*ast.Automation{{
							Name:         "Notifier",
							TargetContext: "Billing",
						}},
					}},
				}},
			}},
		}
		raw, err := diagram.ExportMermaid(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "tf 01 pcr Billing.Notifier")
	})

	t.Run("sequential numbering increments across all slices", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{
						{
							Name:     "S1",
							Commands: []*ast.Command{{Name: "CmdA"}},
						},
						{
							Name:     "S2",
							Commands: []*ast.Command{{Name: "CmdB"}},
						},
					},
				}},
			}},
		}
		raw, err := diagram.ExportMermaid(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "tf 01 cmd Ctx.CmdA")
		require.Contains(t, output, "tf 02 cmd Ctx.CmdB")
	})

	t.Run("model name appears as a comment", func(t *testing.T) {
		model := &ast.Model{
			Name: "MyModel",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{{
						Name:     "S1",
						Commands: []*ast.Command{{Name: "Cmd"}},
					}},
				}},
			}},
		}
		raw, err := diagram.ExportMermaid(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "% MyModel")
	})

	t.Run("slice names appear as comments", func(t *testing.T) {
		model := minimalModel("Test", "InitOrder")
		raw, err := diagram.ExportMermaid(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "% Slice: InitOrder")
	})

	t.Run("complete model with all element types produces well-formed output", func(t *testing.T) {
		model := fullModel()
		raw, err := diagram.ExportMermaid(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "eventmodeling")
		require.Contains(t, output, "% FullModel")
		require.Contains(t, output, "% Slice: Create Order")
		require.Contains(t, output, "% Slice: Ship Order")

		// First slice: Create Order — UI trigger, commands, events, view, automation
		require.Contains(t, output, "tf 01 ui Orders.PlaceOrderForm")
		require.Contains(t, output, "tf 02 cmd Orders.CreateOrder")
		require.Contains(t, output, "tf 03 cmd Orders.ValidatePayment")
		require.Contains(t, output, "tf 04 evt Orders.OrderCreated")
		require.Contains(t, output, "tf 05 evt Orders.PaymentValidated")
		require.Contains(t, output, "tf 06 rmo Orders.OrderSummary")
		require.Contains(t, output, "tf 07 pcr Orders.InventoryUpdater")
		require.Contains(t, output, "tf 08 pcr Orders.PaymentGW")

		// Second slice: Ship Order — schedule trigger, command, event
		require.Contains(t, output, "tf 09 pcr Orders.ShipTimer")
		require.Contains(t, output, "tf 10 cmd Orders.ShipOrder")
		require.Contains(t, output, "tf 11 evt Orders.OrderShipped")
	})

	t.Run("DCB context with direct slices renders timeframe entries", func(t *testing.T) {
		model := &ast.Model{
			Name: "DCBTest",
			Contexts: []*ast.Context{{
				Name: "DCBCtx",
				Slices: []*ast.Slice{{
					Name:     "DirectSlice",
					Commands: []*ast.Command{{Name: "DirectCmd"}},
				}},
			}},
		}

		raw, err := diagram.ExportMermaid(model, diagram.StyleDCB)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "eventmodeling")
		require.Contains(t, output, "tf 01 cmd DCBCtx.DirectCmd")
	})

	// --- Projected style (tag-based) tests ---

	t.Run("aggregate-only context with projected style produces identical output", func(t *testing.T) {
		model := fullModel()
		autoRaw, err := diagram.ExportMermaid(model, diagram.StyleAuto)
		require.NoError(t, err)
		projRaw, err := diagram.ExportMermaid(model, diagram.StyleProjected)
		require.NoError(t, err)
		require.Equal(t, autoRaw, projRaw, "aggregate-only context must produce identical output")
	})

	t.Run("DCB context with tagged events renders tag sections", func(t *testing.T) {
		model := &ast.Model{
			Name: "TagTest",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Slices: []*ast.Slice{{
					Name: "S1",
					Events: []*ast.Event{
						{Name: "OrderPlaced", Tags: []ast.TagEntry{{Key: "priority"}}},
						{Name: "OrderShipped", Tags: []ast.TagEntry{{Key: "region"}}},
					},
				}},
			}},
		}

		raw, err := diagram.ExportMermaid(model, diagram.StyleProjected)
		require.NoError(t, err)

		output := string(raw)
		// Should have tag-key section markers
		require.Contains(t, output, "% Tag: priority")
		require.Contains(t, output, "% Tag: region")
		// Should have tag-key namespace prefixes on events
		require.Contains(t, output, "tf 01 evt priority.OrderPlaced")
		require.Contains(t, output, "tf 02 evt region.OrderShipped")
		// Should have the Commands / Triggers section header
		require.Contains(t, output, "% Commands / Triggers")
	})

	t.Run("single-tag event appears only in that tag's section", func(t *testing.T) {
		model := &ast.Model{
			Name: "SingleTag",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Slices: []*ast.Slice{{
					Name: "S1",
					Events: []*ast.Event{
						{Name: "OrderPlaced", Tags: []ast.TagEntry{{Key: "priority"}}},
					},
				}},
			}},
		}

		raw, err := diagram.ExportMermaid(model, diagram.StyleProjected)
		require.NoError(t, err)

		output := string(raw)
		// Event appears only with the "priority" tag prefix
		require.Contains(t, output, "priority.OrderPlaced")
		// Should not appear with any other tag prefix
		require.NotContains(t, output, "region.OrderPlaced")
	})

	t.Run("multi-tag event appears in multiple sections with connector comment", func(t *testing.T) {
		model := &ast.Model{
			Name: "MultiTag",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Slices: []*ast.Slice{{
					Name: "S1",
					Events: []*ast.Event{
						{Name: "OrderPlaced", Tags: []ast.TagEntry{{Key: "priority"}, {Key: "region"}}},
					},
				}},
			}},
		}

		raw, err := diagram.ExportMermaid(model, diagram.StyleProjected)
		require.NoError(t, err)

		output := string(raw)
		// Event appears with both tag prefixes
		require.Contains(t, output, "tf 01 evt priority.OrderPlaced")
		require.Contains(t, output, "tf 02 evt region.OrderPlaced")
		// Connector comment on one of them (priority is first tag, region is second alphabetically)
		require.Contains(t, output, "%   connector: also in")
	})

	t.Run("commands and triggers appear in Commands / Triggers section", func(t *testing.T) {
		model := &ast.Model{
			Name: "CmdTest",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Slices: []*ast.Slice{{
					Name: "S1",
					Trigger: &ast.Trigger{Kind: "UI", Name: "Submit", Actor: "User"},
					Commands: []*ast.Command{{Name: "ProcessOrder"}},
					Events: []*ast.Event{
						{Name: "OrderPlaced", Tags: []ast.TagEntry{{Key: "priority"}}},
					},
				}},
			}},
		}

		raw, err := diagram.ExportMermaid(model, diagram.StyleProjected)
		require.NoError(t, err)

		output := string(raw)
		// Commands and triggers appear in Commands/Triggers section
		require.Contains(t, output, "% Commands / Triggers")
		require.Contains(t, output, "tf 01 ui Ctx.Submit")
		require.Contains(t, output, "tf 02 cmd Ctx.ProcessOrder")
		// Event appears in tag section (not in commands section)
		require.Contains(t, output, "tf 03 evt priority.OrderPlaced")
	})

	t.Run("DCB context without tags renders everything in general section", func(t *testing.T) {
		model := &ast.Model{
			Name: "NoTag",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Slices: []*ast.Slice{{
					Name: "S1",
					Commands: []*ast.Command{{Name: "DoSomething"}},
					Events:   []*ast.Event{{Name: "SomethingHappened"}},
				}},
			}},
		}

		raw, err := diagram.ExportMermaid(model, diagram.StyleProjected)
		require.NoError(t, err)

		output := string(raw)
		// Since there are no tag keys, projected mode is not activated
		// Output should be same as standard mode
		require.Contains(t, output, "% Slice: S1")
		require.Contains(t, output, "tf 01 cmd Ctx.DoSomething")
		require.Contains(t, output, "tf 02 evt Ctx.SomethingHappened")
	})

	t.Run("mixed mode shows both aggregate events and tag-grouped events", func(t *testing.T) {
		model := &ast.Model{
			Name: "Mixed",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{{
						Name: "AggSlice",
						Events: []*ast.Event{{Name: "AggEvent"}},
					}},
				}},
				Slices: []*ast.Slice{{
					Name: "DCBSlice",
					Events: []*ast.Event{
						{Name: "DCBEvent", Tags: []ast.TagEntry{{Key: "dcbTag"}}},
					},
				}},
			}},
		}

		projRaw, err := diagram.ExportMermaid(model, diagram.StyleProjected)
		require.NoError(t, err)

		projOut := string(projRaw)

		// Aggregate event appears in general section with context prefix
		require.Contains(t, projOut, "evt Ctx.AggEvent")
		// Tag event appears in tag section
		require.Contains(t, projOut, "% Tag: dcbTag")
		require.Contains(t, projOut, "evt dcbTag.DCBEvent")
	})

	t.Run("projected output starts with eventmodeling", func(t *testing.T) {
		model := &ast.Model{
			Name: "Validity",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Slices: []*ast.Slice{{
					Name: "S1",
					Events: []*ast.Event{
						{Name: "EventA", Tags: []ast.TagEntry{{Key: "alpha"}}},
					},
				}},
			}},
		}

		raw, err := diagram.ExportMermaid(model, diagram.StyleProjected)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "eventmodeling")
		require.Contains(t, output, "% Validity")
		// All timeframe entries use the tf pattern
		require.Contains(t, output, "tf 01 evt alpha.EventA")
	})

	t.Run("tag sections are in alphabetical order", func(t *testing.T) {
		model := &ast.Model{
			Name: "OrderTest",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Slices: []*ast.Slice{{
					Name: "S1",
					Events: []*ast.Event{
						{Name: "EventA", Tags: []ast.TagEntry{{Key: "zeta"}, {Key: "alpha"}}},
						{Name: "EventB", Tags: []ast.TagEntry{{Key: "beta"}}},
					},
				}},
			}},
		}

		raw, err := diagram.ExportMermaid(model, diagram.StyleProjected)
		require.NoError(t, err)

		output := string(raw)
		// alpha section should appear before beta, beta before zeta
		alphaIdx := strings.Index(output, "% Tag: alpha")
		betaIdx := strings.Index(output, "% Tag: beta")
		zetaIdx := strings.Index(output, "% Tag: zeta")
		require.True(t, alphaIdx < betaIdx, "alpha should come before beta")
		require.True(t, betaIdx < zetaIdx, "beta should come before zeta")
	})
}

// --- DCB style (query-lens) tests ---

func TestExportMermaidDCB(t *testing.T) {
	t.Run("DCB context with StyleDCB renders flat events section", func(t *testing.T) {
		model := &ast.Model{
			Name: "DCBFlat",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Slices: []*ast.Slice{{
					Name: "S1",
					Events: []*ast.Event{
						{Name: "EventA", Tags: []ast.TagEntry{{Key: "priority"}}},
						{Name: "EventB", Tags: []ast.TagEntry{{Key: "region"}}},
					},
				}},
			}},
		}

		raw, err := diagram.ExportMermaid(model, diagram.StyleDCB)
		require.NoError(t, err)

		output := string(raw)
		// Should have a single % Events section (not tag sections)
		require.Contains(t, output, "% Events")
		// Should NOT have tag section markers
		require.NotContains(t, output, "% Tag: priority")
		require.NotContains(t, output, "% Tag: region")
		// Should have Commands / Triggers section
		require.Contains(t, output, "% Commands / Triggers")
		// Events should not have tag-key namespace prefix
		require.Contains(t, output, "Ctx.EventA")
		require.Contains(t, output, "Ctx.EventB")
	})

	t.Run("command with decides_on shows annotation in DCB mode", func(t *testing.T) {
		model := &ast.Model{
			Name: "DecidesOn",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Slices: []*ast.Slice{{
					Name: "S1",
					Commands: []*ast.Command{{
						Name: "ProcessOrder",
						DecidesOn: &ast.DecidesOnClause{
							Events: []string{"OrderProcessed"},
						},
					}},
					Events: []*ast.Event{{Name: "OrderProcessed"}},
					Flows:  []*ast.Flow{{CommandName: "ProcessOrder", EventName: "OrderProcessed"}},
				}},
			}},
		}

		raw, err := diagram.ExportMermaid(model, diagram.StyleDCB)
		require.NoError(t, err)

		output := string(raw)
		// Command should be present with decides_on comment
		require.Contains(t, output, "Ctx.ProcessOrder")
		require.Contains(t, output, "decides_on: OrderProcessed")
	})

	t.Run("event with tags shows tag badges in DCB mode", func(t *testing.T) {
		model := &ast.Model{
			Name: "TagBadges",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Slices: []*ast.Slice{{
					Name: "S1",
					Events: []*ast.Event{
						{Name: "OrderPlaced", Tags: []ast.TagEntry{{Key: "priority", FieldRef: "high"}}},
					},
				}},
			}},
		}

		raw, err := diagram.ExportMermaid(model, diagram.StyleDCB)
		require.NoError(t, err)

		output := string(raw)
		// Event with tags should have tags comment
		require.Contains(t, output, "Ctx.OrderPlaced")
		require.Contains(t, output, "tags: [priority: high]")
	})

	t.Run("non-DCB context with StyleDCB renders without error", func(t *testing.T) {
		model := fullModel()
		raw, err := diagram.ExportMermaid(model, diagram.StyleDCB)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "eventmodeling")
		require.NotContains(t, output, "% Events")
	})

	t.Run("DCB mode output starts with eventmodeling", func(t *testing.T) {
		model := &ast.Model{
			Name: "DCBTest",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Slices: []*ast.Slice{{
					Name: "S1",
					Events: []*ast.Event{
						{Name: "EventA", Tags: []ast.TagEntry{{Key: "alpha"}}},
					},
				}},
			}},
		}

		raw, err := diagram.ExportMermaid(model, diagram.StyleDCB)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "eventmodeling")
		require.Contains(t, output, "% DCBTest")
		require.Contains(t, output, "tf 01 evt Ctx.EventA")
	})

	t.Run("command with decides_on and predicate shows full annotation", func(t *testing.T) {
		model := &ast.Model{
			Name: "PredicateDCB",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Slices: []*ast.Slice{{
					Name: "S1",
					Commands: []*ast.Command{{
						Name: "PrioritizeOrder",
						DecidesOn: &ast.DecidesOnClause{
							Events: []string{"OrderPrioritized", "OrderFlagged"},
							Predicate: ast.TagPredicate{
								Field:    "priority",
								Operator: "==",
								Value:    "high",
							},
						},
					}},
					Events: []*ast.Event{
						{Name: "OrderPrioritized"},
						{Name: "OrderFlagged"},
					},
					Flows: []*ast.Flow{
						{CommandName: "PrioritizeOrder", EventName: "OrderPrioritized"},
					},
				}},
			}},
		}

		raw, err := diagram.ExportMermaid(model, diagram.StyleDCB)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "Ctx.PrioritizeOrder")
		require.Contains(t, output, "decides_on: OrderPrioritized, OrderFlagged")
		require.Contains(t, output, "where tag(priority == high)")
	})
}

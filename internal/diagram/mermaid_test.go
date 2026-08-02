//go:build unit

package diagram_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagram"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
)

// libraryLendingProcessorTimeframes is the processor line every style writes for
// each automation of test.AutomationScheduleLibraryLending, both slice homes
// together and in declaration order: the ones running on a cadence name it where
// the ones activating on an event name that event.
var libraryLendingProcessorTimeframes = []string{
	`pcr Lending.RemindMemberEachMorning (every "0 9 * * *" → RemindMember)`,
	`pcr Lending.RecallOnSecondReminder (MemberReminded → RecallCopy)`,
	`pcr Lending.SweepOverdueLoans (every "15m" → RecallCopy)`,
	`pcr Reading Room.CloseDesksAtNight (every "0 22 * * *" → ReleaseDesk)`,
	`pcr Reading Room.RemindReaderOfLoans (DeskReleased → RemindMember)`,
	`pcr Reading Room.SweepIdleDesks (every "45m" → ReleaseDesk)`,
}

// timeframeSlotAssignments is the timeframe each of timeframeSlotModel's
// non-event elements is written into, every style alike, both slice homes
// together and in declaration order: a trigger belongs to the timeframe a
// person acts in and an automation or a translation reactor to the processor's.
var timeframeSlotAssignments = []string{
	`ui Lending.Lending Desk`,
	`cmd Lending.BorrowCopy`,
	`rmo Lending.MemberLoansView`,
	`pcr Lending.RemindOnDueDate (CopyBorrowed → RemindMember)`,
	`pcr Lending.PartnerImport`,
	`ui Lending.ReturnsBinWatcher`,
	`cmd Lending.ReturnCopy`,
	`ui Lending.NightlySweep`,
	`cmd Lending.RecallCopy`,
	`pcr Lending.SweepOverdueLoans (every "15m" → RecallCopy)`,
}

// timeframeSlotModel declares a trigger in each home a slice has — one named
// for the screen it is, one named as a schedule and one named as a background
// watcher — beside the automation and the translation reactor that share the
// processor timeframe with each other. Its directly declared slice tags an
// event, without which the DCB and the projected style both fall back to the
// standard layout and answer nothing about themselves.
func timeframeSlotModel() *ast.Model {
	return &ast.Model{
		Name: "Library Lending",
		Contexts: []*ast.Context{{
			Name: "Lending",
			Aggregates: []*ast.Aggregate{{
				Name: "Loan",
				Slices: []*ast.Slice{
					{
						Name:     "Borrow Copy",
						Trigger:  &ast.Trigger{Name: "Lending Desk", Actor: "Member"},
						Commands: []*ast.Command{command("BorrowCopy")},
						Events:   []*ast.Event{event("CopyBorrowed")},
						Views:    []*ast.View{{Name: "MemberLoansView", Subscribes: []string{"CopyBorrowed"}}},
						Automations: []*ast.Automation{{
							Name:    "RemindOnDueDate",
							OnEvent: "CopyBorrowed",
							Command: "RemindMember",
						}},
						Translations: []*ast.Translation{{
							Name:           "PartnerImport",
							ExternalSystem: "LoanRegistry",
							Command:        "BorrowCopy",
							Event:          &ast.Event{Name: "PartnerLoanReceived"},
						}},
					},
					{
						Name:     "Empty The Returns Bin",
						Trigger:  &ast.Trigger{Name: "ReturnsBinWatcher"},
						Commands: []*ast.Command{command("ReturnCopy")},
					},
				},
			}},
			Slices: []*ast.Slice{{
				Name:     "Sweep Overdue Loans",
				Trigger:  &ast.Trigger{Name: "NightlySweep"},
				Commands: []*ast.Command{command("RecallCopy")},
				Events: []*ast.Event{
					{Name: "CopyRecalled", Tags: []ast.TagEntry{{Key: "loan", FieldRef: "loanId"}}},
				},
				Automations: []*ast.Automation{{
					Name:     "SweepOverdueLoans",
					Schedule: "15m",
					Command:  "RecallCopy",
				}},
			}},
		}},
	}
}

// timeframeEntry is one tf line split into the number the style gave it and the
// slot it wrote: the timeframe letter together with the element that letter
// places.
type timeframeEntry struct {
	number int
	slot   string
}

func timeframeEntries(t *testing.T, output string) []timeframeEntry {
	t.Helper()

	var entries []timeframeEntry
	for _, line := range strings.Split(output, "\n") {
		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 3 || parts[0] != "tf" {
			continue
		}

		number, err := strconv.Atoi(parts[1])
		require.NoError(t, err, "timeframe entry %q must carry a number", line)
		entries = append(entries, timeframeEntry{number: number, slot: parts[2]})
	}

	return entries
}

// timeframeSlots returns the slot of every entry keep accepts, dropping the
// sequence numbers: the styles number the same entries differently, and no
// expectation reading slots back is about the numbering.
func timeframeSlots(t *testing.T, output string, keep func(slot string) bool) []string {
	t.Helper()

	var slots []string
	for _, entry := range timeframeEntries(t, output) {
		if keep(entry.slot) {
			slots = append(slots, entry.slot)
		}
	}

	return slots
}

// nonEventTimeframes keeps every slot but the events': each style places events
// in a section of its own, and no expectation here is about where.
func nonEventTimeframes(t *testing.T, output string) []string {
	return timeframeSlots(t, output, func(slot string) bool {
		return !strings.HasPrefix(slot, "evt ")
	})
}

func eventTimeframes(t *testing.T, output string) []string {
	return timeframeSlots(t, output, func(slot string) bool {
		return strings.HasPrefix(slot, "evt ")
	})
}

func processorTimeframes(t *testing.T, output string) []string {
	return timeframeSlots(t, output, func(slot string) bool {
		return strings.HasPrefix(slot, "pcr ")
	})
}

// renderTimeframeSlotModel renders timeframeSlotModel in the given style, having
// first read back the events to see that this style laid them out its own way.
// The DCB and the projected style fall back to the standard layout for a model
// they find nothing of their own to lay out, and a fallback would leave the
// standard layout answering for the style named here.
func renderTimeframeSlotModel(t *testing.T, style diagram.Style, events []string) string {
	t.Helper()

	raw, err := diagram.ExportMermaid(timeframeSlotModel(), style)
	require.NoError(t, err)

	output := string(raw)
	require.Equal(t, events, eventTimeframes(t, output))

	return output
}

func timeframeNumbers(t *testing.T, output string) []int {
	t.Helper()

	var numbers []int
	for _, entry := range timeframeEntries(t, output) {
		numbers = append(numbers, entry.number)
	}

	return numbers
}

func countingFromOne(count int) []int {
	var numbers []int
	for n := 1; n <= count; n++ {
		numbers = append(numbers, n)
	}

	return numbers
}

func TestExportMermaid(t *testing.T) {
	t.Run("auto and projected styles", func(t *testing.T) {
		t.Run("empty model (no contexts) returns output starting with eventmodeling and no timeframe entries", func(t *testing.T) {
			model := &ast.Model{Name: "Empty"}
			raw, err := diagram.ExportMermaid(model, diagram.StyleAuto)
			require.NoError(t, err)

			output := string(raw)
			require.Contains(t, output, "eventmodeling")
			require.NotContains(t, output, "tf ")
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
								Name:          "Notifier",
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

		t.Run("a scheduled automation names its cadence where an event-activated one names its event", func(t *testing.T) {
			for _, style := range []struct {
				name  string
				style diagram.Style
			}{
				{name: "auto", style: diagram.StyleAuto},
				{name: "projected", style: diagram.StyleProjected},
			} {
				t.Run(style.name, func(t *testing.T) {
					raw, err := diagram.ExportMermaid(test.AutomationScheduleLibraryLendingModel(t), style.style)
					require.NoError(t, err)

					require.Equal(t, libraryLendingProcessorTimeframes, processorTimeframes(t, string(raw)))
				})
			}
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
			require.Contains(t, output, "tf 09 ui Orders.ShipTimer")
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
						Name:     "S1",
						Trigger:  &ast.Trigger{Name: "Submit", Actor: "User"},
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
						Name:     "S1",
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
							Name:   "AggSlice",
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
			sections := []string{"% Tag: alpha", "% Tag: beta", "% Tag: zeta"}

			require.Equal(t, sections, appearanceOrder(t, output, sections...))
		})
	})

	t.Run("dcb style", func(t *testing.T) {
		t.Run("a scheduled automation names its cadence where an event-activated one names its event", func(t *testing.T) {
			raw, err := diagram.ExportMermaid(test.AutomationScheduleLibraryLendingModel(t), diagram.StyleDCB)
			require.NoError(t, err)

			require.Equal(t, libraryLendingProcessorTimeframes, processorTimeframes(t, string(raw)))
		})

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
	})

	t.Run("every style", func(t *testing.T) {
		for _, style := range []struct {
			name  string
			style diagram.Style
			// events is where this style writes timeframeSlotModel's events, and
			// no two styles write them alike: the standard layout leaves each
			// event in the slice declaring it, the DCB layout gathers them all
			// into a section of their own and is the only style to give the
			// event a translation nests a line, and the projected layout files a
			// tagged event under its tag key instead.
			events []string
		}{
			{name: "standard", style: diagram.StyleAuto, events: []string{
				"evt Lending.CopyBorrowed",
				"evt Lending.CopyRecalled",
			}},
			{name: "dcb", style: diagram.StyleDCB, events: []string{
				"evt Lending.CopyBorrowed",
				"evt Lending.PartnerLoanReceived",
				"evt Lending.CopyRecalled",
			}},
			{name: "projected", style: diagram.StyleProjected, events: []string{
				"evt Lending.CopyBorrowed",
				"evt loan.CopyRecalled",
			}},
		} {
			t.Run(style.name, func(t *testing.T) {
				t.Run("a trigger takes the ui timeframe whatever its name reads, and an automation and a translation the pcr timeframe", func(t *testing.T) {
					output := renderTimeframeSlotModel(t, style.style, style.events)

					require.Equal(t, timeframeSlotAssignments, nonEventTimeframes(t, output))
				})

				t.Run("timeframe numbers run unbroken from one", func(t *testing.T) {
					output := renderTimeframeSlotModel(t, style.style, style.events)

					require.Equal(t, countingFromOne(len(timeframeSlotAssignments)+len(style.events)), timeframeNumbers(t, output),
						"a repeated number draws two elements in the same place, a gap leaves a hole, and a short sequence is an element that lost its line")
				})
			})
		}
	})
}

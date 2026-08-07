//go:build unit

package importer_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/export"
	"github.com/hpcsc/emod/internal/formatter"
	"github.com/hpcsc/emod/internal/importer"
	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/parser"
	"github.com/hpcsc/emod/internal/test"
	"github.com/hpcsc/emod/internal/validator"
	"github.com/stretchr/testify/require"
)

func parseModel(t *testing.T, source string) *ast.Model {
	t.Helper()
	tokens, _ := lexer.Scan(source, "test.emod")
	model, _ := parser.New(tokens, "test.emod").Parse()
	require.NotNil(t, model)
	return model
}

func importFrom(t *testing.T, source string) *ast.Model {
	t.Helper()
	return importExported(t, parseModel(t, source))
}

// importExported runs model through the export→import path the viewer uses:
// the model becomes diagram JSON, the diagram JSON becomes an AST again.
func importExported(t *testing.T, model *ast.Model) *ast.Model {
	t.Helper()
	diagramJSON, err := export.ExportDiagramJSON(model)
	require.NoError(t, err)
	return importDiagram(t, string(diagramJSON))
}

func importDiagram(t *testing.T, document string) *ast.Model {
	t.Helper()
	model, err := importer.ImportDiagram([]byte(document))
	require.NoError(t, err)
	return model
}

// parseSaved reads back the text a save writes for model, so what the two
// helpers below report on is the file the next open would see.
func parseSaved(model *ast.Model) (*ast.Model, []*diagnostic.Entry) {
	tokens, scanDiags := lexer.Scan(formatter.Format(model), "saved.emod")
	saved, parseDiags := parser.New(tokens, "saved.emod").Parse()
	return saved, append(scanDiags, parseDiags...)
}

func savedTextDiagnostics(model *ast.Model) []*diagnostic.Entry {
	_, diagnostics := parseSaved(model)
	return diagnostics
}

func savedModelDiagnostics(model *ast.Model) []*diagnostic.Entry {
	saved, _ := parseSaved(model)
	return validator.Validate(saved)
}

// automationNodeKeying binds the label and the value of a document's single
// automation node once, so the pair of documents a caller renders from it
// differs in nothing but the key the reader is asked for.
func automationNodeKeying(label, value string) func(key string) string {
	return func(key string) string {
		return fmt.Sprintf(`{
      "model_name": "M",
      "nodes": [
        {"id": "context-1", "type": "context", "label": "C", "parentId": null},
        {"id": "slice-1", "type": "slice", "label": "S", "parentId": "context-1"},
        {"id": "auto-1", "type": "automation", "label": %q, "parentId": "slice-1",
         %q: %q}
      ],
      "edges": []
    }`, label, key, value)
	}
}

func TestImportDiagram(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		t.Run("reproduces the formatted source for every documented pattern", func(t *testing.T) {
			source, err := os.ReadFile("../parser/testdata/all_patterns.emod")
			require.NoError(t, err)

			// Comments do not survive diagram JSON, so the baseline is the
			// formatted model with comments stripped rather than the file.
			original := parseModel(t, string(source))
			stripComments(original)
			expected := formatter.Format(original)

			require.Equal(t, expected, formatter.Format(importFrom(t, string(source))))
		})

		t.Run("reproduces the formatted source across multiple contexts", func(t *testing.T) {
			source, err := os.ReadFile("../parser/testdata/multi_context.emod")
			require.NoError(t, err)

			original := parseModel(t, string(source))
			stripComments(original)
			expected := formatter.Format(original)

			require.Equal(t, expected, formatter.Format(importFrom(t, string(source))))
		})

		t.Run("preserves slices declared directly under a context", func(t *testing.T) {
			source := `emod 1
model "M"

context "C" {
  slice "Direct" {
    command DoThing {
      fields {
        id string required
      }
    }

    event ThingDone {
      fields {
        id string required
      }
    }

    flow {
      command -> event: DoThing -> ThingDone
    }
  }
}
`
			require.Equal(t, source, formatter.Format(importFrom(t, source)))
		})

		t.Run("preserves a translation without duplicating its nested event", func(t *testing.T) {
			source := `emod 1
model "M"

context "C" {
  aggregate "A" {
    slice "S" {
      command RecordPayment {
        fields {
          amount int required
        }
      }

      translation StripeWebhook {
        external_system "Stripe"
        command RecordPayment
        event PaymentReceived {
          fields {
            amount int required
          }
        }
      }
    }
  }
}
`
			require.Equal(t, source, formatter.Format(importFrom(t, source)))
		})

		t.Run("preserves the view an automation reads, leaving the one beside it reading nothing", func(t *testing.T) {
			source := `emod 1
model "M"

context "C" {
  aggregate "A" {
    slice "Review Member Loans" {
      view MemberLoansView {
        fields {
          loanId string required
        }
      }
    }

    slice "Chase Overdue Copy" {
      command RecallCopy {
        fields {
          loanId string required
        }
      }

      automation RecallOverdueCopy {
        on CopyBorrowed
        reads MemberLoansView
        command RecallCopy
      }

      automation RemindMember {
        on CopyBorrowed
        command RecallCopy
      }
    }
  }
}
`
			require.Equal(t, source, formatter.Format(importFrom(t, source)))
		})

		t.Run("preserves the cadence a scheduled automation runs on, beside the event the automation under it activates on", func(t *testing.T) {
			source := `emod 1
model "M"

context "C" {
  aggregate "A" {
    slice "Chase Overdue Copy" {
      command RemindMember {
        fields {
          loanId string required
        }
      }

      event MemberReminded {
        fields {
          loanId string required
        }
      }

      automation RemindMemberEachMorning {
        every "0 9 * * *"
        command RemindMember
      }

      automation RecallOnSecondReminder {
        on MemberReminded
        command RemindMember
      }

      flow {
        command -> event: RemindMember -> MemberReminded
      }
    }
  }
}
`
			require.Equal(t, source, formatter.Format(importFrom(t, source)))
		})

		t.Run("keeps the cadence every scheduled automation runs on and the event the rest activate on, from both slice homes", func(t *testing.T) {
			scheduled := test.AutomationScheduleLibraryLendingModel(t)

			require.Equal(t, test.AutomationScheduleLibraryLendingSchedules, test.DeclaredSchedules(scheduled),
				"the model has to run on a schedule in both slice homes, or the comparisons below run over a copy of itself")
			require.Equal(t, test.AutomationScheduleLibraryLendingActivationEvents, test.DeclaredActivationEvents(scheduled),
				"the model has to activate on an event beside those cadences, or the comparisons below say nothing about the two forms travelling together")

			imported := importExported(t, scheduled)

			require.Equal(t, test.AutomationScheduleLibraryLendingSchedules, test.DeclaredSchedules(imported))
			require.Equal(t, test.AutomationScheduleLibraryLendingActivationEvents, test.DeclaredActivationEvents(imported))

			carried := test.AutomationScheduleLibraryLendingModel(t)
			stripWhatDiagramJSONDrops(carried)
			require.Equal(t, formatter.Format(carried), formatter.Format(imported))
		})

		t.Run("keeps the view every automation reads and the event each activates on, from both slice homes", func(t *testing.T) {
			reading := test.AutomationReadsLibraryLendingModel(t)
			unread := test.WithoutAutomationReads(reading)

			require.Equal(t, test.AutomationReadsLibraryLendingViewNames, test.DeclaredAutomationReads(reading),
				"the model has to keep reading a view in both slice homes once the twin is taken, or the comparisons below run over a copy of itself")
			require.Equal(t, test.AutomationReadsLibraryLendingActivationEvents, test.DeclaredActivationEvents(reading),
				"the model has to activate on an event in both slice homes, or the comparisons below run over a copy of itself")
			require.Empty(t, test.DeclaredAutomationReads(unread),
				"the twin has to lose the reads of both slice homes, or whichever home it kept answers the comparisons below")

			imported := importExported(t, reading)

			require.Equal(t, test.AutomationReadsLibraryLendingViewNames, test.DeclaredAutomationReads(imported))
			require.Equal(t, test.AutomationReadsLibraryLendingActivationEvents, test.DeclaredActivationEvents(imported))
			require.Empty(t, test.DeclaredAutomationReads(importExported(t, unread)))
		})

		t.Run("keeps the view every trigger and every automation reads, from both slice homes and across contexts", func(t *testing.T) {
			reading := test.TriggerReadsLibraryLendingModel(t)
			unreadTriggers := test.WithoutTriggerReads(reading)
			unreadAutomations := test.WithoutAutomationReads(reading)

			require.Equal(t, test.TriggerReadsLibraryLendingTriggerViewNames, test.DeclaredTriggerReads(reading),
				"the model has to keep its triggers reading a view in both slice homes once the twins are taken, or the comparisons below run over a copy of itself")
			require.Equal(t, test.TriggerReadsLibraryLendingAutomationViewNames, test.DeclaredAutomationReads(reading),
				"the model has to keep its automations reading a view in both slice homes once the twins are taken, or the comparisons below run over a copy of itself")
			require.Empty(t, test.DeclaredTriggerReads(unreadTriggers),
				"the trigger twin has to lose the reads of both slice homes, or whichever home it kept answers the comparison below")
			require.Empty(t, test.DeclaredAutomationReads(unreadAutomations),
				"the automation twin has to lose the reads of both slice homes, or whichever home it kept answers the comparison below")

			imported := importExported(t, reading)

			require.Equal(t, test.TriggerReadsLibraryLendingTriggerViewNames, test.DeclaredTriggerReads(imported))
			require.Equal(t, test.TriggerReadsLibraryLendingAutomationViewNames, test.DeclaredAutomationReads(imported))
			require.Empty(t, test.DeclaredTriggerReads(importExported(t, unreadTriggers)))
			require.Empty(t, test.DeclaredAutomationReads(importExported(t, unreadAutomations)))

			carried := test.TriggerReadsLibraryLendingModel(t)
			stripWhatDiagramJSONDrops(carried)
			require.Equal(t, formatter.Format(carried), formatter.Format(imported))
		})

		t.Run("preserves the view a trigger and an automation read from a sibling slice, leaving the pair beside them reading nothing", func(t *testing.T) {
			source := `emod 1
model "M"

context "C" {
  aggregate "A" {
    slice "Review Member Loans" {
      view MemberLoansView {
        fields {
          loanId string required
        }
      }
    }

    slice "Chase Overdue Copy" {
      trigger "Overdue Report" {
        actor Librarian
        reads MemberLoansView
      }

      command RecallCopy {
        fields {
          loanId string required
        }
      }

      automation RecallOverdueCopy {
        on CopyBorrowed
        reads MemberLoansView
        command RecallCopy
      }
    }

    slice "Return Copy" {
      trigger "Returns Counter" {
        actor Member
      }

      command ReturnCopy {
        fields {
          loanId string required
        }
      }

      automation RemindMember {
        on CopyBorrowed
        command ReturnCopy
      }
    }
  }
}
`
			require.Equal(t, source, formatter.Format(importFrom(t, source)))
		})

		t.Run("preserves a trigger's name, actor and reads through the viewer save path", func(t *testing.T) {
			source := `emod 1
model "M"

context "C" {
  aggregate "A" {
    slice "S" {
      trigger "Reservation Form" {
        actor Guest
        reads AvailableRoomsView
      }
    }
  }
}
`
			require.Equal(t, source, formatter.Format(importFrom(t, source)))
		})

		t.Run("ignores a stale kind key on a trigger node", func(t *testing.T) {
			withoutKind := `{
              "model_name": "M",
              "nodes": [
                {"id": "context-1", "type": "context", "label": "C", "parentId": null},
                {"id": "slice-1", "type": "slice", "label": "S", "parentId": "context-1"},
                {"id": "trigger-1", "type": "trigger", "label": "Form", "parentId": "slice-1",
                 "actor": "Guest", "reads": "MyView"}
              ],
              "edges": []
            }`
			withKind := `{
              "model_name": "M",
              "nodes": [
                {"id": "context-1", "type": "context", "label": "C", "parentId": null},
                {"id": "slice-1", "type": "slice", "label": "S", "parentId": "context-1"},
                {"id": "trigger-1", "type": "trigger", "label": "Form", "parentId": "slice-1",
                 "kind": "UI", "actor": "Guest", "reads": "MyView"}
              ],
              "edges": []
            }`

			a := importDiagram(t, withoutKind)
			b := importDiagram(t, withKind)
			require.Equal(t, a.Contexts[0].Slices[0].Trigger, b.Contexts[0].Slices[0].Trigger)
		})

		t.Run("re-emits the description every construct states, and writes none for the twin that states none", func(t *testing.T) {
			require.Equal(t, test.DescribedHotelReservationDescriptions,
				test.DeclaredDescriptions(test.DescribedHotelReservationModel(t)),
				"the fixture has to describe every construct, or the round trips below say nothing about prose")
			require.Empty(t, test.DeclaredDescriptions(test.HotelReservationModel(t)),
				"and its twin has to describe none, or they cannot show prose being invented")

			requireSaveKeepsAllButItsLosses(t, test.DescribedHotelReservationModel)
			requireSaveKeepsAllButItsLosses(t, test.HotelReservationModel)
		})

		t.Run("preserves external event sources", func(t *testing.T) {
			source := `emod 1
model "M"

context "C" {
  aggregate "A" {
    slice "S" {
      event PaymentSettled {
        source external "Stripe"
        fields {
          id string required
        }
      }
    }
  }
}
`
			require.Equal(t, source, formatter.Format(importFrom(t, source)))
		})
	})

	t.Run("node metadata", func(t *testing.T) {
		t.Run("an automation activates on the event stated under on_event and on none stated under trigger_event", func(t *testing.T) {
			documentKeying := automationNodeKeying("RecallOverdueCopy", "CopyBorrowed")

			keyedOnEvent := importDiagram(t, documentKeying("on_event"))
			require.Equal(t, "CopyBorrowed", keyedOnEvent.Contexts[0].Slices[0].Automations[0].OnEvent)

			keyedTriggerEvent := importDiagram(t, documentKeying("trigger_event"))
			require.Empty(t, keyedTriggerEvent.Contexts[0].Slices[0].Automations[0].OnEvent,
				"a document keyed by anything but on_event states no activation event, or the assertion above passes on a reader that takes both")
		})

		t.Run("an automation runs on the cadence stated under every and on none stated under schedule", func(t *testing.T) {
			documentKeying := automationNodeKeying("SweepOverdueLoans", "15m")

			keyedEvery := importDiagram(t, documentKeying("every"))
			require.Equal(t, "15m", keyedEvery.Contexts[0].Slices[0].Automations[0].Schedule)

			keyedSchedule := importDiagram(t, documentKeying("schedule"))
			require.Empty(t, keyedSchedule.Contexts[0].Slices[0].Automations[0].Schedule,
				"a document keyed by anything but every states no cadence, or the assertion above passes on a reader that takes both")
		})
	})

	t.Run("edges", func(t *testing.T) {
		t.Run("a flow edge drawn in the viewer becomes a flow entry", func(t *testing.T) {
			diagram := `{
              "model_name": "M",
              "nodes": [
                {"id": "context-1", "type": "context", "label": "C", "parentId": null},
                {"id": "slice-1", "type": "slice", "label": "S", "parentId": "context-1"},
                {"id": "command-1", "type": "command", "label": "DoThing", "parentId": "slice-1"},
                {"id": "event-1", "type": "event", "label": "ThingDone", "parentId": "slice-1"}
              ],
              "edges": [{"source": "command-1", "target": "event-1", "type": "flow"}]
            }`

			model := importDiagram(t, diagram)

			require.Equal(t,
				[]*ast.Flow{{CommandName: "DoThing", EventName: "ThingDone"}},
				model.Contexts[0].Slices[0].Flows)
		})

		t.Run("a subscription edge drawn in the viewer becomes a subscribes entry", func(t *testing.T) {
			diagram := `{
              "model_name": "M",
              "nodes": [
                {"id": "context-1", "type": "context", "label": "C", "parentId": null},
                {"id": "slice-1", "type": "slice", "label": "S", "parentId": "context-1"},
                {"id": "event-1", "type": "event", "label": "ThingDone", "parentId": "slice-1"},
                {"id": "view-1", "type": "view", "label": "ThingsView", "parentId": "slice-1"}
              ],
              "edges": [{"source": "event-1", "target": "view-1", "type": "subscription"}]
            }`

			model := importDiagram(t, diagram)

			require.Equal(t, []string{"ThingDone"}, model.Contexts[0].Slices[0].Views[0].Subscribes)
		})

		t.Run("an event to automation edge drawn in the viewer becomes the automation's activation event", func(t *testing.T) {
			diagram := `{
              "model_name": "M",
              "nodes": [
                {"id": "context-1", "type": "context", "label": "C", "parentId": null},
                {"id": "slice-1", "type": "slice", "label": "S", "parentId": "context-1"},
                {"id": "event-1", "type": "event", "label": "CopyBorrowed", "parentId": "slice-1"},
                {"id": "auto-1", "type": "automation", "label": "RecallOverdueCopy", "parentId": "slice-1"}
              ],
              "edges": [{"source": "event-1", "target": "auto-1", "type": "automation_trigger"}]
            }`

			model := importDiagram(t, diagram)

			require.Equal(t, "CopyBorrowed", model.Contexts[0].Slices[0].Automations[0].OnEvent)
		})

		t.Run("an event to automation edge drawn onto a scheduled automation leaves it on its cadence and unactivated", func(t *testing.T) {
			diagram := `{
              "model_name": "M",
              "nodes": [
                {"id": "context-1", "type": "context", "label": "C", "parentId": null},
                {"id": "slice-1", "type": "slice", "label": "S", "parentId": "context-1"},
                {"id": "event-1", "type": "event", "label": "CopyBorrowed", "parentId": "slice-1"},
                {"id": "command-1", "type": "command", "label": "RecallCopy", "parentId": "slice-1"},
                {"id": "auto-1", "type": "automation", "label": "SweepOverdueLoans", "parentId": "slice-1",
                 "every": "15m", "command": "RecallCopy"}
              ],
              "edges": [{"source": "event-1", "target": "auto-1", "type": "automation_trigger"}]
            }`

			model := importDiagram(t, diagram)

			require.Equal(t,
				&ast.Automation{Name: "SweepOverdueLoans", Schedule: "15m", Command: "RecallCopy"},
				model.Contexts[0].Slices[0].Automations[0])

			require.Empty(t, savedTextDiagnostics(model),
				"an automation stating a cadence and an activation event together is text emod rejects")
		})

		t.Run("a reads edge drawn in the viewer becomes the reads of the trigger, the automation and the translation it points at", func(t *testing.T) {
			diagram := `{
              "model_name": "M",
              "nodes": [
                {"id": "context-1", "type": "context", "label": "C", "parentId": null},
                {"id": "slice-1", "type": "slice", "label": "S", "parentId": "context-1"},
                {"id": "view-1", "type": "view", "label": "MemberLoansView", "parentId": "slice-1"},
                {"id": "trigger-1", "type": "trigger", "label": "Overdue Report", "parentId": "slice-1"},
                {"id": "auto-1", "type": "automation", "label": "RecallOverdueCopy", "parentId": "slice-1"},
                {"id": "trans-1", "type": "translation", "label": "AcknowledgeOverdueNotice", "parentId": "slice-1"}
              ],
              "edges": [
                {"source": "view-1", "target": "trigger-1", "type": "reads"},
                {"source": "view-1", "target": "auto-1", "type": "reads"},
                {"source": "view-1", "target": "trans-1", "type": "reads"}
              ]
            }`

			model := importDiagram(t, diagram)

			slice := model.Contexts[0].Slices[0]
			require.Equal(t, "MemberLoansView", slice.Trigger.Reads)
			require.Equal(t, "MemberLoansView", slice.Automations[0].Reads)
			require.Equal(t, "MemberLoansView", slice.Translations[0].Reads)
		})

		t.Run("a reads edge naming another view leaves the trigger and the automation reading the one their node recorded", func(t *testing.T) {
			diagram := `{
              "model_name": "M",
              "nodes": [
                {"id": "context-1", "type": "context", "label": "C", "parentId": null},
                {"id": "slice-1", "type": "slice", "label": "S", "parentId": "context-1"},
                {"id": "view-1", "type": "view", "label": "MemberLoansView", "parentId": "slice-1"},
                {"id": "view-2", "type": "view", "label": "DeskOccupancyView", "parentId": "slice-1"},
                {"id": "trigger-1", "type": "trigger", "label": "Overdue Report", "parentId": "slice-1",
                 "reads": "MemberLoansView"},
                {"id": "auto-1", "type": "automation", "label": "RecallOverdueCopy", "parentId": "slice-1",
                 "reads": "MemberLoansView"}
              ],
              "edges": [
                {"source": "view-2", "target": "trigger-1", "type": "reads"},
                {"source": "view-2", "target": "auto-1", "type": "reads"}
              ]
            }`

			model := importDiagram(t, diagram)

			slice := model.Contexts[0].Slices[0]
			require.Equal(t, "MemberLoansView", slice.Trigger.Reads)
			require.Equal(t, "MemberLoansView", slice.Automations[0].Reads)
		})

		t.Run("a reads edge dragged off the view onto a command or an event leaves the trigger and the automation reading nothing", func(t *testing.T) {
			diagram := `{
              "model_name": "M",
              "nodes": [
                {"id": "context-1", "type": "context", "label": "C", "parentId": null},
                {"id": "slice-1", "type": "slice", "label": "Chase Overdue Copy", "parentId": "context-1"},
                {"id": "view-1", "type": "view", "label": "MemberLoansView", "parentId": "slice-1"},
                {"id": "command-1", "type": "command", "label": "RecallCopy", "parentId": "slice-1"},
                {"id": "event-1", "type": "event", "label": "CopyRecalled", "parentId": "slice-1"},
                {"id": "trigger-1", "type": "trigger", "label": "Overdue Report", "parentId": "slice-1"},
                {"id": "auto-1", "type": "automation", "label": "RecallOverdueCopy", "parentId": "slice-1"},
                {"id": "slice-2", "type": "slice", "label": "Borrow Copy", "parentId": "context-1"},
                {"id": "trigger-2", "type": "trigger", "label": "Lending Desk", "parentId": "slice-2"},
                {"id": "auto-2", "type": "automation", "label": "RemindOnDueDate", "parentId": "slice-2"}
              ],
              "edges": [
                {"source": "command-1", "target": "event-1", "type": "flow"},
                {"source": "event-1", "target": "trigger-1", "type": "reads"},
                {"source": "command-1", "target": "auto-1", "type": "reads"},
                {"source": "view-1", "target": "trigger-2", "type": "reads"},
                {"source": "view-1", "target": "auto-2", "type": "reads"}
              ]
            }`

			model := importDiagram(t, diagram)

			repointed, drawnFromTheView := model.Contexts[0].Slices[0], model.Contexts[0].Slices[1]
			require.Equal(t, "MemberLoansView", drawnFromTheView.Trigger.Reads)
			require.Equal(t, "MemberLoansView", drawnFromTheView.Automations[0].Reads)
			require.Empty(t, repointed.Trigger.Reads)
			require.Empty(t, repointed.Automations[0].Reads)

			require.Empty(t, savedModelDiagnostics(model),
				"an automation reading a command rather than a view is text emod validate rejects")
		})

		t.Run("an edge already recorded in node metadata is not duplicated", func(t *testing.T) {
			diagram := `{
              "model_name": "M",
              "nodes": [
                {"id": "context-1", "type": "context", "label": "C", "parentId": null},
                {"id": "slice-1", "type": "slice", "label": "S", "parentId": "context-1"},
                {"id": "event-1", "type": "event", "label": "ThingDone", "parentId": "slice-1"},
                {"id": "view-1", "type": "view", "label": "ThingsView", "parentId": "slice-1",
                 "subscribes": ["ThingDone"]}
              ],
              "edges": [{"source": "event-1", "target": "view-1", "type": "subscription"}]
            }`

			model := importDiagram(t, diagram)

			require.Equal(t, []string{"ThingDone"}, model.Contexts[0].Slices[0].Views[0].Subscribes)
		})

		t.Run("an edge referencing a deleted node is ignored", func(t *testing.T) {
			diagram := `{
              "model_name": "M",
              "nodes": [
                {"id": "context-1", "type": "context", "label": "C", "parentId": null},
                {"id": "slice-1", "type": "slice", "label": "S", "parentId": "context-1"},
                {"id": "command-1", "type": "command", "label": "DoThing", "parentId": "slice-1"}
              ],
              "edges": [{"source": "command-1", "target": "event-gone", "type": "flow"}]
            }`

			model := importDiagram(t, diagram)

			require.Empty(t, model.Contexts[0].Slices[0].Flows)
		})

		t.Run("a flow edge crossing slice boundaries is ignored", func(t *testing.T) {
			diagram := `{
              "model_name": "M",
              "nodes": [
                {"id": "context-1", "type": "context", "label": "C", "parentId": null},
                {"id": "slice-1", "type": "slice", "label": "S1", "parentId": "context-1"},
                {"id": "slice-2", "type": "slice", "label": "S2", "parentId": "context-1"},
                {"id": "command-1", "type": "command", "label": "DoThing", "parentId": "slice-1"},
                {"id": "event-1", "type": "event", "label": "ThingDone", "parentId": "slice-2"}
              ],
              "edges": [{"source": "command-1", "target": "event-1", "type": "flow"}]
            }`

			model := importDiagram(t, diagram)

			require.Empty(t, model.Contexts[0].Slices[0].Flows)
			require.Empty(t, model.Contexts[0].Slices[1].Flows)
		})
	})

	t.Run("malformed input", func(t *testing.T) {
		t.Run("invalid JSON reports an error", func(t *testing.T) {
			_, err := importer.ImportDiagram([]byte("{not json"))

			require.Error(t, err)
			require.Contains(t, err.Error(), "invalid diagram JSON")
		})

		t.Run("a trigger node without kind formats to a parseable kindless header", func(t *testing.T) {
			diagram := `{
              "model_name": "M",
              "nodes": [
                {"id": "context-1", "type": "context", "label": "C", "parentId": null},
                {"id": "slice-1", "type": "slice", "label": "S", "parentId": "context-1"},
                {"id": "trigger-1", "type": "trigger", "label": "Form", "parentId": "slice-1"}
              ],
              "edges": []
            }`

			model := importDiagram(t, diagram)
			formatted := formatter.Format(model)

			require.Contains(t, formatted, "trigger \"Form\" {")
			require.Empty(t, savedTextDiagnostics(model))
		})

		t.Run("an empty document yields a model with no contexts", func(t *testing.T) {
			model := importDiagram(t, `{"model_name":"Empty","nodes":[],"edges":[]}`)

			require.Equal(t, &ast.Model{Name: "Empty"}, model)
		})
	})
}

// stripWhatDiagramJSONDrops removes what ImportDiagram states a diagram document
// does not carry — comments, context modes, event tags and decides_on clauses —
// so what remains is the most a round trip through that document can reproduce.
func stripWhatDiagramJSONDrops(model *ast.Model) {
	stripComments(model)
	for _, c := range model.Contexts {
		c.Mode = ""
	}
	forEachSlice(model, stripSliceDecisionMetadata)
}

// stripWhatASaveStillLoses removes, beyond what a diagram document was designed
// to leave behind, two things a save drops that it was not: the model's own
// description, which the document files under no node, and a flow naming an event
// the slice does not itself declare, which no edge between two of that slice's
// nodes stands for. They are losses rather than choices, so they stay out of
// stripWhatDiagramJSONDrops — a model keeping its flows within one slice must
// still round-trip every one of them.
func stripWhatASaveStillLoses(model *ast.Model) {
	stripWhatDiagramJSONDrops(model)
	model.Description = ""
	forEachSlice(model, stripFlowsLeavingTheSlice)
}

// requireSaveKeepsAllButItsLosses requires that a viewer save of the fixture —
// export to a diagram document, import it back, format — rewrites the file the
// fixture states, minus what a save loses. The fixture is read twice because the
// strip rewrites the model it is handed, and the round trip has to run on one no
// strip has touched.
func requireSaveKeepsAllButItsLosses(t *testing.T, fixture func(*testing.T) *ast.Model) {
	t.Helper()

	carried := fixture(t)
	stripWhatASaveStillLoses(carried)

	require.Equal(t, formatter.Format(carried), formatter.Format(importExported(t, fixture(t))))
}

func stripFlowsLeavingTheSlice(s *ast.Slice) {
	declared := make(map[string]bool)
	for _, e := range s.Events {
		declared[e.Name] = true
	}

	var kept []*ast.Flow
	for _, f := range s.Flows {
		if declared[f.EventName] {
			kept = append(kept, f)
		}
	}
	s.Flows = kept
}

// forEachSlice visits both homes a slice has — nested in an aggregate and
// declared directly on a context — so a strip reaching only one of them leaves
// the other stating what a round trip cannot reproduce.
func forEachSlice(model *ast.Model, visit func(*ast.Slice)) {
	for _, c := range model.Contexts {
		for _, agg := range c.Aggregates {
			for _, s := range agg.Slices {
				visit(s)
			}
		}
		for _, s := range c.Slices {
			visit(s)
		}
	}
}

func stripSliceDecisionMetadata(s *ast.Slice) {
	for _, c := range s.Commands {
		c.DecidesOn = nil
	}
	for _, e := range s.Events {
		e.Tags = nil
	}
	for _, tr := range s.Translations {
		if tr.Event != nil {
			tr.Event.Tags = nil
		}
	}
}

func stripComments(model *ast.Model) {
	model.Comments = nil
	for _, a := range model.Actors {
		a.Comments = nil
	}
	for _, c := range model.Contexts {
		c.Comments = nil
		for _, agg := range c.Aggregates {
			agg.Comments = nil
		}
	}
	forEachSlice(model, stripSliceComments)
}

func stripSliceComments(s *ast.Slice) {
	s.Comments = nil
	if s.Trigger != nil {
		s.Trigger.Comments = nil
	}
	for _, c := range s.Commands {
		c.Comments = nil
	}
	for _, e := range s.Events {
		e.Comments = nil
	}
	for _, v := range s.Views {
		v.Comments = nil
	}
	for _, a := range s.Automations {
		a.Comments = nil
	}
	for _, tr := range s.Translations {
		tr.Comments = nil
		if tr.Event != nil {
			tr.Event.Comments = nil
		}
	}
	for _, f := range s.Flows {
		f.Comments = nil
	}
}

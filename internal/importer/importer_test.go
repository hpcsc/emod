//go:build unit

package importer_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/export"
	"github.com/hpcsc/emod/internal/formatter"
	"github.com/hpcsc/emod/internal/importer"
	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/parser"
	"github.com/hpcsc/emod/internal/test"
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
			documentKeying := func(activationEventKey string) string {
				return fmt.Sprintf(`{
              "model_name": "M",
              "nodes": [
                {"id": "context-1", "type": "context", "label": "C", "parentId": null},
                {"id": "slice-1", "type": "slice", "label": "S", "parentId": "context-1"},
                {"id": "auto-1", "type": "automation", "label": "RecallOverdueCopy", "parentId": "slice-1",
                 %q: "CopyBorrowed"}
              ],
              "edges": []
            }`, activationEventKey)
			}

			keyedOnEvent := importDiagram(t, documentKeying("on_event"))
			require.Equal(t, "CopyBorrowed", keyedOnEvent.Contexts[0].Slices[0].Automations[0].OnEvent)

			keyedTriggerEvent := importDiagram(t, documentKeying("trigger_event"))
			require.Empty(t, keyedTriggerEvent.Contexts[0].Slices[0].Automations[0].OnEvent,
				"a document keyed by anything but on_event states no activation event, or the assertion above passes on a reader that takes both")
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

		t.Run("a reads edge drawn in the viewer becomes a translation's reads and not an automation's", func(t *testing.T) {
			diagram := `{
              "model_name": "M",
              "nodes": [
                {"id": "context-1", "type": "context", "label": "C", "parentId": null},
                {"id": "slice-1", "type": "slice", "label": "S", "parentId": "context-1"},
                {"id": "view-1", "type": "view", "label": "MemberLoansView", "parentId": "slice-1"},
                {"id": "auto-1", "type": "automation", "label": "RecallOverdueCopy", "parentId": "slice-1"},
                {"id": "trans-1", "type": "translation", "label": "AcknowledgeOverdueNotice", "parentId": "slice-1"}
              ],
              "edges": [
                {"source": "view-1", "target": "auto-1", "type": "reads"},
                {"source": "view-1", "target": "trans-1", "type": "reads"}
              ]
            }`

			model := importDiagram(t, diagram)

			slice := model.Contexts[0].Slices[0]
			require.Equal(t, "MemberLoansView", slice.Translations[0].Reads)
			require.Empty(t, slice.Automations[0].Reads,
				"US-005 owns the view→automation edge; folding one back here is that story arriving early, not a regression")
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

		t.Run("a trigger with no kind falls back to a parseable one", func(t *testing.T) {
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

			require.Equal(t, "UI", model.Contexts[0].Slices[0].Trigger.Kind)
		})

		t.Run("an empty document yields a model with no contexts", func(t *testing.T) {
			model := importDiagram(t, `{"model_name":"Empty","nodes":[],"edges":[]}`)

			require.Equal(t, &ast.Model{Name: "Empty"}, model)
		})
	})
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
			for _, s := range agg.Slices {
				stripSliceComments(s)
			}
		}
		for _, s := range c.Slices {
			stripSliceComments(s)
		}
	}
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

//go:build unit

package oracle_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/oracle"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
)

// Check runs the whole pipeline, so it uses the pipeline-wide fixtures.
const (
	validEmod   = test.HotelReservation
	invalidEmod = test.Unparseable
)

func TestCheck(t *testing.T) {
	t.Run("clean input", func(t *testing.T) {
		t.Run("returns an empty diagnostic list for a fully valid model", func(t *testing.T) {
			diagnostics := oracle.Check(validEmod, "valid.emod")

			require.Empty(t, diagnostics)
		})

		t.Run("returns an empty diagnostic list for a model that describes every construct", func(t *testing.T) {
			diagnostics := oracle.Check(test.DescribedHotelReservation, "described.emod")

			require.Empty(t, diagnostics)
		})

		t.Run("returns an empty diagnostic list for a model naming its fields after keywords", func(t *testing.T) {
			diagnostics := oracle.Check(test.KeywordFieldSearchCatalog, "keyword-fields.emod")

			require.Empty(t, diagnostics)
		})

		t.Run("reports no errors for a model naming a field after every keyword", func(t *testing.T) {
			keywords := lexer.Keywords()
			require.NotEmpty(t, keywords)

			diagnostics := oracle.Check(modelWithFieldPerKeyword(keywords), "keywords.emod")

			require.Empty(t, errorsIn(diagnostics))
		})

		t.Run("returns an empty diagnostic list for a model stating specs in an aggregate slice and on a context slice", func(t *testing.T) {
			diagnostics := oracle.Check(test.SpecLibraryLending, "specs.emod")

			require.Empty(t, diagnostics)
		})

		t.Run("returns an empty diagnostic list for a model stating example payloads in an aggregate slice and on a context slice", func(t *testing.T) {
			diagnostics := oracle.Check(test.PayloadLibraryLending, "payloads.emod")

			require.Empty(t, diagnostics)
		})

		t.Run("returns an empty diagnostic list for a model stating rejection edges in an aggregate slice and on a context slice", func(t *testing.T) {
			diagnostics := oracle.Check(test.RejectionLibraryLending, "rejections.emod")

			require.Empty(t, diagnostics)
		})

		t.Run("returns an empty diagnostic list for the model stating every construct this batch adds", func(t *testing.T) {
			diagnostics := oracle.Check(test.EveryConstructLibraryLending, "every-construct.emod")

			require.Empty(t, diagnostics)
		})

		t.Run("returns an empty diagnostic list for a model stating a spec for every slice pattern in both homes", func(t *testing.T) {
			diagnostics := oracle.Check(test.SlicePatternLibraryLending, "slice-patterns.emod")

			require.Empty(t, diagnostics)
		})

		t.Run("returns an empty diagnostic list for a model firing automations after a delay in an aggregate slice and on a context slice", func(t *testing.T) {
			diagnostics := oracle.Check(test.AutomationDelayLibraryLending, "automation-delay.emod")

			require.Empty(t, reportedLines(diagnostics))
		})

		t.Run("returns an empty diagnostic list for a model binding wire types in an aggregate slice and on a context slice", func(t *testing.T) {
			diagnostics := oracle.Check(test.WireTypeLibraryLending, "wire-types.emod")

			require.Empty(t, reportedLines(diagnostics))
		})
	})

	t.Run("documented models", func(t *testing.T) {
		for _, document := range []string{"README.md", "docs/dsl-reference.md"} {
			t.Run(document, func(t *testing.T) {
				blocks := emodBlocksIn(t, document)
				require.NotEmpty(t, blocks, "%s fences no emod block", document)

				for _, block := range blocks {
					t.Run(fmt.Sprintf("reports nothing about the model at line %d", block.line), func(t *testing.T) {
						origin := fmt.Sprintf("%s:%d", document, block.line)

						diagnostics := oracle.Check(block.source, origin)

						require.Empty(t, reportedLines(diagnostics))
					})
				}
			})
		}
	})

	t.Run("automations reading no view", func(t *testing.T) {
		t.Run("names the one automation a model of reading automations leaves without a view", func(t *testing.T) {
			diagnostics := oracle.Check(test.AutomationReadsLibraryLending, "automation-reads.emod")

			require.Equal(t, []string{
				`automation-reads.emod:69: [automation/missing-todo-list] automation "RemindOnDueDate" reads no view, so nothing in the model shows what work is outstanding; project a view of pending work and read it`,
			}, reportedLines(diagnostics))
		})

		t.Run("names the one automation a model of reading triggers leaves without a view", func(t *testing.T) {
			diagnostics := oracle.Check(test.TriggerReadsLibraryLending, "trigger-reads.emod")

			require.Equal(t, []string{
				`trigger-reads.emod:95: [automation/missing-todo-list] automation "RemindOnDueDate" reads no view, so nothing in the model shows what work is outstanding; project a view of pending work and read it`,
			}, reportedLines(diagnostics))
		})

		t.Run("names every automation a model of scheduled automations leaves without a view, in both slice homes", func(t *testing.T) {
			diagnostics := oracle.Check(test.AutomationScheduleLibraryLending, "automation-schedule.emod")

			require.Equal(t, []string{
				`automation-schedule.emod:74: [automation/missing-todo-list] automation "RecallOnSecondReminder" reads no view, so nothing in the model shows what work is outstanding; project a view of pending work and read it`,
				`automation-schedule.emod:78: [automation/missing-todo-list] automation "SweepOverdueLoans" reads no view, so the model does not state what the processor acts on; project a view of pending work and read it`,
				`automation-schedule.emod:161: [automation/missing-todo-list] automation "SweepIdleDesks" reads no view, so the model does not state what the processor acts on; project a view of pending work and read it`,
			}, reportedLines(diagnostics))
		})
	})

	t.Run("invariants never exercised", func(t *testing.T) {
		t.Run("names every invariant a model of lending declares without a rejection", func(t *testing.T) {
			diagnostics := oracle.Check(test.InvariantLibraryLending, "invariants.emod")

			require.Equal(t, []string{
				`invariants.emod:8: [spec/invariant-never-exercised] invariant "OneCopyPerLoan" in aggregate "Loan" is not referenced by any rejection`,
				`invariants.emod:33: [spec/invariant-never-exercised] invariant "FiveCopiesPerMember" in aggregate "Loan" is not referenced by any rejection`,
				`invariants.emod:48: [spec/invariant-never-exercised] invariant "OneReaderPerDesk" in context "Reading Room" is not referenced by any rejection`,
				`invariants.emod:49: [spec/invariant-never-exercised] invariant "OneDeskPerReader" in context "Reading Room" is not referenced by any rejection`,
				`invariants.emod:73: [spec/invariant-never-exercised] invariant "DeskFreeAtClosing" in context "Reading Room" is not referenced by any rejection`,
			}, reportedLines(diagnostics))
		})
	})

	t.Run("a misspelled view name", func(t *testing.T) {
		t.Run("is reported once, at the reads that spells it, and not again at the view it was meant to name", func(t *testing.T) {
			// The model exists to trip one rule. Its event carries a domain field
			// beside its identifier so clickbait-event stays quiet, and its command
			// is flowed so neither orphan check fires.
			const source = `model "Case Handling"

actor "Worker"

context "Cases" {
  aggregate "Case" {
    slice "Review Case" {
      trigger "Case Desk" {
        actor Worker
        reads CaseWorkspacveView
      }
      view CaseWorkspaceView {
        fields {
          caseId   string required
          openedBy string required
        }
        subscribes [CaseOpened]
      }
    }
    slice "Open Case" {
      command OpenCase {
        fields {
          caseId   string required
          openedBy string required
        }
      }
      event CaseOpened {
        fields {
          caseId   string    required
          openedBy string    required
          openedAt timestamp required
        }
      }
      flow {
        command -> event: OpenCase -> CaseOpened
      }
    }
  }
}
`

			diagnostics := oracle.Check(source, "cases.emod")

			require.Equal(t, []string{
				`cases.emod:10: view "CaseWorkspacveView" does not exist`,
			}, reportedLines(diagnostics))
		})
	})

	t.Run("unparseable input", func(t *testing.T) {
		t.Run("returns entries carrying position, severity, and message", func(t *testing.T) {
			diagnostics := oracle.Check(invalidEmod, "invalid.emod")

			require.NotEmpty(t, diagnostics)
			first := diagnostics[0]
			require.Equal(t, 1, first.Line)
			require.NotEmpty(t, first.Message)
			require.Equal(t, diagnostic.Error, first.Severity)
		})

		t.Run("propagates the passed-in filename into every entry", func(t *testing.T) {
			const filename = "my-model.emod"

			diagnostics := oracle.Check(invalidEmod, filename)

			require.NotEmpty(t, diagnostics)
			for _, d := range diagnostics {
				require.Equal(t, filename, d.Filename)
			}
		})
	})

	t.Run("unsupported version", func(t *testing.T) {
		t.Run("reports the version error alone, leaving the validator and linter silent", func(t *testing.T) {
			const source = `model "Test"
context "Orders" {
  aggregate "Order" {
    slice "Process Order" {
      command OrderPlaced {}
      event OrderUpdated {}
      view OrderList {}
      automation OrderNotifier {
        on OrderPlaced
        command NotifyCustomer
        target context NonExistent
      }
      flow {
        command -> event: OrderPlaced -> OrderUpdated
      }
    }
  }
}
`

			rejected := oracle.Check("emod 2\n"+source, "unsupported.emod")
			supported := oracle.Check("emod 1\n"+source, "supported.emod")

			require.NotNil(t, findMentioning(supported, "NonExistent"), "the same source under a supported version reports the missing context")
			for _, rule := range []string{"command-past-tense", "state-obsession", "view-naming"} {
				require.True(t, hasRule(supported, rule), "the same source under a supported version reports %s", rule)
			}

			require.Len(t, rejected, 1)
			require.Equal(t, 1, rejected[0].Line)
			require.Equal(t, diagnostic.Error, rejected[0].Severity)
			require.Empty(t, rejected[0].RuleName)
		})
	})

	t.Run("validator faults", func(t *testing.T) {
		t.Run("surface when a parseable model targets a nonexistent context", func(t *testing.T) {
			input := `model "Test"
context "Orders" {
  aggregate "Order" {
    slice "Process Order" {
      automation OrderNotifier {
        on OrderPlaced
        command NotifyCustomer
        target context NonExistent
      }
    }
  }
}
`

			diagnostics := oracle.Check(input, "bad_target.emod")

			found := findMentioning(diagnostics, "NonExistent")
			require.NotNil(t, found, "expected a validator diagnostic mentioning the missing context")
			require.Contains(t, found.Message, "does not exist")
		})
	})

	t.Run("linter faults", func(t *testing.T) {
		t.Run("surface on an otherwise valid model", func(t *testing.T) {
			input := `model "Test"
context "Test" {
  aggregate "Test" {
    slice "Test" {
      command OrderPlaced {}
      event OrderUpdated {}
      view OrderList {}
      flow {
        command -> event: OrderPlaced -> OrderUpdated
      }
    }
  }
}
`

			diagnostics := oracle.Check(input, "lint_only.emod")

			require.True(t, hasRule(diagnostics, "command-past-tense"), "expected command-past-tense rule")
			require.True(t, hasRule(diagnostics, "state-obsession"), "expected state-obsession rule")
			require.True(t, hasRule(diagnostics, "view-naming"), "expected view-naming rule")
		})
	})

	t.Run("combined faults", func(t *testing.T) {
		t.Run("surface both a validator error and linter warnings together", func(t *testing.T) {
			input := `model "Test"
context "Orders" {
  aggregate "Order" {
    slice "Process Order" {
      command OrderPlaced {}
      event OrderUpdated {}
      view OrderList {}
      automation OrderNotifier {
        on OrderPlaced
        command NotifyCustomer
        target context NonExistent
      }
      flow {
        command -> event: OrderPlaced -> OrderUpdated
      }
    }
  }
}
`

			diagnostics := oracle.Check(input, "combined.emod")

			found := findMentioning(diagnostics, "NonExistent")
			require.NotNil(t, found, "expected the validator diagnostic for the missing context")
			require.Contains(t, found.Message, "does not exist")
			for _, rule := range []string{"command-past-tense", "state-obsession", "view-naming"} {
				require.True(t, hasRule(diagnostics, rule), "expected %s alongside the validator error", rule)
			}
		})
	})

	t.Run("severity", func(t *testing.T) {
		t.Run("reports a single-id event at error severity", func(t *testing.T) {
			input := `model "Test"
context "Orders" {
  aggregate "Order" {
    slice "Events" {
      command PlaceOrder {
        fields {
          orderId string required
        }
      }
      event SingleIdEvent {
        fields {
          orderId string required
        }
      }
      flow {
        command -> event: PlaceOrder -> SingleIdEvent
      }
    }
  }
}
`

			diagnostics := oracle.Check(input, "errors.emod")

			found := findRule(diagnostics, "clickbait-event")
			require.NotNil(t, found, "expected a clickbait-event diagnostic")
			require.Equal(t, diagnostic.Error, found.Severity)
		})
	})
}

func modelWithFieldPerKeyword(keywords []string) string {
	var fields strings.Builder
	for _, keyword := range keywords {
		fmt.Fprintf(&fields, "          %s string required\n", keyword)
	}

	return fmt.Sprintf(`model "Keyword Fields"

context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      command DoThing {
        fields {
%s        }
      }
      event ThingDone {
        fields {
%s        }
      }
      flow {
        command -> event: DoThing -> ThingDone
      }
    }
  }
}
`, fields.String(), fields.String())
}

const (
	repositoryRoot = "../.."
	markdownFence  = "```"
	emodFence      = markdownFence + "emod"
)

type emodBlock struct {
	line   int
	source string
}

func emodBlocksIn(t *testing.T, document string) []emodBlock {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(repositoryRoot, document))
	require.NoError(t, err)

	lines := strings.Split(string(content), "\n")

	var blocks []emodBlock
	for opening := 0; opening < len(lines); opening++ {
		if strings.TrimSpace(lines[opening]) != emodFence {
			continue
		}

		closing := opening + 1
		for closing < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[closing]), markdownFence) {
			closing++
		}

		blocks = append(blocks, emodBlock{
			line:   opening + 1,
			source: strings.Join(lines[opening+1:closing], "\n") + "\n",
		})
		opening = closing
	}

	return blocks
}

func reportedLines(diagnostics []*diagnostic.Entry) []string {
	lines := make([]string, 0, len(diagnostics))
	for _, d := range diagnostics {
		lines = append(lines, d.String())
	}

	return lines
}

func errorsIn(diagnostics []*diagnostic.Entry) []*diagnostic.Entry {
	var errors []*diagnostic.Entry
	for _, d := range diagnostics {
		if d.Severity == diagnostic.Error {
			errors = append(errors, d)
		}
	}
	return errors
}

func findMentioning(diagnostics []*diagnostic.Entry, text string) *diagnostic.Entry {
	for _, d := range diagnostics {
		if strings.Contains(d.Message, text) {
			return d
		}
	}
	return nil
}

func findRule(diagnostics []*diagnostic.Entry, ruleName string) *diagnostic.Entry {
	for _, d := range diagnostics {
		if d.RuleName == ruleName {
			return d
		}
	}
	return nil
}

func hasRule(diagnostics []*diagnostic.Entry, ruleName string) bool {
	return findRule(diagnostics, ruleName) != nil
}

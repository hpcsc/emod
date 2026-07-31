//go:build unit

package oracle_test

import (
	"fmt"
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

		t.Run("returns an empty diagnostic list for a model declaring invariants on an aggregate and on a context", func(t *testing.T) {
			diagnostics := oracle.Check(test.InvariantLibraryLending, "invariants.emod")

			require.Empty(t, diagnostics)
		})

		t.Run("returns an empty diagnostic list for a model stating specs in an aggregate slice and on a context slice", func(t *testing.T) {
			diagnostics := oracle.Check(test.SpecLibraryLending, "specs.emod")

			require.Empty(t, diagnostics)
		})

		t.Run("returns an empty diagnostic list for a model whose automations read views in an aggregate slice, on a context slice and across a context boundary", func(t *testing.T) {
			diagnostics := oracle.Check(test.AutomationReadsLibraryLending, "automation-reads.emod")

			require.Empty(t, diagnostics)
		})

		t.Run("reports nothing about an invariant a context declares outside dcb mode", func(t *testing.T) {
			tests := []struct {
				mode   string
				clause string
			}{
				{mode: "mode aggregate", clause: " mode aggregate"},
				{mode: "no mode clause", clause: ""},
			}

			for _, tc := range tests {
				t.Run(tc.mode, func(t *testing.T) {
					source := fmt.Sprintf(`model "Library Lending"

context "Lending"%s {
  invariant FiveCopiesPerMember "A member holds at most five copies at one time"
  aggregate "Loan" {
    slice "Borrow Copy" {
      command BorrowCopy {
        fields {
          memberId string required
          copyId   string required
        }
      }
      event CopyBorrowed {
        fields {
          loanId   string required
          memberId string required
          copyId   string required
        }
      }
      flow {
        command -> event: BorrowCopy -> CopyBorrowed
      }
    }
  }
}
`, tc.clause)

					diagnostics := oracle.Check(source, "lending.emod")

					require.Empty(t, diagnostics)
				})
			}
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

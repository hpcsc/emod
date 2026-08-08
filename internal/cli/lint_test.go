//go:build unit

package cli_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/hpcsc/emod/internal/cli"
	"github.com/stretchr/testify/require"
)

// singleTagDCBEmod uses one tag key across every decides_on predicate, which
// the dcb/single-tag-everywhere rule reports at info severity — the only rule
// that produces info diagnostics.
const singleTagDCBEmod = `model "Orders"

context "Fulfillment" mode dcb {
  slice "Place Order" {
    command PlaceOrder {
      fields {
        customerId string required
        total      int    required
      }
    }

    event OrderPlaced {
      tags {
        entity: customerId
      }
      fields {
        orderId    string required
        customerId string required
        total      int    required
      }
    }

    flow {
      command -> event: PlaceOrder -> OrderPlaced
    }
  }

  slice "Authorize Payment" {
    command AuthorizePayment {
      decides_on {
        events [OrderPlaced]
        where tag(entity = customerId)
      }
      fields {
        authCode string required
      }
    }

    event PaymentAuthorized {
      tags {
        entity: customerId
      }
      fields {
        paymentId  string required
        customerId string required
        authCode   string required
      }
    }

    flow {
      command -> event: AuthorizePayment -> PaymentAuthorized
    }
  }
}
`

// singleTagWithClickbaitEmod adds a single-ID event to the same model, so the
// info diagnostic above is joined by a clickbait-event error.
const singleTagWithClickbaitEmod = `model "Orders"

context "Fulfillment" mode dcb {
  slice "Place Order" {
    command PlaceOrder {
      fields {
        customerId string required
        total      int    required
      }
    }

    event OrderPlaced {
      tags {
        entity: customerId
      }
      fields {
        orderId    string required
        customerId string required
        total      int    required
      }
    }

    flow {
      command -> event: PlaceOrder -> OrderPlaced
    }
  }

  slice "Authorize Payment" {
    command AuthorizePayment {
      decides_on {
        events [OrderPlaced]
        where tag(entity = customerId)
      }
      fields {
        authCode string required
      }
    }

    event PaymentAuthorized {
      tags {
        entity: paymentId
      }
      fields {
        paymentId string required
      }
    }

    flow {
      command -> event: AuthorizePayment -> PaymentAuthorized
    }
  }
}
`

// automationWithoutViewEmod wires an automation straight from an event to a
// command, which automation/missing-todo-list reports and no other rule does.
const automationWithoutViewEmod = `model "Lending"

context "Lending" {
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
    slice "Chase Overdue Copy" {
      command RemindMember {
        fields {
          loanId   string required
          memberId string required
        }
      }
      event MemberReminded {
        fields {
          loanId     string    required
          memberId   string    required
          remindedAt timestamp required
        }
      }
      automation RemindOnDueDate {
        on CopyBorrowed
        command RemindMember
      }
      flow {
        command -> event: RemindMember -> MemberReminded
      }
    }
  }
}
`

// commandWithoutSpecEmod states a spec for one command and leaves another
// uncovered, so spec/command-without-spec reports the uncovered command at info
// severity and no other rule fires.
const commandWithoutSpecEmod = `model "Lending"

context "Lending" {
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
      spec "borrows a copy no one holds" {
        when BorrowCopy
        then [CopyBorrowed]
      }
    }
    slice "Return Copy" {
      command ReturnCopy {
        fields {
          loanId string required
          copyId string required
        }
      }
      event CopyReturned {
        fields {
          loanId   string required
          copyId   string required
          returnedAt timestamp required
        }
      }
      flow {
        command -> event: ReturnCopy -> CopyReturned
      }
    }
  }
}
`

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	fn()

	err = w.Close()
	require.NoError(t, err)
	os.Stdout = old

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	return buf.String()
}

func TestLint(t *testing.T) {
	t.Run("clean file produces no error", func(t *testing.T) {
		path := writeTemp(t, "clean.emod", validEmod)

		err := cli.RunLint(path, "text")

		require.NoError(t, err)
	})

	t.Run("file with naming violations returns error with file path, line number, rule name, and explanation", func(t *testing.T) {
		input := `model "Test"
context "Orders" {
  aggregate "Order" {
    slice "Update Order" {
      command UpdateOrder {
        fields {
          orderId string required
        }
      }
      event OrderUpdated {
        fields {
          orderId string required
        }
      }
    }
  }
}
`
		path := writeTemp(t, "problematic.emod", input)

		err := cli.RunLint(path, "text")

		require.Error(t, err)
		require.Contains(t, err.Error(), path)
		require.Contains(t, err.Error(), ":10:")
		require.Contains(t, err.Error(), "state-obsession")
		require.Contains(t, err.Error(), "OrderUpdated")
	})

	t.Run("missing file argument returns error", func(t *testing.T) {
		err := cli.RunLint("", "text")

		require.ErrorIs(t, err, cli.ErrMissingFileArgument)
	})

	t.Run("nonexistent file returns error naming the file", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "nonexistent.emod")

		err := cli.RunLint(missing, "text")

		require.Error(t, err)
		require.Contains(t, err.Error(), missing)
	})

	t.Run("unparseable file returns error with file path and line number", func(t *testing.T) {
		path := writeTemp(t, "invalid.emod", invalidEmod)

		err := cli.RunLint(path, "text")

		require.Error(t, err)
		require.Contains(t, err.Error(), path)
		require.Contains(t, err.Error(), ":1:")
	})

	t.Run("multiple lint violations are all reported", func(t *testing.T) {
		input := `model "Test"
context "Orders" {
  aggregate "Order" {
    slice "Bad Events" {
      command UpdateOrder {
        fields {
          orderId string required
        }
      }
      event OrderUpdated {
        fields {
          orderId string required
        }
      }
      event PaymentInitiated {
        fields {
          paymentId string required
        }
      }
    }
  }
}
`
		path := writeTemp(t, "multiple.emod", input)

		err := cli.RunLint(path, "text")

		require.Error(t, err)
		require.Contains(t, err.Error(), "state-obsession")
		require.Contains(t, err.Error(), "command-in-disguise")
	})

	t.Run("json format on clean file outputs empty array", func(t *testing.T) {
		path := writeTemp(t, "clean.emod", validEmod)

		output := captureStdout(t, func() {
			err := cli.RunLint(path, "json")
			require.NoError(t, err)
		})

		require.Equal(t, "[]\n", output)
	})

	t.Run("json format on warning-only file outputs warning severity and exit code 1", func(t *testing.T) {
		input := `model "Test"
context "Orders" {
  aggregate "Order" {
    slice "Update Order" {
      command PlaceOrder {
        fields {
          orderId string required
          reason  string required
        }
      }
      event OrderUpdated {
        fields {
          orderId string required
          reason  string required
        }
      }
      flow {
        command -> event: PlaceOrder -> OrderUpdated
      }
    }
  }
}
`
		path := writeTemp(t, "warnings.emod", input)

		var output string
		output = captureStdout(t, func() {
			err := cli.RunLint(path, "json")
			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 1, lintErr.ExitCode)
			require.Empty(t, lintErr.Message, "json output carries the diagnostics; the error only carries the exit code")
		})

		var entries []map[string]interface{}
		err := json.Unmarshal([]byte(output), &entries)
		require.NoError(t, err)
		require.Len(t, entries, 1)
		require.Equal(t, "warning", entries[0]["severity"])
		require.Equal(t, "state-obsession", entries[0]["rule"])
	})

	t.Run("json format on error-only file outputs error severity and exit code 2", func(t *testing.T) {
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
		path := writeTemp(t, "errors.emod", input)

		var output string
		output = captureStdout(t, func() {
			err := cli.RunLint(path, "json")
			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 2, lintErr.ExitCode)
			require.Empty(t, lintErr.Message, "json output carries the diagnostics; the error only carries the exit code")
		})

		var entries []map[string]interface{}
		err := json.Unmarshal([]byte(output), &entries)
		require.NoError(t, err)
		require.Len(t, entries, 1)
		require.Equal(t, "error", entries[0]["severity"])
		require.Equal(t, "clickbait-event", entries[0]["rule"])
	})

	t.Run("json format on mixed warnings and errors outputs both severities and exit code 2", func(t *testing.T) {
		input := `model "Test"
context "Orders" {
  aggregate "Order" {
    slice "Bad Events" {
      command UpdateOrder {
        fields {
          orderId string required
        }
      }
      event OrderUpdated {
        fields {
          orderId string required
        }
      }
    }
  }
}
`
		path := writeTemp(t, "mixed.emod", input)

		var output string
		output = captureStdout(t, func() {
			err := cli.RunLint(path, "json")
			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 2, lintErr.ExitCode)
			require.Empty(t, lintErr.Message, "json output carries the diagnostics; the error only carries the exit code")
		})

		var entries []map[string]interface{}
		err := json.Unmarshal([]byte(output), &entries)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(entries), 2)

		hasWarning := false
		hasError := false
		for _, e := range entries {
			sev, _ := e["severity"].(string)
			if sev == "warning" {
				hasWarning = true
			}
			if sev == "error" {
				hasError = true
			}
		}
		require.True(t, hasWarning, "expected at least one warning severity entry")
		require.True(t, hasError, "expected at least one error severity entry")
	})

	t.Run("json format reports all file and line fields", func(t *testing.T) {
		input := `model "Test"
context "Orders" {
  aggregate "Order" {
    slice "Bad Events" {
      event OrderUpdated {
        fields {
          orderId string required
        }
      }
    }
  }
}
`
		path := writeTemp(t, "fields.emod", input)

		output := captureStdout(t, func() {
			_ = cli.RunLint(path, "json")
		})

		var entries []map[string]interface{}
		err := json.Unmarshal([]byte(output), &entries)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(entries), 1)

		entry := entries[0]
		require.Equal(t, path, entry["file"])
		require.NotEqual(t, 0, entry["line"])
		require.NotEmpty(t, entry["message"])
	})

	t.Run("invalid format returns error", func(t *testing.T) {
		path := writeTemp(t, "clean.emod", validEmod)

		err := cli.RunLint(path, "unknown")

		require.ErrorIs(t, err, cli.ErrUnsupportedFormat)
		var lintErr *cli.LintError
		require.True(t, errors.As(err, &lintErr))
		require.Equal(t, 1, lintErr.ExitCode)
	})

	t.Run("info severity", func(t *testing.T) {
		t.Run("json output reports info severity and exits 1", func(t *testing.T) {
			path := writeTemp(t, "single_tag.emod", singleTagDCBEmod)

			var err error
			output := captureStdout(t, func() {
				err = cli.RunLint(path, "json")
			})

			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 1, lintErr.ExitCode)

			var entries []map[string]interface{}
			require.NoError(t, json.Unmarshal([]byte(output), &entries))
			require.Len(t, entries, 1)
			require.Equal(t, "info", entries[0]["severity"])
			require.Equal(t, "dcb/single-tag-everywhere", entries[0]["rule"])
			require.Equal(t, path, entries[0]["file"])
			require.Equal(t, float64(3), entries[0]["line"])
			require.Contains(t, entries[0]["message"], "entity")
		})

		t.Run("an error alongside info entries raises the exit code to 2", func(t *testing.T) {
			path := writeTemp(t, "info_and_error.emod", singleTagWithClickbaitEmod)

			var err error
			output := captureStdout(t, func() {
				err = cli.RunLint(path, "json")
			})

			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 2, lintErr.ExitCode)

			var entries []map[string]interface{}
			require.NoError(t, json.Unmarshal([]byte(output), &entries))
			severities := make([]any, 0, len(entries))
			for _, e := range entries {
				severities = append(severities, e["severity"])
			}
			require.Equal(t, []any{"info", "error"}, severities)
		})

		t.Run("text output formats info entries like other severities", func(t *testing.T) {
			path := writeTemp(t, "single_tag.emod", singleTagDCBEmod)

			err := cli.RunLint(path, "text")

			require.Error(t, err)
			require.Contains(t, err.Error(), path+":3:")
			require.Contains(t, err.Error(), "[dcb/single-tag-everywhere]")
		})
	})

	t.Run("automation reading no view", func(t *testing.T) {
		t.Run("json output reports the rule at warning severity and exits 1", func(t *testing.T) {
			path := writeTemp(t, "no_todo_list.emod", automationWithoutViewEmod)

			var err error
			output := captureStdout(t, func() {
				err = cli.RunLint(path, "json")
			})

			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 1, lintErr.ExitCode)

			var entries []map[string]interface{}
			require.NoError(t, json.Unmarshal([]byte(output), &entries))
			require.Len(t, entries, 1)
			require.Equal(t, "warning", entries[0]["severity"])
			require.Equal(t, "automation/missing-todo-list", entries[0]["rule"])
			require.Equal(t, path, entries[0]["file"])
			require.Equal(t, float64(37), entries[0]["line"])
			require.Contains(t, entries[0]["message"], "RemindOnDueDate")
		})

		t.Run("text output names the rule, the automation and the line it is declared on", func(t *testing.T) {
			path := writeTemp(t, "no_todo_list.emod", automationWithoutViewEmod)

			err := cli.RunLint(path, "text")

			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 1, lintErr.ExitCode)
			require.Equal(t, path+`:37: [automation/missing-todo-list] automation "RemindOnDueDate" reads no view, so nothing in the model shows what work is outstanding; project a view of pending work and read it`, err.Error())
		})
	})

	t.Run("spec/command-without-spec", func(t *testing.T) {
		t.Run("json output reports the rule at info severity and exits 1", func(t *testing.T) {
			path := writeTemp(t, "uncovered_command.emod", commandWithoutSpecEmod)

			var err error
			output := captureStdout(t, func() {
				err = cli.RunLint(path, "json")
			})

			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 1, lintErr.ExitCode)

			var entries []map[string]interface{}
			require.NoError(t, json.Unmarshal([]byte(output), &entries))
			require.Len(t, entries, 1)
			require.Equal(t, "info", entries[0]["severity"])
			require.Equal(t, "spec/command-without-spec", entries[0]["rule"])
			require.Equal(t, path, entries[0]["file"])
			require.Equal(t, float64(28), entries[0]["line"])
			require.Contains(t, entries[0]["message"], "ReturnCopy")
		})

		t.Run("text output names the rule, the command and the line it is declared on", func(t *testing.T) {
			path := writeTemp(t, "uncovered_command.emod", commandWithoutSpecEmod)

			err := cli.RunLint(path, "text")

			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 1, lintErr.ExitCode)
			require.Contains(t, err.Error(), path)
			require.Contains(t, err.Error(), "[spec/command-without-spec]")
			require.Contains(t, err.Error(), `command "ReturnCopy" is not exercised by any spec`)
		})
	})
}

func TestLintExplain(t *testing.T) {
	t.Run("known rule prints description and returns no error", func(t *testing.T) {
		output := captureStdout(t, func() {
			err := cli.RunLintExplain("state-obsession")
			require.NoError(t, err)
		})

		require.Contains(t, output, "generic state-change suffixes")
		require.Contains(t, output, "OrderUpdated")
	})

	t.Run("dcb rule prints description and returns no error", func(t *testing.T) {
		output := captureStdout(t, func() {
			err := cli.RunLintExplain("dcb/query-too-broad")
			require.NoError(t, err)
		})

		require.Contains(t, output, "decides_on")
	})

	t.Run("automation rule prints description and returns no error", func(t *testing.T) {
		output := captureStdout(t, func() {
			err := cli.RunLintExplain("automation/missing-todo-list")
			require.NoError(t, err)
		})

		require.Contains(t, output, "todo list")
		require.Regexp(t, `\breads\b`, output)
	})

	t.Run("spec rule prints description and returns no error", func(t *testing.T) {
		output := captureStdout(t, func() {
			err := cli.RunLintExplain("spec/command-without-spec")
			require.NoError(t, err)
		})

		require.Contains(t, output, "no spec exercises")
		require.Contains(t, output, "not adopted specs")
	})

	t.Run("validator-emitted orphan rules print descriptions and return no error", func(t *testing.T) {
		for _, rule := range []string{"orphan-command", "orphan-event"} {
			output := captureStdout(t, func() {
				err := cli.RunLintExplain(rule)
				require.NoError(t, err)
			})

			require.Contains(t, output, "flow")
		}
	})

	t.Run("unknown rule returns error", func(t *testing.T) {
		err := cli.RunLintExplain("dcb/nonexistent")

		require.Error(t, err)
		require.Contains(t, err.Error(), "unknown rule")
		require.Contains(t, err.Error(), "dcb/nonexistent")
	})

	t.Run("all rules have descriptions", func(t *testing.T) {
		rules := []string{
			"orphan-command",
			"orphan-event",
			"state-obsession",
			"property-sourcing",
			"command-in-disguise",
			"command-past-tense",
			"view-naming",
			"left-chair",
			"god-view",
			"clickbait-event",
			"dcb-in-aggregate-mode",
			"aggregate-in-dcb-mode",
			"dcb/untagged-event",
			"dcb/query-too-broad",
			"dcb/single-tag-everywhere",
			"dcb/orphan-tag-key",
			"automation/missing-todo-list",
			"spec/command-without-spec",
		}
		for _, rule := range rules {
			t.Run(rule, func(t *testing.T) {
				output := captureStdout(t, func() {
					err := cli.RunLintExplain(rule)
					require.NoError(t, err)
				})
				require.NotEmpty(t, output)
			})
		}
	})
}

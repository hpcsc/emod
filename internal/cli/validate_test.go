//go:build unit

package cli_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/cli"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
)

// The cli tests drive whole commands, so they share the pipeline-wide fixtures.
const (
	validEmod        = test.HotelReservation
	describedEmod    = test.DescribedHotelReservation
	keywordFieldEmod = test.KeywordFieldSearchCatalog
	invariantEmod    = test.InvariantLibraryLending
	specEmod         = test.SpecLibraryLending
	invalidEmod      = test.Unparseable
)

func TestValidate(t *testing.T) {
	t.Run("returns no error for valid input", func(t *testing.T) {
		path := writeTemp(t, "valid.emod", validEmod)

		err := cli.RunValidate(path, "text")

		require.NoError(t, err)
	})

	t.Run("returns no error for each parser fixture the repository ships as valid", func(t *testing.T) {
		for _, shipped := range []string{
			"internal/parser/testdata/all_patterns.emod",
			"internal/parser/testdata/minimal.emod",
			"internal/parser/testdata/multi_context.emod",
		} {
			t.Run(shipped, func(t *testing.T) {
				err := cli.RunValidate(filepath.Join("../..", shipped), "text")

				require.NoError(t, err)
			})
		}
	})

	t.Run("examples", func(t *testing.T) {
		authoredToValidate, authoredToFail := examplePaths(t)

		t.Run("returns no error for every example authored to validate", func(t *testing.T) {
			for _, path := range authoredToValidate {
				t.Run(filepath.Base(path), func(t *testing.T) {
					err := cli.RunValidate(path, "text")

					require.NoError(t, err)
				})
			}
		})

		t.Run("returns the diagnostics every example authored to fail demonstrates", func(t *testing.T) {
			demonstrated := map[string][]string{
				"error_diagnostics_test.emod": {
					`event "GuestCheckedOut" does not exist`,
					`[orphan-command] command "SendEmail" is orphaned`,
				},
			}

			for _, path := range authoredToFail {
				name := filepath.Base(path)
				t.Run(name, func(t *testing.T) {
					expected, declared := demonstrated[name]
					require.True(t, declared, "list the diagnostics %s is authored to demonstrate", name)

					err := cli.RunValidate(path, "text")

					require.Error(t, err)
					require.Contains(t, err.Error(), path)
					for _, diagnostic := range expected {
						require.Contains(t, err.Error(), diagnostic)
					}
				})
			}
		})
	})

	t.Run("returns error for invalid input", func(t *testing.T) {
		path := writeTemp(t, "invalid.emod", invalidEmod)

		err := cli.RunValidate(path, "text")

		require.Error(t, err)
		require.Contains(t, err.Error(), path)
		require.Contains(t, err.Error(), ":1:")
	})

	t.Run("returns error naming retired trigger kind replacement", func(t *testing.T) {
		input := `model "Test"
context "Ctx" {
  aggregate "Agg" {
    slice "Slice" {
      trigger Schedule "Nightly Sweep" {
        reads PendingExpiries
      }
    }
  }
}
`
		path := writeTemp(t, "retired_trigger_kind.emod", input)

		err := cli.RunValidate(path, "text")

		require.Error(t, err)
		require.Contains(t, err.Error(), "Schedule")
		require.Contains(t, err.Error(), "automation")
		require.Regexp(t, `\bevery\b`, err.Error())
	})

	t.Run("returns error naming the file when it does not exist", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "nonexistent.emod")

		err := cli.RunValidate(missing, "text")

		require.Error(t, err)
		require.Contains(t, err.Error(), missing)
	})

	t.Run("returns error when no file argument given", func(t *testing.T) {
		err := cli.RunValidate("", "text")

		require.ErrorIs(t, err, cli.ErrMissingFileArgument)
	})

	t.Run("returns semantic error for automation targeting nonexistent context", func(t *testing.T) {
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
		path := writeTemp(t, "bad_target.emod", input)

		err := cli.RunValidate(path, "text")

		require.Error(t, err)
		require.Contains(t, err.Error(), "NonExistent")
		require.Contains(t, err.Error(), "does not exist")
	})

	t.Run("returns error for automation activation event referencing nonexistent event", func(t *testing.T) {
		input := `model "Test"
context "Orders" {
  aggregate "Order" {
    slice "Process Order" {
      automation OrderNotifier {
        on NonExistentEvent
        command NotifyCustomer
      }
    }
  }
}
`
		path := writeTemp(t, "bad_activation_event.emod", input)

		err := cli.RunValidate(path, "text")

		require.Error(t, err)
		require.Contains(t, err.Error(), "NonExistentEvent")
		require.Contains(t, err.Error(), "does not exist")
	})

	t.Run("returns error naming the schedule expression of neither accepted form", func(t *testing.T) {
		input := `model "Reservations"
context "Reservations" {
  aggregate "Reservation" {
    slice "Expire Stale Holds" {
      command ExpireHold {
        fields {
          holdId string required
        }
      }
      event HoldExpired {
        fields {
          holdId    string    required
          expiredAt timestamp required
        }
      }
      automation StaleHoldExpirer {
        every "nightly"
        command ExpireHold
      }
      flow {
        command -> event: ExpireHold -> HoldExpired
      }
    }
  }
}
`
		path := writeTemp(t, "malformed_schedule.emod", input)

		err := cli.RunValidate(path, "text")

		var lintErr *cli.LintError
		require.True(t, errors.As(err, &lintErr))
		require.Equal(t, 1, lintErr.ExitCode)
		require.Contains(t, err.Error(), `schedule expression "nightly" is neither a Go duration nor a five-field cron expression`)
	})

	t.Run("returns error naming the invariant an aggregate declares twice", func(t *testing.T) {
		input := `model "Library Lending"
context "Lending" {
  aggregate "Loan" {
    invariant OneCopyPerLoan "A loan covers exactly one copy of one title"
    invariant OneCopyPerLoan "A loan is settled once"
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
`
		path := writeTemp(t, "duplicate_invariant.emod", input)

		err := cli.RunValidate(path, "text")

		var lintErr *cli.LintError
		require.True(t, errors.As(err, &lintErr))
		require.Equal(t, 1, lintErr.ExitCode)
		require.Contains(t, err.Error(), `invariant "OneCopyPerLoan" is already declared in aggregate "Loan"`)
	})

	t.Run("returns error naming the event a spec misspells", func(t *testing.T) {
		input := `model "Library Lending"
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
      spec "borrows a copy the member returned" {
        given [CopyBorroed]
        when BorrowCopy
        then [CopyBorrowed]
      }
      flow {
        command -> event: BorrowCopy -> CopyBorrowed
      }
    }
  }
}
`
		path := writeTemp(t, "misspelled_spec_event.emod", input)

		err := cli.RunValidate(path, "text")

		var lintErr *cli.LintError
		require.True(t, errors.As(err, &lintErr))
		require.Equal(t, 1, lintErr.ExitCode)
		require.Contains(t, err.Error(), `event "CopyBorroed" does not exist`)
	})

	t.Run("returns error naming the invariant a spec rejects from outside the declaring scope", func(t *testing.T) {
		input := `model "Library Lending"
context "Lending" {
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
      spec "refuses a member who already holds five copies" {
        given [CopyBorrowed]
        when BorrowCopy
        then rejected FiveCopiesPerMember
      }
      flow {
        command -> event: BorrowCopy -> CopyBorrowed
      }
    }
  }
}
`
		path := writeTemp(t, "rejected_out_of_scope.emod", input)

		err := cli.RunValidate(path, "text")

		var lintErr *cli.LintError
		require.True(t, errors.As(err, &lintErr))
		require.Equal(t, 1, lintErr.ExitCode)
		require.Contains(t, err.Error(), `invariant "FiveCopiesPerMember" is not declared in aggregate "Loan"`)
	})

	t.Run("returns error naming the view a spec outcome names and the kind it was looked up as", func(t *testing.T) {
		input := `model "Library Lending"
context "Lending" {
  aggregate "Loan" {
    slice "Review Member Loans" {
      view MemberLoansView {
        fields {
          loanId string required
        }
        subscribes [CopyBorrowed]
      }
      spec "lists loans no one holds" {
        then view MissingView
      }
    }
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
`
		path := writeTemp(t, "misspelled_spec_view.emod", input)

		err := cli.RunValidate(path, "text")

		var lintErr *cli.LintError
		require.True(t, errors.As(err, &lintErr))
		require.Equal(t, 1, lintErr.ExitCode)
		require.Contains(t, err.Error(), `view "MissingView" does not exist`)
	})

	t.Run("returns no error for valid multi-context model", func(t *testing.T) {
		input := `model "Multi Context Test"

context "Orders" {
  aggregate "Order" {
    slice "Place Order" {
      command PlaceOrder {
        fields {
          orderId     string required
          totalAmount string required
        }
      }
      event OrderPlaced {
        fields {
          orderId     string required
          totalAmount string required
        }
      }
      flow {
        command -> event: PlaceOrder -> OrderPlaced
      }
    }
    slice "Browse Orders" {
      view PlacedOrdersView {
        fields {
          orderId     string required
          totalAmount string required
        }
        subscribes [OrderPlaced]
      }
    }
    slice "Notify On Order" {
      automation OrderNotifier {
        on OrderPlaced
        reads PlacedOrdersView
        command SendNotification
        target context Notifications
      }
    }
  }
}
context "Notifications" {
  aggregate "Notification" {
    slice "Send Notification" {
      command SendNotification {
        fields {
          message string required
        }
      }
      flow {
        command -> event: SendNotification -> NotificationReceived
      }
      event NotificationReceived {
        source external "Email Provider"
        fields {
          notificationId string required
          receivedAt     timestamp required
        }
      }
    }
  }
}
`
		path := writeTemp(t, "multi_context.emod", input)

		err := cli.RunValidate(path, "text")

		require.NoError(t, err)
	})

	t.Run("returns no error for automation targeting existing context", func(t *testing.T) {
		input := `model "Test"
context "Orders" {
  aggregate "Order" {
    slice "Place Order" {
      command PlaceOrder {
        fields {
          orderId     string required
          totalAmount string required
        }
      }
      event OrderPlaced {
        fields {
          orderId     string required
          totalAmount string required
        }
      }
      flow {
        command -> event: PlaceOrder -> OrderPlaced
      }
    }
    slice "Browse Orders" {
      view PlacedOrdersView {
        fields {
          orderId     string required
          totalAmount string required
        }
        subscribes [OrderPlaced]
      }
    }
    slice "Notify On Order" {
      automation OrderNotifier {
        on OrderPlaced
        reads PlacedOrdersView
        command SendNotification
        target context Notifications
      }
    }
  }
}
context "Notifications" {
  aggregate "Notification" {
    slice "Send Notification" {
      command SendNotification {
        fields {
          message string required
        }
      }
      command SendEmail {
        fields {
          to string required
        }
      }
      flow {
        command -> event: SendNotification -> NotificationRequested
      }
      flow {
        command -> event: SendEmail -> NotificationRequested
      }
      event NotificationRequested {
        fields {
          notificationId string required
          message        string required
        }
      }
      automation Sender {
        on NotificationRequested
        reads PlacedOrdersView
        command SendEmail
      }
    }
  }
}
`
		path := writeTemp(t, "valid_target.emod", input)

		err := cli.RunValidate(path, "text")

		require.NoError(t, err)
	})

	t.Run("returns error for model with only lint warnings", func(t *testing.T) {
		input := `model "Test"
context "Test" {
  aggregate "Test" {
    slice "Test" {
      command OrderPlaced {}
      event OrderUpdated {}
      view OrderList {}
      automation OrderNotifier {
        on OrderUpdated
        command OrderPlaced
      }
      flow {
        command -> event: OrderPlaced -> OrderUpdated
      }
    }
  }
}
`
		path := writeTemp(t, "lint_only.emod", input)

		err := cli.RunValidate(path, "text")

		require.Error(t, err)
		require.Contains(t, err.Error(), "command-past-tense")
		require.Contains(t, err.Error(), "OrderPlaced")
		require.Contains(t, err.Error(), "state-obsession")
		require.Contains(t, err.Error(), "OrderUpdated")
		require.Contains(t, err.Error(), "view-naming")
		require.Contains(t, err.Error(), "OrderList")
		require.Contains(t, err.Error(), "automation/missing-todo-list")
		require.Contains(t, err.Error(), "OrderNotifier")
	})

	t.Run("returns both lint warnings and validation errors", func(t *testing.T) {
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
		path := writeTemp(t, "combined.emod", input)

		err := cli.RunValidate(path, "text")

		require.Error(t, err)
		// lint warnings
		require.Contains(t, err.Error(), "command-past-tense")
		require.Contains(t, err.Error(), "state-obsession")
		require.Contains(t, err.Error(), "view-naming")
		// validation errors
		require.Contains(t, err.Error(), "NonExistent")
		require.Contains(t, err.Error(), "does not exist")
	})

	t.Run("rejects a file declaring an unsupported version", func(t *testing.T) {
		const unsupportedVersionEmod = "emod 2\n" + validEmod

		t.Run("text output is the version diagnostic and nothing else", func(t *testing.T) {
			path := writeTemp(t, "unsupported.emod", unsupportedVersionEmod)

			err := cli.RunValidate(path, "text")

			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.NotZero(t, lintErr.ExitCode)
			require.Len(t, strings.Split(err.Error(), "\n"), 1)
			require.Contains(t, err.Error(), path)
			require.Contains(t, err.Error(), ":1:")
		})

		t.Run("json output is a single entry at error severity", func(t *testing.T) {
			path := writeTemp(t, "unsupported.emod", unsupportedVersionEmod)

			output := captureStdout(t, func() {
				err := cli.RunValidate(path, "json")
				var lintErr *cli.LintError
				require.True(t, errors.As(err, &lintErr))
				require.Equal(t, 2, lintErr.ExitCode)
			})

			var entries []map[string]interface{}
			require.NoError(t, json.Unmarshal([]byte(output), &entries))
			require.Len(t, entries, 1)
			require.Equal(t, path, entries[0]["file"])
			require.Equal(t, float64(1), entries[0]["line"])
			require.Equal(t, "error", entries[0]["severity"])
		})
	})

	t.Run("json format on clean file outputs empty array", func(t *testing.T) {
		path := writeTemp(t, "clean.emod", validEmod)

		output := captureStdout(t, func() {
			err := cli.RunValidate(path, "json")
			require.NoError(t, err)
		})

		require.Equal(t, "[]\n", output)
	})

	t.Run("json format on a file naming its fields after keywords outputs empty array", func(t *testing.T) {
		path := writeTemp(t, "keyword-fields.emod", keywordFieldEmod)

		output := captureStdout(t, func() {
			err := cli.RunValidate(path, "json")
			require.NoError(t, err)
		})

		require.Equal(t, "[]\n", output)
	})

	t.Run("json format on invalid input outputs structured diagnostics", func(t *testing.T) {
		path := writeTemp(t, "invalid.emod", invalidEmod)

		output := captureStdout(t, func() {
			err := cli.RunValidate(path, "json")
			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 2, lintErr.ExitCode)
			require.Equal(t, "", lintErr.Message)
		})

		var entries []map[string]interface{}
		err := json.Unmarshal([]byte(output), &entries)
		require.NoError(t, err)
		require.Greater(t, len(entries), 0)
	})

	t.Run("json entries contain file, line, severity, and message fields", func(t *testing.T) {
		path := writeTemp(t, "invalid.emod", invalidEmod)

		output := captureStdout(t, func() {
			_ = cli.RunValidate(path, "json")
		})

		var entries []map[string]interface{}
		err := json.Unmarshal([]byte(output), &entries)
		require.NoError(t, err)
		require.Greater(t, len(entries), 0)

		entry := entries[0]
		require.Equal(t, path, entry["file"])
		require.NotEqual(t, 0, entry["line"])
		require.NotEmpty(t, entry["severity"])
		require.NotEmpty(t, entry["message"])
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
			err := cli.RunValidate(path, "json")
			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 1, lintErr.ExitCode)
			require.Equal(t, "", lintErr.Message)
		})

		var entries []map[string]interface{}
		err := json.Unmarshal([]byte(output), &entries)
		require.NoError(t, err)
		require.Len(t, entries, 1)
		require.Equal(t, "warning", entries[0]["severity"])
		require.Equal(t, "state-obsession", entries[0]["rule"])
	})

	t.Run("json format on file with errors outputs error severity and exit code 2", func(t *testing.T) {
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
			err := cli.RunValidate(path, "json")
			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 2, lintErr.ExitCode)
			require.Equal(t, "", lintErr.Message)
		})

		var entries []map[string]interface{}
		err := json.Unmarshal([]byte(output), &entries)
		require.NoError(t, err)
		require.Len(t, entries, 1)
		require.Equal(t, "error", entries[0]["severity"])
		require.Equal(t, "clickbait-event", entries[0]["rule"])
	})

	t.Run("json format reports all file and line fields", func(t *testing.T) {
		path := writeTemp(t, "invalid.emod", invalidEmod)

		output := captureStdout(t, func() {
			_ = cli.RunValidate(path, "json")
		})

		var entries []map[string]interface{}
		err := json.Unmarshal([]byte(output), &entries)
		require.NoError(t, err)
		require.Greater(t, len(entries), 0)

		entry := entries[0]
		require.Equal(t, path, entry["file"])
		require.NotEqual(t, 0, entry["line"])
		require.NotEmpty(t, entry["message"])
	})

	t.Run("unsupported format returns error", func(t *testing.T) {
		path := writeTemp(t, "clean.emod", validEmod)

		err := cli.RunValidate(path, "unknown")

		require.ErrorIs(t, err, cli.ErrUnsupportedFormat)
		var lintErr *cli.LintError
		require.True(t, errors.As(err, &lintErr))
		require.Equal(t, 1, lintErr.ExitCode)
	})

	t.Run("text format is the default and unchanged for existing behaviors", func(t *testing.T) {
		path := writeTemp(t, "invalid.emod", invalidEmod)

		err := cli.RunValidate(path, "text")

		require.Error(t, err)
		require.Contains(t, err.Error(), path)
		require.Contains(t, err.Error(), ":1:")
	})
}

func examplePaths(t *testing.T) (authoredToValidate, authoredToFail []string) {
	t.Helper()
	const dir = "../../examples"

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	for _, entry := range entries {
		name := entry.Name()
		if filepath.Ext(name) != ".emod" {
			continue
		}
		path := filepath.Join(dir, name)
		if strings.HasSuffix(name, "_test.emod") {
			authoredToFail = append(authoredToFail, path)
			continue
		}
		authoredToValidate = append(authoredToValidate, path)
	}
	require.NotEmpty(t, authoredToValidate)
	require.NotEmpty(t, authoredToFail)

	return authoredToValidate, authoredToFail
}

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)
	return path
}

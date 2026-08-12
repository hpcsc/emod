//go:build unit

package cli_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
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

const commandWithoutSpecEmod = `model "Lending"

context "Lending" {
  aggregate "Loan" {
    invariant OneCopyPerLoan "A loan covers exactly one copy"
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
      spec "refuses a copy already on loan" {
        given [CopyBorrowed]
        when BorrowCopy
        then rejected OneCopyPerLoan
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

const noRejectionPathEmod = `model "Lending"

context "Lending" {
  aggregate "Loan" {
    invariant OneCopyPerLoan "A loan covers exactly one copy"
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
      spec "refuses a copy already on loan" {
        given [CopyBorrowed]
        when BorrowCopy
        then rejected OneCopyPerLoan
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
      spec "returns a copy the member holds" {
        given [CopyBorrowed]
        when ReturnCopy
        then [CopyReturned]
      }
    }
  }
}
`

const invariantNeverExercisedEmod = `model "Lending"

context "Lending" {
  aggregate "Loan" {
    invariant OneCopyPerLoan "A loan covers exactly one copy"
    invariant FiveCopiesPerMember "A member holds at most five copies at one time"
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
      spec "refuses a copy already on loan" {
        given [CopyBorrowed]
        when BorrowCopy
        then rejected OneCopyPerLoan
      }
    }
  }
}
`

const givenOutsideBoundaryEmod = `model "Lending"

context "Lending" {
  aggregate "Loan" {
    invariant OneCopyPerLoan "A loan covers exactly one copy"
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
      spec "borrows a copy" {
        when BorrowCopy
        then [CopyBorrowed]
      }
      spec "refuses a copy already on loan" {
        given [CopyBorrowed]
        when BorrowCopy
        then rejected OneCopyPerLoan
      }
    }
  }
  aggregate "Reader" {
    invariant OneReaderPerFloor "A reader stays on one floor"
    slice "Claim Floor" {
      command ClaimFloor {
        fields {
          readerId string required
          floorId  string required
        }
      }
      event FloorClaimed {
        fields {
          claimId  string required
          readerId string required
          floorId  string required
        }
      }
      flow {
        command -> event: ClaimFloor -> FloorClaimed
      }
      spec "claims a floor" {
        when ClaimFloor
        then [FloorClaimed]
      }
      spec "refuses a floor another reader holds" {
        when ClaimFloor
        then rejected OneReaderPerFloor
      }
      spec "refuses when the member already borrowed a copy" {
        given [CopyBorrowed]
        when ClaimFloor
        then rejected OneReaderPerFloor
      }
    }
  }
}
`

// givenValueOutsideBoundaryEmod states a given payload value the when payload
// contradicts on the field ClaimDesk's lone tag predicate routes on, which the
// value-level arm of spec/given-outside-boundary reports and no other rule does.
// Every command declares the field its predicate tags and carries a happy-path
// spec and a rejection, both events are tagged, the context spends two distinct
// tag keys and every declared key is routed on, and each slice states a flow, so
// the four dcb/* rules, the three sibling spec/* rules and the orphan checks all
// stay quiet.
const givenValueOutsideBoundaryEmod = `model "Library Lending"

context "Reading Room" mode dcb {
  invariant OneDeskPerReader "A reader holds at most one desk"
  slice "Claim Desk" {
    command ClaimDesk {
      decides_on {
        events [DeskClaimed]
        where tag(desk = deskId)
      }
      fields {
        memberId string required
        deskId   string required
      }
    }
    event DeskClaimed {
      tags {
        desk: deskId
      }
      fields {
        sessionId string required
        deskId    string required
        memberId  string required
      }
    }
    flow {
      command -> event: ClaimDesk -> DeskClaimed
    }
    spec "claims a free desk" {
      when ClaimDesk
      then [DeskClaimed]
    }
    spec "refuses a desk another reader holds" {
      given [DeskClaimed { deskId: "D-4210" }]
      when ClaimDesk { deskId: "D-5817" }
      then rejected OneDeskPerReader
    }
  }
  slice "Release Desk" {
    command ReleaseDesk {
      decides_on {
        events [DeskReleased]
        where tag(reader = memberId)
      }
      fields {
        sessionId string required
        memberId  string required
      }
    }
    event DeskReleased {
      tags {
        reader: memberId
      }
      fields {
        sessionId  string    required
        memberId   string    required
        releasedAt timestamp required
      }
    }
    flow {
      command -> event: ReleaseDesk -> DeskReleased
    }
    spec "frees a desk its reader holds" {
      when ReleaseDesk
      then [DeskReleased]
    }
    spec "refuses to free a desk nobody holds" {
      when ReleaseDesk
      then rejected OneDeskPerReader
    }
  }
}
`

const givenOutsideBoundaryDCBEmod = `model "Library Lending"

context "Reading Room" mode dcb {
  invariant OneDeskPerReader "A reader holds at most one desk"
  slice "Desk Operations" {
    command ClaimDesk {
      decides_on {
        events [DeskClaimed]
        where tag(desk = deskId) and tag(region = regionId)
      }
      fields {
        memberId string required
        deskId   string required
      }
    }
    command ReleaseDesk {
      fields {
        sessionId string required
      }
    }
    event DeskClaimed {
      tags {
        desk  : deskId
        region: regionId
      }
      fields {
        sessionId string required
        deskId    string required
        memberId  string required
        regionId  string required
      }
    }
    event DeskReleased {
      tags {
        desk  : deskId
        region: regionId
      }
      fields {
        sessionId  string    required
        deskId     string    required
        memberId   string    required
        regionId   string    required
        releasedAt timestamp required
      }
    }
    flow {
      command -> event: ClaimDesk -> DeskClaimed
    }
    flow {
      command -> event: ReleaseDesk -> DeskReleased
    }
    spec "claims a free desk" {
      when ClaimDesk
      then [DeskClaimed]
    }
    spec "refuses when reader is seated" {
      when ClaimDesk
      then rejected OneDeskPerReader
    }
    spec "claims a desk after release" {
      given [DeskReleased]
      when ClaimDesk
      then [DeskClaimed]
    }
    spec "releases a desk" {
      when ReleaseDesk
      then [DeskReleased]
    }
    spec "refuses to release a free desk" {
      when ReleaseDesk
      then rejected OneDeskPerReader
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

	t.Run("flow/rejection-without-spec", func(t *testing.T) {
		// Written to fire flow/rejection-without-spec and nothing else. Both
		// invariants are declared and both are exercised by a rejection spec
		// somewhere in the aggregate, so spec/invariant-never-exercised stays
		// quiet; both commands carry a spec and a rejection scenario, so
		// spec/command-without-spec and spec/no-rejection-path do too; and both
		// have a flow, so neither command nor event is orphaned. What is missing
		// is the one thing under test: nothing on "Borrow Copy" exercises the
		// rejection its own flow block states.
		const rejectionWithoutSpecEmod = `model "Library Lending"

context "Lending" {
  aggregate "Loan" {
    invariant OneCopyPerLoan "A loan covers exactly one copy of one title"
    invariant FiveCopiesPerMember "A member holds at most five copies at one time"

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
        command -> rejected: BorrowCopy -> FiveCopiesPerMember
      }

      spec "refuses a copy already on loan" {
        when BorrowCopy
        then rejected OneCopyPerLoan
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
          loanId     string    required
          copyId     string    required
          returnedAt timestamp required
        }
      }

      flow {
        command -> event: ReturnCopy -> CopyReturned
      }

      spec "refuses a return of a copy the member does not hold" {
        when ReturnCopy
        then rejected FiveCopiesPerMember
      }
    }
  }
}
`

		t.Run("text output names the rule, the command, the invariant and the line", func(t *testing.T) {
			path := writeTemp(t, "rejection-without-spec.emod", rejectionWithoutSpecEmod)

			err := cli.RunLint(path, "text")

			require.Error(t, err)
			require.Len(t, strings.Split(strings.TrimSpace(err.Error()), "\n"), 1,
				"the fixture is written to trip this rule and no other")
			require.Contains(t, err.Error(), "flow/rejection-without-spec")
			require.Contains(t, err.Error(), "BorrowCopy")
			require.Contains(t, err.Error(), "FiveCopiesPerMember")
			require.Contains(t, err.Error(), ":26:",
				"the diagnostic sits on the rejection entry's invariant name")
		})

		t.Run("json output reports info severity and exit code 1", func(t *testing.T) {
			path := writeTemp(t, "rejection-without-spec.emod", rejectionWithoutSpecEmod)

			output := captureStdout(t, func() {
				err := cli.RunLint(path, "json")
				var lintErr *cli.LintError
				require.True(t, errors.As(err, &lintErr))
				require.Equal(t, 1, lintErr.ExitCode)
			})

			var entries []map[string]interface{}
			require.NoError(t, json.Unmarshal([]byte(output), &entries))
			require.Len(t, entries, 1)
			require.Equal(t, "info", entries[0]["severity"])
			require.Equal(t, "flow/rejection-without-spec", entries[0]["rule"])
		})

		t.Run("an info diagnostic is still a diagnostic, so validate fails too", func(t *testing.T) {
			path := writeTemp(t, "rejection-without-spec.emod", rejectionWithoutSpecEmod)

			err := cli.RunValidate(path, "text")

			require.Error(t, err)
			require.Contains(t, err.Error(), "flow/rejection-without-spec")
		})

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
			require.Equal(t, float64(34), entries[0]["line"])
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

	t.Run("spec/no-rejection-path", func(t *testing.T) {
		t.Run("json output reports the rule at info severity and exits 1", func(t *testing.T) {
			path := writeTemp(t, "no_rejection.emod", noRejectionPathEmod)

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
			require.Equal(t, "spec/no-rejection-path", entries[0]["rule"])
			require.Equal(t, path, entries[0]["file"])
			require.Equal(t, float64(34), entries[0]["line"])
			require.Contains(t, entries[0]["message"], "ReturnCopy")
		})

		t.Run("text output names the rule, the command and the line it is declared on", func(t *testing.T) {
			path := writeTemp(t, "no_rejection.emod", noRejectionPathEmod)

			err := cli.RunLint(path, "text")

			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 1, lintErr.ExitCode)
			require.Contains(t, err.Error(), path)
			require.Contains(t, err.Error(), "[spec/no-rejection-path]")
			require.Contains(t, err.Error(), `command "ReturnCopy" is exercised by specs but none states a rejection`)
		})
	})

	t.Run("spec/invariant-never-exercised", func(t *testing.T) {
		t.Run("json output reports the rule at warning severity and exits 1", func(t *testing.T) {
			path := writeTemp(t, "unexercised_invariant.emod", invariantNeverExercisedEmod)

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
			require.Equal(t, "spec/invariant-never-exercised", entries[0]["rule"])
			require.Equal(t, path, entries[0]["file"])
			require.Equal(t, float64(6), entries[0]["line"])
			require.Contains(t, entries[0]["message"], "FiveCopiesPerMember")
			require.Contains(t, entries[0]["message"], "Loan")
		})

		t.Run("text output names the rule, the invariant and the line it is declared on", func(t *testing.T) {
			path := writeTemp(t, "unexercised_invariant.emod", invariantNeverExercisedEmod)

			err := cli.RunLint(path, "text")

			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 1, lintErr.ExitCode)
			require.Contains(t, err.Error(), path)
			require.Contains(t, err.Error(), "[spec/invariant-never-exercised]")
			require.Contains(t, err.Error(), `invariant "FiveCopiesPerMember" in aggregate "Loan" is not referenced by any rejection`)
		})
	})

	t.Run("spec/given-outside-boundary", func(t *testing.T) {
		t.Run("json output reports the rule at warning severity and exits 1", func(t *testing.T) {
			path := writeTemp(t, "outside_boundary.emod", givenOutsideBoundaryEmod)

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
			require.Equal(t, "spec/given-outside-boundary", entries[0]["rule"])
			require.Equal(t, path, entries[0]["file"])
			require.Contains(t, entries[0]["message"], "CopyBorrowed")
			require.Contains(t, entries[0]["message"], "aggregate")
		})

		t.Run("text output names the rule, the event and the line it is written on", func(t *testing.T) {
			path := writeTemp(t, "outside_boundary.emod", givenOutsideBoundaryEmod)

			err := cli.RunLint(path, "text")

			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 1, lintErr.ExitCode)
			require.Contains(t, err.Error(), path)
			require.Contains(t, err.Error(), "[spec/given-outside-boundary]")
			require.Contains(t, err.Error(), `given event "CopyBorrowed" names an event declared by aggregate "Loan" instead of aggregate "Reader"`)
		})

		t.Run("DCB arm: json output reports the DCB-arm message at warning severity and exits 1", func(t *testing.T) {
			path := writeTemp(t, "outside_boundary_dcb.emod", givenOutsideBoundaryDCBEmod)

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
			require.Equal(t, "spec/given-outside-boundary", entries[0]["rule"])
			require.Equal(t, path, entries[0]["file"])
			require.Contains(t, entries[0]["message"], "DeskReleased")
			require.Contains(t, entries[0]["message"], "decides_on")
		})

		t.Run("DCB arm: text output names the rule, the event and the line it is written on", func(t *testing.T) {
			path := writeTemp(t, "outside_boundary_dcb.emod", givenOutsideBoundaryDCBEmod)

			err := cli.RunLint(path, "text")

			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 1, lintErr.ExitCode)
			require.Contains(t, err.Error(), path)
			require.Contains(t, err.Error(), "[spec/given-outside-boundary]")
			require.Contains(t, err.Error(), `given event "DeskReleased" names an event command "ClaimDesk"'s decides_on does not list`)
		})

		t.Run("value arm: json output reports the value-level message at warning severity and exits 1", func(t *testing.T) {
			path := writeTemp(t, "value_outside_boundary.emod", givenValueOutsideBoundaryEmod)

			var err error
			output := captureStdout(t, func() {
				err = cli.RunLint(path, "json")
			})

			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 1, lintErr.ExitCode)

			var entries []map[string]interface{}
			require.NoError(t, json.Unmarshal([]byte(output), &entries))
			require.Len(t, entries, 1, "the fixture is written to trip this arm and no other rule")
			require.Equal(t, "warning", entries[0]["severity"])
			require.Equal(t, "spec/given-outside-boundary", entries[0]["rule"])
			require.Equal(t, path, entries[0]["file"])
			require.Equal(t, float64(34), entries[0]["line"],
				"the line the given payload is written on")
			require.Equal(t, `given event "DeskClaimed" states deskId "D-4210" while command "ClaimDesk"'s `+
				`when payload states "D-5817", so tag "desk" excludes it from the query`, entries[0]["message"])
		})

		t.Run("value arm: text output names the rule, the field and the line the given payload is written on", func(t *testing.T) {
			path := writeTemp(t, "value_outside_boundary.emod", givenValueOutsideBoundaryEmod)

			err := cli.RunLint(path, "text")

			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 1, lintErr.ExitCode)
			require.Equal(t, path+`:34: [spec/given-outside-boundary] given event "DeskClaimed" states deskId "D-4210" `+
				`while command "ClaimDesk"'s when payload states "D-5817", so tag "desk" excludes it from the query`,
				err.Error())
		})

		t.Run("value arm: validate fails on the same model", func(t *testing.T) {
			path := writeTemp(t, "value_outside_boundary.emod", givenValueOutsideBoundaryEmod)

			err := cli.RunValidate(path, "text")

			var lintErr *cli.LintError
			require.True(t, errors.As(err, &lintErr))
			require.Equal(t, 1, lintErr.ExitCode)
			require.Contains(t, err.Error(), "[spec/given-outside-boundary]")
			require.Contains(t, err.Error(), `states deskId "D-4210"`)
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

	t.Run("rejection-path rule prints description and returns no error", func(t *testing.T) {
		output := captureStdout(t, func() {
			err := cli.RunLintExplain("spec/no-rejection-path")
			require.NoError(t, err)
		})

		require.Contains(t, output, "rejection")
		require.Contains(t, output, "happy-path")
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

	t.Run("the boundary rule is explained once, covering its value-level arm alongside the two type-level ones", func(t *testing.T) {
		output := captureStdout(t, func() {
			err := cli.RunLintExplain("spec/given-outside-boundary")
			require.NoError(t, err)
		})

		require.Contains(t, output, "the boundary is the aggregate")
		require.Contains(t, output, "must appear in the list of events the command queries")
		require.Contains(t, output, "a different value for the tagged field")
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
			"flow/rejection-without-spec",
			"spec/command-without-spec",
			"spec/no-rejection-path",
			"spec/invariant-never-exercised",
			"spec/given-outside-boundary",
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

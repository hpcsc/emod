//go:build unit

package cli_test

import (
	"os"
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/cli"
	"github.com/stretchr/testify/require"
)

// unarrangedEmod trails BorrowView behind the slice whose trigger reads it, so
// arranging has something to move.
const unarrangedEmod = `emod 1
model "Library Lending"

context "Lending" {
  aggregate "Loan" {
    slice "Borrow a Copy" {
      command BorrowCopy {
        fields {
          copyId string required
        }
      }

      event CopyBorrowed {
        fields {
          copyId string required
        }
      }

      flow {
        command -> event: BorrowCopy -> CopyBorrowed
      }
    }

    slice "Return a Copy" {
      trigger "Return Desk" {
        actor Member
        reads BorrowView
      }

      command ReturnCopy {
        fields {
          copyId string required
        }
      }

      event CopyReturned {
        fields {
          copyId string required
        }
      }

      flow {
        command -> event: ReturnCopy -> CopyReturned
      }
    }

    # The comment belongs to the view and has to travel with it.
    slice "Borrowed Copies" {
      view BorrowView {
        fields {
          copyId string required
        }
        subscribes [CopyBorrowed]
      }
    }
  }
}
`

func TestSlicesArrange(t *testing.T) {
	t.Run("rewriting the file", func(t *testing.T) {
		t.Run("moves the view ahead of its reader and keeps the comment with it", func(t *testing.T) {
			path := writeTemp(t, "unarranged.emod", unarrangedEmod)

			output := captureStdout(t, func() {
				require.NoError(t, cli.RunSlicesArrange(path, false))
			})

			arranged, err := os.ReadFile(path)
			require.NoError(t, err)
			body := string(arranged)

			require.Less(t, strings.Index(body, `slice "Borrowed Copies"`), strings.Index(body, `slice "Return a Copy"`))
			require.Contains(t, body, "# The comment belongs to the view and has to travel with it.\n    slice \"Borrowed Copies\"")
			require.Contains(t, output, "moved 1 slice")
			require.Contains(t, output, "backward references 1 -> 0")
		})

		t.Run("leaves an already arranged file untouched on disk", func(t *testing.T) {
			path := writeTemp(t, "unarranged.emod", unarrangedEmod)
			captureStdout(t, func() {
				require.NoError(t, cli.RunSlicesArrange(path, false))
			})
			first, err := os.ReadFile(path)
			require.NoError(t, err)

			output := captureStdout(t, func() {
				require.NoError(t, cli.RunSlicesArrange(path, false))
			})

			second, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, string(first), string(second))
			require.Contains(t, output, "already arranged")
		})

		t.Run("reports the references no ordering can turn forward", func(t *testing.T) {
			// Two slices produce CopyBorrowed and both write, so the second
			// producer points back at the declaration whatever the order.
			path := writeTemp(t, "shared-event.emod", `emod 1
model "Library Lending"

context "Lending" {
  aggregate "Loan" {
    slice "Borrow a Copy" {
      command BorrowCopy {
        fields {
          copyId string required
        }
      }

      event CopyBorrowed {
        fields {
          copyId string required
        }
      }

      flow {
        command -> event: BorrowCopy -> CopyBorrowed
      }
    }

    slice "Renew a Copy" {
      command RenewCopy {
        fields {
          copyId string required
        }
      }

      flow {
        command -> event: RenewCopy -> CopyBorrowed
      }
    }
  }
}
`)

			output := captureStdout(t, func() {
				require.NoError(t, cli.RunSlicesArrange(path, false))
			})

			require.Contains(t, output, "[flow] RenewCopy -> CopyBorrowed (Renew a Copy -> Borrow a Copy)")
		})
	})

	t.Run("check", func(t *testing.T) {
		t.Run("fails without rewriting when slices are out of order", func(t *testing.T) {
			path := writeTemp(t, "unarranged.emod", unarrangedEmod)

			err := cli.RunSlicesArrange(path, true)

			require.Error(t, err)
			require.Contains(t, err.Error(), "is not arranged")
			unchanged, readErr := os.ReadFile(path)
			require.NoError(t, readErr)
			require.Equal(t, unarrangedEmod, string(unchanged))
		})

		t.Run("passes for an arranged file", func(t *testing.T) {
			path := writeTemp(t, "unarranged.emod", unarrangedEmod)
			captureStdout(t, func() {
				require.NoError(t, cli.RunSlicesArrange(path, false))
			})

			captureStdout(t, func() {
				require.NoError(t, cli.RunSlicesArrange(path, true))
			})
		})
	})

	t.Run("fails with descriptive message for parse errors", func(t *testing.T) {
		path := writeTemp(t, "invalid.emod", invalidEmod)

		err := cli.RunSlicesArrange(path, false)

		require.Error(t, err)
		require.Contains(t, err.Error(), path)
	})

	t.Run("fails for missing file argument", func(t *testing.T) {
		err := cli.RunSlicesArrange("", false)

		require.ErrorIs(t, err, cli.ErrMissingFileArgument)
	})
}

//go:build unit

package cli_test

import (
	"testing"

	"github.com/hpcsc/emod/internal/cli"
	"github.com/stretchr/testify/require"
)

func TestLint(t *testing.T) {
	t.Run("clean file produces no error", func(t *testing.T) {
		path := writeTemp(t, "clean.emod", validEmod)

		err := cli.RunLint(path)

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

		err := cli.RunLint(path)

		require.Error(t, err)
		require.Contains(t, err.Error(), path)
		require.Contains(t, err.Error(), ":10:")
		require.Contains(t, err.Error(), "state-obsession")
		require.Contains(t, err.Error(), "OrderUpdated")
	})

	t.Run("missing file argument returns error", func(t *testing.T) {
		err := cli.RunLint("")

		require.Error(t, err)
		require.Equal(t, "lint requires exactly one file argument", err.Error())
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		err := cli.RunLint("/tmp/nonexistent-emod-lint-file-abc123.emod")

		require.Error(t, err)
	})

	t.Run("unparseable file returns error with file path and line number", func(t *testing.T) {
		path := writeTemp(t, "invalid.emod", invalidEmod)

		err := cli.RunLint(path)

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

		err := cli.RunLint(path)

		require.Error(t, err)
		require.Contains(t, err.Error(), "state-obsession")
		require.Contains(t, err.Error(), "command-in-disguise")
	})
}
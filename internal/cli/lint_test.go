//go:build unit

package cli_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/hpcsc/emod/internal/cli"
	"github.com/stretchr/testify/require"
)

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

		require.Error(t, err)
		require.Equal(t, "lint requires exactly one file argument", err.Error())
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		err := cli.RunLint("/tmp/nonexistent-emod-lint-file-abc123.emod", "text")

		require.Error(t, err)
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
      event OrderUpdated {
        fields {
          orderId string required
          reason  string required
        }
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
			if errors.As(err, &lintErr) {
				require.Equal(t, 1, lintErr.ExitCode)
				require.Equal(t, "", lintErr.Message)
			} else {
				require.Error(t, err)
			}
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
      event SingleIdEvent {
        fields {
          orderId string required
        }
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
			if errors.As(err, &lintErr) {
				require.Equal(t, 2, lintErr.ExitCode)
				require.Equal(t, "", lintErr.Message)
			} else {
				require.Error(t, err)
			}
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
			if errors.As(err, &lintErr) {
				require.Equal(t, 2, lintErr.ExitCode)
				require.Equal(t, "", lintErr.Message)
			} else {
				require.Error(t, err)
			}
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

	t.Run("text format is the default and unchanged", func(t *testing.T) {
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
		path := writeTemp(t, "default.emod", input)

		errText := cli.RunLint(path, "text")
		errDefault := cli.RunLint(path, "text")

		require.Equal(t, errDefault.Error(), errText.Error())
	})

	t.Run("invalid format returns error", func(t *testing.T) {
		path := writeTemp(t, "clean.emod", validEmod)

		err := cli.RunLint(path, "unknown")

		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported format")
		var lintErr *cli.LintError
		if errors.As(err, &lintErr) {
			require.Equal(t, 1, lintErr.ExitCode)
		}
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

	t.Run("unknown rule returns error", func(t *testing.T) {
		err := cli.RunLintExplain("dcb/nonexistent")

		require.Error(t, err)
		require.Contains(t, err.Error(), "unknown rule")
		require.Contains(t, err.Error(), "dcb/nonexistent")
	})

	t.Run("all rules have descriptions", func(t *testing.T) {
		rules := []string{
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

//go:build unit

package formatter_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/formatter"
	"github.com/hpcsc/emod/internal/parser"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
)

func TestFormatHCL(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		models := map[string]string{
			"billing payments":      test.BillingPayments,
			"hotel reservation":     test.HotelReservation,
			"described hotel":       test.DescribedHotelReservation,
			"keyword field catalog": test.KeywordFieldSearchCatalog,
			"invariants":            test.InvariantLibraryLending,
			"specs":                 test.SpecLibraryLending,
			"rejections":            test.RejectionLibraryLending,
			"every slice pattern":   test.SlicePatternLibraryLending,
			"automation reads":      test.AutomationReadsLibraryLending,
			"trigger reads":         test.TriggerReadsLibraryLending,
			"automation schedule":   test.AutomationScheduleLibraryLending,
			"automation delay":      test.AutomationDelayLibraryLending,
			"payloads":              test.PayloadLibraryLending,
			"wire types":            test.WireTypeLibraryLending,
			"every construct":       test.EveryConstructLibraryLending,
		}

		for name, source := range models {
			t.Run(name+" means the same after it is written and read back", func(t *testing.T) {
				written, parseDiags := parser.Parse(source, "test.emod")
				require.Empty(t, parseDiags)

				rendered := formatter.Format(written)
				read, readDiags := parser.Parse(rendered, "test.emod")
				for _, entry := range readDiags {
					t.Log(entry.String())
				}
				require.Empty(t, readDiags)

				test.RequireEqual(t, written, read,
					cmpopts.IgnoreTypes(ast.Position{}),
					cmpopts.IgnoreFields(ast.Model{}, "VersionDeclared"))
			})
		}
	})

	t.Run("the repository's own examples", func(t *testing.T) {
		paths, err := filepath.Glob(filepath.Join("..", "..", "examples", "*.emod"))
		require.NoError(t, err)
		require.NotEmpty(t, paths)

		for _, path := range paths {
			t.Run(filepath.Base(path)+" means the same after it is written and read back", func(t *testing.T) {
				source, err := os.ReadFile(path)
				require.NoError(t, err)

				written, parseDiags := parser.Parse(string(source), path)
				if len(parseDiags) > 0 {
					t.Skip("the example is the fixture for diagnostics, so it does not parse")
				}

				read, readDiags := parser.Parse(formatter.Format(written), path)

				require.Empty(t, readDiags)
				test.RequireEqual(t, written, read,
					cmpopts.IgnoreTypes(ast.Position{}),
					cmpopts.IgnoreFields(ast.Model{}, "VersionDeclared"))
			})
		}
	})

	t.Run("layout", func(t *testing.T) {
		t.Run("the heredoc lines up with the attribute that opens it", func(t *testing.T) {
			model, _ := parser.Parse(test.BillingPayments, "test.emod")

			rendered := formatter.Format(model)

			require.Contains(t, rendered, "      flow = <<-FLOW\n")
			require.Contains(t, rendered, "        command -> event:    TakePayment -> PaymentTaken\n")
			require.Contains(t, rendered, "      FLOW\n")
		})

		t.Run("a model built rather than read is still pinned to the supported version", func(t *testing.T) {
			rendered := formatter.Format(&ast.Model{Name: "Built"})

			require.Contains(t, rendered, "emod = 1")
		})

		t.Run("what it writes is read back by emod itself", func(t *testing.T) {
			model, _ := parser.Parse(test.BillingPayments, "test.emod")

			_, diags := parser.Parse(formatter.Format(model), "test.emod")

			require.Empty(t, diags)
		})
	})
}

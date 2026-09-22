//go:build unit

package formatter_test

import (
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/formatter"
	"github.com/hpcsc/emod/internal/lexer"
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
			t.Run(name+" means the same after it is written as HCL and read back", func(t *testing.T) {
				tokens, lexDiags := lexer.Scan(source, "test.emod")
				written, parseDiags := parser.New(tokens, "test.emod").Parse()
				require.Empty(t, lexDiags)
				require.Empty(t, parseDiags)

				rendered := formatter.FormatHCL(written)
				read, readDiags := parser.ParseHCL(rendered, "test.emod")
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

	t.Run("layout", func(t *testing.T) {
		t.Run("the heredoc lines up with the attribute that opens it", func(t *testing.T) {
			tokens, _ := lexer.Scan(test.BillingPayments, "test.emod")
			model, _ := parser.New(tokens, "test.emod").Parse()

			rendered := formatter.FormatHCL(model)

			require.Contains(t, rendered, "      flow = <<-FLOW\n")
			require.Contains(t, rendered, "        command -> event:    TakePayment -> PaymentTaken\n")
			require.Contains(t, rendered, "      FLOW\n")
		})

		t.Run("a model built rather than read is still pinned to the supported version", func(t *testing.T) {
			rendered := formatter.FormatHCL(&ast.Model{Name: "Built"})

			require.Contains(t, rendered, "emod = 1")
		})

		t.Run("what it writes is read back by emod itself", func(t *testing.T) {
			tokens, _ := lexer.Scan(test.BillingPayments, "test.emod")
			model, _ := parser.New(tokens, "test.emod").Parse()

			_, diags := parser.ParseHCL(formatter.FormatHCL(model), "test.emod")

			require.Empty(t, diags)
		})
	})
}

package test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/parser"
)

func HotelReservationModel(t *testing.T) *ast.Model {
	t.Helper()

	return parseFixture(t, HotelReservation, "hotel.emod")
}

func DescribedHotelReservationModel(t *testing.T) *ast.Model {
	t.Helper()

	return parseFixture(t, DescribedHotelReservation, "described.emod")
}

func KeywordFieldSearchCatalogModel(t *testing.T) *ast.Model {
	t.Helper()

	return parseFixture(t, KeywordFieldSearchCatalog, "keyword-fields.emod")
}

func InvariantLibraryLendingModel(t *testing.T) *ast.Model {
	t.Helper()

	return parseFixture(t, InvariantLibraryLending, "invariants.emod")
}

func SpecLibraryLendingModel(t *testing.T) *ast.Model {
	t.Helper()

	return parseFixture(t, SpecLibraryLending, "specs.emod")
}

func RejectionLibraryLendingModel(t *testing.T) *ast.Model {
	t.Helper()

	return parseFixture(t, RejectionLibraryLending, "rejections.emod")
}

func SlicePatternLibraryLendingModel(t *testing.T) *ast.Model {
	t.Helper()

	return parseFixture(t, SlicePatternLibraryLending, "slice-patterns.emod")
}

func AutomationReadsLibraryLendingModel(t *testing.T) *ast.Model {
	t.Helper()

	return parseFixture(t, AutomationReadsLibraryLending, "automation-reads.emod")
}

func TriggerReadsLibraryLendingModel(t *testing.T) *ast.Model {
	t.Helper()

	return parseFixture(t, TriggerReadsLibraryLending, "trigger-reads.emod")
}

func AutomationScheduleLibraryLendingModel(t *testing.T) *ast.Model {
	t.Helper()

	return parseFixture(t, AutomationScheduleLibraryLending, "automation-schedule.emod")
}

// parseFixture runs a fixture through the lexer and parser rather than handing
// back a model built in Go, so what a test reads back cannot drift from what an
// author writing that source would get.
func parseFixture(t *testing.T, source, filename string) *ast.Model {
	t.Helper()

	tokens, scanErrs := lexer.Scan(source, filename)
	require.Empty(t, scanErrs)

	model, parseErrs := parser.New(tokens, filename).Parse()
	require.Empty(t, parseErrs)

	return model
}

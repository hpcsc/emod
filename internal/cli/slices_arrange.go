package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/hpcsc/emod/internal/arrange"
	"github.com/hpcsc/emod/internal/formatter"
	"github.com/hpcsc/emod/internal/oracle"
)

func RunSlicesArrange(path string, check bool) error {
	if path == "" {
		return fmt.Errorf("slices arrange %w", ErrMissingFileArgument)
	}

	source, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	model, diagnostics := oracle.Parse(string(source), path)
	if len(diagnostics) > 0 {
		var sb strings.Builder
		for _, d := range diagnostics {
			fmt.Fprintln(&sb, d.String())
		}
		return errors.New(strings.TrimRight(sb.String(), "\n"))
	}

	report := arrange.Model(model)

	if check {
		if report.Changed() {
			return fmt.Errorf("%s is not arranged", path)
		}
		printArrangeReport(report)
		return nil
	}

	// Writing means reprinting the whole file, which would reformat a model
	// whose order was already right. A file nothing moved in is left untouched.
	if report.Changed() {
		if err := os.WriteFile(path, []byte(formatter.Format(model)), 0o644); err != nil {
			return err
		}
	}

	printArrangeReport(report)
	return nil
}

func printArrangeReport(report arrange.Report) {
	if report.Changed() {
		fmt.Printf("moved %d %s; backward references %d -> %d\n",
			report.Moved, pluralSlices(report.Moved), report.BackwardBefore, report.BackwardAfter)
	} else {
		fmt.Printf("already arranged; %d backward %s\n",
			report.BackwardAfter, pluralReferences(report.BackwardAfter))
	}

	for _, r := range report.Backward {
		fmt.Printf("  [%s] %s (%s -> %s)\n", r.Kind, r.Label, r.From.Name, r.To.Name)
	}
}

func pluralSlices(n int) string {
	if n == 1 {
		return "slice"
	}
	return "slices"
}

func pluralReferences(n int) string {
	if n == 1 {
		return "reference"
	}
	return "references"
}

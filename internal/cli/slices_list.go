package cli

import (
	"encoding/json"
	"fmt"

	"github.com/hpcsc/emod/internal/ast"
)

func RunSlicesList(path, format string) error {
	if format != "text" && format != "json" {
		return &LintError{
			Message:  fmt.Sprintf("unsupported format %q; supported formats: text, json", format),
			ExitCode: 1,
			Cause:    ErrUnsupportedFormat,
		}
	}

	model, err := parseModelFile("slices", path)
	if err != nil {
		return err
	}

	records := ast.SliceRecords(model)

	if format == "json" {
		return formatSlicesJSON(records)
	}

	if len(records) == 0 {
		fmt.Println("No slices found.")
		return nil
	}

	maxSlice := len("SLICE")
	maxPattern := len("PATTERN")
	maxContext := len("CONTEXT")
	for _, r := range records {
		maxSlice = max(maxSlice, len(r.Name))
		maxPattern = max(maxPattern, len(r.Pattern))
		maxContext = max(maxContext, len(r.Context))
	}

	fmt.Printf("%-*s  %-*s  %-*s  %s\n", maxSlice, "SLICE", maxPattern, "PATTERN", maxContext, "CONTEXT", "KEY ELEMENTS")
	for _, r := range records {
		fmt.Printf("%-*s  %-*s  %-*s  %s\n", maxSlice, r.Name, maxPattern, r.Pattern, maxContext, r.Context, r.KeyElements)
	}

	return nil
}

func formatSlicesJSON(records []ast.SliceRecord) error {
	out, err := json.Marshal(records)
	if err != nil {
		return &LintError{
			Message:  fmt.Sprintf("json encoding: %s", err),
			ExitCode: 1,
		}
	}
	fmt.Println(string(out))
	return nil
}

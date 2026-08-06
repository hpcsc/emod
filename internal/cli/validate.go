package cli

import (
	"fmt"

	"github.com/hpcsc/emod/internal/oracle"
)

// RunValidate reads the file at path, runs the correctness oracle over its
// contents, and returns an error if there are any diagnostics. The format
// parameter controls output: "text" for human-readable diagnostics (default)
// or "json" for structured output. An empty path is treated as a missing argument.
func RunValidate(path, format string) error {
	if format != "text" && format != "json" {
		return &LintError{
			Message:  fmt.Sprintf("unsupported format %q; supported formats: text, json", format),
			ExitCode: 1,
			Cause:    ErrUnsupportedFormat,
		}
	}

	source, err := readSourceFile("validate", path)
	if err != nil {
		return err
	}

	diagnostics := oracle.Check(source, path)

	if format == "json" {
		return formatJSON(diagnostics)
	}

	if len(diagnostics) > 0 {
		return formatText(diagnostics)
	}

	return nil
}

package cli

import (
	"fmt"

	"github.com/hpcsc/emod/internal/cue"
)

// RunSchema prints the bundled CUE schema to stdout for the "cue" format.
// An unsupported format returns a LintError with exit code 1.
func RunSchema(format string) error {
	if format != "cue" {
		return &LintError{
			Message:  fmt.Sprintf("unsupported format %q; supported format: cue", format),
			ExitCode: 1,
			Cause:    ErrUnsupportedFormat,
		}
	}

	fmt.Print(cue.Schema)
	return nil
}

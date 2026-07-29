package cli

import (
	"fmt"
	"strings"

	"github.com/hpcsc/emod/internal/glossary"
)

func RunGlossary(path, format string) error {
	if format != "markdown" {
		return &LintError{
			Message:  fmt.Sprintf("unsupported format %q; supported formats: markdown", format),
			ExitCode: 1,
			Cause:    ErrUnsupportedFormat,
		}
	}

	model, err := parseModelFile("glossary", path)
	if err != nil {
		return err
	}

	rendered, err := glossary.RenderMarkdown(model)
	if err != nil {
		return &LintError{
			Message:  fmt.Sprintf("rendering glossary: %s", err),
			ExitCode: 1,
		}
	}

	fmt.Print(string(rendered))
	return nil
}

// urfave/cli v2 stops parsing flags at the first positional argument, so a
// format written after the file never reaches the parsed flag set and has to be
// recovered from the leftover arguments.
func glossaryPathAndFormat(args []string, parsedFormat string) (string, string) {
	path := ""
	format := parsedFormat
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-f" || arg == "--format":
			if i+1 < len(args) {
				format = args[i+1]
				i++
			}
		case strings.HasPrefix(arg, "-f="):
			format = strings.TrimPrefix(arg, "-f=")
		case strings.HasPrefix(arg, "--format="):
			format = strings.TrimPrefix(arg, "--format=")
		case path == "":
			path = arg
		}
	}
	return path, format
}

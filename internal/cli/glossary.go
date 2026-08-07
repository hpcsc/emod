package cli

import (
	"fmt"

	"github.com/hpcsc/emod/internal/glossary"
)

func RunGlossary(path, format string) error {
	if format != "markdown" && format != "json" {
		return &LintError{
			Message:  fmt.Sprintf("unsupported format %q; supported formats: markdown, json", format),
			ExitCode: 1,
			Cause:    ErrUnsupportedFormat,
		}
	}

	model, err := parseModelFile("glossary", path)
	if err != nil {
		return err
	}

	render := glossary.RenderMarkdown
	if format == "json" {
		render = glossary.RenderJSON
	}

	rendered, err := render(model)
	if err != nil {
		return &LintError{
			Message:  fmt.Sprintf("rendering glossary: %s", err),
			ExitCode: 1,
		}
	}

	fmt.Print(string(rendered))
	return nil
}

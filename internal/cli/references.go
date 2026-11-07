package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/hpcsc/emod/internal/query"
)

var (
	ErrMissingName     = errors.New("requires a name")
	ErrUnsupportedKind = errors.New("unsupported kind")
)

func RunReferences(path, name, kind, format string) error {
	if format != "text" && format != "json" {
		return &LintError{
			Message:  fmt.Sprintf("unsupported format %q; supported formats: text, json", format),
			ExitCode: 1,
			Cause:    ErrUnsupportedFormat,
		}
	}
	if kind != "" && !slices.Contains(query.ReferenceKinds, query.Kind(kind)) {
		return &LintError{
			Message:  fmt.Sprintf("unsupported kind %q; supported kinds: command, event, view, invariant", kind),
			ExitCode: 1,
			Cause:    ErrUnsupportedKind,
		}
	}
	if path == "" || name == "" {
		cause := ErrMissingName
		if path == "" {
			cause = ErrMissingFileArgument
		}
		return &LintError{
			Message:  "references requires a file and a name: emod references <file> <name>",
			ExitCode: 1,
			Cause:    cause,
		}
	}

	model, err := parseModelFile("references", path)
	if err != nil {
		return err
	}

	found, err := query.References(model, name, query.Kind(kind))
	if err != nil {
		return &LintError{Message: err.Error(), ExitCode: 1, Cause: err}
	}

	if format == "json" {
		out, err := json.Marshal(found)
		if err != nil {
			return &LintError{Message: fmt.Sprintf("json encoding: %s", err), ExitCode: 1}
		}
		fmt.Println(string(out))
		return nil
	}

	blocks := make([]string, len(found))
	for i, f := range found {
		blocks[i] = formatNameReferences(f)
	}
	fmt.Print(strings.Join(blocks, "\n"))
	return nil
}

func formatNameReferences(f query.NameReferences) string {
	var b strings.Builder

	place := query.ScopeName(f.Context, f.Aggregate)
	if f.Slice != "" {
		place += fmt.Sprintf(", slice %q", f.Slice)
	}
	fmt.Fprintf(&b, "%s %s   %s   :%d\n\n", f.Kind, f.Name, place, f.Line)

	lineWidth, keywordWidth, elementWidth, specKeywordWidth := 0, 0, 0, 0
	for _, ref := range f.References {
		lineWidth = max(lineWidth, len(fmt.Sprintf(":%d", ref.Line)))
		if ref.Group == query.GroupNamedInSpecs {
			specKeywordWidth = max(specKeywordWidth, len(ref.Keyword))
			continue
		}
		keywordWidth = max(keywordWidth, len(ref.Keyword))
		elementWidth = max(elementWidth, len(referenceElement(f, ref)))
	}

	for _, group := range query.GroupsOf(f.Kind) {
		fmt.Fprintln(&b, group)
		written := false
		for _, ref := range f.References {
			if ref.Group != group {
				continue
			}
			written = true
			line := fmt.Sprintf(":%d", ref.Line)
			if group == query.GroupNamedInSpecs {
				fmt.Fprintf(&b, "  %-*s  %-*s  %q\n", lineWidth, line, specKeywordWidth, ref.Keyword, ref.Spec)
				continue
			}
			fmt.Fprintf(&b, "  %-*s  %-*s  %-*s  %q\n", lineWidth, line, keywordWidth, ref.Keyword, elementWidth, referenceElement(f, ref), ref.Slice)
		}
		if !written {
			fmt.Fprintln(&b, "  none")
		}
	}

	return b.String()
}

func referenceElement(f query.NameReferences, ref query.Reference) string {
	switch ref.Keyword {
	case "flow", "rejected", "translation":
		if f.Kind == query.KindCommand {
			return f.Name + " -> " + ref.Element.Name
		}
		return ref.Element.Name + " -> " + f.Name
	}
	return string(ref.Element.Kind) + " " + displayName(ref.Element.Name)
}

var identifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func displayName(name string) string {
	if identifier.MatchString(name) {
		return name
	}
	return fmt.Sprintf("%q", name)
}

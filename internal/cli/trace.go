package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/hpcsc/emod/internal/query"
)

func RunTrace(path, from, format string, options query.TraceOptions) error {
	if format != "text" && format != "json" {
		return &LintError{
			Message:  fmt.Sprintf("unsupported format %q; supported formats: text, json", format),
			ExitCode: 1,
			Cause:    ErrUnsupportedFormat,
		}
	}
	if options.Kind != "" && !slices.Contains(query.TraceKinds, options.Kind) {
		return &LintError{
			Message:  fmt.Sprintf("unsupported kind %q; supported kinds: %s", options.Kind, kindList(query.TraceKinds)),
			ExitCode: 1,
			Cause:    ErrUnsupportedKind,
		}
	}
	if path == "" || from == "" {
		cause := ErrMissingName
		if path == "" {
			cause = ErrMissingFileArgument
		}
		return &LintError{
			Message:  "trace requires a file and an element: emod trace <file> <from>",
			ExitCode: 1,
			Cause:    cause,
		}
	}

	model, err := parseModelFile("trace", path)
	if err != nil {
		return err
	}

	tree, err := query.Trace(model, from, options)
	if err != nil {
		message := err.Error()
		var ambiguous *query.AmbiguousNameError
		if errors.As(err, &ambiguous) && len(ambiguous.Kinds()) > 1 {
			message += "; pick one with --kind"
		}
		return &LintError{Message: message, ExitCode: 1, Cause: err}
	}

	if format == "json" {
		out, err := json.Marshal(tree)
		if err != nil {
			return &LintError{Message: fmt.Sprintf("json encoding: %s", err), ExitCode: 1}
		}
		fmt.Println(string(out))
		return nil
	}

	fmt.Print(formatTree(tree, options.Upstream))
	return nil
}

func kindList(kinds []query.Kind) string {
	words := make([]string, len(kinds))
	for i, kind := range kinds {
		words[i] = string(kind)
	}
	return strings.Join(words, ", ")
}

type treeLine struct {
	element string
	node    query.Node
	crosses string
}

func formatTree(tree query.Tree, upstream bool) string {
	children := map[query.Element][]query.Step{}
	nodes := map[query.Element]query.Node{tree.Root.Element: tree.Root}
	for _, step := range tree.Steps {
		children[step.From] = append(children[step.From], step)
		nodes[step.Element] = step.Node
	}

	lines := []treeLine{{element: displayName(tree.Root.Name), node: tree.Root}}
	var walk func(parent query.Node, depth int)
	walk = func(parent query.Node, depth int) {
		for _, step := range children[parent.Element] {
			line := treeLine{
				element: strings.Repeat("  ", depth) + stepArrow(step, upstream) + " " + displayName(step.Name),
				node:    step.Node,
			}
			if step.CrossesContext {
				from, to := parent.Context, step.Context
				if upstream {
					from, to = to, from
				}
				line.crosses = fmt.Sprintf("  (%s -> %s)", from, to)
			}
			lines = append(lines, line)
			walk(nodes[step.Element], depth+1)
		}
	}
	walk(tree.Root, 1)

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", treeHeading(len(tree.Steps), displayName(tree.Root.Name), upstream))
	writeNodeLines(&b, lines)
	return b.String()
}

func treeHeading(count int, root string, upstream bool) string {
	if upstream {
		switch count {
		case 0:
			return "nothing leads to " + root
		case 1:
			return "1 element leads to " + root
		}
		return fmt.Sprintf("%d elements lead to %s", count, root)
	}

	switch count {
	case 0:
		return "nothing follows from " + root
	case 1:
		return "1 element follows from " + root
	}
	return fmt.Sprintf("%d elements follow from %s", count, root)
}

func stepArrow(step query.Step, upstream bool) string {
	label := step.Label
	if step.After != "" {
		label += ", after " + step.After
	}
	if upstream {
		return "<-[" + label + "]-"
	}
	return "-[" + label + "]->"
}

func writeNodeLines(b *strings.Builder, lines []treeLine) {
	elementWidth, kindWidth, contextWidth, sliceWidth := 0, 0, 0, 0
	for _, line := range lines {
		elementWidth = max(elementWidth, len(line.element))
		kindWidth = max(kindWidth, len(line.node.Kind))
		contextWidth = max(contextWidth, len(line.node.Context))
		sliceWidth = max(sliceWidth, len(sliceLabel(line.node)))
	}
	for _, line := range lines {
		fmt.Fprintf(b, "%-*s  %-*s  %-*s  %-*s  :%d%s\n",
			elementWidth, line.element,
			kindWidth, line.node.Kind,
			contextWidth, line.node.Context,
			sliceWidth, sliceLabel(line.node),
			line.node.Line, line.crosses)
	}
}

func sliceLabel(node query.Node) string {
	if node.Slice == "" {
		return ""
	}
	return fmt.Sprintf("%q", node.Slice)
}

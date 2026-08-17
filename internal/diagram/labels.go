package diagram

import (
	"strings"
	"unicode/utf8"

	"github.com/hpcsc/emod/internal/ast"
)

const (
	gearMarking  = "⚙"
	clockMarking = "⏱"
)

func reactorLabel(name string) string {
	return gearMarking + " " + name
}

func cadenceLabel(schedule string) string {
	return `every "` + schedule + `"`
}

// delayLabel renders the delay an automation fires after, in the DSL's own
// spelling, for painting on the arrow from the event it is measured from. The
// clock badge cadenceLabel feeds stays reserved for every: a relative delay and
// a wall-clock schedule are told apart by where each is drawn.
func delayLabel(after string) string {
	if after == "" {
		return ""
	}

	return `after "` + after + `"`
}

func automationLabel(auto *ast.Automation, lineBreak string) string {
	label := reactorLabel(auto.Name)
	if auto.Schedule == "" {
		return label
	}

	return label + lineBreak + clockMarking + " " + auto.Schedule
}

// specCardLines states each of a slice's scenarios in the DSL's own spelling —
// the spec's name quoted as the source quotes it, then whichever of given, when
// and then it writes. The quotes are what separate one scenario from the next:
// every other line opens with a lowercase keyword, and a blank line cannot do
// the job, because an empty line carries no glyph for the format to advance
// past and so collapses into the line above it. A spec that omits given and one
// that writes an empty given both state no given line, which is what emod fmt
// writes for the two spellings, so the card and the formatted source agree. A
// rejection names its invariant and never the prose that invariant states.
func specCardLines(specs []*ast.Spec) []string {
	var lines []string
	for _, spec := range specs {
		if spec == nil {
			continue
		}
		lines = append(lines, `"`+spec.Name+`"`)
		if given := specElementNames(spec.Given); len(given) > 0 {
			lines = append(lines, "given ["+strings.Join(given, ", ")+"]")
		}
		if spec.When != nil && spec.When.Name != "" {
			lines = append(lines, "when "+spec.When.Name)
		}
		if outcome := specOutcomeLine(spec.Then); outcome != "" {
			lines = append(lines, outcome)
		}
	}

	return lines
}

// specOutcomeLine states a scenario's outcome, one spelling per shape a then
// clause takes, so the four read as four rather than as one line whose meaning
// depends on the slice it sits under.
func specOutcomeLine(then ast.ThenClause) string {
	switch outcome := then.(type) {
	case *ast.ThenEvents:
		events := specElementNames(outcome.Events)
		if len(events) == 0 {
			return ""
		}
		return "then [" + strings.Join(events, ", ") + "]"
	case *ast.ThenRejected:
		return "then rejected " + outcome.InvariantName
	case *ast.ThenView:
		return "then view " + outcome.ViewName
	case *ast.ThenCommand:
		return "then command " + outcome.CommandName
	default:
		return ""
	}
}

// specCardLineBudget is how many characters a card's line may hold, standing in
// for a text measurement neither format exposes. A card is sliceWidth-20 = 260px
// and its text is drawn at specCardFontSize, where the widest mix a card states
// measures about 5.7px per character — a line of CamelCase construct names, not
// the prose of a scenario's own name, which runs nearer 4.6px. 44 therefore
// occupies about 250px at the wide end and leaves the card a margin. Calibrate
// against the wide end, not the average: a line past the card is drawn outside
// it by the SVG writer, which never wraps, and wrapped into more lines than the
// card was measured for by draw.io, which does. The figures come from Helvetica;
// a viewer resolving sans-serif to a wider face has less margin, which is the
// other reason the budget is short of the arithmetic.
const specCardLineBudget = 44

// wrapCardLines breaks any line wider than the card into several, so the height
// a card is measured for is the height its text occupies. Breaking at a space
// keeps a scenario's name readable; a single word longer than the budget is cut,
// there being nowhere else to break it.
func wrapCardLines(lines []string) []string {
	wrapped := make([]string, 0, len(lines))
	for _, line := range lines {
		wrapped = append(wrapped, wrapToBudget(line)...)
	}

	return wrapped
}

func wrapToBudget(line string) []string {
	if utf8.RuneCountInString(line) <= specCardLineBudget {
		return []string{line}
	}

	var (
		out     []string
		current string
	)
	flush := func() {
		for utf8.RuneCountInString(current) > specCardLineBudget {
			runes := []rune(current)
			out = append(out, string(runes[:specCardLineBudget]))
			current = string(runes[specCardLineBudget:])
		}
	}

	for _, word := range strings.Fields(line) {
		switch {
		case current == "":
			current = word
		case utf8.RuneCountInString(current)+1+utf8.RuneCountInString(word) <= specCardLineBudget:
			current += " " + word
		default:
			out = append(out, current)
			current = word
		}
		flush()
	}
	if current != "" {
		out = append(out, current)
	}

	return out
}

// specElementNames names the constructs a spec's list refers to. The example
// values an element may carry are left out: a card states what a scenario is
// about, not the data it runs on.
func specElementNames(elements []*ast.SpecElement) []string {
	names := make([]string, 0, len(elements))
	for _, element := range elements {
		if element == nil || element.Name == "" {
			continue
		}
		names = append(names, element.Name)
	}

	return names
}

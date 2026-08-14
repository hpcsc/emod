package diagram

import "github.com/hpcsc/emod/internal/ast"

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

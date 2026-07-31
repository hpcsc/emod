package diagram

import "github.com/hpcsc/emod/internal/ast"

const (
	gearMarking  = "⚙"
	clockMarking = "⏱"
)

func reactorLabel(name string) string {
	return gearMarking + " " + name
}

func automationLabel(auto *ast.Automation, lineBreak string) string {
	label := reactorLabel(auto.Name)
	if auto.Schedule == "" {
		return label
	}

	return label + lineBreak + clockMarking + " " + auto.Schedule
}

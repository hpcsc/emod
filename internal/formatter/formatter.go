package formatter

import (
	"fmt"
	"strings"

	"github.com/hpcsc/emod/internal/ast"
)

func Format(model *ast.Model) string {
	var b strings.Builder
	f := &writer{b: &b}
	f.writeModel(model)
	return b.String()
}

type writer struct {
	b *strings.Builder
}

func (w *writer) line(level int, format string, args ...any) {
	fmt.Fprintf(w.b, "%s%s\n", indent(level), fmt.Sprintf(format, args...))
}

func (w *writer) blankLine() {
	w.b.WriteString("\n")
}

func indent(level int) string {
	return strings.Repeat("  ", level)
}

func (w *writer) writeModel(model *ast.Model) {
	w.line(0, "model %q", model.Name)

	for _, actor := range model.Actors {
		w.blankLine()
		w.line(0, "actor %q", actor.Name)
	}

	for _, ctx := range model.Contexts {
		w.blankLine()
		w.writeContext(ctx, 0)
	}
}

func (w *writer) writeContext(ctx *ast.Context, level int) {
	w.line(level, "context %q {", ctx.Name)
	for i, agg := range ctx.Aggregates {
		if i > 0 {
			w.blankLine()
		}
		w.writeAggregate(agg, level+1)
	}
	w.line(level, "}")
}

func (w *writer) writeAggregate(agg *ast.Aggregate, level int) {
	w.line(level, "aggregate %q {", agg.Name)
	for i, slice := range agg.Slices {
		if i > 0 {
			w.blankLine()
		}
		w.writeSlice(slice, level+1)
	}
	w.line(level, "}")
}

func (w *writer) writeSlice(slice *ast.Slice, level int) {
	w.line(level, "slice %q {", slice.Name)

	inner := level + 1
	needsBlank := false

	if slice.Trigger != nil {
		w.writeTrigger(slice.Trigger, inner)
		needsBlank = true
	}

	if len(slice.Commands) > 0 {
		for i, cmd := range slice.Commands {
			if needsBlank || i > 0 {
				w.blankLine()
			}
			w.writeCommand(cmd, inner)
			needsBlank = true
		}
	}

	if len(slice.Events) > 0 {
		for i, evt := range slice.Events {
			if needsBlank || i > 0 {
				w.blankLine()
			}
			w.writeEvent(evt, inner)
			needsBlank = true
		}
	}

	if len(slice.Views) > 0 {
		for i, view := range slice.Views {
			if needsBlank || i > 0 {
				w.blankLine()
			}
			w.writeView(view, inner)
			needsBlank = true
		}
	}

	if len(slice.Automations) > 0 {
		for i, auto := range slice.Automations {
			if needsBlank || i > 0 {
				w.blankLine()
			}
			w.writeAutomation(auto, inner)
			needsBlank = true
		}
	}

	if len(slice.Translations) > 0 {
		for i, trans := range slice.Translations {
			if needsBlank || i > 0 {
				w.blankLine()
			}
			w.writeTranslation(trans, inner)
			needsBlank = true
		}
	}

	if len(slice.Flows) > 0 {
		if needsBlank {
			w.blankLine()
		}
		w.writeFlows(slice.Flows, inner)
	}

	w.line(level, "}")
}

func (w *writer) writeTrigger(trigger *ast.Trigger, level int) {
	w.line(level, "trigger %s %q {", trigger.Kind, trigger.Name)
	if trigger.Actor != "" {
		w.line(level+1, "actor %s", trigger.Actor)
	}
	if trigger.Reads != "" {
		w.line(level+1, "reads %s", trigger.Reads)
	}
	w.line(level, "}")
}

func (w *writer) writeCommand(cmd *ast.Command, level int) {
	w.line(level, "command %s {", cmd.Name)
	if len(cmd.Fields) > 0 {
		w.writeFields(cmd.Fields, level+1)
	}
	w.line(level, "}")
}

func (w *writer) writeEvent(evt *ast.Event, level int) {
	w.line(level, "event %s {", evt.Name)
	if evt.Source == "external" && evt.ExternalName != "" {
		w.line(level+1, "source external %q", evt.ExternalName)
	}
	if len(evt.Fields) > 0 {
		w.writeFields(evt.Fields, level+1)
	}
	w.line(level, "}")
}

func (w *writer) writeFields(fields []*ast.Field, level int) {
	nameWidth, typeWidth := fieldColumnWidths(fields)

	w.line(level, "fields {")
	for _, f := range fields {
		if f.Modifier != "" {
			w.line(level+1, "%-*s %-*s %s", nameWidth, f.Name, typeWidth, f.Type, f.Modifier)
		} else {
			w.line(level+1, "%-*s %s", nameWidth, f.Name, f.Type)
		}
	}
	w.line(level, "}")
}

func fieldColumnWidths(fields []*ast.Field) (nameWidth, typeWidth int) {
	for _, f := range fields {
		if len(f.Name) > nameWidth {
			nameWidth = len(f.Name)
		}
		if len(f.Type) > typeWidth {
			typeWidth = len(f.Type)
		}
	}
	return nameWidth, typeWidth
}

func (w *writer) writeFlows(flows []*ast.Flow, level int) {
	w.line(level, "flow {")
	for _, flow := range flows {
		w.line(level+1, "command -> event: %s -> %s", flow.CommandName, flow.EventName)
	}
	w.line(level, "}")
}

func (w *writer) writeView(view *ast.View, level int) {
	w.line(level, "view %s {", view.Name)
	if len(view.Fields) > 0 {
		w.writeFields(view.Fields, level+1)
	}
	if len(view.Subscribes) > 0 {
		w.line(level+1, "subscribes [%s]", strings.Join(view.Subscribes, ", "))
	}
	w.line(level, "}")
}

func (w *writer) writeAutomation(auto *ast.Automation, level int) {
	w.line(level, "automation %s {", auto.Name)
	if auto.TriggerEvent != "" {
		w.line(level+1, "trigger %s", auto.TriggerEvent)
	}
	if auto.Command != "" {
		w.line(level+1, "command %s", auto.Command)
	}
	if auto.TargetContext != "" {
		w.line(level+1, "target context %s", auto.TargetContext)
	}
	w.line(level, "}")
}

func (w *writer) writeTranslation(trans *ast.Translation, level int) {
	w.line(level, "translation %s {", trans.Name)
	if trans.ExternalSystem != "" {
		w.line(level+1, "external_system %q", trans.ExternalSystem)
	}
	if trans.Reads != "" {
		w.line(level+1, "reads %s", trans.Reads)
	}
	if trans.Command != "" {
		w.line(level+1, "command %s", trans.Command)
	}
	if trans.Event != nil {
		w.writeEvent(trans.Event, level+1)
	}
	w.line(level, "}")
}

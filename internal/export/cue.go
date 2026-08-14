package export

import (
	"fmt"
	"strings"

	"github.com/hpcsc/emod/internal/ast"
)

// ExportCUE serializes the given AST model to CUE text output.
// The output conforms to the schema defined in internal/cue/schema.cue.
func ExportCUE(model *ast.Model) ([]byte, error) {
	var b strings.Builder
	w := &cueWriter{b: &b}
	w.writeModel(model)
	return []byte(b.String()), nil
}

// cueWriter writes CUE text output by traversing the AST.
type cueWriter struct {
	b     *strings.Builder
	level int
}

func (w *cueWriter) line(format string, args ...any) {
	fmt.Fprintf(w.b, "%s%s\n", strings.Repeat("  ", w.level), fmt.Sprintf(format, args...))
}

func (w *cueWriter) lineIfSet(field, value string) {
	if value == "" {
		return
	}
	w.line("%s: %q", field, value)
}

func (w *cueWriter) listIfSet(field string, items []string) {
	if len(items) == 0 {
		return
	}
	w.line("%s: %s", field, formatStringList(items))
}

func (w *cueWriter) writeObject(field string, writeBody func()) {
	w.line("%s: {", field)
	w.level++
	writeBody()
	w.level--
	w.line("}")
}

func writeCUEList[T any](w *cueWriter, field string, items []T, writeItem func(T)) {
	if len(items) == 0 {
		return
	}
	w.line("%s: [", field)
	w.level++
	for i, item := range items {
		if i > 0 {
			w.line("}, {")
		} else {
			w.line("{")
		}
		w.level++
		writeItem(item)
		w.level--
	}
	w.line("}]")
	w.level--
}

func (w *cueWriter) writeComments(comments []*ast.Comment) {
	writeCUEList(w, "comments", comments, func(c *ast.Comment) {
		w.line("text: %q", c.Text)
	})
}

func (w *cueWriter) writeModel(m *ast.Model) {
	if m == nil {
		return
	}
	w.writeComments(m.Comments)
	w.line("name: %q", m.Name)
	w.lineIfSet("description", m.Description)
	writeCUEList(w, "actors", m.Actors, w.writeActor)
	writeCUEList(w, "contexts", m.Contexts, w.writeContext)
}

func (w *cueWriter) writeActor(a *ast.Actor) {
	w.writeComments(a.Comments)
	w.line("name: %q", a.Name)
	w.lineIfSet("description", a.Description)
}

func (w *cueWriter) writeContext(c *ast.Context) {
	w.writeComments(c.Comments)
	w.line("name: %q", c.Name)
	w.lineIfSet("description", c.Description)
	writeCUEList(w, "invariants", c.Invariants, w.writeInvariant)
	writeCUEList(w, "aggregates", c.Aggregates, w.writeAggregate)
	writeCUEList(w, "slices", c.Slices, w.writeSlice)
}

func (w *cueWriter) writeAggregate(a *ast.Aggregate) {
	w.writeComments(a.Comments)
	w.line("name: %q", a.Name)
	w.lineIfSet("description", a.Description)
	writeCUEList(w, "invariants", a.Invariants, w.writeInvariant)
	writeCUEList(w, "slices", a.Slices, w.writeSlice)
}

func (w *cueWriter) writeInvariant(inv *ast.Invariant) {
	w.writeComments(inv.Comments)
	w.line("name: %q", inv.Name)
	w.line("statement: %q", inv.Statement)
}

func (w *cueWriter) writeSlice(s *ast.Slice) {
	w.writeComments(s.Comments)
	w.line("name: %q", s.Name)
	w.lineIfSet("description", s.Description)
	if s.Trigger != nil {
		w.writeObject("trigger", func() { w.writeTrigger(s.Trigger) })
	}
	writeCUEList(w, "commands", s.Commands, w.writeCommand)
	writeCUEList(w, "events", s.Events, w.writeEvent)
	writeCUEList(w, "fields", s.Fields, w.writeField)
	writeCUEList(w, "flows", s.Flows, w.writeFlow)
	writeCUEList(w, "rejections", s.Rejections, w.writeRejection)
	writeCUEList(w, "views", s.Views, w.writeView)
	writeCUEList(w, "automations", s.Automations, w.writeAutomation)
	writeCUEList(w, "translations", s.Translations, w.writeTranslation)
	writeCUEList(w, "specs", s.Specs, w.writeSpec)
}

func (w *cueWriter) writeSpec(s *ast.Spec) {
	w.writeComments(s.Comments)
	w.line("name: %q", s.Name)
	writeCUEList(w, "given", s.Given, w.writeSpecElement)
	if s.When != nil {
		w.writeObject("when", func() { w.writeSpecElement(s.When) })
	}
	if s.Then != nil {
		w.writeObject("then", func() { w.writeSpecOutcome(s.Then) })
	}
}

func (w *cueWriter) writeSpecElement(element *ast.SpecElement) {
	w.line("name: %q", element.Name)
	writeCUEList(w, "payload", element.Payload, w.writePayloadField)
}

func (w *cueWriter) writePayloadField(field *ast.PayloadField) {
	w.line("name: %q", field.Name)
	w.line("value: %s", cueLiteral(field))
}

// cueLiteral escapes a string, unlike the formatter's quoted: CUE does have
// escape sequences, so a backslash or a tab in a payload value must be written
// as one or the document will not parse back.
func cueLiteral(field *ast.PayloadField) string {
	switch field.Kind {
	case ast.IntegerLiteral, ast.DecimalLiteral:
		return canonicalNumber(field.Value)
	case ast.BooleanLiteral:
		return field.Value
	default:
		return fmt.Sprintf("%q", field.Value)
	}
}

func (w *cueWriter) writeSpecOutcome(then ast.ThenClause) {
	switch t := then.(type) {
	case *ast.ThenEvents:
		writeCUEList(w, "events", t.Events, w.writeSpecElement)
	case *ast.ThenRejected:
		w.lineIfSet("rejected", t.InvariantName)
	case *ast.ThenView:
		w.lineIfSet("view", t.ViewName)
	case *ast.ThenCommand:
		w.lineIfSet("command", t.CommandName)
	}
}

func (w *cueWriter) writeTrigger(t *ast.Trigger) {
	w.writeComments(t.Comments)
	w.line("name: %q", t.Name)
	w.lineIfSet("description", t.Description)
	w.lineIfSet("actor", t.Actor)
	w.lineIfSet("reads", t.Reads)
}

func (w *cueWriter) writeCommand(c *ast.Command) {
	w.writeComments(c.Comments)
	w.line("name: %q", c.Name)
	w.lineIfSet("description", c.Description)
	writeCUEList(w, "fields", c.Fields, w.writeField)
}

func (w *cueWriter) writeEvent(e *ast.Event) {
	w.writeComments(e.Comments)
	w.line("name: %q", e.Name)
	w.lineIfSet("description", e.Description)
	w.lineIfSet("type", e.WireType)
	w.lineIfSet("source", e.Source)
	w.lineIfSet("external_name", e.ExternalName)
	writeCUEList(w, "fields", e.Fields, w.writeField)
}

func (w *cueWriter) writeField(f *ast.Field) {
	w.line("name: %q", f.Name)
	w.line("type: %q", f.Type)
	w.lineIfSet("modifier", f.Modifier)
}

func (w *cueWriter) writeFlow(f *ast.Flow) {
	w.writeComments(f.Comments)
	w.line("command_name: %q", f.CommandName)
	w.line("event_name: %q", f.EventName)
}

func (w *cueWriter) writeRejection(r *ast.Rejection) {
	w.writeComments(r.Comments)
	w.line("command_name: %q", r.CommandName)
	w.line("invariant_name: %q", r.InvariantName)
}

func (w *cueWriter) writeView(v *ast.View) {
	w.writeComments(v.Comments)
	w.line("name: %q", v.Name)
	w.lineIfSet("description", v.Description)
	writeCUEList(w, "fields", v.Fields, w.writeField)
	w.listIfSet("subscribes", v.Subscribes)
}

func (w *cueWriter) writeAutomation(a *ast.Automation) {
	w.writeComments(a.Comments)
	w.line("name: %q", a.Name)
	w.lineIfSet("description", a.Description)
	w.lineIfSet("on_event", a.OnEvent)
	w.lineIfSet("every", a.Schedule)
	w.lineIfSet("after", a.After)
	w.lineIfSet("reads", a.Reads)
	w.lineIfSet("command", a.Command)
	w.lineIfSet("target_context", a.TargetContext)
}

func (w *cueWriter) writeTranslation(t *ast.Translation) {
	w.writeComments(t.Comments)
	w.line("name: %q", t.Name)
	w.lineIfSet("description", t.Description)
	w.lineIfSet("external_system", t.ExternalSystem)
	w.lineIfSet("reads", t.Reads)
	w.lineIfSet("command", t.Command)
	if t.Event != nil {
		w.writeObject("event", func() { w.writeEvent(t.Event) })
	}
}

// formatStringList formats a []string as a CUE list literal.
func formatStringList(items []string) string {
	parts := make([]string, len(items))
	for i, item := range items {
		parts[i] = fmt.Sprintf("%q", item)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

package export

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hpcsc/emod/internal/ast"
)

// JSON intermediate types with struct tags for serialization.
// These are separate from AST types to avoid coupling serialization concerns
// into the domain types.

type jsonComment struct {
	Text string `json:"text"`
}

type jsonModel struct {
	Name     string         `json:"name"`
	Comments []*jsonComment `json:"comments,omitempty"`
	Actors   []*jsonActor   `json:"actors,omitempty"`
	Contexts []*jsonContext `json:"contexts,omitempty"`
}

type jsonActor struct {
	Name     string         `json:"name"`
	Comments []*jsonComment `json:"comments,omitempty"`
}

type jsonContext struct {
	Name       string           `json:"name"`
	Comments   []*jsonComment   `json:"comments,omitempty"`
	Aggregates []*jsonAggregate `json:"aggregates,omitempty"`
}

type jsonAggregate struct {
	Name     string       `json:"name"`
	Comments []*jsonComment `json:"comments,omitempty"`
	Slices   []*jsonSlice `json:"slices,omitempty"`
}

type jsonSlice struct {
	Name         string             `json:"name"`
	Comments     []*jsonComment     `json:"comments,omitempty"`
	Trigger      *jsonTrigger       `json:"trigger,omitempty"`
	Commands     []*jsonCommand     `json:"commands,omitempty"`
	Events       []*jsonEvent       `json:"events,omitempty"`
	Fields       []*jsonField       `json:"fields,omitempty"`
	Flows        []*jsonFlow        `json:"flows,omitempty"`
	Views        []*jsonView        `json:"views,omitempty"`
	Automations  []*jsonAutomation  `json:"automations,omitempty"`
	Translations []*jsonTranslation `json:"translations,omitempty"`
}

type jsonCommand struct {
	Name     string         `json:"name"`
	Comments []*jsonComment `json:"comments,omitempty"`
	Fields   []*jsonField   `json:"fields,omitempty"`
}

type jsonEvent struct {
	Name         string         `json:"name"`
	Comments     []*jsonComment `json:"comments,omitempty"`
	Source       string         `json:"source,omitempty"`
	ExternalName string         `json:"external_name,omitempty"`
	Fields       []*jsonField   `json:"fields,omitempty"`
}

type jsonField struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Modifier string `json:"modifier,omitempty"`
}

type jsonFlow struct {
	Comments    []*jsonComment `json:"comments,omitempty"`
	CommandName string         `json:"command_name"`
	EventName   string         `json:"event_name"`
}

type jsonTrigger struct {
	Comments []*jsonComment `json:"comments,omitempty"`
	Kind     string         `json:"kind"`
	Name     string         `json:"name"`
	Actor    string         `json:"actor,omitempty"`
	Reads    string         `json:"reads,omitempty"`
}

type jsonView struct {
	Name       string         `json:"name"`
	Comments   []*jsonComment `json:"comments,omitempty"`
	Fields     []*jsonField   `json:"fields,omitempty"`
	Subscribes []string       `json:"subscribes,omitempty"`
}

type jsonAutomation struct {
	Name          string         `json:"name"`
	Comments      []*jsonComment `json:"comments,omitempty"`
	TriggerEvent  string         `json:"trigger_event,omitempty"`
	Command       string         `json:"command,omitempty"`
	TargetContext string         `json:"target_context,omitempty"`
}

type jsonTranslation struct {
	Name           string         `json:"name"`
	Comments       []*jsonComment `json:"comments,omitempty"`
	ExternalSystem string         `json:"external_system,omitempty"`
	Reads          string         `json:"reads,omitempty"`
	Command        string         `json:"command,omitempty"`
	Event          *jsonEvent     `json:"event,omitempty"`
}

// ExportJSON serializes the given AST model to a JSON byte slice.
func ExportJSON(model *ast.Model) ([]byte, error) {
	j := convertModel(model)
	return json.Marshal(j)
}

func convertModel(m *ast.Model) *jsonModel {
	if m == nil {
		return nil
	}
	out := &jsonModel{
		Name:     m.Name,
		Comments: convertComments(m.Comments),
		Actors:   convertActors(m.Actors),
		Contexts: convertContexts(m.Contexts),
	}
	return out
}

func convertComments(comments []*ast.Comment) []*jsonComment {
	if comments == nil {
		return nil
	}
	out := make([]*jsonComment, 0, len(comments))
	for _, c := range comments {
		out = append(out, &jsonComment{Text: c.Text})
	}
	return out
}

func convertActors(actors []*ast.Actor) []*jsonActor {
	if actors == nil {
		return nil
	}
	out := make([]*jsonActor, 0, len(actors))
	for _, a := range actors {
		out = append(out, convertActor(a))
	}
	return out
}

func convertActor(a *ast.Actor) *jsonActor {
	if a == nil {
		return nil
	}
	return &jsonActor{
		Name:     a.Name,
		Comments: convertComments(a.Comments),
	}
}

func convertContexts(ctxs []*ast.Context) []*jsonContext {
	if ctxs == nil {
		return nil
	}
	out := make([]*jsonContext, 0, len(ctxs))
	for _, c := range ctxs {
		out = append(out, convertContext(c))
	}
	return out
}

func convertContext(c *ast.Context) *jsonContext {
	if c == nil {
		return nil
	}
	return &jsonContext{
		Name:       c.Name,
		Comments:   convertComments(c.Comments),
		Aggregates: convertAggregates(c.Aggregates),
	}
}

func convertAggregates(aggs []*ast.Aggregate) []*jsonAggregate {
	if aggs == nil {
		return nil
	}
	out := make([]*jsonAggregate, 0, len(aggs))
	for _, a := range aggs {
		out = append(out, convertAggregate(a))
	}
	return out
}

func convertAggregate(a *ast.Aggregate) *jsonAggregate {
	if a == nil {
		return nil
	}
	return &jsonAggregate{
		Name:     a.Name,
		Comments: convertComments(a.Comments),
		Slices:   convertSlices(a.Slices),
	}
}

func convertSlices(slices []*ast.Slice) []*jsonSlice {
	if slices == nil {
		return nil
	}
	out := make([]*jsonSlice, 0, len(slices))
	for _, s := range slices {
		out = append(out, convertSlice(s))
	}
	return out
}

func convertSlice(s *ast.Slice) *jsonSlice {
	if s == nil {
		return nil
	}
	return &jsonSlice{
		Name:         s.Name,
		Comments:     convertComments(s.Comments),
		Trigger:      convertTrigger(s.Trigger),
		Commands:     convertCommands(s.Commands),
		Events:       convertEvents(s.Events),
		Fields:       convertFields(s.Fields),
		Flows:        convertFlows(s.Flows),
		Views:        convertViews(s.Views),
		Automations:  convertAutomations(s.Automations),
		Translations: convertTranslations(s.Translations),
	}
}

func convertCommands(cmds []*ast.Command) []*jsonCommand {
	if cmds == nil {
		return nil
	}
	out := make([]*jsonCommand, 0, len(cmds))
	for _, c := range cmds {
		out = append(out, convertCommand(c))
	}
	return out
}

func convertCommand(c *ast.Command) *jsonCommand {
	if c == nil {
		return nil
	}
	return &jsonCommand{
		Name:     c.Name,
		Comments: convertComments(c.Comments),
		Fields:   convertFields(c.Fields),
	}
}

func convertEvents(events []*ast.Event) []*jsonEvent {
	if events == nil {
		return nil
	}
	out := make([]*jsonEvent, 0, len(events))
	for _, e := range events {
		out = append(out, convertEvent(e))
	}
	return out
}

func convertEvent(e *ast.Event) *jsonEvent {
	if e == nil {
		return nil
	}
	return &jsonEvent{
		Name:         e.Name,
		Comments:     convertComments(e.Comments),
		Source:       e.Source,
		ExternalName: e.ExternalName,
		Fields:       convertFields(e.Fields),
	}
}

func convertFields(fields []*ast.Field) []*jsonField {
	if fields == nil {
		return nil
	}
	out := make([]*jsonField, 0, len(fields))
	for _, f := range fields {
		out = append(out, convertField(f))
	}
	return out
}

func convertField(f *ast.Field) *jsonField {
	if f == nil {
		return nil
	}
	return &jsonField{
		Name:     f.Name,
		Type:     f.Type,
		Modifier: f.Modifier,
	}
}

func convertFlows(flows []*ast.Flow) []*jsonFlow {
	if flows == nil {
		return nil
	}
	out := make([]*jsonFlow, 0, len(flows))
	for _, f := range flows {
		out = append(out, convertFlow(f))
	}
	return out
}

func convertFlow(f *ast.Flow) *jsonFlow {
	if f == nil {
		return nil
	}
	return &jsonFlow{
		Comments:    convertComments(f.Comments),
		CommandName: f.CommandName,
		EventName:   f.EventName,
	}
}

func convertTrigger(t *ast.Trigger) *jsonTrigger {
	if t == nil {
		return nil
	}
	return &jsonTrigger{
		Comments: convertComments(t.Comments),
		Kind:     t.Kind,
		Name:     t.Name,
		Actor:    t.Actor,
		Reads:    t.Reads,
	}
}

func convertViews(views []*ast.View) []*jsonView {
	if views == nil {
		return nil
	}
	out := make([]*jsonView, 0, len(views))
	for _, v := range views {
		out = append(out, convertView(v))
	}
	return out
}

func convertView(v *ast.View) *jsonView {
	if v == nil {
		return nil
	}
	return &jsonView{
		Name:       v.Name,
		Comments:   convertComments(v.Comments),
		Fields:     convertFields(v.Fields),
		Subscribes: v.Subscribes,
	}
}

func convertAutomations(autos []*ast.Automation) []*jsonAutomation {
	if autos == nil {
		return nil
	}
	out := make([]*jsonAutomation, 0, len(autos))
	for _, a := range autos {
		out = append(out, convertAutomation(a))
	}
	return out
}

func convertAutomation(a *ast.Automation) *jsonAutomation {
	if a == nil {
		return nil
	}
	return &jsonAutomation{
		Name:          a.Name,
		Comments:      convertComments(a.Comments),
		TriggerEvent:  a.TriggerEvent,
		Command:       a.Command,
		TargetContext: a.TargetContext,
	}
}

func convertTranslations(trans []*ast.Translation) []*jsonTranslation {
	if trans == nil {
		return nil
	}
	out := make([]*jsonTranslation, 0, len(trans))
	for _, t := range trans {
		out = append(out, convertTranslation(t))
	}
	return out
}

func convertTranslation(t *ast.Translation) *jsonTranslation {
	if t == nil {
		return nil
	}
	return &jsonTranslation{
		Name:           t.Name,
		Comments:       convertComments(t.Comments),
		ExternalSystem: t.ExternalSystem,
		Reads:          t.Reads,
		Command:        t.Command,
		Event:          convertEvent(t.Event),
	}
}

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

func (w *cueWriter) writeModel(m *ast.Model) {
	if m == nil {
		return
	}
	w.writeCommentsList("comments", m.Comments)
	w.line("name: %q", m.Name)
	if len(m.Actors) > 0 {
		w.writeActorList("actors", m.Actors)
	}
	if len(m.Contexts) > 0 {
		w.writeContextList("contexts", m.Contexts)
	}
}

func (w *cueWriter) writeCommentsList(field string, comments []*ast.Comment) {
	if len(comments) == 0 {
		return
	}
	w.line("%s: [", field)
	w.level++
	for i, c := range comments {
		if i > 0 {
			w.line("}, {")
		} else {
			w.line("{")
		}
		w.level++
		w.line("text: %q", c.Text)
		w.level--
	}
	w.line("}]")
	w.level--
}

func (w *cueWriter) writeActorList(field string, actors []*ast.Actor) {
	w.line("%s: [", field)
	w.level++
	for i, a := range actors {
		if i > 0 {
			w.line("}, {")
		} else {
			w.line("{")
		}
		w.level++
		w.writeCommentsList("comments", a.Comments)
		w.line("name: %q", a.Name)
		w.level--
	}
	w.line("}]")
	w.level--
}

func (w *cueWriter) writeContextList(field string, contexts []*ast.Context) {
	w.line("%s: [", field)
	w.level++
	for i, c := range contexts {
		if i > 0 {
			w.line("}, {")
		} else {
			w.line("{")
		}
		w.level++
		w.writeCommentsList("comments", c.Comments)
		w.line("name: %q", c.Name)
		if len(c.Aggregates) > 0 {
			w.writeAggregateList("aggregates", c.Aggregates)
		}
		w.level--
	}
	w.line("}]")
	w.level--
}

func (w *cueWriter) writeAggregateList(field string, aggregates []*ast.Aggregate) {
	w.line("%s: [", field)
	w.level++
	for i, a := range aggregates {
		if i > 0 {
			w.line("}, {")
		} else {
			w.line("{")
		}
		w.level++
		w.writeCommentsList("comments", a.Comments)
		w.line("name: %q", a.Name)
		if len(a.Slices) > 0 {
			w.writeSliceList("slices", a.Slices)
		}
		w.level--
	}
	w.line("}]")
	w.level--
}

func (w *cueWriter) writeSliceList(field string, slices []*ast.Slice) {
	w.line("%s: [", field)
	w.level++
	for i, s := range slices {
		if i > 0 {
			w.line("}, {")
		} else {
			w.line("{")
		}
		w.level++
		w.writeCommentsList("comments", s.Comments)
		w.line("name: %q", s.Name)
		if s.Trigger != nil {
			w.writeTrigger("trigger", s.Trigger)
		}
		if len(s.Commands) > 0 {
			w.writeCommandList("commands", s.Commands)
		}
		if len(s.Events) > 0 {
			w.writeEventList("events", s.Events)
		}
		if len(s.Fields) > 0 {
			w.writeFieldList("fields", s.Fields)
		}
		if len(s.Flows) > 0 {
			w.writeFlowList("flows", s.Flows)
		}
		if len(s.Views) > 0 {
			w.writeViewList("views", s.Views)
		}
		if len(s.Automations) > 0 {
			w.writeAutomationList("automations", s.Automations)
		}
		if len(s.Translations) > 0 {
			w.writeTranslationList("translations", s.Translations)
		}
		w.level--
	}
	w.line("}]")
	w.level--
}

func (w *cueWriter) writeTrigger(field string, t *ast.Trigger) {
	w.line("%s: {", field)
	w.level++
	w.writeCommentsList("comments", t.Comments)
	w.line("kind: %q", t.Kind)
	w.line("name: %q", t.Name)
	if t.Actor != "" {
		w.line("actor: %q", t.Actor)
	}
	if t.Reads != "" {
		w.line("reads: %q", t.Reads)
	}
	w.level--
	w.line("}")
}

func (w *cueWriter) writeCommandList(field string, commands []*ast.Command) {
	w.line("%s: [", field)
	w.level++
	for i, c := range commands {
		if i > 0 {
			w.line("}, {")
		} else {
			w.line("{")
		}
		w.level++
		w.writeCommentsList("comments", c.Comments)
		w.line("name: %q", c.Name)
		if len(c.Fields) > 0 {
			w.writeFieldList("fields", c.Fields)
		}
		w.level--
	}
	w.line("}]")
	w.level--
}

func (w *cueWriter) writeEventList(field string, events []*ast.Event) {
	w.line("%s: [", field)
	w.level++
	for i, e := range events {
		if i > 0 {
			w.line("}, {")
		} else {
			w.line("{")
		}
		w.level++
		w.writeCommentsList("comments", e.Comments)
		w.line("name: %q", e.Name)
		if e.Source != "" {
			w.line("source: %q", e.Source)
		}
		if e.ExternalName != "" {
			w.line("external_name: %q", e.ExternalName)
		}
		if len(e.Fields) > 0 {
			w.writeFieldList("fields", e.Fields)
		}
		w.level--
	}
	w.line("}]")
	w.level--
}

func (w *cueWriter) writeFieldList(field string, fields []*ast.Field) {
	w.line("%s: [", field)
	w.level++
	for i, f := range fields {
		if i > 0 {
			w.line("}, {")
		} else {
			w.line("{")
		}
		w.level++
		w.line("name: %q", f.Name)
		w.line("type: %q", f.Type)
		if f.Modifier != "" {
			w.line("modifier: %q", f.Modifier)
		}
		w.level--
	}
	w.line("}]")
	w.level--
}

func (w *cueWriter) writeFlowList(field string, flows []*ast.Flow) {
	w.line("%s: [", field)
	w.level++
	for i, f := range flows {
		if i > 0 {
			w.line("}, {")
		} else {
			w.line("{")
		}
		w.level++
		w.writeCommentsList("comments", f.Comments)
		w.line("command_name: %q", f.CommandName)
		w.line("event_name: %q", f.EventName)
		w.level--
	}
	w.line("}]")
	w.level--
}

func (w *cueWriter) writeViewList(field string, views []*ast.View) {
	w.line("%s: [", field)
	w.level++
	for i, v := range views {
		if i > 0 {
			w.line("}, {")
		} else {
			w.line("{")
		}
		w.level++
		w.writeCommentsList("comments", v.Comments)
		w.line("name: %q", v.Name)
		if len(v.Fields) > 0 {
			w.writeFieldList("fields", v.Fields)
		}
		if len(v.Subscribes) > 0 {
			w.line("subscribes: %s", formatStringList(v.Subscribes))
		}
		w.level--
	}
	w.line("}]")
	w.level--
}

func (w *cueWriter) writeAutomationList(field string, automations []*ast.Automation) {
	w.line("%s: [", field)
	w.level++
	for i, a := range automations {
		if i > 0 {
			w.line("}, {")
		} else {
			w.line("{")
		}
		w.level++
		w.writeCommentsList("comments", a.Comments)
		w.line("name: %q", a.Name)
		if a.TriggerEvent != "" {
			w.line("trigger_event: %q", a.TriggerEvent)
		}
		if a.Command != "" {
			w.line("command: %q", a.Command)
		}
		if a.TargetContext != "" {
			w.line("target_context: %q", a.TargetContext)
		}
		w.level--
	}
	w.line("}]")
	w.level--
}

func (w *cueWriter) writeTranslationList(field string, translations []*ast.Translation) {
	w.line("%s: [", field)
	w.level++
	for i, t := range translations {
		if i > 0 {
			w.line("}, {")
		} else {
			w.line("{")
		}
		w.level++
		w.writeCommentsList("comments", t.Comments)
		w.line("name: %q", t.Name)
		if t.ExternalSystem != "" {
			w.line("external_system: %q", t.ExternalSystem)
		}
		if t.Reads != "" {
			w.line("reads: %q", t.Reads)
		}
		if t.Command != "" {
			w.line("command: %q", t.Command)
		}
		if t.Event != nil {
			w.writeInlineEvent("event", t.Event)
		}
		w.level--
	}
	w.line("}]")
	w.level--
}

func (w *cueWriter) writeInlineEvent(field string, e *ast.Event) {
	w.line("%s: {", field)
	w.level++
	w.writeCommentsList("comments", e.Comments)
	w.line("name: %q", e.Name)
	if e.Source != "" {
		w.line("source: %q", e.Source)
	}
	if e.ExternalName != "" {
		w.line("external_name: %q", e.ExternalName)
	}
	if len(e.Fields) > 0 {
		w.writeFieldList("fields", e.Fields)
	}
	w.level--
	w.line("}")
}

// formatStringList formats a []string as a CUE list literal.
func formatStringList(items []string) string {
	parts := make([]string, len(items))
	for i, item := range items {
		parts[i] = fmt.Sprintf("%q", item)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

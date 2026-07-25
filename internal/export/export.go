package export

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagnostic"
)

// JSON intermediate types with struct tags for serialization.
// These are separate from AST types to avoid coupling serialization concerns
// into the domain types.

// jsonPosition represents source position information for JSON output.
type jsonPosition struct {
	Filename string `json:"filename,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
}

type jsonComment struct {
	Text     string         `json:"text"`
	Position *jsonPosition  `json:"position,omitempty"`
}

type jsonModel struct {
	Name     string         `json:"name"`
	Position *jsonPosition  `json:"position,omitempty"`
	Comments []*jsonComment `json:"comments,omitempty"`
	Actors   []*jsonActor   `json:"actors,omitempty"`
	Contexts []*jsonContext `json:"contexts,omitempty"`
}

type jsonActor struct {
	Name     string         `json:"name"`
	Position *jsonPosition  `json:"position,omitempty"`
	Comments []*jsonComment `json:"comments,omitempty"`
}

type jsonContext struct {
	Name          string           `json:"name"`
	Position      *jsonPosition    `json:"position,omitempty"`
	OpenPosition  *jsonPosition    `json:"open_position,omitempty"`
	ClosePosition *jsonPosition    `json:"close_position,omitempty"`
	Comments      []*jsonComment   `json:"comments,omitempty"`
	Aggregates    []*jsonAggregate `json:"aggregates,omitempty"`
}

type jsonAggregate struct {
	Name          string         `json:"name"`
	Position      *jsonPosition  `json:"position,omitempty"`
	OpenPosition  *jsonPosition  `json:"open_position,omitempty"`
	ClosePosition *jsonPosition  `json:"close_position,omitempty"`
	Comments      []*jsonComment `json:"comments,omitempty"`
	Slices        []*jsonSlice   `json:"slices,omitempty"`
}

type jsonSlice struct {
	Name          string              `json:"name"`
	Position      *jsonPosition       `json:"position,omitempty"`
	OpenPosition  *jsonPosition       `json:"open_position,omitempty"`
	ClosePosition *jsonPosition       `json:"close_position,omitempty"`
	Comments      []*jsonComment      `json:"comments,omitempty"`
	Trigger       *jsonTrigger        `json:"trigger,omitempty"`
	Commands      []*jsonCommand      `json:"commands,omitempty"`
	Events        []*jsonEvent        `json:"events,omitempty"`
	Fields        []*jsonField        `json:"fields,omitempty"`
	Flows         []*jsonFlow         `json:"flows,omitempty"`
	Views         []*jsonView         `json:"views,omitempty"`
	Automations   []*jsonAutomation   `json:"automations,omitempty"`
	Translations  []*jsonTranslation  `json:"translations,omitempty"`
}

type jsonCommand struct {
	Name          string         `json:"name"`
	Position      *jsonPosition  `json:"position,omitempty"`
	OpenPosition  *jsonPosition  `json:"open_position,omitempty"`
	ClosePosition *jsonPosition  `json:"close_position,omitempty"`
	Comments      []*jsonComment `json:"comments,omitempty"`
	Fields        []*jsonField   `json:"fields,omitempty"`
}

type jsonEvent struct {
	Name                 string         `json:"name"`
	Position             *jsonPosition  `json:"position,omitempty"`
	SourcePosition       *jsonPosition  `json:"source_position,omitempty"`
	ExternalNamePosition *jsonPosition  `json:"external_name_position,omitempty"`
	OpenPosition         *jsonPosition  `json:"open_position,omitempty"`
	ClosePosition        *jsonPosition  `json:"close_position,omitempty"`
	Comments             []*jsonComment `json:"comments,omitempty"`
	Source               string         `json:"source,omitempty"`
	ExternalName         string         `json:"external_name,omitempty"`
	Fields               []*jsonField   `json:"fields,omitempty"`
}

type jsonField struct {
	Name             string        `json:"name"`
	Position         *jsonPosition `json:"position,omitempty"`
	TypePosition     *jsonPosition `json:"type_position,omitempty"`
	ModifierPosition *jsonPosition `json:"modifier_position,omitempty"`
	Type             string        `json:"type"`
	Modifier         string        `json:"modifier,omitempty"`
}

type jsonFlow struct {
	Comments        []*jsonComment `json:"comments,omitempty"`
	CommandName     string         `json:"command_name"`
	CommandPosition *jsonPosition  `json:"command_position,omitempty"`
	EventName       string         `json:"event_name"`
	EventPosition   *jsonPosition  `json:"event_position,omitempty"`
}

type jsonTrigger struct {
	Comments      []*jsonComment `json:"comments,omitempty"`
	Kind          string         `json:"kind"`
	KindPosition  *jsonPosition  `json:"kind_position,omitempty"`
	Name          string         `json:"name"`
	Position      *jsonPosition  `json:"position,omitempty"`
	Actor         string         `json:"actor,omitempty"`
	ActorPosition *jsonPosition  `json:"actor_position,omitempty"`
	Reads         string         `json:"reads,omitempty"`
	ReadsPosition *jsonPosition  `json:"reads_position,omitempty"`
	OpenPosition  *jsonPosition  `json:"open_position,omitempty"`
	ClosePosition *jsonPosition  `json:"close_position,omitempty"`
}

type jsonView struct {
	Name          string         `json:"name"`
	Position      *jsonPosition  `json:"position,omitempty"`
	OpenPosition  *jsonPosition  `json:"open_position,omitempty"`
	ClosePosition *jsonPosition  `json:"close_position,omitempty"`
	Comments      []*jsonComment `json:"comments,omitempty"`
	Fields        []*jsonField   `json:"fields,omitempty"`
	Subscribes    []string       `json:"subscribes,omitempty"`
}

type jsonAutomation struct {
	Name                  string         `json:"name"`
	Position              *jsonPosition  `json:"position,omitempty"`
	TriggerEventPosition  *jsonPosition  `json:"trigger_event_position,omitempty"`
	CommandPosition       *jsonPosition  `json:"command_position,omitempty"`
	TargetContextPosition *jsonPosition  `json:"target_context_position,omitempty"`
	OpenPosition          *jsonPosition  `json:"open_position,omitempty"`
	ClosePosition         *jsonPosition  `json:"close_position,omitempty"`
	Comments              []*jsonComment `json:"comments,omitempty"`
	TriggerEvent          string         `json:"trigger_event,omitempty"`
	Command               string         `json:"command,omitempty"`
	TargetContext         string         `json:"target_context,omitempty"`
}

type jsonTranslation struct {
	Name               string         `json:"name"`
	Position           *jsonPosition  `json:"position,omitempty"`
	ExternalPosition   *jsonPosition  `json:"external_position,omitempty"`
	ReadsPosition      *jsonPosition  `json:"reads_position,omitempty"`
	CommandPosition    *jsonPosition  `json:"command_position,omitempty"`
	OpenPosition       *jsonPosition  `json:"open_position,omitempty"`
	ClosePosition      *jsonPosition  `json:"close_position,omitempty"`
	Comments           []*jsonComment `json:"comments,omitempty"`
	ExternalSystem     string         `json:"external_system,omitempty"`
	Reads              string         `json:"reads,omitempty"`
	Command            string         `json:"command,omitempty"`
	Event              *jsonEvent     `json:"event,omitempty"`
}

// jsonDiagnosticsWrapper is the top-level envelope for JSON diagnostics output.
type jsonDiagnosticsWrapper struct {
	Diagnostics []*jsonDiagnosticEntry `json:"diagnostics"`
	Model       json.RawMessage        `json:"model"`
}

// jsonDiagnosticEntry maps a diagnostic.Entry to JSON.
type jsonDiagnosticEntry struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
	RuleName string `json:"rule_name,omitempty"`
}

// ExportJSONDiagnostics wraps the model JSON and a diagnostics slice into a structured envelope
// with top-level diagnostics array and model object.
func ExportJSONDiagnostics(model *ast.Model, diagnostics []*diagnostic.Entry) ([]byte, error) {
	modelJSON, err := ExportJSON(model)
	if err != nil {
		return nil, err
	}

	wrapper := jsonDiagnosticsWrapper{
		Diagnostics: convertDiagnostics(diagnostics),
		Model:       json.RawMessage(modelJSON),
	}

	return json.Marshal(wrapper)
}

// convertDiagnostics converts a slice of diagnostic Entries to JSON diagnostic entries.
// Nil input produces an empty slice (not null) to ensure diagnostics: [] in output.
func convertDiagnostics(diags []*diagnostic.Entry) []*jsonDiagnosticEntry {
	if diags == nil {
		return []*jsonDiagnosticEntry{}
	}

	out := make([]*jsonDiagnosticEntry, 0, len(diags))
	for _, d := range diags {
		out = append(out, convertDiagnostic(d))
	}
	return out
}

func convertDiagnostic(d *diagnostic.Entry) *jsonDiagnosticEntry {
	return &jsonDiagnosticEntry{
		File:     d.Filename,
		Line:     d.Line,
		Column:   d.Column,
		Message:  d.Message,
		Severity: d.Severity.String(),
		RuleName: d.RuleName,
	}
}

// convertPosition converts an AST Position to a *jsonPosition for serialization.
// Returns nil for zero-value positions so omitempty excludes them from output.
func convertPosition(p ast.Position) *jsonPosition {
	if p == (ast.Position{}) {
		return nil
	}
	return &jsonPosition{
		Filename: p.Filename,
		Line:     p.Line,
		Column:   p.Column,
	}
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
		Position: convertPosition(m.NamePos),
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
		out = append(out, &jsonComment{
			Text:     c.Text,
			Position: convertPosition(c.Position),
		})
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
		Position: convertPosition(a.NamePos),
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
		Name:          c.Name,
		Position:      convertPosition(c.NamePos),
		OpenPosition:  convertPosition(c.OpenPos),
		ClosePosition: convertPosition(c.ClosePos),
		Comments:      convertComments(c.Comments),
		Aggregates:    convertAggregates(c.Aggregates),
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
		Name:          a.Name,
		Position:      convertPosition(a.NamePos),
		OpenPosition:  convertPosition(a.OpenPos),
		ClosePosition: convertPosition(a.ClosePos),
		Comments:      convertComments(a.Comments),
		Slices:        convertSlices(a.Slices),
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
		Name:          s.Name,
		Position:      convertPosition(s.NamePos),
		OpenPosition:  convertPosition(s.OpenPos),
		ClosePosition: convertPosition(s.ClosePos),
		Comments:      convertComments(s.Comments),
		Trigger:       convertTrigger(s.Trigger),
		Commands:      convertCommands(s.Commands),
		Events:        convertEvents(s.Events),
		Fields:        convertFields(s.Fields),
		Flows:         convertFlows(s.Flows),
		Views:         convertViews(s.Views),
		Automations:   convertAutomations(s.Automations),
		Translations:  convertTranslations(s.Translations),
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
		Name:          c.Name,
		Position:      convertPosition(c.NamePos),
		OpenPosition:  convertPosition(c.OpenPos),
		ClosePosition: convertPosition(c.ClosePos),
		Comments:      convertComments(c.Comments),
		Fields:        convertFields(c.Fields),
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
		Name:                 e.Name,
		Position:             convertPosition(e.NamePos),
		SourcePosition:       convertPosition(e.SourcePos),
		ExternalNamePosition: convertPosition(e.ExternalNamePos),
		OpenPosition:         convertPosition(e.OpenPos),
		ClosePosition:        convertPosition(e.ClosePos),
		Comments:             convertComments(e.Comments),
		Source:               e.Source,
		ExternalName:         e.ExternalName,
		Fields:               convertFields(e.Fields),
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
		Name:        f.Name,
		Position:    convertPosition(f.NamePos),
		TypePosition: convertPosition(f.TypePos),
		ModifierPosition: convertPosition(f.ModPos),
		Type:        f.Type,
		Modifier:    f.Modifier,
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
		CommandPosition: convertPosition(f.CommandPos),
		EventName:      f.EventName,
		EventPosition:  convertPosition(f.EventPos),
	}
}

func convertTrigger(t *ast.Trigger) *jsonTrigger {
	if t == nil {
		return nil
	}
	return &jsonTrigger{
		Comments:      convertComments(t.Comments),
		Kind:          t.Kind,
		KindPosition:  convertPosition(t.KindPos),
		Name:          t.Name,
		Position:      convertPosition(t.NamePos),
		Actor:         t.Actor,
		ActorPosition: convertPosition(t.ActorPos),
		Reads:         t.Reads,
		ReadsPosition: convertPosition(t.ReadsPos),
		OpenPosition:  convertPosition(t.OpenPos),
		ClosePosition: convertPosition(t.ClosePos),
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
		Name:          v.Name,
		Position:      convertPosition(v.NamePos),
		OpenPosition:  convertPosition(v.OpenPos),
		ClosePosition: convertPosition(v.ClosePos),
		Comments:      convertComments(v.Comments),
		Fields:        convertFields(v.Fields),
		Subscribes:    v.Subscribes,
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
		Name:                 a.Name,
		Position:             convertPosition(a.NamePos),
		TriggerEventPosition: convertPosition(a.TriggerEventPos),
		CommandPosition:      convertPosition(a.CommandPos),
		TargetContextPosition: convertPosition(a.TargetContextPos),
		OpenPosition:         convertPosition(a.OpenPos),
		ClosePosition:        convertPosition(a.ClosePos),
		Comments:             convertComments(a.Comments),
		TriggerEvent:         a.TriggerEvent,
		Command:              a.Command,
		TargetContext:        a.TargetContext,
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
		Name:             t.Name,
		Position:         convertPosition(t.NamePos),
		ExternalPosition: convertPosition(t.ExternalPos),
		ReadsPosition:    convertPosition(t.ReadsPos),
		CommandPosition:  convertPosition(t.CommandPos),
		OpenPosition:     convertPosition(t.OpenPos),
		ClosePosition:    convertPosition(t.ClosePos),
		Comments:         convertComments(t.Comments),
		ExternalSystem:   t.ExternalSystem,
		Reads:            t.Reads,
		Command:          t.Command,
		Event:            convertEvent(t.Event),
	}
}

// Diagram JSON intermediate types.

type jsonDiagramDocument struct {
	ModelName string             `json:"model_name"`
	Nodes     []*jsonDiagramNode `json:"nodes"`
	Edges     []*jsonDiagramEdge `json:"edges"`
}

type jsonDiagramNode struct {
	ID       string              `json:"id"`
	Type     string              `json:"type"`
	Label    string              `json:"label"`
	ParentID *string             `json:"parentId"`
	Fields   []*jsonDiagramField `json:"fields,omitempty"`
	Position *jsonPosition       `json:"position,omitempty"`
	// Type-specific metadata for trigger, event, view, automation, translation
	Kind           string     `json:"kind,omitempty"`
	Actor          string     `json:"actor,omitempty"`
	Reads          string     `json:"reads,omitempty"`
	Subscribes     []string   `json:"subscribes,omitempty"`
	TriggerEvent   string     `json:"trigger_event,omitempty"`
	Command        string     `json:"command,omitempty"`
	TargetContext  string     `json:"target_context,omitempty"`
	ExternalSystem string     `json:"external_system,omitempty"`
	Source         string     `json:"source,omitempty"`
	ExternalName   string     `json:"external_name,omitempty"`
	Event          *jsonEvent `json:"event,omitempty"`
}

type jsonDiagramEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
}

type jsonDiagramField struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Modifier string `json:"modifier,omitempty"`
}

// jsonDiagramDiagnosticsWrapper is the top-level envelope for diagram JSON diagnostics output.
type jsonDiagramDiagnosticsWrapper struct {
	Diagnostics []*jsonDiagnosticEntry `json:"diagnostics"`
	Diagram     json.RawMessage        `json:"diagram"`
}

// ExportDiagramJSON serializes the given AST model to a diagram-oriented JSON byte slice.
func ExportDiagramJSON(model *ast.Model) ([]byte, error) {
	j := convertModelToDiagram(model)
	return json.Marshal(j)
}

// ExportDiagramJSONDiagnostics wraps the diagram JSON and a diagnostics slice into a structured envelope
// with top-level diagnostics array and diagram object.
func ExportDiagramJSONDiagnostics(model *ast.Model, diagnostics []*diagnostic.Entry) ([]byte, error) {
	diagramJSON, err := ExportDiagramJSON(model)
	if err != nil {
		return nil, err
	}

	wrapper := jsonDiagramDiagnosticsWrapper{
		Diagnostics: convertDiagnostics(diagnostics),
		Diagram:     json.RawMessage(diagramJSON),
	}

	return json.Marshal(wrapper)
}

// diagramIDGenerator generates deterministic sequential IDs for diagram nodes.
type diagramIDGenerator struct {
	counters map[string]int
}

func newDiagramIDGenerator() *diagramIDGenerator {
	return &diagramIDGenerator{
		counters: make(map[string]int),
	}
}

func (g *diagramIDGenerator) next(typ string) string {
	g.counters[typ]++
	return fmt.Sprintf("%s-%d", typ, g.counters[typ])
}

func collectSliceNodes(
	s *ast.Slice,
	sliceID string,
	g *diagramIDGenerator,
	doc *jsonDiagramDocument,
	cmdIDs, evtIDs, triggerIDs, viewIDs, autoIDs, transIDs map[string]string,
) {
	// Commands
	for _, cmd := range s.Commands {
		if cmd == nil {
			continue
		}
		cmdID := g.next("command")
		cmdIDs[cmd.Name] = cmdID
		node := &jsonDiagramNode{
			ID:       cmdID,
			Type:     "command",
			Label:    cmd.Name,
			ParentID: &sliceID,
			Position: convertPosition(cmd.NamePos),
		}
		if len(cmd.Fields) > 0 {
			node.Fields = convertFieldsToDiagram(cmd.Fields)
		}
		doc.Nodes = append(doc.Nodes, node)
	}

	// Events
	for _, evt := range s.Events {
		if evt == nil {
			continue
		}
		evtID := g.next("event")
		evtIDs[evt.Name] = evtID
		node := &jsonDiagramNode{
			ID:           evtID,
			Type:         "event",
			Label:        evt.Name,
			ParentID:     &sliceID,
			Position:     convertPosition(evt.NamePos),
			Source:       evt.Source,
			ExternalName: evt.ExternalName,
		}
		if len(evt.Fields) > 0 {
			node.Fields = convertFieldsToDiagram(evt.Fields)
		}
		doc.Nodes = append(doc.Nodes, node)
	}

	// Trigger node (single per slice)
	if s.Trigger != nil {
		tID := g.next("trigger")
		triggerIDs[s.Trigger.Name] = tID
		doc.Nodes = append(doc.Nodes, &jsonDiagramNode{
			ID:       tID,
			Type:     "trigger",
			Label:    s.Trigger.Name,
			ParentID: &sliceID,
			Position: convertPosition(s.Trigger.NamePos),
			Kind:     s.Trigger.Kind,
			Actor:    s.Trigger.Actor,
			Reads:    s.Trigger.Reads,
		})
	}

	// View nodes
	for _, v := range s.Views {
		if v == nil {
			continue
		}
		vID := g.next("view")
		viewIDs[v.Name] = vID
		node := &jsonDiagramNode{
			ID:         vID,
			Type:       "view",
			Label:      v.Name,
			ParentID:   &sliceID,
			Position:   convertPosition(v.NamePos),
			Subscribes: v.Subscribes,
		}
		if len(v.Fields) > 0 {
			node.Fields = convertFieldsToDiagram(v.Fields)
		}
		doc.Nodes = append(doc.Nodes, node)
	}

	// Automation nodes
	for _, a := range s.Automations {
		if a == nil {
			continue
		}
		aID := g.next("auto")
		autoIDs[a.Name] = aID
		doc.Nodes = append(doc.Nodes, &jsonDiagramNode{
			ID:            aID,
			Type:          "automation",
			Label:         a.Name,
			ParentID:      &sliceID,
			Position:      convertPosition(a.NamePos),
			TriggerEvent:  a.TriggerEvent,
			Command:       a.Command,
			TargetContext: a.TargetContext,
		})
	}

	// Translation nodes (with optional standalone nested event node)
	for _, t := range s.Translations {
		if t == nil {
			continue
		}
		tID := g.next("trans")
		transIDs[t.Name] = tID
		node := &jsonDiagramNode{
			ID:             tID,
			Type:           "translation",
			Label:          t.Name,
			ParentID:       &sliceID,
			Position:       convertPosition(t.NamePos),
			ExternalSystem: t.ExternalSystem,
			Reads:          t.Reads,
			Command:        t.Command,
		}
		if t.Event != nil {
			node.Event = convertEvent(t.Event)

			// Create a standalone event node for the translation_event edge target
			evtID := g.next("event")
			evtIDs[t.Event.Name] = evtID
			evtNode := &jsonDiagramNode{
				ID:           evtID,
				Type:         "event",
				Label:        t.Event.Name,
				ParentID:     &sliceID,
				Position:     convertPosition(t.Event.NamePos),
				Source:       t.Event.Source,
				ExternalName: t.Event.ExternalName,
			}
			if len(t.Event.Fields) > 0 {
				evtNode.Fields = convertFieldsToDiagram(t.Event.Fields)
			}
			doc.Nodes = append(doc.Nodes, evtNode)
		}
		doc.Nodes = append(doc.Nodes, node)
	}
}

func convertModelToDiagram(m *ast.Model) *jsonDiagramDocument {
	if m == nil {
		return nil
	}

	g := newDiagramIDGenerator()
	doc := &jsonDiagramDocument{
		ModelName: m.Name,
		Nodes:     make([]*jsonDiagramNode, 0),
		Edges:     make([]*jsonDiagramEdge, 0),
	}

	// Global name→ID maps for two-pass name resolution (Pass 1: collect, Pass 2: resolve)
	cmdIDs := make(map[string]string)
	evtIDs := make(map[string]string)
	viewIDs := make(map[string]string)
	autoIDs := make(map[string]string)
	transIDs := make(map[string]string)
	triggerIDs := make(map[string]string)

	// ---- Pass 1: Create all nodes and build global name→ID maps ----

	for _, a := range m.Actors {
		if a == nil {
			continue
		}
		doc.Nodes = append(doc.Nodes, &jsonDiagramNode{
			ID:       g.next("actor"),
			Type:     "actor",
			Label:    a.Name,
			ParentID: nil,
			Position: convertPosition(a.NamePos),
		})
	}

	for _, c := range m.Contexts {
		if c == nil {
			continue
		}
		ctxID := g.next("context")
		doc.Nodes = append(doc.Nodes, &jsonDiagramNode{
			ID:       ctxID,
			Type:     "context",
			Label:    c.Name,
			ParentID: nil,
			Position: convertPosition(c.NamePos),
		})

		for _, agg := range c.Aggregates {
			if agg == nil {
				continue
			}
			aggID := g.next("aggregate")
			doc.Nodes = append(doc.Nodes, &jsonDiagramNode{
				ID:       aggID,
				Type:     "aggregate",
				Label:    agg.Name,
				ParentID: &ctxID,
				Position: convertPosition(agg.NamePos),
			})

			for _, s := range agg.Slices {
				if s == nil {
					continue
				}
				sliceID := g.next("slice")
				doc.Nodes = append(doc.Nodes, &jsonDiagramNode{
					ID:       sliceID,
					Type:     "slice",
					Label:    s.Name,
					ParentID: &aggID,
					Position: convertPosition(s.NamePos),
				})
				collectSliceNodes(s, sliceID, g, doc, cmdIDs, evtIDs, triggerIDs, viewIDs, autoIDs, transIDs)
			}
		}

		for _, s := range c.Slices {
			if s == nil {
				continue
			}
			sliceID := g.next("slice")
			doc.Nodes = append(doc.Nodes, &jsonDiagramNode{
				ID:       sliceID,
				Type:     "slice",
				Label:    s.Name,
				ParentID: &ctxID,
				Position: convertPosition(s.NamePos),
			})
			collectSliceNodes(s, sliceID, g, doc, cmdIDs, evtIDs, triggerIDs, viewIDs, autoIDs, transIDs)
		}
	}

	// ---- Pass 2: Resolve references and emit edges ----
	// Uses global name→ID maps collected in Pass 1.
	// Unresolved references are silently skipped (no panic, no broken output).

	forEachSlice := func(c *ast.Context, fn func(s *ast.Slice)) {
		for _, agg := range c.Aggregates {
			if agg == nil {
				continue
			}
			for _, s := range agg.Slices {
				if s == nil {
					continue
				}
				fn(s)
			}
		}
		for _, s := range c.Slices {
			if s == nil {
				continue
			}
			fn(s)
		}
	}

	for _, c := range m.Contexts {
		if c == nil {
			continue
		}
		forEachSlice(c, func(s *ast.Slice) {
			// Flow edges
			for _, f := range s.Flows {
				if f == nil {
					return
				}
				if srcID, ok := cmdIDs[f.CommandName]; ok {
					if tgtID, ok := evtIDs[f.EventName]; ok {
						doc.Edges = append(doc.Edges, &jsonDiagramEdge{
							Source: srcID,
							Target: tgtID,
							Type:   "flow",
						})
					}
				}
			}

			// trigger_command: trigger → each command within the same slice
			if s.Trigger != nil {
				srcID, ok := triggerIDs[s.Trigger.Name]
				if ok {
					for _, cmd := range s.Commands {
						if cmd == nil {
							continue
						}
						if tgtID, ok := cmdIDs[cmd.Name]; ok {
							doc.Edges = append(doc.Edges, &jsonDiagramEdge{
								Source: srcID,
								Target: tgtID,
								Type:   "trigger_command",
							})
						}
					}
				}
			}

			// subscription: event → subscribing view (cross-boundary)
			for _, v := range s.Views {
				if v == nil {
					continue
				}
				tgtID, ok := viewIDs[v.Name]
				if !ok {
					continue
				}
				for _, sub := range v.Subscribes {
					if srcID, ok := evtIDs[sub]; ok {
						doc.Edges = append(doc.Edges, &jsonDiagramEdge{
							Source: srcID,
							Target: tgtID,
							Type:   "subscription",
						})
					}
				}
			}

			// Automation edges
			for _, a := range s.Automations {
				if a == nil {
					continue
				}
				autoID, ok := autoIDs[a.Name]
				if !ok {
					continue
				}

				if a.TriggerEvent != "" {
					if srcID, ok := evtIDs[a.TriggerEvent]; ok {
						doc.Edges = append(doc.Edges, &jsonDiagramEdge{
							Source: srcID,
							Target: autoID,
							Type:   "automation_trigger",
						})
					}
				}

				if a.Command != "" {
					if tgtID, ok := cmdIDs[a.Command]; ok {
						doc.Edges = append(doc.Edges, &jsonDiagramEdge{
							Source: autoID,
							Target: tgtID,
							Type:   "automation_command",
						})
					}
				}
			}

			// translation edges
			for _, t := range s.Translations {
				if t == nil {
					continue
				}
				srcID, ok := transIDs[t.Name]
				if !ok {
					continue
				}

				if t.Reads != "" {
					if viewID, ok := viewIDs[t.Reads]; ok {
						doc.Edges = append(doc.Edges, &jsonDiagramEdge{
							Source: viewID,
							Target: srcID,
							Type:   "reads",
						})
					}
				}

				if t.Command != "" {
					if cmdID, ok := cmdIDs[t.Command]; ok {
						doc.Edges = append(doc.Edges, &jsonDiagramEdge{
							Source: srcID,
							Target: cmdID,
							Type:   "translation_command",
						})

						if t.Event != nil && t.Event.Name != "" {
							if evtID, ok := evtIDs[t.Event.Name]; ok {
								doc.Edges = append(doc.Edges, &jsonDiagramEdge{
									Source: cmdID,
									Target: evtID,
									Type:   "flow",
								})
							}
						}
					}
				}
			}
		})
	}

	return doc
}

func convertFieldsToDiagram(fields []*ast.Field) []*jsonDiagramField {
	if fields == nil {
		return nil
	}
	out := make([]*jsonDiagramField, 0, len(fields))
	for _, f := range fields {
		if f == nil {
			continue
		}
		out = append(out, &jsonDiagramField{
			Name:     f.Name,
			Type:     f.Type,
			Modifier: f.Modifier,
		})
	}
	return out
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

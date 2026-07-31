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
	Text     string        `json:"text"`
	Position *jsonPosition `json:"position,omitempty"`
}

type jsonModel struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Position    *jsonPosition  `json:"position,omitempty"`
	Comments    []*jsonComment `json:"comments,omitempty"`
	Actors      []*jsonActor   `json:"actors,omitempty"`
	Contexts    []*jsonContext `json:"contexts,omitempty"`
}

type jsonActor struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Position    *jsonPosition  `json:"position,omitempty"`
	Comments    []*jsonComment `json:"comments,omitempty"`
}

type jsonContext struct {
	Name          string           `json:"name"`
	Description   string           `json:"description,omitempty"`
	Position      *jsonPosition    `json:"position,omitempty"`
	OpenPosition  *jsonPosition    `json:"open_position,omitempty"`
	ClosePosition *jsonPosition    `json:"close_position,omitempty"`
	Comments      []*jsonComment   `json:"comments,omitempty"`
	Invariants    []*jsonInvariant `json:"invariants,omitempty"`
	Aggregates    []*jsonAggregate `json:"aggregates,omitempty"`
	Slices        []*jsonSlice     `json:"slices,omitempty"`
}

type jsonAggregate struct {
	Name          string           `json:"name"`
	Description   string           `json:"description,omitempty"`
	Position      *jsonPosition    `json:"position,omitempty"`
	OpenPosition  *jsonPosition    `json:"open_position,omitempty"`
	ClosePosition *jsonPosition    `json:"close_position,omitempty"`
	Comments      []*jsonComment   `json:"comments,omitempty"`
	Invariants    []*jsonInvariant `json:"invariants,omitempty"`
	Slices        []*jsonSlice     `json:"slices,omitempty"`
}

type jsonInvariant struct {
	Comments  []*jsonComment `json:"comments,omitempty"`
	Name      string         `json:"name"`
	Statement string         `json:"statement"`
}

type jsonSlice struct {
	Name          string             `json:"name"`
	Description   string             `json:"description,omitempty"`
	Position      *jsonPosition      `json:"position,omitempty"`
	OpenPosition  *jsonPosition      `json:"open_position,omitempty"`
	ClosePosition *jsonPosition      `json:"close_position,omitempty"`
	Comments      []*jsonComment     `json:"comments,omitempty"`
	Trigger       *jsonTrigger       `json:"trigger,omitempty"`
	Commands      []*jsonCommand     `json:"commands,omitempty"`
	Events        []*jsonEvent       `json:"events,omitempty"`
	Fields        []*jsonField       `json:"fields,omitempty"`
	Flows         []*jsonFlow        `json:"flows,omitempty"`
	Views         []*jsonView        `json:"views,omitempty"`
	Automations   []*jsonAutomation  `json:"automations,omitempty"`
	Translations  []*jsonTranslation `json:"translations,omitempty"`
	Specs         []*jsonSpec        `json:"specs,omitempty"`
}

type jsonCommand struct {
	Name          string         `json:"name"`
	Description   string         `json:"description,omitempty"`
	Position      *jsonPosition  `json:"position,omitempty"`
	OpenPosition  *jsonPosition  `json:"open_position,omitempty"`
	ClosePosition *jsonPosition  `json:"close_position,omitempty"`
	Comments      []*jsonComment `json:"comments,omitempty"`
	Fields        []*jsonField   `json:"fields,omitempty"`
}

type jsonEvent struct {
	Name                 string         `json:"name"`
	Description          string         `json:"description,omitempty"`
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
	Description   string         `json:"description,omitempty"`
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
	Description   string         `json:"description,omitempty"`
	Position      *jsonPosition  `json:"position,omitempty"`
	OpenPosition  *jsonPosition  `json:"open_position,omitempty"`
	ClosePosition *jsonPosition  `json:"close_position,omitempty"`
	Comments      []*jsonComment `json:"comments,omitempty"`
	Fields        []*jsonField   `json:"fields,omitempty"`
	Subscribes    []string       `json:"subscribes,omitempty"`
}

type jsonAutomation struct {
	Name                  string         `json:"name"`
	Description           string         `json:"description,omitempty"`
	Position              *jsonPosition  `json:"position,omitempty"`
	OnEventPosition       *jsonPosition  `json:"on_event_position,omitempty"`
	ReadsPosition         *jsonPosition  `json:"reads_position,omitempty"`
	CommandPosition       *jsonPosition  `json:"command_position,omitempty"`
	TargetContextPosition *jsonPosition  `json:"target_context_position,omitempty"`
	OpenPosition          *jsonPosition  `json:"open_position,omitempty"`
	ClosePosition         *jsonPosition  `json:"close_position,omitempty"`
	Comments              []*jsonComment `json:"comments,omitempty"`
	OnEvent               string         `json:"on_event,omitempty"`
	Reads                 string         `json:"reads,omitempty"`
	Command               string         `json:"command,omitempty"`
	TargetContext         string         `json:"target_context,omitempty"`
}

type jsonTranslation struct {
	Name             string         `json:"name"`
	Description      string         `json:"description,omitempty"`
	Position         *jsonPosition  `json:"position,omitempty"`
	ExternalPosition *jsonPosition  `json:"external_position,omitempty"`
	ReadsPosition    *jsonPosition  `json:"reads_position,omitempty"`
	CommandPosition  *jsonPosition  `json:"command_position,omitempty"`
	OpenPosition     *jsonPosition  `json:"open_position,omitempty"`
	ClosePosition    *jsonPosition  `json:"close_position,omitempty"`
	Comments         []*jsonComment `json:"comments,omitempty"`
	ExternalSystem   string         `json:"external_system,omitempty"`
	Reads            string         `json:"reads,omitempty"`
	Command          string         `json:"command,omitempty"`
	Event            *jsonEvent     `json:"event,omitempty"`
}

type jsonSpec struct {
	Name          string           `json:"name"`
	Position      *jsonPosition    `json:"position,omitempty"`
	WhenPosition  *jsonPosition    `json:"when_position,omitempty"`
	OpenPosition  *jsonPosition    `json:"open_position,omitempty"`
	ClosePosition *jsonPosition    `json:"close_position,omitempty"`
	Comments      []*jsonComment   `json:"comments,omitempty"`
	Given         []string         `json:"given,omitempty"`
	When          string           `json:"when,omitempty"`
	Then          *jsonSpecOutcome `json:"then,omitempty"`
}

type jsonSpecOutcome struct {
	RejectedPosition *jsonPosition `json:"rejected_position,omitempty"`
	Events           []string      `json:"events,omitempty"`
	Rejected         string        `json:"rejected,omitempty"`
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

func convertList[T, U any](items []T, convert func(T) U) []U {
	if items == nil {
		return nil
	}
	out := make([]U, 0, len(items))
	for _, item := range items {
		out = append(out, convert(item))
	}
	return out
}

func convertModel(m *ast.Model) *jsonModel {
	if m == nil {
		return nil
	}
	out := &jsonModel{
		Name:        m.Name,
		Description: m.Description,
		Position:    convertPosition(m.NamePos),
		Comments:    convertComments(m.Comments),
		Actors:      convertActors(m.Actors),
		Contexts:    convertContexts(m.Contexts),
	}
	return out
}

func convertComments(comments []*ast.Comment) []*jsonComment {
	return convertList(comments, convertComment)
}

func convertComment(c *ast.Comment) *jsonComment {
	return &jsonComment{
		Text:     c.Text,
		Position: convertPosition(c.Position),
	}
}

func convertActors(actors []*ast.Actor) []*jsonActor {
	return convertList(actors, convertActor)
}

func convertActor(a *ast.Actor) *jsonActor {
	if a == nil {
		return nil
	}
	return &jsonActor{
		Name:        a.Name,
		Description: a.Description,
		Position:    convertPosition(a.NamePos),
		Comments:    convertComments(a.Comments),
	}
}

func convertContexts(ctxs []*ast.Context) []*jsonContext {
	return convertList(ctxs, convertContext)
}

func convertContext(c *ast.Context) *jsonContext {
	if c == nil {
		return nil
	}
	return &jsonContext{
		Name:          c.Name,
		Description:   c.Description,
		Position:      convertPosition(c.NamePos),
		OpenPosition:  convertPosition(c.OpenPos),
		ClosePosition: convertPosition(c.ClosePos),
		Comments:      convertComments(c.Comments),
		Invariants:    convertInvariants(c.Invariants),
		Aggregates:    convertAggregates(c.Aggregates),
		Slices:        convertSlices(c.Slices),
	}
}

func convertInvariants(invariants []*ast.Invariant) []*jsonInvariant {
	return convertList(invariants, convertInvariant)
}

func convertInvariant(inv *ast.Invariant) *jsonInvariant {
	return &jsonInvariant{
		Comments:  convertComments(inv.Comments),
		Name:      inv.Name,
		Statement: inv.Statement,
	}
}

func convertAggregates(aggs []*ast.Aggregate) []*jsonAggregate {
	return convertList(aggs, convertAggregate)
}

func convertAggregate(a *ast.Aggregate) *jsonAggregate {
	if a == nil {
		return nil
	}
	return &jsonAggregate{
		Name:          a.Name,
		Description:   a.Description,
		Position:      convertPosition(a.NamePos),
		OpenPosition:  convertPosition(a.OpenPos),
		ClosePosition: convertPosition(a.ClosePos),
		Comments:      convertComments(a.Comments),
		Invariants:    convertInvariants(a.Invariants),
		Slices:        convertSlices(a.Slices),
	}
}

func convertSlices(slices []*ast.Slice) []*jsonSlice {
	return convertList(slices, convertSlice)
}

func convertSlice(s *ast.Slice) *jsonSlice {
	if s == nil {
		return nil
	}
	return &jsonSlice{
		Name:          s.Name,
		Description:   s.Description,
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
		Specs:         convertSpecs(s.Specs),
	}
}

func convertSpecs(specs []*ast.Spec) []*jsonSpec {
	return convertList(specs, convertSpec)
}

func convertSpec(s *ast.Spec) *jsonSpec {
	if s == nil {
		return nil
	}
	return &jsonSpec{
		Name:          s.Name,
		Position:      convertPosition(s.NamePos),
		WhenPosition:  specElementPosition(s.When),
		OpenPosition:  convertPosition(s.OpenPos),
		ClosePosition: convertPosition(s.ClosePos),
		Comments:      convertComments(s.Comments),
		Given:         specElementNames(s.Given),
		When:          specElementName(s.When),
		Then:          convertSpecOutcome(s.Then),
	}
}

func convertSpecOutcome(then ast.ThenClause) *jsonSpecOutcome {
	switch t := then.(type) {
	case *ast.ThenEvents:
		return &jsonSpecOutcome{Events: specElementNames(t.Events)}
	case *ast.ThenRejected:
		return &jsonSpecOutcome{
			RejectedPosition: convertPosition(t.InvariantPos),
			Rejected:         t.InvariantName,
		}
	}
	return nil
}

func specElementNames(elements []*ast.SpecElement) []string {
	if len(elements) == 0 {
		return nil
	}
	names := make([]string, 0, len(elements))
	for _, element := range elements {
		names = append(names, element.Name)
	}
	return names
}

func specElementName(element *ast.SpecElement) string {
	if element == nil {
		return ""
	}
	return element.Name
}

func specElementPosition(element *ast.SpecElement) *jsonPosition {
	if element == nil {
		return nil
	}
	return convertPosition(element.NamePos)
}

func convertCommands(cmds []*ast.Command) []*jsonCommand {
	return convertList(cmds, convertCommand)
}

func convertCommand(c *ast.Command) *jsonCommand {
	if c == nil {
		return nil
	}
	return &jsonCommand{
		Name:          c.Name,
		Description:   c.Description,
		Position:      convertPosition(c.NamePos),
		OpenPosition:  convertPosition(c.OpenPos),
		ClosePosition: convertPosition(c.ClosePos),
		Comments:      convertComments(c.Comments),
		Fields:        convertFields(c.Fields),
	}
}

func convertEvents(events []*ast.Event) []*jsonEvent {
	return convertList(events, convertEvent)
}

func convertEvent(e *ast.Event) *jsonEvent {
	if e == nil {
		return nil
	}
	return &jsonEvent{
		Name:                 e.Name,
		Description:          e.Description,
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
	return convertList(fields, convertField)
}

func convertField(f *ast.Field) *jsonField {
	if f == nil {
		return nil
	}
	return &jsonField{
		Name:             f.Name,
		Position:         convertPosition(f.NamePos),
		TypePosition:     convertPosition(f.TypePos),
		ModifierPosition: convertPosition(f.ModPos),
		Type:             f.Type,
		Modifier:         f.Modifier,
	}
}

func convertFlows(flows []*ast.Flow) []*jsonFlow {
	return convertList(flows, convertFlow)
}

func convertFlow(f *ast.Flow) *jsonFlow {
	if f == nil {
		return nil
	}
	return &jsonFlow{
		Comments:        convertComments(f.Comments),
		CommandName:     f.CommandName,
		CommandPosition: convertPosition(f.CommandPos),
		EventName:       f.EventName,
		EventPosition:   convertPosition(f.EventPos),
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
		Description:   t.Description,
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
	return convertList(views, convertView)
}

func convertView(v *ast.View) *jsonView {
	if v == nil {
		return nil
	}
	return &jsonView{
		Name:          v.Name,
		Description:   v.Description,
		Position:      convertPosition(v.NamePos),
		OpenPosition:  convertPosition(v.OpenPos),
		ClosePosition: convertPosition(v.ClosePos),
		Comments:      convertComments(v.Comments),
		Fields:        convertFields(v.Fields),
		Subscribes:    v.Subscribes,
	}
}

func convertAutomations(autos []*ast.Automation) []*jsonAutomation {
	return convertList(autos, convertAutomation)
}

func convertAutomation(a *ast.Automation) *jsonAutomation {
	if a == nil {
		return nil
	}
	return &jsonAutomation{
		Name:                  a.Name,
		Description:           a.Description,
		Position:              convertPosition(a.NamePos),
		OnEventPosition:       convertPosition(a.OnEventPos),
		ReadsPosition:         convertPosition(a.ReadsPos),
		CommandPosition:       convertPosition(a.CommandPos),
		TargetContextPosition: convertPosition(a.TargetContextPos),
		OpenPosition:          convertPosition(a.OpenPos),
		ClosePosition:         convertPosition(a.ClosePos),
		Comments:              convertComments(a.Comments),
		OnEvent:               a.OnEvent,
		Reads:                 a.Reads,
		Command:               a.Command,
		TargetContext:         a.TargetContext,
	}
}

func convertTranslations(trans []*ast.Translation) []*jsonTranslation {
	return convertList(trans, convertTranslation)
}

func convertTranslation(t *ast.Translation) *jsonTranslation {
	if t == nil {
		return nil
	}
	return &jsonTranslation{
		Name:             t.Name,
		Description:      t.Description,
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
	Kind           string            `json:"kind,omitempty"`
	Actor          string            `json:"actor,omitempty"`
	Reads          string            `json:"reads,omitempty"`
	Subscribes     []string          `json:"subscribes,omitempty"`
	OnEvent        string            `json:"on_event,omitempty"`
	Command        string            `json:"command,omitempty"`
	TargetContext  string            `json:"target_context,omitempty"`
	ExternalSystem string            `json:"external_system,omitempty"`
	Source         string            `json:"source,omitempty"`
	ExternalName   string            `json:"external_name,omitempty"`
	Event          *jsonDiagramEvent `json:"event,omitempty"`
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

type jsonDiagramEvent struct {
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
			OnEvent:       a.OnEvent,
			Reads:         a.Reads,
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
			node.Event = convertEventToDiagram(t.Event)

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

				if a.OnEvent != "" {
					if srcID, ok := evtIDs[a.OnEvent]; ok {
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

func convertEventToDiagram(e *ast.Event) *jsonDiagramEvent {
	if e == nil {
		return nil
	}
	return &jsonDiagramEvent{
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
	writeCUEList(w, "views", s.Views, w.writeView)
	writeCUEList(w, "automations", s.Automations, w.writeAutomation)
	writeCUEList(w, "translations", s.Translations, w.writeTranslation)
	writeCUEList(w, "specs", s.Specs, w.writeSpec)
}

func (w *cueWriter) writeSpec(s *ast.Spec) {
	w.writeComments(s.Comments)
	w.line("name: %q", s.Name)
	w.listIfSet("given", specElementNames(s.Given))
	w.lineIfSet("when", specElementName(s.When))
	if s.Then != nil {
		w.writeObject("then", func() { w.writeSpecOutcome(s.Then) })
	}
}

func (w *cueWriter) writeSpecOutcome(then ast.ThenClause) {
	switch t := then.(type) {
	case *ast.ThenEvents:
		w.listIfSet("events", specElementNames(t.Events))
	case *ast.ThenRejected:
		w.lineIfSet("rejected", t.InvariantName)
	}
}

func (w *cueWriter) writeTrigger(t *ast.Trigger) {
	w.writeComments(t.Comments)
	w.line("kind: %q", t.Kind)
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

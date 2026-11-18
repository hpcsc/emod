package export

import (
	"encoding/json"
	"fmt"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/diagram"
)

// Diagram JSON intermediate types.

type jsonDiagramDocument struct {
	ModelName        string             `json:"model_name"`
	ModelDescription string             `json:"model_description,omitempty"`
	Nodes            []*jsonDiagramNode `json:"nodes"`
	Edges            []*jsonDiagramEdge `json:"edges"`
}

type jsonDiagramReads struct {
	views  []string
	isList bool
}

func readsList(views []string) jsonDiagramReads {
	return jsonDiagramReads{views: views, isList: true}
}

func readsName(view string) jsonDiagramReads {
	if view == "" {
		return jsonDiagramReads{}
	}
	return jsonDiagramReads{views: []string{view}}
}

func (r jsonDiagramReads) IsZero() bool {
	return len(r.views) == 0
}

func (r jsonDiagramReads) MarshalJSON() ([]byte, error) {
	if r.isList {
		return json.Marshal(r.views)
	}
	return json.Marshal(r.views[0])
}

type jsonDiagramNode struct {
	ID          string              `json:"id"`
	Type        string              `json:"type"`
	Label       string              `json:"label"`
	Description string              `json:"description,omitempty"`
	ParentID    *string             `json:"parentId"`
	Fields      []*jsonDiagramField `json:"fields,omitempty"`
	Position    *jsonPosition       `json:"position,omitempty"`
	Comments    []*jsonComment      `json:"comments,omitempty"`
	// Type-specific metadata for trigger, event, view, automation, translation
	Reads          jsonDiagramReads  `json:"reads,omitzero"`
	Subscribes     []string          `json:"subscribes,omitempty"`
	OnEvent        string            `json:"on_event,omitempty"`
	Schedule       string            `json:"every,omitempty"`
	After          string            `json:"after,omitempty"`
	Command        string            `json:"command,omitempty"`
	TargetContext  string            `json:"target_context,omitempty"`
	ExternalSystem string            `json:"external_system,omitempty"`
	Source         string            `json:"source,omitempty"`
	ExternalName   string            `json:"external_name,omitempty"`
	Event          *jsonDiagramEvent `json:"event,omitempty"`
	CopyOf         string            `json:"copy_of,omitempty"`
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

// jsonDiagramDiagnosticsWrapper is the top-level envelope for diagram JSON diagnostics output.
type jsonDiagramDiagnosticsWrapper struct {
	Diagnostics []*jsonDiagnosticEntry `json:"diagnostics"`
	Diagram     json.RawMessage        `json:"diagram"`
}

type jsonDiagramParsedWrapper struct {
	jsonDiagramDiagnosticsWrapper
	Parsed bool `json:"parsed"`
}

// ExportDiagramJSON serializes the given AST model to a diagram-oriented JSON byte slice.
func ExportDiagramJSON(model *ast.Model) ([]byte, error) {
	j := convertModelToDiagram(model)
	return json.Marshal(j)
}

// ExportDiagramJSONDiagnostics wraps the diagram JSON and a diagnostics slice into a structured envelope
// with top-level diagnostics array and diagram object.
func ExportDiagramJSONDiagnostics(model *ast.Model, diagnostics []*diagnostic.Entry) ([]byte, error) {
	wrapper, err := diagramDiagnosticsWrapper(model, diagnostics)
	if err != nil {
		return nil, err
	}

	return json.Marshal(wrapper)
}

// ExportDiagramJSONParsed is the ExportDiagramJSONDiagnostics envelope with a
// parsed key beside it, saying whether lexing and parsing reported nothing.
func ExportDiagramJSONParsed(model *ast.Model, diagnostics []*diagnostic.Entry, parsed bool) ([]byte, error) {
	wrapper, err := diagramDiagnosticsWrapper(model, diagnostics)
	if err != nil {
		return nil, err
	}

	return json.Marshal(jsonDiagramParsedWrapper{jsonDiagramDiagnosticsWrapper: wrapper, Parsed: parsed})
}

func diagramDiagnosticsWrapper(model *ast.Model, diagnostics []*diagnostic.Entry) (jsonDiagramDiagnosticsWrapper, error) {
	diagramJSON, err := ExportDiagramJSON(model)
	if err != nil {
		return jsonDiagramDiagnosticsWrapper{}, err
	}

	return jsonDiagramDiagnosticsWrapper{
		Diagnostics: convertDiagnostics(diagnostics),
		Diagram:     json.RawMessage(diagramJSON),
	}, nil
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

var copyTypes = map[diagram.ElementKind]string{
	diagram.ElementEvent:   "event_copy",
	diagram.ElementView:    "view_copy",
	diagram.ElementCommand: "command_copy",
}

// diagramBuilder accumulates the nodes and edges of one diagram document. The
// name→ID maps are filled while nodes are drawn and read only once every node
// exists: an edge names its endpoints by construct name, and a slice may name a
// construct another slice declares, so no edge can be drawn while nodes still
// are.
type diagramBuilder struct {
	doc *jsonDiagramDocument
	ids *diagramIDGenerator

	commandIDs     map[string]string
	eventIDs       map[string]string
	triggerIDs     map[*ast.Slice]string
	viewIDs        map[string]string
	automationIDs  map[string]string
	translationIDs map[string]string

	// declaredIDs keeps the node of each event, view and command by the slice
	// that declares it, so that an edge and a copy find the element that
	// ElementCopies names; eventIDs, viewIDs and commandIDs keep the last element
	// of a name.
	declaredIDs map[diagram.DeclaredElement]string
	copies      diagram.ElementCopies
	copyIDs     map[*ast.Slice]map[diagram.DeclaredElement]string
	copyNodes   map[*jsonDiagramNode]diagram.DeclaredElement
}

func newDiagramBuilder(modelName, modelDescription string, copies diagram.ElementCopies) *diagramBuilder {
	return &diagramBuilder{
		doc: &jsonDiagramDocument{
			ModelName:        modelName,
			ModelDescription: modelDescription,
			Nodes:            make([]*jsonDiagramNode, 0),
			Edges:            make([]*jsonDiagramEdge, 0),
		},
		ids:            newDiagramIDGenerator(),
		commandIDs:     make(map[string]string),
		eventIDs:       make(map[string]string),
		triggerIDs:     make(map[*ast.Slice]string),
		viewIDs:        make(map[string]string),
		automationIDs:  make(map[string]string),
		translationIDs: make(map[string]string),
		declaredIDs:    make(map[diagram.DeclaredElement]string),
		copies:         copies,
		copyIDs:        make(map[*ast.Slice]map[diagram.DeclaredElement]string),
		copyNodes:      make(map[*jsonDiagramNode]diagram.DeclaredElement),
	}
}

func (b *diagramBuilder) appendNode(n *jsonDiagramNode) {
	b.doc.Nodes = append(b.doc.Nodes, n)
}

func (b *diagramBuilder) appendContext(c *ast.Context) {
	if c == nil {
		return
	}
	ctxID := b.ids.next("context")
	b.appendNode(&jsonDiagramNode{
		ID:          ctxID,
		Type:        "context",
		Label:       c.Name,
		Description: c.Description,
		ParentID:    nil,
		Position:    convertPosition(c.NamePos),
		Comments:    convertComments(c.Comments),
	})

	for _, agg := range c.Aggregates {
		b.appendAggregate(agg, ctxID)
	}
	for _, s := range c.Slices {
		b.appendSlice(s, ctxID)
	}
}

func (b *diagramBuilder) appendAggregate(agg *ast.Aggregate, ctxID string) {
	if agg == nil {
		return
	}
	aggID := b.ids.next("aggregate")
	b.appendNode(&jsonDiagramNode{
		ID:          aggID,
		Type:        "aggregate",
		Label:       agg.Name,
		Description: agg.Description,
		ParentID:    &ctxID,
		Position:    convertPosition(agg.NamePos),
		Comments:    convertComments(agg.Comments),
	})

	for _, s := range agg.Slices {
		b.appendSlice(s, aggID)
	}
}

func (b *diagramBuilder) appendSlice(s *ast.Slice, parentID string) {
	if s == nil {
		return
	}
	sliceID := b.ids.next("slice")
	b.appendNode(&jsonDiagramNode{
		ID:          sliceID,
		Type:        "slice",
		Label:       s.Name,
		Description: s.Description,
		ParentID:    &parentID,
		Position:    convertPosition(s.NamePos),
		Comments:    convertComments(s.Comments),
	})

	b.appendCommands(s, sliceID)
	b.appendEvents(s, sliceID)
	b.appendTrigger(s, sliceID)
	b.appendViews(s, sliceID)
	b.appendAutomations(s.Automations, sliceID)
	b.appendTranslations(s, sliceID)
	b.appendCopies(s, sliceID)
}

func (b *diagramBuilder) appendCopies(s *ast.Slice, sliceID string) {
	copies := b.copies.In(s)
	if len(copies) == 0 {
		return
	}

	b.copyIDs[s] = make(map[diagram.DeclaredElement]string, len(copies))
	for _, c := range copies {
		typ := copyTypes[c.Kind]
		node := &jsonDiagramNode{
			ID:       b.ids.next(typ),
			Type:     typ,
			Label:    c.Name,
			ParentID: &sliceID,
		}
		b.copyIDs[s][c] = node.ID
		b.copyNodes[node] = c
		b.appendNode(node)
	}
}

// pointCopiesAtOriginals fills in the element each copy stands for: the element
// in the slice the copy names, which the SVG names too. A copy can be drawn in a
// slice written before the one that declares its element, so this waits until
// every element has a node.
func (b *diagramBuilder) pointCopiesAtOriginals() {
	for node, c := range b.copyNodes {
		node.CopyOf = b.declaredIDs[c]
	}
}

// keepDeclared records id as the node of the element of kind declared under name
// in slice s, unless one came before it there.
func (b *diagramBuilder) keepDeclared(s *ast.Slice, kind diagram.ElementKind, name, id string) {
	key := diagram.DeclaredElement{Kind: kind, Name: name, Slice: s}
	if _, seen := b.declaredIDs[key]; !seen {
		b.declaredIDs[key] = id
	}
}

func (b *diagramBuilder) appendCommands(s *ast.Slice, sliceID string) {
	for _, cmd := range s.Commands {
		if cmd == nil {
			continue
		}
		cmdID := b.ids.next("command")
		b.commandIDs[cmd.Name] = cmdID
		b.keepDeclared(s, diagram.ElementCommand, cmd.Name, cmdID)
		node := &jsonDiagramNode{
			ID:          cmdID,
			Type:        "command",
			Label:       cmd.Name,
			Description: cmd.Description,
			ParentID:    &sliceID,
			Position:    convertPosition(cmd.NamePos),
			Comments:    convertComments(cmd.Comments),
		}
		if len(cmd.Fields) > 0 {
			node.Fields = convertFieldsToDiagram(cmd.Fields)
		}
		b.appendNode(node)
	}
}

func (b *diagramBuilder) appendEvents(s *ast.Slice, sliceID string) {
	for _, evt := range s.Events {
		b.appendEvent(s, evt, sliceID)
	}
}

func (b *diagramBuilder) appendEvent(s *ast.Slice, evt *ast.Event, sliceID string) {
	if evt == nil {
		return
	}
	evtID := b.ids.next("event")
	b.eventIDs[evt.Name] = evtID
	b.keepDeclared(s, diagram.ElementEvent, evt.Name, evtID)
	node := &jsonDiagramNode{
		ID:           evtID,
		Type:         "event",
		Label:        evt.Name,
		Description:  evt.Description,
		ParentID:     &sliceID,
		Position:     convertPosition(evt.NamePos),
		Comments:     convertComments(evt.Comments),
		Source:       evt.Source,
		ExternalName: evt.ExternalName,
	}
	if len(evt.Fields) > 0 {
		node.Fields = convertFieldsToDiagram(evt.Fields)
	}
	b.appendNode(node)
}

func (b *diagramBuilder) appendTrigger(s *ast.Slice, sliceID string) {
	t := s.Trigger
	if t == nil {
		return
	}
	tID := b.ids.next("trigger")
	b.triggerIDs[s] = tID
	b.appendNode(&jsonDiagramNode{
		ID:          tID,
		Type:        "trigger",
		Label:       t.Name,
		Description: t.Description,
		ParentID:    &sliceID,
		Position:    convertPosition(t.NamePos),
		Comments:    convertComments(t.Comments),
		Reads:       readsList(t.Reads),
	})
}

func (b *diagramBuilder) appendViews(s *ast.Slice, sliceID string) {
	for _, v := range s.Views {
		if v == nil {
			continue
		}
		vID := b.ids.next("view")
		b.viewIDs[v.Name] = vID
		b.keepDeclared(s, diagram.ElementView, v.Name, vID)
		node := &jsonDiagramNode{
			ID:          vID,
			Type:        "view",
			Label:       v.Name,
			Description: v.Description,
			ParentID:    &sliceID,
			Position:    convertPosition(v.NamePos),
			Comments:    convertComments(v.Comments),
			Subscribes:  v.Subscribes,
		}
		if len(v.Fields) > 0 {
			node.Fields = convertFieldsToDiagram(v.Fields)
		}
		b.appendNode(node)
	}
}

func (b *diagramBuilder) appendAutomations(automations []*ast.Automation, sliceID string) {
	for _, a := range automations {
		if a == nil {
			continue
		}
		aID := b.ids.next("auto")
		b.automationIDs[a.Name] = aID
		b.appendNode(&jsonDiagramNode{
			ID:            aID,
			Type:          "automation",
			Label:         a.Name,
			Description:   a.Description,
			ParentID:      &sliceID,
			Position:      convertPosition(a.NamePos),
			Comments:      convertComments(a.Comments),
			OnEvent:       a.OnEvent,
			Schedule:      a.Schedule,
			After:         a.After,
			Reads:         readsName(a.Reads),
			Command:       a.Command,
			TargetContext: a.TargetContext,
		})
	}
}

func (b *diagramBuilder) appendTranslations(s *ast.Slice, sliceID string) {
	for _, t := range s.Translations {
		if t == nil {
			continue
		}
		tID := b.ids.next("trans")
		b.translationIDs[t.Name] = tID
		node := &jsonDiagramNode{
			ID:             tID,
			Type:           "translation",
			Label:          t.Name,
			Description:    t.Description,
			ParentID:       &sliceID,
			Position:       convertPosition(t.NamePos),
			Comments:       convertComments(t.Comments),
			ExternalSystem: t.ExternalSystem,
			Reads:          readsName(t.Reads),
			Command:        t.Command,
		}
		if t.Event != nil {
			node.Event = convertEventToDiagram(t.Event)

			// The event a translation nests also gets a node of its own: the
			// translation_event edge needs an endpoint to point at.
			b.appendEvent(s, t.Event, sliceID)
		}
		b.appendNode(node)
	}
}

// appendEdges draws the arrows every slice of the model declares. They come
// from diagram.SliceEdges, the same derivation the renderers draw from, so this
// JSON and the drawn diagrams cannot disagree about which arrows exist.
func (b *diagramBuilder) appendEdges(m *ast.Model) {
	for _, ref := range m.SliceRefs() {
		for _, e := range diagram.SliceEdges(ref.Slice) {
			switch e.Kind {
			case diagram.EdgeFlow, diagram.EdgeTranslationFlow:
				b.link(b.copiedEndIDs(ref.Slice, e, b.commandIDs), e.From, b.eventIDs, e.To, "flow")

			case diagram.EdgeTriggerReads:
				b.link(b.copiedEndIDs(ref.Slice, e, b.viewIDs), e.From, b.triggerIDsOf(ref.Slice), e.To, "reads")

			case diagram.EdgeAutomationReads:
				b.link(b.copiedEndIDs(ref.Slice, e, b.viewIDs), e.From, b.automationIDs, e.To, "reads")

			case diagram.EdgeTranslationReads:
				b.link(b.copiedEndIDs(ref.Slice, e, b.viewIDs), e.From, b.translationIDs, e.To, "reads")

			case diagram.EdgeTriggerCommand:
				command := diagram.DeclaredElement{Kind: diagram.ElementCommand, Name: e.To, Slice: ref.Slice}
				b.link(b.triggerIDsOf(ref.Slice), e.From, b.declaredIDsOf(command), e.To, "trigger_command")

			case diagram.EdgeSubscription:
				b.link(b.copiedEndIDs(ref.Slice, e, b.eventIDs), e.From, b.viewIDs, e.To, "subscription")

			case diagram.EdgeAutomationTrigger:
				b.link(b.copiedEndIDs(ref.Slice, e, b.eventIDs), e.From, b.automationIDs, e.To, "automation_trigger")

			case diagram.EdgeAutomationCommand:
				b.link(b.automationIDs, e.From, b.copiedEndIDs(ref.Slice, e, b.commandIDs), e.To, "automation_command")

			case diagram.EdgeTranslationCommand:
				b.link(b.translationIDs, e.From, b.copiedEndIDs(ref.Slice, e, b.commandIDs), e.To, "translation_command")

			case diagram.EdgeTranslationExternal:
				// External systems are not diagram-JSON nodes; the translation
				// node stands in for them, so this edge has no representation.

			case diagram.EdgeRejection:
				// Invariants are not diagram-JSON nodes, so this edge has no
				// representation either. Giving one a node would oblige a
				// palette entry, an EDGE_TYPE_BY_ENDS pairing, an edgeConfig
				// pair, a detail-panel section and a foldEdges arm for a
				// construct the viewer cannot edit.
			}
		}
	}
}

func (b *diagramBuilder) triggerIDsOf(s *ast.Slice) map[string]string {
	return map[string]string{s.Trigger.Name: b.triggerIDs[s]}
}

func (b *diagramBuilder) declaredIDsOf(element diagram.DeclaredElement) map[string]string {
	return map[string]string{element.Name: b.declaredIDs[element]}
}

// copiedEndIDs returns the node ids to find an end of edge, an arrow of slice s,
// in: the node of the element that ElementCopies.Joined names, or of the copy of
// it that s draws. A name no slice declares is found in ids. Pass it for the end
// of edge that a copy can stand in for.
func (b *diagramBuilder) copiedEndIDs(s *ast.Slice, edge diagram.Edge, ids map[string]string) map[string]string {
	element, copied, found := b.copies.Joined(s, edge)
	switch {
	case copied:
		return map[string]string{element.Name: b.copyIDs[s][element]}
	case found:
		return b.declaredIDsOf(element)
	}

	return ids
}

// link draws an edge between the nodes two named constructs were drawn as. An
// endpoint naming a construct no node was drawn for leaves the edge out rather
// than dangling: an export runs on models that have not passed validation, so
// an unresolved name is a diagnostic elsewhere, not broken output here.
func (b *diagramBuilder) link(fromIDs map[string]string, from string, toIDs map[string]string, to string, edgeType string) {
	fromID, fromDrawn := fromIDs[from]
	toID, toDrawn := toIDs[to]
	if !fromDrawn || !toDrawn {
		return
	}
	b.doc.Edges = append(b.doc.Edges, &jsonDiagramEdge{
		Source: fromID,
		Target: toID,
		Type:   edgeType,
	})
}

func convertModelToDiagram(m *ast.Model) *jsonDiagramDocument {
	if m == nil {
		return nil
	}

	b := newDiagramBuilder(m.Name, m.Description, diagram.NewElementCopies(m))
	for _, c := range m.Contexts {
		b.appendContext(c)
	}
	b.pointCopiesAtOriginals()
	b.appendEdges(m)

	return b.doc
}

func convertEventToDiagram(e *ast.Event) *jsonDiagramEvent {
	if e == nil {
		return nil
	}
	return &jsonDiagramEvent{
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

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
	Actor          string            `json:"actor,omitempty"`
	Reads          string            `json:"reads,omitempty"`
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

type declaredElement struct {
	slice string
	kind  diagram.CopyKind
	name  string
}

var copyTypes = map[diagram.CopyKind]string{
	diagram.CopyOfEvent:   "event_copy",
	diagram.CopyOfView:    "view_copy",
	diagram.CopyOfCommand: "command_copy",
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
	triggerIDs     map[string]string
	viewIDs        map[string]string
	automationIDs  map[string]string
	translationIDs map[string]string

	// declaredIDs keeps the node of each element a copy can stand for, by the
	// slice that declares it, so that a copy points at the element in the slice
	// it names; eventIDs, viewIDs and commandIDs keep the last element of a name.
	declaredIDs   map[declaredElement]string
	sliceNameByID map[string]string
	copies        diagram.ElementCopies
	copyIDs       map[*ast.Slice]map[diagram.ElementCopy]string
	copyNodes     map[*jsonDiagramNode]diagram.ElementCopy
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
		triggerIDs:     make(map[string]string),
		viewIDs:        make(map[string]string),
		automationIDs:  make(map[string]string),
		translationIDs: make(map[string]string),
		declaredIDs:    make(map[declaredElement]string),
		sliceNameByID:  make(map[string]string),
		copies:         copies,
		copyIDs:        make(map[*ast.Slice]map[diagram.ElementCopy]string),
		copyNodes:      make(map[*jsonDiagramNode]diagram.ElementCopy),
	}
}

func (b *diagramBuilder) appendNode(n *jsonDiagramNode) {
	b.doc.Nodes = append(b.doc.Nodes, n)
}

func (b *diagramBuilder) appendActor(a *ast.Actor) {
	if a == nil {
		return
	}
	b.appendNode(&jsonDiagramNode{
		ID:          b.ids.next("actor"),
		Type:        "actor",
		Label:       a.Name,
		Description: a.Description,
		ParentID:    nil,
		Position:    convertPosition(a.NamePos),
		Comments:    convertComments(a.Comments),
	})
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
	b.sliceNameByID[sliceID] = s.Name
	b.appendNode(&jsonDiagramNode{
		ID:          sliceID,
		Type:        "slice",
		Label:       s.Name,
		Description: s.Description,
		ParentID:    &parentID,
		Position:    convertPosition(s.NamePos),
		Comments:    convertComments(s.Comments),
	})

	b.appendCommands(s.Commands, sliceID)
	b.appendEvents(s.Events, sliceID)
	b.appendTrigger(s.Trigger, sliceID)
	b.appendViews(s.Views, sliceID)
	b.appendAutomations(s.Automations, sliceID)
	b.appendTranslations(s.Translations, sliceID)
	b.appendCopies(s, sliceID)
}

func (b *diagramBuilder) appendCopies(s *ast.Slice, sliceID string) {
	copies := b.copies.In(s)
	if len(copies) == 0 {
		return
	}

	b.copyIDs[s] = make(map[diagram.ElementCopy]string, len(copies))
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
		node.CopyOf = b.nodeOf(c)
	}
}

// nodeOf is the id of the node drawn for the element c stands for, in the slice
// c names.
func (b *diagramBuilder) nodeOf(c diagram.ElementCopy) string {
	return b.declaredIDs[declaredElement{slice: c.DeclaredIn, kind: c.Kind, name: c.Name}]
}

// keepDeclared records id as the node of the element of kind declared under name
// in the slice whose node is sliceID, unless one came before it there.
func (b *diagramBuilder) keepDeclared(sliceID string, kind diagram.CopyKind, name, id string) {
	key := declaredElement{slice: b.sliceNameByID[sliceID], kind: kind, name: name}
	if _, seen := b.declaredIDs[key]; !seen {
		b.declaredIDs[key] = id
	}
}

func (b *diagramBuilder) appendCommands(commands []*ast.Command, sliceID string) {
	for _, cmd := range commands {
		if cmd == nil {
			continue
		}
		cmdID := b.ids.next("command")
		b.commandIDs[cmd.Name] = cmdID
		b.keepDeclared(sliceID, diagram.CopyOfCommand, cmd.Name, cmdID)
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

func (b *diagramBuilder) appendEvents(events []*ast.Event, sliceID string) {
	for _, evt := range events {
		b.appendEvent(evt, sliceID)
	}
}

func (b *diagramBuilder) appendEvent(evt *ast.Event, sliceID string) {
	if evt == nil {
		return
	}
	evtID := b.ids.next("event")
	b.eventIDs[evt.Name] = evtID
	b.keepDeclared(sliceID, diagram.CopyOfEvent, evt.Name, evtID)
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

func (b *diagramBuilder) appendTrigger(t *ast.Trigger, sliceID string) {
	if t == nil {
		return
	}
	tID := b.ids.next("trigger")
	b.triggerIDs[t.Name] = tID
	b.appendNode(&jsonDiagramNode{
		ID:          tID,
		Type:        "trigger",
		Label:       t.Name,
		Description: t.Description,
		ParentID:    &sliceID,
		Position:    convertPosition(t.NamePos),
		Comments:    convertComments(t.Comments),
		Actor:       t.Actor,
		Reads:       t.Reads,
	})
}

func (b *diagramBuilder) appendViews(views []*ast.View, sliceID string) {
	for _, v := range views {
		if v == nil {
			continue
		}
		vID := b.ids.next("view")
		b.viewIDs[v.Name] = vID
		b.keepDeclared(sliceID, diagram.CopyOfView, v.Name, vID)
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
			Reads:         a.Reads,
			Command:       a.Command,
			TargetContext: a.TargetContext,
		})
	}
}

func (b *diagramBuilder) appendTranslations(translations []*ast.Translation, sliceID string) {
	for _, t := range translations {
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
			Reads:          t.Reads,
			Command:        t.Command,
		}
		if t.Event != nil {
			node.Event = convertEventToDiagram(t.Event)

			// The event a translation nests also gets a node of its own: the
			// translation_event edge needs an endpoint to point at.
			b.appendEvent(t.Event, sliceID)
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
				b.link(b.copiedEndIDs(ref.Slice, e, b.viewIDs), e.From, b.triggerIDs, e.To, "reads")

			case diagram.EdgeAutomationReads:
				b.link(b.copiedEndIDs(ref.Slice, e, b.viewIDs), e.From, b.automationIDs, e.To, "reads")

			case diagram.EdgeTranslationReads:
				b.link(b.copiedEndIDs(ref.Slice, e, b.viewIDs), e.From, b.translationIDs, e.To, "reads")

			case diagram.EdgeTriggerCommand:
				b.link(b.triggerIDs, e.From, b.commandIDs, e.To, "trigger_command")

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

// copiedEndIDs returns the node ids to find an end of edge, an arrow of slice s,
// in: the copy s draws in place of the element at that end, or else the element
// in the first slice that declares it, which the copy rule measures from. A name
// no slice declares is found in ids. Pass it for the end of edge that a copy can
// stand in for.
func (b *diagramBuilder) copiedEndIDs(s *ast.Slice, edge diagram.Edge, ids map[string]string) map[string]string {
	if copied, found := b.copies.At(s, edge); found {
		return map[string]string{copied.Name: b.copyIDs[s][copied]}
	}
	if element, declared := b.copies.Joined(s, edge); declared {
		return map[string]string{element.Name: b.nodeOf(element)}
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
	for _, a := range m.Actors {
		b.appendActor(a)
	}
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

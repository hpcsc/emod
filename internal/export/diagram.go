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
	Actor          string            `json:"actor,omitempty"`
	Reads          string            `json:"reads,omitempty"`
	Subscribes     []string          `json:"subscribes,omitempty"`
	OnEvent        string            `json:"on_event,omitempty"`
	Schedule       string            `json:"every,omitempty"`
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
			Schedule:      a.Schedule,
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
	// The semantic edges come from diagram.SliceEdges, the same derivation the
	// renderers draw from, so this JSON and the drawn diagrams cannot disagree
	// about which arrows exist. Unresolved references are silently skipped
	// (no panic, no broken output).

	appendEdge := func(srcID string, srcOK bool, tgtID string, tgtOK bool, edgeType string) {
		if !srcOK || !tgtOK {
			return
		}
		doc.Edges = append(doc.Edges, &jsonDiagramEdge{
			Source: srcID,
			Target: tgtID,
			Type:   edgeType,
		})
	}
	readsEdge := func(viewName string, readerID string, readerOK bool) {
		viewID, declared := viewIDs[viewName]
		appendEdge(viewID, declared, readerID, readerOK, "reads")
	}
	resolved := func(ids map[string]string, name string) (string, bool) {
		id, ok := ids[name]
		return id, ok
	}

	for _, ref := range m.SliceRefs() {
		for _, e := range diagram.SliceEdges(ref.Slice) {
			switch e.Kind {
			case diagram.EdgeFlow, diagram.EdgeTranslationFlow:
				srcID, srcOK := resolved(cmdIDs, e.From)
				tgtID, tgtOK := resolved(evtIDs, e.To)
				appendEdge(srcID, srcOK, tgtID, tgtOK, "flow")

			case diagram.EdgeTriggerReads:
				readerID, ok := resolved(triggerIDs, e.To)
				readsEdge(e.From, readerID, ok)

			case diagram.EdgeAutomationReads:
				readerID, ok := resolved(autoIDs, e.To)
				readsEdge(e.From, readerID, ok)

			case diagram.EdgeTranslationReads:
				readerID, ok := resolved(transIDs, e.To)
				readsEdge(e.From, readerID, ok)

			case diagram.EdgeTriggerCommand:
				srcID, srcOK := resolved(triggerIDs, e.From)
				tgtID, tgtOK := resolved(cmdIDs, e.To)
				appendEdge(srcID, srcOK, tgtID, tgtOK, "trigger_command")

			case diagram.EdgeSubscription:
				srcID, srcOK := resolved(evtIDs, e.From)
				tgtID, tgtOK := resolved(viewIDs, e.To)
				appendEdge(srcID, srcOK, tgtID, tgtOK, "subscription")

			case diagram.EdgeAutomationTrigger:
				srcID, srcOK := resolved(evtIDs, e.From)
				tgtID, tgtOK := resolved(autoIDs, e.To)
				appendEdge(srcID, srcOK, tgtID, tgtOK, "automation_trigger")

			case diagram.EdgeAutomationCommand:
				srcID, srcOK := resolved(autoIDs, e.From)
				tgtID, tgtOK := resolved(cmdIDs, e.To)
				appendEdge(srcID, srcOK, tgtID, tgtOK, "automation_command")

			case diagram.EdgeTranslationCommand:
				srcID, srcOK := resolved(transIDs, e.From)
				tgtID, tgtOK := resolved(cmdIDs, e.To)
				appendEdge(srcID, srcOK, tgtID, tgtOK, "translation_command")

			case diagram.EdgeTranslationExternal:
				// External systems are not diagram-JSON nodes; the translation
				// node stands in for them, so this edge has no representation.
			}
		}
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

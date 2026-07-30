// Package importer converts diagram JSON — the {model_name, nodes, edges}
// document produced by export.ExportDiagramJSON and edited by the viewer —
// back into an AST model, so that formatter.Format is the single writer of
// .emod text.
package importer

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/hpcsc/emod/internal/ast"
)

// defaultTriggerKind is used when a trigger node carries no kind, so that the
// formatted output stays parseable instead of emitting `trigger  "name"`.
const defaultTriggerKind = "UI"

type diagramDocument struct {
	ModelName string         `json:"model_name"`
	Nodes     []*diagramNode `json:"nodes"`
	Edges     []*diagramEdge `json:"edges"`
}

type diagramNode struct {
	ID             string          `json:"id"`
	Type           string          `json:"type"`
	Label          string          `json:"label"`
	ParentID       *string         `json:"parentId"`
	Fields         []*diagramField `json:"fields,omitempty"`
	Kind           string          `json:"kind,omitempty"`
	Actor          string          `json:"actor,omitempty"`
	Reads          string          `json:"reads,omitempty"`
	Subscribes     []string        `json:"subscribes,omitempty"`
	TriggerEvent   string          `json:"trigger_event,omitempty"`
	Command        string          `json:"command,omitempty"`
	TargetContext  string          `json:"target_context,omitempty"`
	ExternalSystem string          `json:"external_system,omitempty"`
	Source         string          `json:"source,omitempty"`
	ExternalName   string          `json:"external_name,omitempty"`
	Event          *diagramEvent   `json:"event,omitempty"`
}

type diagramEvent struct {
	Name         string          `json:"name"`
	Source       string          `json:"source,omitempty"`
	ExternalName string          `json:"external_name,omitempty"`
	Fields       []*diagramField `json:"fields,omitempty"`
}

type diagramField struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Modifier string `json:"modifier,omitempty"`
}

type diagramEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
}

// ImportDiagram converts a diagram JSON document into an AST model.
//
// Diagram JSON carries no comments, context modes, event tags, or decides_on
// clauses, so those are absent from the result even if the model the diagram
// was exported from had them.
func ImportDiagram(data []byte) (*ast.Model, error) {
	var doc diagramDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("importer: invalid diagram JSON: %w", err)
	}

	b := &builder{
		nodeByID:   make(map[string]*diagramNode, len(doc.Nodes)),
		childrenOf: make(map[string][]*diagramNode),
		sliceOf:    make(map[string]*ast.Slice),
		byNode:     make(map[string]any),
	}

	for _, n := range doc.Nodes {
		if n == nil || n.ID == "" {
			continue
		}
		b.nodeByID[n.ID] = n
		if n.ParentID != nil {
			b.childrenOf[*n.ParentID] = append(b.childrenOf[*n.ParentID], n)
		}
	}

	model := &ast.Model{Name: doc.ModelName}

	for _, n := range doc.Nodes {
		if n == nil || n.ParentID != nil {
			continue
		}
		switch n.Type {
		case "actor":
			model.Actors = append(model.Actors, &ast.Actor{Name: n.Label})
		case "context":
			model.Contexts = append(model.Contexts, b.buildContext(n))
		}
	}

	b.foldEdges(doc.Edges)

	return model, nil
}

type builder struct {
	nodeByID   map[string]*diagramNode
	childrenOf map[string][]*diagramNode

	// sliceOf maps a node ID to the slice that node belongs to, so edges can be
	// attributed to the right slice without a second tree walk.
	sliceOf map[string]*ast.Slice
	// byNode maps a node ID to the AST value built from it (*ast.Command,
	// *ast.View, *ast.Automation, *ast.Translation, ...).
	byNode map[string]any
}

func (b *builder) children(parentID, typ string) []*diagramNode {
	var out []*diagramNode
	for _, n := range b.childrenOf[parentID] {
		if n.Type == typ {
			out = append(out, n)
		}
	}
	return out
}

func (b *builder) buildContext(n *diagramNode) *ast.Context {
	ctx := &ast.Context{Name: n.Label}
	for _, aggNode := range b.children(n.ID, "aggregate") {
		agg := &ast.Aggregate{Name: aggNode.Label}
		for _, sliceNode := range b.children(aggNode.ID, "slice") {
			agg.Slices = append(agg.Slices, b.buildSlice(sliceNode))
		}
		ctx.Aggregates = append(ctx.Aggregates, agg)
	}
	for _, sliceNode := range b.children(n.ID, "slice") {
		ctx.Slices = append(ctx.Slices, b.buildSlice(sliceNode))
	}
	return ctx
}

func (b *builder) buildSlice(n *diagramNode) *ast.Slice {
	slice := &ast.Slice{Name: n.Label}

	// Translation events are re-emitted inside their translation block, so the
	// standalone event node the exporter adds for them must not become a
	// second, duplicate top-level event declaration.
	translationEvents := make(map[string]bool)
	for _, t := range b.children(n.ID, "translation") {
		if t.Event != nil && t.Event.Name != "" {
			translationEvents[t.Event.Name] = true
		}
	}

	if triggers := b.children(n.ID, "trigger"); len(triggers) > 0 {
		t := triggers[0]
		kind := t.Kind
		if kind == "" {
			kind = defaultTriggerKind
		}
		slice.Trigger = &ast.Trigger{
			Kind:  kind,
			Name:  t.Label,
			Actor: t.Actor,
			Reads: t.Reads,
		}
		b.register(t.ID, slice, slice.Trigger)
	}

	for _, c := range b.children(n.ID, "command") {
		cmd := &ast.Command{Name: c.Label, Fields: convertFields(c.Fields)}
		slice.Commands = append(slice.Commands, cmd)
		b.register(c.ID, slice, cmd)
	}

	for _, e := range b.children(n.ID, "event") {
		if translationEvents[e.Label] {
			b.register(e.ID, slice, nil)
			continue
		}
		evt := &ast.Event{
			Name:         e.Label,
			Source:       e.Source,
			ExternalName: e.ExternalName,
			Fields:       convertFields(e.Fields),
		}
		slice.Events = append(slice.Events, evt)
		b.register(e.ID, slice, evt)
	}

	for _, v := range b.children(n.ID, "view") {
		view := &ast.View{
			Name:       v.Label,
			Fields:     convertFields(v.Fields),
			Subscribes: append([]string(nil), v.Subscribes...),
		}
		slice.Views = append(slice.Views, view)
		b.register(v.ID, slice, view)
	}

	for _, a := range b.children(n.ID, "automation") {
		auto := &ast.Automation{
			Name:          a.Label,
			TriggerEvent:  a.TriggerEvent,
			Reads:         a.Reads,
			Command:       a.Command,
			TargetContext: a.TargetContext,
		}
		slice.Automations = append(slice.Automations, auto)
		b.register(a.ID, slice, auto)
	}

	for _, t := range b.children(n.ID, "translation") {
		trans := &ast.Translation{
			Name:           t.Label,
			ExternalSystem: t.ExternalSystem,
			Reads:          t.Reads,
			Command:        t.Command,
		}
		if t.Event != nil {
			trans.Event = &ast.Event{
				Name:         t.Event.Name,
				Source:       t.Event.Source,
				ExternalName: t.Event.ExternalName,
				Fields:       convertFields(t.Event.Fields),
			}
		}
		slice.Translations = append(slice.Translations, trans)
		b.register(t.ID, slice, trans)
	}

	b.register(n.ID, slice, slice)
	return slice
}

func (b *builder) register(nodeID string, slice *ast.Slice, value any) {
	b.sliceOf[nodeID] = slice
	if value != nil {
		b.byNode[nodeID] = value
	}
}

// foldEdges writes edge-carried relationships back into the AST. Node metadata
// already covers relationships that survived a round trip through the exporter;
// edges are the only record of connections drawn in the viewer, which never
// update that metadata.
func (b *builder) foldEdges(edges []*diagramEdge) {
	for _, e := range edges {
		if e == nil {
			continue
		}
		src, srcOK := b.nodeByID[e.Source]
		tgt, tgtOK := b.nodeByID[e.Target]
		if !srcOK || !tgtOK {
			continue
		}

		switch e.Type {
		case "flow":
			b.foldFlow(src, tgt)
		case "subscription":
			if view, ok := b.byNode[tgt.ID].(*ast.View); ok {
				view.Subscribes = appendMissing(view.Subscribes, src.Label)
			}
		case "automation_trigger":
			if auto, ok := b.byNode[tgt.ID].(*ast.Automation); ok && auto.TriggerEvent == "" {
				auto.TriggerEvent = src.Label
			}
		case "automation_command":
			if auto, ok := b.byNode[src.ID].(*ast.Automation); ok && auto.Command == "" {
				auto.Command = tgt.Label
			}
		case "translation_command":
			if trans, ok := b.byNode[src.ID].(*ast.Translation); ok && trans.Command == "" {
				trans.Command = tgt.Label
			}
		case "reads":
			if trans, ok := b.byNode[tgt.ID].(*ast.Translation); ok && trans.Reads == "" {
				trans.Reads = src.Label
			}
		}
	}
}

func (b *builder) foldFlow(src, tgt *diagramNode) {
	if src.Type != "command" || tgt.Type != "event" {
		return
	}
	slice, ok := b.sliceOf[src.ID]
	if !ok || slice != b.sliceOf[tgt.ID] {
		return
	}
	// The exporter derives a command→event flow edge for every translation, but
	// the translation block already expresses that pairing; re-emitting it as a
	// flow entry would add a line the source never had.
	for _, trans := range slice.Translations {
		if trans.Command == src.Label && trans.Event != nil && trans.Event.Name == tgt.Label {
			return
		}
	}
	for _, f := range slice.Flows {
		if f.CommandName == src.Label && f.EventName == tgt.Label {
			return
		}
	}
	slice.Flows = append(slice.Flows, &ast.Flow{CommandName: src.Label, EventName: tgt.Label})
}

func appendMissing(existing []string, value string) []string {
	if slices.Contains(existing, value) {
		return existing
	}
	return append(existing, value)
}

func convertFields(fields []*diagramField) []*ast.Field {
	if len(fields) == 0 {
		return nil
	}
	out := make([]*ast.Field, 0, len(fields))
	for _, f := range fields {
		if f == nil {
			continue
		}
		out = append(out, &ast.Field{Name: f.Name, Type: f.Type, Modifier: f.Modifier})
	}
	return out
}

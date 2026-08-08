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

type diagramDocument struct {
	ModelName        string         `json:"model_name"`
	ModelDescription string         `json:"model_description,omitempty"`
	Nodes            []*diagramNode `json:"nodes"`
	Edges            []*diagramEdge `json:"edges"`
}

type diagramNode struct {
	ID             string            `json:"id"`
	Type           string            `json:"type"`
	Label          string            `json:"label"`
	Description    string            `json:"description,omitempty"`
	ParentID       *string           `json:"parentId"`
	Fields         []*diagramField   `json:"fields,omitempty"`
	Comments       []*diagramComment `json:"comments,omitempty"`
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
	Event          *diagramEvent     `json:"event,omitempty"`
}

type diagramEvent struct {
	Name         string            `json:"name"`
	Description  string            `json:"description,omitempty"`
	Comments     []*diagramComment `json:"comments,omitempty"`
	Source       string            `json:"source,omitempty"`
	ExternalName string            `json:"external_name,omitempty"`
	Fields       []*diagramField   `json:"fields,omitempty"`
}

// diagramComment drops the position the exporter writes beside each comment: it
// points into the file the diagram was exported from, and a viewer save rewrites
// every line of that file.
type diagramComment struct {
	Text string `json:"text"`
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
// Diagram JSON carries no context modes, event tags or decides_on clauses, so
// those are absent from the result even if the model the diagram was exported
// from had them. A comment survives wherever a node stands for the construct it
// was written on, and is lost where none does — on the model itself, on an
// invariant, a spec, a flow or a decides_on clause.
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

	model := &ast.Model{Name: doc.ModelName, Description: doc.ModelDescription}

	for _, n := range doc.Nodes {
		if n == nil || n.ParentID != nil {
			continue
		}
		switch n.Type {
		case "actor":
			model.Actors = append(model.Actors, &ast.Actor{
				Name:        n.Label,
				Description: n.Description,
				Comments:    convertComments(n.Comments),
			})
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
	ctx := &ast.Context{
		Name:        n.Label,
		Description: n.Description,
		Comments:    convertComments(n.Comments),
	}
	for _, aggNode := range b.children(n.ID, "aggregate") {
		ctx.Aggregates = append(ctx.Aggregates, b.buildAggregate(aggNode))
	}
	for _, sliceNode := range b.children(n.ID, "slice") {
		ctx.Slices = append(ctx.Slices, b.buildSlice(sliceNode))
	}
	return ctx
}

func (b *builder) buildAggregate(n *diagramNode) *ast.Aggregate {
	agg := &ast.Aggregate{
		Name:        n.Label,
		Description: n.Description,
		Comments:    convertComments(n.Comments),
	}
	for _, sliceNode := range b.children(n.ID, "slice") {
		agg.Slices = append(agg.Slices, b.buildSlice(sliceNode))
	}
	return agg
}

func (b *builder) buildSlice(n *diagramNode) *ast.Slice {
	slice := &ast.Slice{
		Name:        n.Label,
		Description: n.Description,
		Comments:    convertComments(n.Comments),
	}

	b.appendTrigger(slice, n.ID)
	b.appendCommands(slice, n.ID)
	b.appendEvents(slice, n.ID)
	b.appendViews(slice, n.ID)
	b.appendAutomations(slice, n.ID)
	b.appendTranslations(slice, n.ID)

	b.register(n.ID, slice, slice)
	return slice
}

func (b *builder) appendTrigger(slice *ast.Slice, sliceID string) {
	triggers := b.children(sliceID, "trigger")
	if len(triggers) == 0 {
		return
	}
	t := triggers[0]
	slice.Trigger = &ast.Trigger{
		Name:        t.Label,
		Description: t.Description,
		Comments:    convertComments(t.Comments),
		Actor:       t.Actor,
		Reads:       t.Reads,
	}
	b.register(t.ID, slice, slice.Trigger)
}

func (b *builder) appendCommands(slice *ast.Slice, sliceID string) {
	for _, c := range b.children(sliceID, "command") {
		cmd := &ast.Command{
			Name:        c.Label,
			Description: c.Description,
			Comments:    convertComments(c.Comments),
			Fields:      convertFields(c.Fields),
		}
		slice.Commands = append(slice.Commands, cmd)
		b.register(c.ID, slice, cmd)
	}
}

func (b *builder) appendEvents(slice *ast.Slice, sliceID string) {
	nestedInTranslation := b.translationEventNames(sliceID)

	for _, e := range b.children(sliceID, "event") {
		if nestedInTranslation[e.Label] {
			b.register(e.ID, slice, nil)
			continue
		}
		evt := &ast.Event{
			Name:         e.Label,
			Description:  e.Description,
			Comments:     convertComments(e.Comments),
			Source:       e.Source,
			ExternalName: e.ExternalName,
			Fields:       convertFields(e.Fields),
		}
		slice.Events = append(slice.Events, evt)
		b.register(e.ID, slice, evt)
	}
}

// translationEventNames names the events the slice's translations nest. Each of
// them is re-emitted inside its translation block, so the standalone event node
// the exporter adds for it must not become a second, duplicate top-level event
// declaration.
func (b *builder) translationEventNames(sliceID string) map[string]bool {
	nested := make(map[string]bool)
	for _, t := range b.children(sliceID, "translation") {
		if t.Event != nil && t.Event.Name != "" {
			nested[t.Event.Name] = true
		}
	}
	return nested
}

func (b *builder) appendViews(slice *ast.Slice, sliceID string) {
	for _, v := range b.children(sliceID, "view") {
		view := &ast.View{
			Name:        v.Label,
			Description: v.Description,
			Comments:    convertComments(v.Comments),
			Fields:      convertFields(v.Fields),
			Subscribes:  append([]string(nil), v.Subscribes...),
		}
		slice.Views = append(slice.Views, view)
		b.register(v.ID, slice, view)
	}
}

func (b *builder) appendAutomations(slice *ast.Slice, sliceID string) {
	for _, a := range b.children(sliceID, "automation") {
		auto := &ast.Automation{
			Name:          a.Label,
			Description:   a.Description,
			Comments:      convertComments(a.Comments),
			OnEvent:       a.OnEvent,
			Schedule:      a.Schedule,
			Reads:         a.Reads,
			Command:       a.Command,
			TargetContext: a.TargetContext,
		}
		slice.Automations = append(slice.Automations, auto)
		b.register(a.ID, slice, auto)
	}
}

func (b *builder) appendTranslations(slice *ast.Slice, sliceID string) {
	for _, t := range b.children(sliceID, "translation") {
		trans := &ast.Translation{
			Name:           t.Label,
			Description:    t.Description,
			Comments:       convertComments(t.Comments),
			ExternalSystem: t.ExternalSystem,
			Reads:          t.Reads,
			Command:        t.Command,
		}
		if t.Event != nil {
			trans.Event = &ast.Event{
				Name:         t.Event.Name,
				Description:  t.Event.Description,
				Comments:     convertComments(t.Event.Comments),
				Source:       t.Event.Source,
				ExternalName: t.Event.ExternalName,
				Fields:       convertFields(t.Event.Fields),
			}
		}
		slice.Translations = append(slice.Translations, trans)
		b.register(t.ID, slice, trans)
	}
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
			// An automation states either an activation event or a schedule, never
			// both, so an edge drawn onto a scheduled one would produce text the
			// parser rejects.
			if auto, ok := b.byNode[tgt.ID].(*ast.Automation); ok && auto.OnEvent == "" && auto.Schedule == "" {
				auto.OnEvent = src.Label
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
			b.foldReads(src, tgt)
		}
	}
}

func (b *builder) foldReads(src, tgt *diagramNode) {
	switch reader := b.byNode[tgt.ID].(type) {
	case *ast.Trigger:
		reader.Reads = readsDrawnFromView(reader.Reads, src)
	case *ast.Automation:
		reader.Reads = readsDrawnFromView(reader.Reads, src)
	case *ast.Translation:
		if reader.Reads == "" {
			reader.Reads = src.Label
		}
	}
}

// readsDrawnFromView takes the source node's label only when that node is a
// view. An edge dragged off the view keeps its "reads" type, so without the
// check the element would come back reading a command or an event name — which
// for an automation is a view reference the validator resolves and rejects.
func readsDrawnFromView(recorded string, src *diagramNode) string {
	if recorded != "" || src.Type != "view" {
		return recorded
	}
	return src.Label
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

func convertComments(comments []*diagramComment) []*ast.Comment {
	if len(comments) == 0 {
		return nil
	}
	out := make([]*ast.Comment, 0, len(comments))
	for _, c := range comments {
		if c == nil {
			continue
		}
		out = append(out, &ast.Comment{Text: c.Text})
	}
	return out
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

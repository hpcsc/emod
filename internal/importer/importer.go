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

// diagramNames reads a key that names other constructs, as one name or as a
// list.
type diagramNames []string

func (n *diagramNames) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err == nil {
		*n = nil
		if name != "" {
			*n = diagramNames{name}
		}
		return nil
	}
	var names []string
	if err := json.Unmarshal(data, &names); err != nil {
		return err
	}
	*n = names
	return nil
}

func (n diagramNames) first() string {
	if len(n) == 0 {
		return ""
	}
	return n[0]
}

type diagramNode struct {
	ID             string            `json:"id"`
	Type           string            `json:"type"`
	Label          string            `json:"label"`
	Description    string            `json:"description,omitempty"`
	Image          string            `json:"image,omitempty"`
	ParentID       *string           `json:"parentId"`
	Position       *ast.Position     `json:"position,omitempty"`
	Fields         []*diagramField   `json:"fields,omitempty"`
	Comments       []*diagramComment `json:"comments,omitempty"`
	Reads          diagramNames      `json:"reads,omitempty"`
	Subscribes     []string          `json:"subscribes,omitempty"`
	OnEvent        string            `json:"on_event,omitempty"`
	Schedule       string            `json:"every,omitempty"`
	After          string            `json:"after,omitempty"`
	Command        diagramNames      `json:"command,omitempty"`
	TargetContext  string            `json:"target_context,omitempty"`
	ExternalSystem string            `json:"external_system,omitempty"`
	Source         string            `json:"source,omitempty"`
	ExternalName   string            `json:"external_name,omitempty"`
	Event          *diagramEvent     `json:"event,omitempty"`
	CopyOf         string            `json:"copy_of,omitempty"`
}

// viewName answers nothing for a node that is not a view: an edge dragged off a
// view keeps its "reads" type, so it can start at a command or an event.
func (n *diagramNode) viewName() string {
	if n.Type != "view" && n.Type != "view_copy" {
		return ""
	}
	return n.Label
}

// commandName answers nothing for a node that is not a command: an edge dragged
// off a command keeps its "shows_command" type, so it can end at a view or an
// event.
func (n *diagramNode) commandName() string {
	if n.Type != "command" && n.Type != "command_copy" {
		return ""
	}
	return n.Label
}

// isCopy reports whether the node draws again an event, a view or a command
// that another slice declares.
func (n *diagramNode) isCopy() bool {
	switch n.Type {
	case "event_copy", "view_copy", "command_copy":
		return true
	default:
		return false
	}
}

func (n *diagramNode) asContext() *ast.Context {
	return &ast.Context{
		Name:        n.Label,
		Description: n.Description,
		Comments:    convertComments(n.Comments),
	}
}

func (n *diagramNode) asAggregate() *ast.Aggregate {
	return &ast.Aggregate{
		Name:        n.Label,
		Description: n.Description,
		Comments:    convertComments(n.Comments),
	}
}

func (n *diagramNode) asSlice() *ast.Slice {
	return &ast.Slice{
		Name:        n.Label,
		Description: n.Description,
		Comments:    convertComments(n.Comments),
	}
}

func (n *diagramNode) asScreen() *ast.Screen {
	return &ast.Screen{
		Name:        n.Label,
		Description: n.Description,
		Image:       n.Image,
		Comments:    convertComments(n.Comments),
	}
}

func (n *diagramNode) asAppearance() *ast.Appearance {
	return &ast.Appearance{
		Screen:   n.Label,
		Comments: convertComments(n.Comments),
		Reads:    slices.Clone([]string(n.Reads)),
		Commands: slices.Clone([]string(n.Command)),
	}
}

func (n *diagramNode) asCommand() *ast.Command {
	return &ast.Command{
		Name:        n.Label,
		Description: n.Description,
		Comments:    convertComments(n.Comments),
		Fields:      convertFields(n.Fields),
	}
}

func (n *diagramNode) asEvent() *ast.Event {
	return &ast.Event{
		Name:         n.Label,
		Description:  n.Description,
		Comments:     convertComments(n.Comments),
		Source:       n.Source,
		ExternalName: n.ExternalName,
		Fields:       convertFields(n.Fields),
	}
}

func (n *diagramNode) asView() *ast.View {
	return &ast.View{
		Name:        n.Label,
		Description: n.Description,
		Comments:    convertComments(n.Comments),
		Fields:      convertFields(n.Fields),
		Subscribes:  append([]string(nil), n.Subscribes...),
	}
}

func (n *diagramNode) asAutomation() *ast.Automation {
	return &ast.Automation{
		Name:          n.Label,
		Description:   n.Description,
		Comments:      convertComments(n.Comments),
		OnEvent:       n.OnEvent,
		Schedule:      n.Schedule,
		After:         n.After,
		Reads:         n.Reads.first(),
		Command:       n.Command.first(),
		TargetContext: n.TargetContext,
	}
}

func (n *diagramNode) asTranslation() *ast.Translation {
	return &ast.Translation{
		Name:           n.Label,
		Description:    n.Description,
		Comments:       convertComments(n.Comments),
		ExternalSystem: n.ExternalSystem,
		Reads:          n.Reads.first(),
		Command:        n.Command.first(),
		Event:          n.Event.asEvent(),
	}
}

type diagramEvent struct {
	Name         string            `json:"name"`
	Description  string            `json:"description,omitempty"`
	Comments     []*diagramComment `json:"comments,omitempty"`
	Source       string            `json:"source,omitempty"`
	ExternalName string            `json:"external_name,omitempty"`
	Fields       []*diagramField   `json:"fields,omitempty"`
}

func (e *diagramEvent) asEvent() *ast.Event {
	if e == nil {
		return nil
	}
	return &ast.Event{
		Name:         e.Name,
		Description:  e.Description,
		Comments:     convertComments(e.Comments),
		Source:       e.Source,
		ExternalName: e.ExternalName,
		Fields:       convertFields(e.Fields),
	}
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

	b := newBuilder(&doc)
	model := &ast.Model{Name: doc.ModelName, Description: doc.ModelDescription}

	for _, n := range doc.Nodes {
		if n == nil || n.ParentID != nil {
			continue
		}
		switch n.Type {
		case "screen":
			model.Screens = append(model.Screens, n.asScreen())
		case "context":
			model.Contexts = append(model.Contexts, b.buildContext(n))
		}
	}

	b.foldEdges(doc.Edges)
	declareShownScreens(model)

	return model, nil
}

// declareShownScreens declares each screen that a shows block names and no
// declaration holds, after the declarations the model has, so that the saved
// text names no screen it does not declare.
func declareShownScreens(model *ast.Model) {
	declared := make(map[string]bool, len(model.Screens))
	for _, screen := range model.Screens {
		declared[screen.Name] = true
	}
	for _, slice := range model.AllSlices() {
		for _, appearance := range slice.Appearances {
			if appearance.Screen != "" && !declared[appearance.Screen] {
				declared[appearance.Screen] = true
				model.Screens = append(model.Screens, &ast.Screen{Name: appearance.Screen})
			}
		}
	}
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

func newBuilder(doc *diagramDocument) *builder {
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

	return b
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
	ctx := n.asContext()
	for _, aggNode := range b.children(n.ID, "aggregate") {
		ctx.Aggregates = append(ctx.Aggregates, b.buildAggregate(aggNode))
	}
	for _, sliceNode := range b.children(n.ID, "slice") {
		ctx.Slices = append(ctx.Slices, b.buildSlice(sliceNode))
	}
	return ctx
}

func (b *builder) buildAggregate(n *diagramNode) *ast.Aggregate {
	agg := n.asAggregate()
	for _, sliceNode := range b.children(n.ID, "slice") {
		agg.Slices = append(agg.Slices, b.buildSlice(sliceNode))
	}
	return agg
}

func (b *builder) buildSlice(n *diagramNode) *ast.Slice {
	slice := n.asSlice()

	b.appendAppearances(slice, n.ID)
	b.appendCommands(slice, n.ID)
	b.appendEvents(slice, n.ID)
	b.appendViews(slice, n.ID)
	b.appendAutomations(slice, n.ID)
	b.appendTranslations(slice, n.ID)

	b.register(n.ID, slice, slice)
	return slice
}

func (b *builder) appendAppearances(slice *ast.Slice, sliceID string) {
	for _, a := range b.children(sliceID, "shows") {
		appearance := a.asAppearance()
		slice.Appearances = append(slice.Appearances, appearance)
		b.register(a.ID, slice, appearance)
	}
}

func (b *builder) appendCommands(slice *ast.Slice, sliceID string) {
	for _, c := range b.children(sliceID, "command") {
		cmd := c.asCommand()
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
		evt := e.asEvent()
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
		view := v.asView()
		slice.Views = append(slice.Views, view)
		b.register(v.ID, slice, view)
	}
}

func (b *builder) appendAutomations(slice *ast.Slice, sliceID string) {
	for _, a := range b.children(sliceID, "automation") {
		auto := a.asAutomation()
		slice.Automations = append(slice.Automations, auto)
		b.register(a.ID, slice, auto)
	}
}

func (b *builder) appendTranslations(slice *ast.Slice, sliceID string) {
	for _, t := range b.children(sliceID, "translation") {
		trans := t.asTranslation()
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
		case "shows_command":
			if appearance, ok := b.byNode[src.ID].(*ast.Appearance); ok {
				if command := tgt.commandName(); command != "" {
					appearance.Commands = appendMissing(appearance.Commands, command)
				}
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
	case *ast.Appearance:
		if view := src.viewName(); view != "" {
			reader.Reads = appendMissing(reader.Reads, view)
		}
	case *ast.Automation:
		reader.Reads = readsDrawnFromView(reader.Reads, src)
	case *ast.Translation:
		if reader.Reads == "" {
			reader.Reads = src.Label
		}
	}
}

func readsDrawnFromView(recorded string, src *diagramNode) string {
	if recorded != "" {
		return recorded
	}
	return src.viewName()
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

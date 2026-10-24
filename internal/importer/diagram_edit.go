package importer

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/export"
)

// ApplyDiagram makes model say what diagram says. diagram must be the document
// the exporter draws from model, edited in the viewer.
func ApplyDiagram(model *ast.Model, diagram []byte) error {
	edit, err := newDiagramEdit(model, diagram)
	if err != nil {
		return err
	}
	edit.applyTo(model)

	return nil
}

type blockGroup struct {
	sliceID, typ string
}

type repoint struct {
	removed, added *diagramEdge
	view           *ast.View
	reserved       string
	slice          *ast.Slice
	flow           *ast.Flow
}

type diagramEdit struct {
	drawn  *diagramDocument
	edited *diagramDocument

	drawnNodes          map[string]*diagramNode
	drawnFrom           map[string]any
	nestedIn            map[string]string
	nestedEventOf       map[*ast.Translation]string
	builder             *builder
	blocks              map[blockGroup][]*diagramNode
	removedFlowComments map[*ast.Slice][]*ast.Comment
}

func newDiagramEdit(model *ast.Model, diagram []byte) (*diagramEdit, error) {
	var edited diagramDocument
	if err := json.Unmarshal(diagram, &edited); err != nil {
		return nil, fmt.Errorf("importer: invalid diagram JSON: %w", err)
	}

	drawnJSON, err := export.ExportDiagramJSON(model)
	if err != nil {
		return nil, fmt.Errorf("importer: drawing the model: %w", err)
	}
	var drawn diagramDocument
	if err := json.Unmarshal(drawnJSON, &drawn); err != nil {
		return nil, fmt.Errorf("importer: reading the drawn model: %w", err)
	}

	e := &diagramEdit{drawn: &drawn, edited: &edited, removedFlowComments: make(map[*ast.Slice][]*ast.Comment)}
	e.dropNullEntries()
	if err := e.matchDrawnNodes(model); err != nil {
		return nil, err
	}
	e.builder = newBuilder(e.edited)
	e.nameCopiesAfterOriginals()
	e.placeBlocks()

	return e, nil
}

// nameCopiesAfterOriginals gives each copy the name its event has now. The
// viewer renames the event and never its copies, and a view that subscribes
// through a copy reads the name off the copy.
func (e *diagramEdit) nameCopiesAfterOriginals() {
	for _, n := range e.edited.Nodes {
		if !n.isEventCopy() {
			continue
		}
		if original := e.builder.nodeByID[n.CopyOf]; original != nil && !original.isEventCopy() {
			n.Label = original.Label
		}
	}
}

func (e *diagramEdit) applyTo(model *ast.Model) {
	removed, kept, added := e.edgeChanges()

	model.Contexts = e.contexts()

	repoints, repointed := e.pairRepoints(removed, added)
	for _, edge := range removed {
		if r, ok := repoints[edge]; ok {
			e.reserveEntry(model, r)
		} else {
			e.clearRelationship(model, edge)
		}
	}
	var renames []func()
	for _, edge := range kept {
		if rename := e.pendingRename(model, edge); rename != nil {
			renames = append(renames, rename)
		}
	}
	for _, rename := range renames {
		rename()
	}
	e.builder.foldEdges(slices.DeleteFunc(slices.Clone(added), func(edge *diagramEdge) bool { return repointed[edge] }))
	for _, edge := range removed {
		if r, ok := repoints[edge]; ok {
			e.fillEntry(r)
		}
	}
	e.restoreFlowComments()
}

func (e *diagramEdit) pairRepoints(removed, added []*diagramEdge) (map[*diagramEdge]*repoint, map[*diagramEdge]bool) {
	repoints := make(map[*diagramEdge]*repoint)
	repointed := make(map[*diagramEdge]bool)
	for _, a := range added {
		for _, r := range removed {
			if repoints[r] != nil || r.Type != a.Type {
				continue
			}
			sameSource, sameTarget := r.Source == a.Source, r.Target == a.Target
			if (r.Type == "subscription" && sameTarget && !sameSource) || (r.Type == "flow" && sameSource != sameTarget) {
				repoints[r] = &repoint{removed: r, added: a}
				repointed[a] = true
				break
			}
		}
	}
	return repoints, repointed
}

func (e *diagramEdit) reserveEntry(model *ast.Model, r *repoint) {
	from, to := e.drawnNodes[r.removed.Source].Label, e.drawnNodes[r.removed.Target].Label

	switch r.removed.Type {
	case "flow":
		for _, s := range model.AllSlices() {
			for _, f := range s.Flows {
				if r.flow == nil && f.CommandName == from && f.EventName == to {
					r.flow, r.slice = f, s
					f.CommandName, f.EventName = "", ""
				}
			}
			e.removeFlows(s, from, to)
		}
	case "subscription":
		view, ok := e.drawnFrom[r.removed.Target].(*ast.View)
		if !ok {
			return
		}
		if at := slices.Index(view.Subscribes, from); at >= 0 {
			r.view, r.reserved = view, fmt.Sprintf("\x00%p", r)
			view.Subscribes[at] = r.reserved
		}
		view.Subscribes = slices.DeleteFunc(view.Subscribes, func(name string) bool { return name == from })
	}
}

func (e *diagramEdit) fillEntry(r *repoint) {
	switch {
	case r.flow != nil:
		before := len(r.slice.Flows)
		e.builder.foldEdges([]*diagramEdge{r.added})
		if len(r.slice.Flows) == before+1 {
			added := r.slice.Flows[before]
			r.flow.CommandName, r.flow.EventName = added.CommandName, added.EventName
			r.slice.Flows = r.slice.Flows[:before]
			return
		}
		r.slice.Flows = slices.DeleteFunc(r.slice.Flows, func(f *ast.Flow) bool { return f == r.flow })
		e.removedFlowComments[r.slice] = append(e.removedFlowComments[r.slice], r.flow.Comments...)
	case r.view != nil:
		before := len(r.view.Subscribes)
		e.builder.foldEdges([]*diagramEdge{r.added})
		at := slices.Index(r.view.Subscribes, r.reserved)
		if len(r.view.Subscribes) == before+1 {
			r.view.Subscribes[at] = r.view.Subscribes[before]
			r.view.Subscribes = r.view.Subscribes[:before]
			return
		}
		r.view.Subscribes = slices.Delete(r.view.Subscribes, at, at+1)
	default:
		e.builder.foldEdges([]*diagramEdge{r.added})
	}
}

func (e *diagramEdit) dropNullEntries() {
	e.edited.Nodes = slices.DeleteFunc(e.edited.Nodes, func(n *diagramNode) bool { return n == nil || n.ID == "" })
	e.edited.Edges = slices.DeleteFunc(e.edited.Edges, func(edge *diagramEdge) bool { return edge == nil })
}

func (e *diagramEdit) matchDrawnNodes(model *ast.Model) error {
	constructs := make(map[ast.Position]any)
	nestedIn := make(map[ast.Position]*ast.Translation)

	for _, a := range model.Actors {
		constructs[a.NamePos] = a
	}
	for _, c := range model.Contexts {
		constructs[c.NamePos] = c
		for _, agg := range c.Aggregates {
			constructs[agg.NamePos] = agg
		}
	}
	for _, s := range model.AllSlices() {
		constructs[s.NamePos] = s
		if s.Trigger != nil {
			constructs[s.Trigger.NamePos] = s.Trigger
		}
		for _, c := range s.Commands {
			constructs[c.NamePos] = c
		}
		for _, ev := range s.Events {
			constructs[ev.NamePos] = ev
		}
		for _, v := range s.Views {
			constructs[v.NamePos] = v
		}
		for _, a := range s.Automations {
			constructs[a.NamePos] = a
		}
		for _, t := range s.Translations {
			constructs[t.NamePos] = t
			if t.Event != nil {
				constructs[t.Event.NamePos] = t.Event
				nestedIn[t.Event.NamePos] = t
			}
		}
	}

	e.drawnNodes = make(map[string]*diagramNode, len(e.drawn.Nodes))
	e.drawnFrom = make(map[string]any, len(e.drawn.Nodes))
	e.nestedIn = make(map[string]string)
	e.nestedEventOf = make(map[*ast.Translation]string)
	for _, n := range e.drawn.Nodes {
		if n.isEventCopy() {
			e.drawnNodes[n.ID] = n
			continue
		}
		if n.Position == nil {
			return fmt.Errorf("importer: the diagram node %s carries no position", n.ID)
		}
		position := *n.Position
		construct, found := constructs[position]
		if !found {
			return fmt.Errorf("importer: the diagram node %s stands for no construct in the model", n.ID)
		}
		e.drawnNodes[n.ID] = n
		e.drawnFrom[n.ID] = construct
		if t, nested := nestedIn[position]; nested {
			e.nestedEventOf[t] = n.ID
		}
	}
	for id, construct := range e.drawnFrom {
		if t, isTranslation := construct.(*ast.Translation); isTranslation {
			if eventID, nests := e.nestedEventOf[t]; nests {
				e.nestedIn[eventID] = id
			}
		}
	}

	return nil
}

func (e *diagramEdit) placeBlocks() {
	declaredIn := make(map[string]string)
	holdsTrigger := make(map[string]bool)
	place := func(n *diagramNode, sliceID string) {
		declaredIn[n.ID] = sliceID
		if n.Type == "trigger" {
			holdsTrigger[sliceID] = true
		}
	}

	var arriving []*diagramNode
	for _, n := range e.edited.Nodes {
		if _, nested := e.nestedIn[n.ID]; nested {
			continue
		}
		if e.parentOf(e.drawnNodes[n.ID]) == e.parentOf(n) {
			place(n, e.parentOf(n))
			continue
		}
		arriving = append(arriving, n)
	}

	for _, n := range arriving {
		target := e.parentOf(n)
		if !e.canHold(target, n, holdsTrigger) {
			target = e.parentOf(e.drawnNodes[n.ID])
		}
		place(n, target)
	}

	e.blocks = make(map[blockGroup][]*diagramNode)
	var arrived []*diagramNode
	for _, n := range e.edited.Nodes {
		sliceID, placed := declaredIn[n.ID]
		if !placed {
			continue
		}
		if e.parentOf(e.drawnNodes[n.ID]) != sliceID {
			arrived = append(arrived, n)
			continue
		}
		group := blockGroup{sliceID, n.Type}
		e.blocks[group] = append(e.blocks[group], n)
	}
	for _, n := range arrived {
		group := blockGroup{declaredIn[n.ID], n.Type}
		e.blocks[group] = append(e.blocks[group], n)
	}
}

func (e *diagramEdit) parentOf(n *diagramNode) string {
	if n == nil || n.ParentID == nil {
		return ""
	}
	return *n.ParentID
}

func (e *diagramEdit) canHold(sliceID string, n *diagramNode, holdsTrigger map[string]bool) bool {
	slice := e.builder.nodeByID[sliceID]
	if slice == nil || slice.Type != "slice" {
		return false
	}
	return n.Type != "trigger" || !holdsTrigger[sliceID]
}

func (e *diagramEdit) blocksIn(sliceID, typ string) []*diagramNode {
	return e.blocks[blockGroup{sliceID, typ}]
}

func (e *diagramEdit) contexts() []*ast.Context {
	var contexts []*ast.Context
	for _, n := range e.edited.Nodes {
		if n.Type == "context" {
			contexts = append(contexts, e.contextFor(n))
		}
	}
	return contexts
}

func (e *diagramEdit) contextFor(n *diagramNode) *ast.Context {
	ctx, kept := e.drawnFrom[n.ID].(*ast.Context)
	if !kept {
		ctx = n.asContext()
	}
	ctx.Name = n.Label

	ctx.Aggregates = nil
	for _, child := range e.builder.children(n.ID, "aggregate") {
		ctx.Aggregates = append(ctx.Aggregates, e.aggregateFor(child))
	}
	ctx.Slices = nil
	for _, child := range e.builder.children(n.ID, "slice") {
		ctx.Slices = append(ctx.Slices, e.sliceFor(child))
	}
	return ctx
}

func (e *diagramEdit) aggregateFor(n *diagramNode) *ast.Aggregate {
	agg, kept := e.drawnFrom[n.ID].(*ast.Aggregate)
	if !kept {
		agg = n.asAggregate()
	}
	agg.Name = n.Label

	agg.Slices = nil
	for _, child := range e.builder.children(n.ID, "slice") {
		agg.Slices = append(agg.Slices, e.sliceFor(child))
	}
	return agg
}

func (e *diagramEdit) sliceFor(n *diagramNode) *ast.Slice {
	s, kept := e.drawnFrom[n.ID].(*ast.Slice)
	if !kept {
		s = n.asSlice()
	}
	s.Name = n.Label

	s.Trigger = nil
	if triggers := e.blocksIn(n.ID, "trigger"); len(triggers) > 0 {
		s.Trigger = e.triggerFor(triggers[0])
		e.builder.register(triggers[0].ID, s, s.Trigger)
	}

	s.Commands = nil
	for _, c := range e.blocksIn(n.ID, "command") {
		cmd := e.commandFor(c)
		s.Commands = append(s.Commands, cmd)
		e.builder.register(c.ID, s, cmd)
	}

	s.Events = nil
	for _, ev := range e.blocksIn(n.ID, "event") {
		evt := e.eventFor(ev)
		s.Events = append(s.Events, evt)
		e.builder.register(ev.ID, s, evt)
	}

	s.Views = nil
	for _, v := range e.blocksIn(n.ID, "view") {
		view := e.viewFor(v)
		s.Views = append(s.Views, view)
		e.builder.register(v.ID, s, view)
	}

	s.Automations = nil
	for _, a := range e.blocksIn(n.ID, "automation") {
		auto := e.automationFor(a)
		s.Automations = append(s.Automations, auto)
		e.builder.register(a.ID, s, auto)
	}

	s.Translations = nil
	for _, t := range e.blocksIn(n.ID, "translation") {
		trans := e.translationFor(t)
		s.Translations = append(s.Translations, trans)
		e.builder.register(t.ID, s, trans)
		if nestedID, nests := e.nestedEventOf[trans]; nests {
			e.builder.register(nestedID, s, nil)
		}
	}

	return s
}

func (e *diagramEdit) triggerFor(n *diagramNode) *ast.Trigger {
	trigger, kept := e.drawnFrom[n.ID].(*ast.Trigger)
	if !kept {
		trigger = n.asTrigger()
	}
	trigger.Name = n.Label
	return trigger
}

func (e *diagramEdit) commandFor(n *diagramNode) *ast.Command {
	cmd, kept := e.drawnFrom[n.ID].(*ast.Command)
	if !kept {
		cmd = n.asCommand()
	}
	cmd.Name = n.Label
	cmd.Fields = e.fieldsOf(n)
	return cmd
}

func (e *diagramEdit) eventFor(n *diagramNode) *ast.Event {
	evt, kept := e.drawnFrom[n.ID].(*ast.Event)
	if !kept {
		evt = n.asEvent()
	}
	evt.Name = n.Label
	evt.Fields = e.fieldsOf(n)
	return evt
}

func (e *diagramEdit) viewFor(n *diagramNode) *ast.View {
	view, kept := e.drawnFrom[n.ID].(*ast.View)
	if !kept {
		view = n.asView()
	}
	view.Name = n.Label
	return view
}

func (e *diagramEdit) automationFor(n *diagramNode) *ast.Automation {
	auto, kept := e.drawnFrom[n.ID].(*ast.Automation)
	if !kept {
		auto = n.asAutomation()
	}
	auto.Name = n.Label
	return auto
}

func (e *diagramEdit) translationFor(n *diagramNode) *ast.Translation {
	trans, kept := e.drawnFrom[n.ID].(*ast.Translation)
	if !kept {
		trans = n.asTranslation()
	}
	trans.Name = n.Label

	if nestedID, nests := e.nestedEventOf[trans]; nests {
		if nested := e.builder.nodeByID[nestedID]; nested != nil {
			trans.Event.Name = nested.Label
			trans.Event.Fields = e.fieldsOf(nested)
		} else {
			trans.Event = nil
		}
	}
	return trans
}

func (e *diagramEdit) fieldsOf(n *diagramNode) []*ast.Field {
	return slices.DeleteFunc(convertFields(n.Fields), func(f *ast.Field) bool { return f.Name == "" })
}

func (e *diagramEdit) edgeChanges() (removed, kept, added []*diagramEdge) {
	inEdited := make(map[diagramEdge]bool)
	for _, edge := range e.edited.Edges {
		if e.inResult(edge.Source) && e.inResult(edge.Target) {
			inEdited[*edge] = true
		}
	}

	inDrawn := make(map[diagramEdge]bool)
	for _, edge := range e.drawn.Edges {
		inDrawn[*edge] = true
		if inEdited[*edge] {
			kept = append(kept, edge)
		} else {
			removed = append(removed, edge)
		}
	}

	for _, edge := range e.edited.Edges {
		if !inDrawn[*edge] && e.inResult(edge.Source) && e.inResult(edge.Target) {
			added = append(added, edge)
		}
	}

	return removed, kept, added
}

// inResult reports whether the node stands in the edited diagram. A copy stands
// only while the event it copies does: deleting the event takes away every
// subscription drawn from its copies, as it takes away those drawn from itself.
func (e *diagramEdit) inResult(id string) bool {
	n := e.builder.nodeByID[id]
	if n == nil {
		return false
	}
	if n.isEventCopy() {
		original := e.builder.nodeByID[n.CopyOf]
		return original != nil && !original.isEventCopy() && e.inResult(original.ID)
	}
	translation, nested := e.nestedIn[id]
	return !nested || e.builder.nodeByID[translation] != nil
}

func (e *diagramEdit) clearRelationship(model *ast.Model, edge *diagramEdge) {
	from, to := e.drawnNodes[edge.Source].Label, e.drawnNodes[edge.Target].Label

	switch edge.Type {
	case "flow":
		for _, s := range model.AllSlices() {
			e.removeFlows(s, from, to)
		}
	case "subscription":
		if view, ok := e.drawnFrom[edge.Target].(*ast.View); ok {
			view.Subscribes = slices.DeleteFunc(view.Subscribes, func(name string) bool { return name == from })
		}
	case "automation_trigger":
		if auto, ok := e.drawnFrom[edge.Target].(*ast.Automation); ok {
			auto.OnEvent = ""
		}
	case "automation_command":
		if auto, ok := e.drawnFrom[edge.Source].(*ast.Automation); ok {
			auto.Command = ""
		}
	case "translation_command":
		if trans, ok := e.drawnFrom[edge.Source].(*ast.Translation); ok {
			trans.Command = ""
		}
	case "reads":
		e.setReads(edge.Target, "")
	}
}

func (e *diagramEdit) removeFlows(s *ast.Slice, command, event string) {
	var kept []*ast.Flow
	for _, f := range s.Flows {
		if f.CommandName == command && f.EventName == event {
			e.removedFlowComments[s] = append(e.removedFlowComments[s], f.Comments...)
			continue
		}
		kept = append(kept, f)
	}
	s.Flows = kept
}

func (e *diagramEdit) restoreFlowComments() {
	for s, comments := range e.removedFlowComments {
		switch {
		case len(s.Flows) > 0:
			s.Flows[0].Comments = append(comments, s.Flows[0].Comments...)
		case len(s.Rejections) > 0:
			s.Rejections[0].Comments = append(comments, s.Rejections[0].Comments...)
		}
	}
}

func (e *diagramEdit) pendingRename(model *ast.Model, edge *diagramEdge) func() {
	from, to := e.drawnNodes[edge.Source].Label, e.drawnNodes[edge.Target].Label
	newFrom, newTo := e.builder.nodeByID[edge.Source].Label, e.builder.nodeByID[edge.Target].Label

	switch edge.Type {
	case "flow":
		var flows []*ast.Flow
		for _, s := range model.AllSlices() {
			for _, f := range s.Flows {
				if f.CommandName == from && f.EventName == to {
					flows = append(flows, f)
				}
			}
		}
		return func() {
			for _, f := range flows {
				f.CommandName, f.EventName = newFrom, newTo
			}
		}
	case "subscription":
		view, ok := e.drawnFrom[edge.Target].(*ast.View)
		if !ok {
			return nil
		}
		var at []int
		for i, name := range view.Subscribes {
			if name == from {
				at = append(at, i)
			}
		}
		return func() {
			for _, i := range at {
				view.Subscribes[i] = newFrom
			}
		}
	case "automation_trigger":
		if auto, ok := e.drawnFrom[edge.Target].(*ast.Automation); ok {
			return func() { auto.OnEvent = newFrom }
		}
	case "automation_command":
		if auto, ok := e.drawnFrom[edge.Source].(*ast.Automation); ok {
			return func() { auto.Command = newTo }
		}
	case "translation_command":
		if trans, ok := e.drawnFrom[edge.Source].(*ast.Translation); ok {
			return func() { trans.Command = newTo }
		}
	case "reads":
		return func() { e.setReads(edge.Target, newFrom) }
	}
	return nil
}

func (e *diagramEdit) setReads(readerID, view string) {
	switch reader := e.drawnFrom[readerID].(type) {
	case *ast.Trigger:
		reader.Reads = view
	case *ast.Automation:
		reader.Reads = view
	case *ast.Translation:
		reader.Reads = view
	}
}

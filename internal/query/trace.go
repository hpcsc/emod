package query

import (
	"fmt"
	"slices"
	"strings"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagram"
)

const KindExternalSystem Kind = "external_system"

var TraceKinds = []Kind{KindCommand, KindEvent, KindView, KindTrigger, KindAutomation, KindTranslation, KindInvariant, KindExternalSystem}

// An invariant has no slice, because a scope declares it. An external system
// has the place of the first translation that names it.
type Node struct {
	Element
	Context   string `json:"context"`
	Aggregate string `json:"aggregate,omitempty"`
	Slice     string `json:"slice,omitempty"`
	Line      int    `json:"line"`
}

// From is the element at the other end of the edge that reaches the step.
type Step struct {
	Label string `json:"label"`
	After string `json:"after,omitempty"`
	Node
	From           Element `json:"from"`
	CrossesContext bool    `json:"crosses_context,omitempty"`
	Depth          int     `json:"depth"`
}

// Tree lists each element that a trace reaches one time, in breadth-first
// order, under the first element that reaches it.
type Tree struct {
	Root  Node   `json:"root"`
	Steps []Step `json:"steps"`
}

type TraceOptions struct {
	Kind     Kind
	Upstream bool
	NoViews  bool
}

type AmbiguousNameError struct {
	Name    string
	Matches []Node
}

func (e *AmbiguousNameError) Error() string {
	places := make([]string, len(e.Matches))
	for i, m := range e.Matches {
		places[i] = fmt.Sprintf("%s in %s", m.Kind, ScopeName(m.Context, m.Aggregate))
	}
	return fmt.Sprintf("the model declares %q more than one time: as %s", e.Name, strings.Join(places, ", as "))
}

func (e *AmbiguousNameError) Kinds() []Kind {
	var kinds []Kind
	for _, m := range e.Matches {
		if !slices.Contains(kinds, m.Kind) {
			kinds = append(kinds, m.Kind)
		}
	}
	return kinds
}

func Trace(model *ast.Model, from string, options TraceOptions) (Tree, error) {
	g := newGraph(model)
	root, err := g.resolve(from, options.Kind)
	if err != nil {
		return Tree{}, err
	}

	tree := Tree{Root: g.nodes[root], Steps: []Step{}}
	type queued struct {
		key   nodeKey
		depth int
	}
	seen := map[nodeKey]bool{root: true}
	queue := []queued{{key: root}}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		parent := g.nodes[current.key]

		for _, edge := range g.edgesOf(current.key, options) {
			next := edge.to
			if options.Upstream {
				next = edge.from
			}
			if seen[next] {
				continue
			}
			seen[next] = true

			node := g.nodes[next]
			tree.Steps = append(tree.Steps, Step{
				Label:          edge.label,
				After:          edge.after,
				Node:           node,
				From:           parent.Element,
				CrossesContext: node.Context != parent.Context,
				Depth:          current.depth + 1,
			})
			queue = append(queue, queued{key: next, depth: current.depth + 1})
		}
	}

	return tree, nil
}

// nodeKey has a scope only for an invariant: two scopes can declare one
// invariant name, and each declaration is its own element.
type nodeKey struct {
	kind  Kind
	scope string
	name  string
}

type graphEdge struct {
	from  nodeKey
	to    nodeKey
	label string
	after string
}

func (e graphEdge) touchesView() bool {
	return e.label == "subscribes" || e.label == "reads"
}

type graph struct {
	nodes map[nodeKey]Node
	order []nodeKey
	out   map[nodeKey][]graphEdge
	in    map[nodeKey][]graphEdge
}

func newGraph(model *ast.Model) *graph {
	g := &graph{
		nodes: map[nodeKey]Node{},
		out:   map[nodeKey][]graphEdge{},
		in:    map[nodeKey][]graphEdge{},
	}

	for _, ref := range model.SliceRefs() {
		slice := ref.Slice
		add := func(kind Kind, name string, pos ast.Position) {
			if name == "" {
				return
			}
			g.addNode(nodeKey{kind: kind, name: name}, Node{
				Element:   Element{Kind: kind, Name: name},
				Context:   ref.Context.Name,
				Aggregate: aggregateName(ref),
				Slice:     slice.Name,
				Line:      pos.Line,
			})
		}

		if slice.Trigger != nil {
			add(KindTrigger, slice.Trigger.Name, slice.Trigger.NamePos)
		}
		for _, cmd := range slice.Commands {
			add(KindCommand, cmd.Name, cmd.NamePos)
		}
		for _, evt := range slice.Events {
			add(KindEvent, evt.Name, evt.NamePos)
		}
		for _, v := range slice.Views {
			add(KindView, v.Name, v.NamePos)
		}
		for _, auto := range slice.Automations {
			add(KindAutomation, auto.Name, auto.NamePos)
		}
		for _, tr := range slice.Translations {
			add(KindTranslation, tr.Name, tr.NamePos)
			if tr.Event != nil {
				add(KindEvent, tr.Event.Name, tr.Event.NamePos)
			}
			if _, named := g.nodes[nodeKey{kind: KindExternalSystem, name: tr.ExternalSystem}]; !named {
				add(KindExternalSystem, tr.ExternalSystem, tr.ExternalPos)
			}
		}
	}

	for _, scope := range InvariantScopes(model) {
		for _, inv := range scope.Declared {
			g.addNode(nodeKey{kind: KindInvariant, scope: scope.Name(), name: inv.Name}, Node{
				Element:   Element{Kind: KindInvariant, Name: inv.Name},
				Context:   scope.Context,
				Aggregate: scope.Aggregate,
				Line:      inv.NamePos.Line,
			})
		}
	}

	for _, ref := range model.SliceRefs() {
		scope := ScopeName(ref.Context.Name, aggregateName(ref))
		for _, e := range diagram.SliceEdges(ref.Slice) {
			if edge, ok := traceEdge(ref.Slice, scope, e); ok {
				g.addEdge(edge)
			}
		}
	}

	return g
}

// addNode keeps the last of two declarations of one name, as DeclaredNames
// does, and the place of the first in the order of names.
func (g *graph) addNode(key nodeKey, node Node) {
	if _, ok := g.nodes[key]; !ok {
		g.order = append(g.order, key)
	}
	g.nodes[key] = node
}

// addEdge leaves out an edge to a name that the model does not declare, as
// the diagram does.
func (g *graph) addEdge(edge graphEdge) {
	_, fromDeclared := g.nodes[edge.from]
	_, toDeclared := g.nodes[edge.to]
	if !fromDeclared || !toDeclared || slices.Contains(g.out[edge.from], edge) {
		return
	}
	g.out[edge.from] = append(g.out[edge.from], edge)
	g.in[edge.to] = append(g.in[edge.to], edge)
}

func traceEdge(slice *ast.Slice, invariantScope string, e diagram.Edge) (graphEdge, bool) {
	key := func(kind Kind, name string) nodeKey { return nodeKey{kind: kind, name: name} }
	edge := func(fromKind Kind, toKind Kind, label string) (graphEdge, bool) {
		return graphEdge{from: key(fromKind, e.From), to: key(toKind, e.To), label: label}, true
	}

	switch e.Kind {
	case diagram.EdgeFlow, diagram.EdgeTranslationFlow:
		return edge(KindCommand, KindEvent, "flow")
	case diagram.EdgeRejection:
		return graphEdge{
			from:  key(KindCommand, e.From),
			to:    nodeKey{kind: KindInvariant, scope: invariantScope, name: e.To},
			label: "rejected",
		}, true
	case diagram.EdgeTriggerCommand:
		return edge(KindTrigger, KindCommand, "issues")
	case diagram.EdgeTriggerReads:
		return edge(KindView, KindTrigger, "reads")
	case diagram.EdgeSubscription:
		return edge(KindEvent, KindView, "subscribes")
	case diagram.EdgeAutomationTrigger:
		on, _ := edge(KindEvent, KindAutomation, "on")
		for _, auto := range slice.Automations {
			if auto.Name == e.To {
				on.after = auto.After
			}
		}
		return on, true
	case diagram.EdgeAutomationReads:
		return edge(KindView, KindAutomation, "reads")
	case diagram.EdgeTranslationReads:
		return edge(KindView, KindTranslation, "reads")
	case diagram.EdgeAutomationCommand:
		return edge(KindAutomation, KindCommand, "command")
	case diagram.EdgeTranslationCommand:
		return edge(KindTranslation, KindCommand, "command")
	case diagram.EdgeTranslationExternal:
		return edge(KindExternalSystem, KindTranslation, "from")
	}
	return graphEdge{}, false
}

func (g *graph) edgesOf(key nodeKey, options TraceOptions) []graphEdge {
	edges := g.out[key]
	if options.Upstream {
		edges = g.in[key]
	}
	if !options.NoViews {
		return edges
	}

	var kept []graphEdge
	for _, edge := range edges {
		if !edge.touchesView() {
			kept = append(kept, edge)
		}
	}
	return kept
}

func (g *graph) resolve(name string, kind Kind) (nodeKey, error) {
	var matches []nodeKey
	for _, key := range g.order {
		if key.name == name && (kind == "" || key.kind == kind) {
			matches = append(matches, key)
		}
	}

	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return nodeKey{}, g.unknownName(name, kind)
	}

	nodes := make([]Node, len(matches))
	for i, key := range matches {
		nodes[i] = g.nodes[key]
	}
	return nodeKey{}, &AmbiguousNameError{Name: name, Matches: nodes}
}

func (g *graph) unknownName(name string, kind Kind) error {
	unknown := &UnknownNameError{Name: name}
	if kind != "" {
		unknown.Kinds = []Kind{kind}
		for _, key := range g.order {
			if key.name == name && !slices.Contains(unknown.DeclaredAs, key.kind) {
				unknown.DeclaredAs = append(unknown.DeclaredAs, key.kind)
			}
		}
		if len(unknown.DeclaredAs) > 0 {
			return unknown
		}
	}

	var candidates []string
	for _, key := range g.order {
		if kind == "" || key.kind == kind {
			candidates = append(candidates, key.name)
		}
	}
	unknown.Closest = closestNames(name, candidates)
	return unknown
}

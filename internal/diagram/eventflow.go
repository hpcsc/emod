package diagram

import "github.com/hpcsc/emod/internal/ast"

// NodeKind identifies what a node of an event flow stands for.
type NodeKind int

const (
	// NodeEvent is a fact the log holds.
	NodeEvent NodeKind = iota
	// NodeAutomation is the system acting on its own: an automation, or the
	// reactor a translation declares.
	NodeAutomation
	// NodeOrigin is what issues a command from outside the automations: a
	// trigger, an external system, or a command nothing in the model issues.
	NodeOrigin
	// NodeRejection is an invariant that refuses a command, so the events that
	// command emits are not appended.
	NodeRejection
)

// Node is one shape an event flow draws. ID is what a hop names it by; Label is
// what the shape says, and Caption what is written under it.
type Node struct {
	ID          string
	Kind        NodeKind
	Label       string
	Caption     string
	Description string
	// DeadEnd marks an event nothing in the model reads: no view subscribes to
	// it, no automation activates on it and no command decides on it.
	DeadEnd bool
}

// Hop is one arrow an event flow draws, named by its endpoints.
type Hop struct {
	From  string
	To    string
	Label string
}

// EventFlow is a model with its command chain collapsed: the events it appends,
// the automations that react to them, and what each reaction appends in turn.
type EventFlow struct {
	Nodes []Node
	Hops  []Hop
}

// Captions naming what an origin box stands for. A trigger states its actor
// instead, so a reader is told who works it rather than which keyword declared
// it.
const (
	externalCaption = "external system"
	commandCaption  = "command"
	triggerCaption  = "trigger"
)

// CollapseEventFlow derives the event flow of a model.
//
// The commands are collapsed away: an automation is joined straight to the
// events its command emits, and a command no automation issues is drawn as the
// origin of the events it emits. Views are left out — a view states what a
// processor reads, not what the system does next — so an event only a view
// reads draws no arrow onward and carries no dead-end mark either.
func CollapseEventFlow(model *ast.Model) EventFlow {
	if model == nil {
		return EventFlow{}
	}

	entries := collectSlices(model)
	b := flowBuilder{
		index:    make(map[string]int),
		seen:     make(map[Hop]struct{}),
		emits:    commandEvents(entries),
		issuers:  commandIssuers(entries),
		consumed: consumedEvents(entries),
	}
	for _, entry := range entries {
		b.walk(entry)
	}

	return b.flow()
}

type flowBuilder struct {
	nodes    []Node
	index    map[string]int
	hops     []Hop
	seen     map[Hop]struct{}
	emits    map[string][]string
	issuers  map[string][]Node
	consumed map[string]struct{}
}

func (b *flowBuilder) walk(entry sliceEntry) {
	s := entry.slice

	for _, auto := range s.Automations {
		if auto == nil {
			continue
		}
		b.add(automationNode(auto))
	}
	for _, tr := range s.Translations {
		if tr == nil {
			continue
		}
		b.add(reactorNode(tr))
		if tr.ExternalSystem != "" {
			b.add(externalNode(tr.ExternalSystem, tr.Description))
		}
	}
	for _, evt := range s.Events {
		if evt != nil {
			b.add(eventNode(evt))
		}
	}
	for _, tr := range s.Translations {
		if tr != nil && tr.Event != nil {
			b.add(eventNode(tr.Event))
		}
	}

	for _, tr := range s.Translations {
		if tr == nil || tr.ExternalSystem == "" || tr.Name == "" {
			continue
		}
		b.hop(Hop{From: tr.ExternalSystem, To: tr.Name})
	}
	for _, auto := range s.Automations {
		if auto == nil || auto.OnEvent == "" {
			continue
		}
		b.hop(Hop{From: auto.OnEvent, To: auto.Name, Label: delayLabel(auto.After)})
	}
	for _, cmd := range s.Commands {
		if cmd == nil {
			continue
		}
		for _, event := range b.emits[cmd.Name] {
			for _, origin := range b.producers(cmd.Name) {
				b.hop(Hop{From: origin, To: event})
			}
		}
	}
	for _, tr := range s.Translations {
		if tr == nil || tr.Name == "" {
			continue
		}
		for _, event := range b.emits[tr.Command] {
			b.hop(Hop{From: tr.Name, To: event})
		}
	}
	for _, rejection := range s.Rejections {
		if rejection == nil || rejection.InvariantName == "" {
			continue
		}
		refusal := rejectionNode(entry, rejection.InvariantName)
		b.add(refusal)
		for _, origin := range b.producers(rejection.CommandName) {
			b.hop(Hop{From: origin, To: refusal.ID})
		}
	}
}

// producers names the nodes a command's events and refusals hang off, and draws
// whichever of them the walk has not drawn yet. A command nothing issues stands
// for itself: leaving it out would take the events it emits away from every
// arrow, and the reader would read them as facts nothing produces.
func (b *flowBuilder) producers(command string) []string {
	if command == "" {
		return nil
	}

	issuers := b.issuers[command]
	if len(issuers) == 0 {
		issuers = []Node{commandNode(command)}
	}

	ids := make([]string, 0, len(issuers))
	for _, issuer := range issuers {
		b.add(issuer)
		ids = append(ids, issuer.ID)
	}

	return ids
}

func (b *flowBuilder) add(n Node) {
	if _, drawn := b.index[n.ID]; drawn {
		return
	}

	b.index[n.ID] = len(b.nodes)
	b.nodes = append(b.nodes, n)
}

func (b *flowBuilder) hop(h Hop) {
	if _, made := b.seen[h]; made {
		return
	}

	b.seen[h] = struct{}{}
	b.hops = append(b.hops, h)
}

// flow marks the events nothing reads and drops the hops whose endpoints no
// node answers for, so what comes back is a graph rather than a list of names.
func (b *flowBuilder) flow() EventFlow {
	nodes := make([]Node, len(b.nodes))
	copy(nodes, b.nodes)
	for i, node := range nodes {
		if node.Kind != NodeEvent {
			continue
		}
		if _, read := b.consumed[node.Label]; !read {
			nodes[i].DeadEnd = true
		}
	}

	hops := make([]Hop, 0, len(b.hops))
	for _, hop := range b.hops {
		_, from := b.index[hop.From]
		_, to := b.index[hop.To]
		if from && to {
			hops = append(hops, hop)
		}
	}

	return EventFlow{Nodes: nodes, Hops: hops}
}

func eventNode(evt *ast.Event) Node {
	return Node{ID: evt.Name, Kind: NodeEvent, Label: evt.Name, Description: evt.Description}
}

func automationNode(auto *ast.Automation) Node {
	caption := ""
	if auto.Schedule != "" {
		caption = cadenceLabel(auto.Schedule)
	}

	return Node{ID: auto.Name, Kind: NodeAutomation, Label: auto.Name, Caption: caption, Description: auto.Description}
}

func reactorNode(tr *ast.Translation) Node {
	return Node{ID: tr.Name, Kind: NodeAutomation, Label: tr.Name, Description: tr.Description}
}

func externalNode(name, description string) Node {
	return Node{ID: name, Kind: NodeOrigin, Label: name, Caption: externalCaption, Description: description}
}

func commandNode(name string) Node {
	return Node{ID: name, Kind: NodeOrigin, Label: name, Caption: commandCaption}
}

func triggerNode(entry sliceEntry, trigger *ast.Trigger) Node {
	caption := trigger.Actor
	if caption == "" {
		caption = triggerCaption
	}

	return Node{
		ID:          entry.ctxName + "." + trigger.Name,
		Kind:        NodeOrigin,
		Label:       trigger.Name,
		Caption:     caption,
		Description: trigger.Description,
	}
}

// rejectionNode keys an invariant by the context declaring it, which is as far
// as one name reaches: two contexts may each declare a rule of that name, and
// one box for both would claim they are one rule.
func rejectionNode(entry sliceEntry, invariant string) Node {
	return Node{
		ID:          entry.ctxName + "." + invariant,
		Kind:        NodeRejection,
		Label:       invariant,
		Description: entry.invariantStatement(invariant),
	}
}

// commandEvents maps each command to the events it emits, in declaration order.
// A translation's nested event is one of them unless the slice already states
// that flow, which would otherwise list the same event twice.
func commandEvents(entries []sliceEntry) map[string][]string {
	emits := make(map[string][]string)
	appendEvent := func(command, event string) {
		if command == "" || event == "" {
			return
		}
		for _, have := range emits[command] {
			if have == event {
				return
			}
		}
		emits[command] = append(emits[command], event)
	}

	for _, entry := range entries {
		for _, flow := range entry.slice.Flows {
			if flow != nil {
				appendEvent(flow.CommandName, flow.EventName)
			}
		}
		for _, tr := range entry.slice.Translations {
			if tr == nil || tr.Event == nil || declaresFlow(entry.slice, tr.Command, tr.Event.Name) {
				continue
			}
			appendEvent(tr.Command, tr.Event.Name)
		}
	}

	return emits
}

// commandIssuers maps each command to what issues it: the trigger of the slice
// declaring it, an automation, or a translation reactor.
func commandIssuers(entries []sliceEntry) map[string][]Node {
	issuers := make(map[string][]Node)
	add := func(command string, node Node) {
		if command == "" {
			return
		}
		for _, have := range issuers[command] {
			if have.ID == node.ID {
				return
			}
		}
		issuers[command] = append(issuers[command], node)
	}

	for _, entry := range entries {
		s := entry.slice
		if s.Trigger != nil {
			trigger := triggerNode(entry, s.Trigger)
			for _, cmd := range s.Commands {
				if cmd != nil {
					add(cmd.Name, trigger)
				}
			}
		}
		for _, auto := range s.Automations {
			if auto != nil {
				add(auto.Command, automationNode(auto))
			}
		}
		for _, tr := range s.Translations {
			if tr != nil {
				add(tr.Command, reactorNode(tr))
			}
		}
	}

	return issuers
}

// consumedEvents names every event the model reads back.
func consumedEvents(entries []sliceEntry) map[string]struct{} {
	consumed := make(map[string]struct{})
	for _, entry := range entries {
		s := entry.slice
		for _, view := range s.Views {
			if view == nil {
				continue
			}
			for _, sub := range view.Subscribes {
				consumed[sub] = struct{}{}
			}
		}
		for _, auto := range s.Automations {
			if auto != nil && auto.OnEvent != "" {
				consumed[auto.OnEvent] = struct{}{}
			}
		}
		for _, cmd := range s.Commands {
			if cmd == nil || cmd.DecidesOn == nil {
				continue
			}
			for _, event := range cmd.DecidesOn.Events {
				consumed[event] = struct{}{}
			}
		}
	}

	return consumed
}

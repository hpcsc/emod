package diagram

import "github.com/hpcsc/emod/internal/ast"

// EdgeKind identifies the semantic relation an Edge carries.
type EdgeKind int

const (
	// EdgeFlow is a command -> event arrow a flow declares.
	EdgeFlow EdgeKind = iota
	// EdgeTriggerReads is a view -> trigger arrow from a trigger's reads entry.
	EdgeTriggerReads
	// EdgeTriggerCommand is a trigger -> command arrow within a slice.
	EdgeTriggerCommand
	// EdgeSubscription is an event -> view arrow from a view's subscribes list.
	EdgeSubscription
	// EdgeAutomationTrigger is an event -> automation arrow from an automation's
	// activating event.
	EdgeAutomationTrigger
	// EdgeAutomationReads is a view -> automation arrow from an automation's
	// reads entry.
	EdgeAutomationReads
	// EdgeAutomationCommand is an automation -> command arrow.
	EdgeAutomationCommand
	// EdgeTranslationReads is a view -> translation reactor arrow from a
	// translation's reads entry. The drawn formats route it into the external
	// system box fronting the reactor.
	EdgeTranslationReads
	// EdgeTranslationExternal is an external system -> translation reactor arrow.
	EdgeTranslationExternal
	// EdgeTranslationCommand is a translation reactor -> command arrow.
	EdgeTranslationCommand
	// EdgeTranslationFlow is the command -> event arrow a translation implies.
	// It is derived only when no flow already declares the same pair, so a
	// consumer never draws that arrow twice.
	EdgeTranslationFlow
)

// Edge is one semantic arrow a slice declares, named by its endpoints.
// Deriving edges in one place keeps every renderer and the diagram-JSON
// exporter describing the same picture.
type Edge struct {
	Kind EdgeKind
	From string
	To   string
}

// SliceEdges returns the edges a slice declares. Endpoints are names, not
// resolved elements: whether an edge is drawn stays with the caller, which
// skips edges whose endpoints it has not drawn.
func SliceEdges(s *ast.Slice) []Edge {
	var edges []Edge

	for _, f := range s.Flows {
		if f == nil {
			continue
		}
		edges = append(edges, Edge{Kind: EdgeFlow, From: f.CommandName, To: f.EventName})
	}

	if s.Trigger != nil {
		if s.Trigger.Reads != "" {
			edges = append(edges, Edge{Kind: EdgeTriggerReads, From: s.Trigger.Reads, To: s.Trigger.Name})
		}
		for _, cmd := range s.Commands {
			if cmd == nil {
				continue
			}
			edges = append(edges, Edge{Kind: EdgeTriggerCommand, From: s.Trigger.Name, To: cmd.Name})
		}
	}

	for _, v := range s.Views {
		if v == nil {
			continue
		}
		for _, sub := range v.Subscribes {
			edges = append(edges, Edge{Kind: EdgeSubscription, From: sub, To: v.Name})
		}
	}

	for _, a := range s.Automations {
		if a == nil {
			continue
		}
		if a.OnEvent != "" {
			edges = append(edges, Edge{Kind: EdgeAutomationTrigger, From: a.OnEvent, To: a.Name})
		}
		if a.Reads != "" {
			edges = append(edges, Edge{Kind: EdgeAutomationReads, From: a.Reads, To: a.Name})
		}
		if a.Command != "" {
			edges = append(edges, Edge{Kind: EdgeAutomationCommand, From: a.Name, To: a.Command})
		}
	}

	for _, tr := range s.Translations {
		if tr == nil {
			continue
		}
		if tr.Reads != "" {
			edges = append(edges, Edge{Kind: EdgeTranslationReads, From: tr.Reads, To: tr.Name})
		}
		edges = append(edges, Edge{Kind: EdgeTranslationExternal, From: tr.ExternalSystem, To: tr.Name})
		if tr.Command != "" {
			edges = append(edges, Edge{Kind: EdgeTranslationCommand, From: tr.Name, To: tr.Command})
		}
		if tr.Command != "" && tr.Event != nil && tr.Event.Name != "" &&
			!declaresFlow(s, tr.Command, tr.Event.Name) {
			edges = append(edges, Edge{Kind: EdgeTranslationFlow, From: tr.Command, To: tr.Event.Name})
		}
	}

	return edges
}

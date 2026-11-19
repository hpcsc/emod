package lsp

import (
	"slices"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/query"
)

// referenceTargetKinds omits query.KindContext: of the names resolved model-wide,
// find-references answers for a command, an event, a view and a screen alone. An
// invariant is resolved per scope instead, by invariantLocations.
var referenceTargetKinds = []query.Kind{query.KindCommand, query.KindEvent, query.KindView, query.KindScreen}

// GetReferences finds all references to the command, event, view or screen name at the
// given cursor position, or — for an invariant — the declaration and every site
// naming it within the one scope that declares it. If the cursor is not on a
// resolvable name (definition or reference), it returns nil.
//
// Positions are 0-based LSP coordinates (line, character).
func GetReferences(text string, line, character int, uri string) []Location {
	if text == "" {
		return nil
	}

	model, _ := parseModel(text, uri)
	if model == nil {
		return nil
	}

	declared := query.NewDeclaredNames(model)
	sites := query.NameSites(model)

	at := cursorAt(line, character)

	target, ok := targetAt(at, declared, sites)
	if !ok {
		return invariantLocations(at, model, uri)
	}

	var locations []Location
	if pos, ok := declared.PositionOf(target.Kind, target.Name); ok {
		locations = append(locations, *locationFor(uri, pos, target.Name))
	}
	for _, site := range sites {
		if site.Kind == target.Kind && site.Name == target.Name {
			locations = append(locations, *locationFor(uri, site.Pos, site.Name))
		}
	}

	return locations
}

// invariantLocations lists an invariant's declaration and every site naming it
// within the one scope that declares it. Invariants stay out of declaredNames
// and referenceTargetKinds because a flat name map cannot hold two scopes
// declaring one name, so they are resolved here rather than through targetAt.
func invariantLocations(at cursor, model *ast.Model, uri string) []Location {
	for _, scope := range query.InvariantScopes(model) {
		name, ok := declaredNameAt(scope, at)
		if !ok {
			if name, ok = referencedNameAt(scope, at); !ok {
				continue
			}
		}
		inv, declared := scope.DeclarationOf(name)
		if !declared {
			return nil
		}

		locations := []Location{*locationFor(uri, inv.NamePos, inv.Name)}
		for _, site := range scope.Sites {
			if site.Name == name {
				locations = append(locations, *locationFor(uri, site.Pos, site.Name))
			}
		}
		return locations
	}

	return nil
}

// targetAt resolves the name the cursor sits on, whether it sits on the
// declaration or on a site referencing it.
func targetAt(at cursor, declared query.DeclaredNames, sites []query.NameSite) (query.NameSite, bool) {
	for _, kind := range referenceTargetKinds {
		for name, pos := range declared[kind] {
			if at.onName(pos, name) {
				return query.NameSite{Kind: kind, Name: name, Pos: pos}, true
			}
		}
	}

	for _, site := range sites {
		if !slices.Contains(referenceTargetKinds, site.Kind) || !at.onName(site.Pos, site.Name) {
			continue
		}
		if _, ok := declared.PositionOf(site.Kind, site.Name); ok {
			return site, true
		}
	}

	return query.NameSite{}, false
}

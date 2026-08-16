package lsp

import (
	"slices"

	"github.com/hpcsc/emod/internal/ast"
)

// referenceTargetKinds omits contextName: of the names resolved model-wide,
// find-references answers for a command, an event and a view alone. An invariant
// is resolved per scope instead, by invariantLocations.
var referenceTargetKinds = []nameKind{commandName, eventName, viewName}

// GetReferences finds all references to the command, event or view name at the
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

	declared := newDeclaredNames(model)
	refs := referencesIn(model)

	at := cursorAt(line, character)

	target, ok := targetAt(at, declared, refs)
	if !ok {
		return invariantLocations(at, model, uri)
	}

	var locations []Location
	if pos, ok := declared.positionOf(target.kind, target.name); ok {
		locations = append(locations, *locationFor(uri, pos, target.name))
	}
	for _, ref := range refs {
		if ref.kind == target.kind && ref.name == target.name {
			locations = append(locations, *locationFor(uri, ref.pos, ref.name))
		}
	}

	return locations
}

// invariantLocations lists an invariant's declaration and every site naming it
// within the one scope that declares it. Invariants stay out of declaredNames
// and referenceTargetKinds because a flat name map cannot hold two scopes
// declaring one name, so they are resolved here rather than through targetAt.
func invariantLocations(at cursor, model *ast.Model, uri string) []Location {
	for _, scope := range invariantScopes(model) {
		name, ok := scope.declaredNameAt(at)
		if !ok {
			if name, ok = scope.referencedNameAt(at); !ok {
				continue
			}
		}
		inv, declared := scope.declarationOf(name)
		if !declared {
			return nil
		}

		locations := []Location{*locationFor(uri, inv.NamePos, inv.Name)}
		for _, ref := range scope.references {
			if ref.name == name {
				locations = append(locations, *locationFor(uri, ref.pos, ref.name))
			}
		}
		return locations
	}

	return nil
}

// targetAt resolves the name the cursor sits on, whether it sits on the
// declaration or on a site referencing it.
func targetAt(at cursor, declared declaredNames, refs []nameRef) (nameRef, bool) {
	for _, kind := range referenceTargetKinds {
		for name, pos := range declared[kind] {
			if at.onName(pos, name) {
				return nameRef{kind: kind, name: name, pos: pos}, true
			}
		}
	}

	for _, ref := range refs {
		if !slices.Contains(referenceTargetKinds, ref.kind) || !at.onName(ref.pos, ref.name) {
			continue
		}
		if _, ok := declared.positionOf(ref.kind, ref.name); ok {
			return ref, true
		}
	}

	return nameRef{}, false
}

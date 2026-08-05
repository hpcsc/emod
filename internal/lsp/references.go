package lsp

import "slices"

// referenceTargetKinds omits contextName: find-references resolves a cursor
// sitting on a command, event or view, and on nothing else.
var referenceTargetKinds = []nameKind{commandName, eventName, viewName}

// GetReferences finds all references to the command, event, or view name at the
// given cursor position. If the cursor is not on a resolvable name (definition
// or reference), it returns nil.
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

	target, ok := targetAt(cursorAt(line, character), declared, refs)
	if !ok {
		return nil
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

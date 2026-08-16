package lsp

// GetDefinition finds the definition location for the reference at the given
// cursor position. If the cursor is not on a known reference, or the referenced
// name has no definition in the document, it returns nil.
//
// Positions are 0-based LSP coordinates (line, character).
func GetDefinition(text string, line, character int, uri string) *Location {
	if text == "" {
		return nil
	}

	model, _ := parseModel(text, uri)
	if model == nil {
		return nil
	}

	declared := newDeclaredNames(model)
	at := cursorAt(line, character)

	for _, ref := range referencesIn(model) {
		if !at.onName(ref.pos, ref.name) {
			continue
		}
		if pos, ok := declared.positionOf(ref.kind, ref.name); ok {
			return locationFor(uri, pos, ref.name)
		}
	}

	for _, scope := range invariantScopes(model) {
		name, ok := scope.referencedNameAt(at)
		if !ok {
			continue
		}
		if inv, declared := scope.declarationOf(name); declared {
			return locationFor(uri, inv.NamePos, inv.Name)
		}
	}

	return nil
}

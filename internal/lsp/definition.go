package lsp

import "github.com/hpcsc/emod/internal/query"

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

	declared := query.NewDeclaredNames(model)
	at := cursorAt(line, character)

	for _, site := range query.NameSites(model) {
		if !at.onName(site.Pos, site.Name) {
			continue
		}
		if pos, ok := declared.PositionOf(site.Kind, site.Name); ok {
			return locationFor(uri, pos, site.Name)
		}
	}

	for _, scope := range query.InvariantScopes(model) {
		name, ok := referencedNameAt(scope, at)
		if !ok {
			continue
		}
		if inv, declared := scope.DeclarationOf(name); declared {
			return locationFor(uri, inv.NamePos, inv.Name)
		}
	}

	return nil
}

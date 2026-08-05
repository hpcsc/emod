package lsp

import "github.com/hpcsc/emod/internal/ast"

// cursor is a caret in the AST's 1-based line and column coordinates, converted
// from the 0-based line and character an LSP request carries.
type cursor struct {
	line   int
	column int
}

func cursorAt(line, character int) cursor {
	return cursor{line: line + 1, column: character + 1}
}

func (c cursor) onName(pos ast.Position, name string) bool {
	if name == "" {
		return false
	}
	return c.line == pos.Line &&
		c.column >= pos.Column &&
		c.column < pos.Column+len(name)
}

// locationFor builds an LSP Location for the given definition position,
// converting from 1-based AST coordinates to 0-based LSP coordinates.
// The range covers the full definition name.
func locationFor(uri string, pos ast.Position, name string) *Location {
	if name == "" {
		return nil
	}
	return &Location{
		URI:   uri,
		Range: *nameRange(pos, name),
	}
}

// nameRange builds an LSP Range for the given name position,
// converting from 1-based AST coordinates to 0-based LSP coordinates.
func nameRange(pos ast.Position, name string) *Range {
	if name == "" {
		return nil
	}
	return &Range{
		Start: Position{
			Line:      pos.Line - 1,
			Character: pos.Column - 1,
		},
		End: Position{
			Line:      pos.Line - 1,
			Character: pos.Column - 1 + len(name),
		},
	}
}

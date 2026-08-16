package lsp

import (
	"unicode/utf8"

	"github.com/hpcsc/emod/internal/ast"
)

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
		c.column < pos.Column+nameWidth(name)
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

// nameWidth is how far a name reaches from its own first character. An LSP
// position counts characters where Go's len counts bytes, so a quoted name
// holding any non-ASCII text would otherwise stretch a character past itself
// per multibyte rune and swallow its own closing quote.
func nameWidth(name string) int {
	return utf8.RuneCountInString(name)
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
			Character: pos.Column - 1 + nameWidth(name),
		},
	}
}

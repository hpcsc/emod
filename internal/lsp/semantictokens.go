package lsp

import (
	"sort"

	"github.com/hpcsc/emod/internal/ast"
)

// tokenTypeIndex returns the legend index for a given semantic token type.
func tokenTypeIndex(t SemanticTokenTypes) uint {
	switch t {
	case TokenTypeFunction:
		return 0
	case TokenTypeEvent:
		return 1
	case TokenTypeClass:
		return 2
	case TokenTypeParameter:
		return 3
	case TokenTypeNamespace:
		return 4
	case TokenTypeStruct:
		return 5
	default:
		return 0
	}
}

// GetSemanticTokensLegend returns the canonical legend for semantic token types
// used by this server. The indices must match tokenTypeIndex.
func GetSemanticTokensLegend() SemanticTokensLegend {
	return SemanticTokensLegend{
		TokenTypes: []string{
			string(TokenTypeFunction),
			string(TokenTypeEvent),
			string(TokenTypeClass),
			string(TokenTypeParameter),
			string(TokenTypeNamespace),
			string(TokenTypeStruct),
		},
		TokenModifiers: []string{},
	}
}

// tokenEntry represents one semantic token with 0-based LSP coordinates.
type tokenEntry struct {
	line      int
	character int
	length    int
	typeIndex uint
}

type tokenEntries []tokenEntry

func (e *tokenEntries) addIdentifier(pos ast.Position, name string, t SemanticTokenTypes) {
	e.add(pos.Line-1, pos.Column-1, name, t)
}

// addQuoted adds a name written in double quotes, whose position points at the
// opening quote rather than at the first character of the name.
func (e *tokenEntries) addQuoted(pos ast.Position, name string, t SemanticTokenTypes) {
	e.add(pos.Line-1, pos.Column, name, t)
}

func (e *tokenEntries) add(line, character int, name string, t SemanticTokenTypes) {
	if name == "" {
		return
	}
	*e = append(*e, tokenEntry{
		line:      line,
		character: character,
		length:    len(name),
		typeIndex: tokenTypeIndex(t),
	})
}

// GetSemanticTokens parses an emod document and returns delta-encoded semantic
// token data mapping named identifiers to their token types.
//
// Named identifiers receive the following token types:
//   - Commands → TokenTypeFunction
//   - Events → TokenTypeEvent
//   - Views → TokenTypeClass
//   - Actors → TokenTypeParameter
//   - Contexts → TokenTypeNamespace
//   - Aggregates → TokenTypeStruct
//
// Documents with parse errors or no named identifiers return empty semantic
// tokens (zero tokens in the data array).
func GetSemanticTokens(doc string) *SemanticTokens {
	if doc == "" {
		return &SemanticTokens{Data: []uint{}}
	}

	model, _ := parseModel(doc, "")
	if model == nil {
		return &SemanticTokens{Data: []uint{}}
	}

	var entries tokenEntries

	for _, ctx := range model.Contexts {
		entries.addQuoted(ctx.NamePos, ctx.Name, TokenTypeNamespace)
	}

	for _, agg := range declaredAggregates(model) {
		entries.addQuoted(agg.NamePos, agg.Name, TokenTypeStruct)
	}

	for _, scoped := range scopedSlices(model) {
		for _, cmd := range scoped.slice.Commands {
			entries.addIdentifier(cmd.NamePos, cmd.Name, TokenTypeFunction)
		}
		for _, evt := range scoped.slice.Events {
			entries.addIdentifier(evt.NamePos, evt.Name, TokenTypeEvent)
		}
		for _, v := range scoped.slice.Views {
			entries.addIdentifier(v.NamePos, v.Name, TokenTypeClass)
		}
	}

	for _, actor := range model.Actors {
		entries.addQuoted(actor.NamePos, actor.Name, TokenTypeParameter)
	}

	if len(entries) == 0 {
		return &SemanticTokens{Data: []uint{}}
	}

	// Sort by position: line first, then character.
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].line != entries[j].line {
			return entries[i].line < entries[j].line
		}
		return entries[i].character < entries[j].character
	})

	// Delta-encode per LSP semantic tokens spec.
	// First token uses absolute position from (0,0).
	// Subsequent tokens use relative deltas from the previous token.
	data := make([]uint, 0, len(entries)*5)
	var prevLine, prevChar int

	for _, e := range entries {
		deltaLine := e.line - prevLine
		deltaChar := e.character
		if deltaLine == 0 {
			deltaChar = e.character - prevChar
		}
		data = append(data, uint(deltaLine), uint(deltaChar), uint(e.length), e.typeIndex, 0)
		prevLine = e.line
		prevChar = e.character
	}

	return &SemanticTokens{Data: data}
}

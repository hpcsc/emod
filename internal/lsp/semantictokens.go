package lsp

import (
	"sort"

	"github.com/hpcsc/emod/internal/lexer"
	"github.com/hpcsc/emod/internal/parser"
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

	tokens, _ := lexer.Scan(doc, "")
	p := parser.New(tokens, "")
	model, _ := p.Parse()
	if model == nil {
		return &SemanticTokens{Data: []uint{}}
	}

	var entries []tokenEntry

	for _, ctx := range model.Contexts {
		if ctx.Name != "" {
			// Context name is a quoted string — shift column past the opening quote.
			entries = append(entries, tokenEntry{
				line:      ctx.NamePos.Line - 1,
				character: ctx.NamePos.Column, // Column points to '"', skip to name content
				length:    len(ctx.Name),
				typeIndex: tokenTypeIndex(TokenTypeNamespace),
			})
		}
		for _, agg := range ctx.Aggregates {
			if agg.Name != "" {
				// Aggregate name is a quoted string — shift column past the opening quote.
				entries = append(entries, tokenEntry{
					line:      agg.NamePos.Line - 1,
					character: agg.NamePos.Column, // Column points to '"', skip to name content
					length:    len(agg.Name),
					typeIndex: tokenTypeIndex(TokenTypeStruct),
				})
			}
			for _, slice := range agg.Slices {
				for _, cmd := range slice.Commands {
					if cmd.Name != "" {
						entries = append(entries, tokenEntry{
							line:      cmd.NamePos.Line - 1,
							character: cmd.NamePos.Column - 1,
							length:    len(cmd.Name),
							typeIndex: tokenTypeIndex(TokenTypeFunction),
						})
					}
				}
				for _, evt := range slice.Events {
					if evt.Name != "" {
						entries = append(entries, tokenEntry{
							line:      evt.NamePos.Line - 1,
							character: evt.NamePos.Column - 1,
							length:    len(evt.Name),
							typeIndex: tokenTypeIndex(TokenTypeEvent),
						})
					}
				}
				for _, v := range slice.Views {
					if v.Name != "" {
						entries = append(entries, tokenEntry{
							line:      v.NamePos.Line - 1,
							character: v.NamePos.Column - 1,
							length:    len(v.Name),
							typeIndex: tokenTypeIndex(TokenTypeClass),
						})
					}
				}
			}
		}
	}

	for _, actor := range model.Actors {
		if actor.Name != "" {
			// Actor name is a quoted string — shift column past the opening quote.
			entries = append(entries, tokenEntry{
				line:      actor.NamePos.Line - 1,
				character: actor.NamePos.Column, // Column points to '"', skip to name content
				length:    len(actor.Name),
				typeIndex: tokenTypeIndex(TokenTypeParameter),
			})
		}
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

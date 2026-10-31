package lsp

import (
	"slices"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/oracle"
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

type semanticDecl struct {
	nameStart ast.Position
	name      string
	tokenType SemanticTokenTypes
}

// GetSemanticTokens parses an emod document and returns delta-encoded semantic
// token data mapping declared names to their token types.
//
// Declared names receive the following token types:
//   - Commands → TokenTypeFunction
//   - Events → TokenTypeEvent
//   - Views → TokenTypeClass
//   - Actors → TokenTypeParameter
//   - Contexts → TokenTypeNamespace
//   - Aggregates → TokenTypeStruct
//
// A document that declares no names, including text the parser recovers nothing
// from, returns empty semantic tokens (zero tokens in the data array).
func GetSemanticTokens(doc string) *SemanticTokens {
	decls := semanticDeclsOf(doc)
	slices.SortFunc(decls, func(a, b semanticDecl) int {
		return a.nameStart.Compare(b.nameStart)
	})

	// Delta-encode per LSP semantic tokens spec.
	// First token uses absolute position from (0,0).
	// Subsequent tokens use relative deltas from the previous token.
	data := []uint{}
	var prevLine, prevChar int

	for _, decl := range decls {
		line, char := decl.nameStart.Line-1, decl.nameStart.Column-1
		deltaLine := line - prevLine
		deltaChar := char
		if deltaLine == 0 {
			deltaChar = char - prevChar
		}
		data = append(data, uint(deltaLine), uint(deltaChar), uint(len(decl.name)), tokenTypeIndex(decl.tokenType), 0)
		prevLine = line
		prevChar = char
	}

	return &SemanticTokens{Data: data}
}

func semanticDeclsOf(doc string) []semanticDecl {
	model, _ := oracle.Parse(doc, "")

	var decls []semanticDecl
	add := func(pos ast.Position, name string, t SemanticTokenTypes) {
		if name == "" {
			return
		}
		decls = append(decls, semanticDecl{nameStart: afterQuote(pos), name: name, tokenType: t})
	}

	for _, ctx := range model.Contexts {
		add(ctx.NamePos, ctx.Name, TokenTypeNamespace)
	}

	for _, agg := range declaredAggregates(model) {
		add(agg.NamePos, agg.Name, TokenTypeStruct)
	}

	for _, slice := range model.AllSlices() {
		for _, cmd := range slice.Commands {
			add(cmd.NamePos, cmd.Name, TokenTypeFunction)
		}
		for _, evt := range slice.Events {
			add(evt.NamePos, evt.Name, TokenTypeEvent)
		}
		for _, v := range slice.Views {
			add(v.NamePos, v.Name, TokenTypeClass)
		}
	}

	for _, actor := range model.Actors {
		add(actor.NamePos, actor.Name, TokenTypeParameter)
	}

	return decls
}

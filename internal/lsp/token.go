package lsp

import (
	"bytes"
	"sort"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// Token is a stretch of a document and the category an editor colours it with.
// Start and End count UTF-16 code units, the unit a browser indexes text in.
type Token struct {
	Start    int    `json:"start"`
	End      int    `json:"end"`
	Category string `json:"category"`
}

const (
	categoryOperator             = "operator"
	categoryPunctuationBracket   = "punctuation.bracket"
	categoryPunctuationDelimiter = "punctuation.delimiter"
	categoryPunctuationSpecial   = "punctuation.special"
	categoryKeyword              = "keyword"
	categoryKeywordRepeat        = "keyword.repeat"
	categoryKeywordConditional   = "keyword.conditional"
	categoryType                 = "type"
	categoryString               = "string"
	categoryNumber               = "number"
	categoryBoolean              = "boolean"
	categoryConstant             = "constant"
	categoryComment              = "comment"
	categoryVariable             = "variable"
	categoryVariableMember       = "variable.member"
	categoryVariableBuiltin      = "variable.builtin"
	categoryFunction             = "function"
	categorySemanticPrefix       = "lsp.type."
)

var lexicalCategories = map[hclsyntax.TokenType]string{
	hclsyntax.TokenBang:            categoryOperator,
	hclsyntax.TokenStar:            categoryOperator,
	hclsyntax.TokenSlash:           categoryOperator,
	hclsyntax.TokenPercent:         categoryOperator,
	hclsyntax.TokenPlus:            categoryOperator,
	hclsyntax.TokenMinus:           categoryOperator,
	hclsyntax.TokenGreaterThan:     categoryOperator,
	hclsyntax.TokenGreaterThanEq:   categoryOperator,
	hclsyntax.TokenLessThan:        categoryOperator,
	hclsyntax.TokenLessThanEq:      categoryOperator,
	hclsyntax.TokenEqualOp:         categoryOperator,
	hclsyntax.TokenNotEqual:        categoryOperator,
	hclsyntax.TokenAnd:             categoryOperator,
	hclsyntax.TokenOr:              categoryOperator,
	hclsyntax.TokenOBrace:          categoryPunctuationBracket,
	hclsyntax.TokenCBrace:          categoryPunctuationBracket,
	hclsyntax.TokenOBrack:          categoryPunctuationBracket,
	hclsyntax.TokenCBrack:          categoryPunctuationBracket,
	hclsyntax.TokenOParen:          categoryPunctuationBracket,
	hclsyntax.TokenCParen:          categoryPunctuationBracket,
	hclsyntax.TokenDot:             categoryPunctuationDelimiter,
	hclsyntax.TokenComma:           categoryPunctuationDelimiter,
	hclsyntax.TokenEllipsis:        categoryPunctuationSpecial,
	hclsyntax.TokenQuestion:        categoryPunctuationSpecial,
	hclsyntax.TokenFatArrow:        categoryPunctuationSpecial,
	hclsyntax.TokenTemplateInterp:  categoryPunctuationSpecial,
	hclsyntax.TokenTemplateControl: categoryPunctuationSpecial,
	hclsyntax.TokenTemplateSeqEnd:  categoryPunctuationSpecial,
	hclsyntax.TokenOQuote:          categoryString,
	hclsyntax.TokenCQuote:          categoryString,
	hclsyntax.TokenQuotedLit:       categoryString,
	hclsyntax.TokenStringLit:       categoryString,
	hclsyntax.TokenOHeredoc:        categoryPunctuationDelimiter,
	hclsyntax.TokenCHeredoc:        categoryPunctuationDelimiter,
	hclsyntax.TokenNumberLit:       categoryNumber,
	hclsyntax.TokenComment:         categoryComment,
	hclsyntax.TokenIdent:           categoryVariable,
}

// GetTokens answers the stretches of an emod document and the category of each,
// as an editor colours them with the HCL grammar and emod lsp's semantic tokens.
// The tokens come in document order and never overlap.
func GetTokens(doc string) []Token {
	src := []byte(doc)
	lexed, _ := hclsyntax.LexConfig(src, "", hcl.InitialPos)

	categories := newTokenCategories(src, lexed)
	if file, _ := hclsyntax.ParseConfig(src, "", hcl.InitialPos); file != nil {
		if body, ok := file.Body.(*hclsyntax.Body); ok {
			categories.addBody(body, categoryKeyword)
		}
	}
	categories.addTemplateKeywords()
	categories.addSemanticDecls(semanticDeclsOf(doc))

	offsets := utf16Offsets(src)
	tokens := []Token{}
	for _, lexedToken := range lexed {
		category, ok := categories.of(lexedToken)
		if !ok {
			continue
		}
		start, end := colouredBytes(lexedToken)
		tokens = append(tokens, Token{Start: offsets[start], End: offsets[end], Category: category})
	}

	return tokens
}

type tokenCategories struct {
	src     []byte
	lexed   hclsyntax.Tokens
	byStart map[int]string
}

func newTokenCategories(src []byte, lexed hclsyntax.Tokens) *tokenCategories {
	return &tokenCategories{src: src, lexed: lexed, byStart: map[int]string{}}
}

func (c *tokenCategories) addBody(body *hclsyntax.Body, blockCategory string) {
	for _, attr := range body.Attributes {
		c.byStart[attr.NameRange.Start.Byte] = categoryVariableMember
		c.addExpression(attr.Expr)
	}
	for _, block := range body.Blocks {
		c.byStart[block.TypeRange.Start.Byte] = blockCategory
		for _, label := range block.LabelRanges {
			if c.src[label.Start.Byte] != '"' {
				c.byStart[label.Start.Byte] = blockCategory
			}
		}
		if block.Body != nil {
			c.addBody(block.Body, categoryType)
		}
	}
}

func (c *tokenCategories) addExpression(expr hclsyntax.Expression) {
	hclsyntax.VisitAll(expr, func(node hclsyntax.Node) hcl.Diagnostics {
		switch e := node.(type) {
		case *hclsyntax.FunctionCallExpr:
			c.byStart[e.NameRange.Start.Byte] = categoryFunction
		case *hclsyntax.LiteralValueExpr:
			c.addLiteral(e)
		case *hclsyntax.ObjectConsExpr:
			c.addObjectKeys(e)
		case *hclsyntax.ScopeTraversalExpr:
			c.addReference(e, e.Traversal)
		case *hclsyntax.RelativeTraversalExpr:
			if root, ok := c.rootOf(e.Source); ok {
				c.addReference(root, e.Traversal)
			}
		case *hclsyntax.SplatExpr:
			c.addSplatMarker(e.MarkerRange)
		case *hclsyntax.ForExpr:
			c.addForKeywords(e)
		}
		return nil
	})
}

func (c *tokenCategories) addLiteral(e *hclsyntax.LiteralValueExpr) {
	switch {
	case e.Val.Type() == cty.Bool:
		c.byStart[e.SrcRange.Start.Byte] = categoryBoolean
	case e.Val.IsNull():
		c.byStart[e.SrcRange.Start.Byte] = categoryConstant
	}
}

func (c *tokenCategories) addObjectKeys(e *hclsyntax.ObjectConsExpr) {
	for _, item := range e.Items {
		if key, ok := item.KeyExpr.(*hclsyntax.ObjectConsKeyExpr); ok {
			if traversal, ok := key.Wrapped.(*hclsyntax.ScopeTraversalExpr); ok {
				c.byStart[traversal.Traversal[0].SourceRange().Start.Byte] = categoryVariableMember
			}
		}
	}
}

func (c *tokenCategories) addReference(root *hclsyntax.ScopeTraversalExpr, steps hcl.Traversal) {
	followed := false
	for _, step := range steps {
		if attr, ok := step.(hcl.TraverseAttr); ok {
			followed = true
			c.byStart[attr.SrcRange.End.Byte-len(attr.Name)] = categoryVariableMember
		}
	}
	if followed {
		c.byStart[root.Traversal[0].SourceRange().Start.Byte] = categoryVariableBuiltin
	}
}

// HCL stretches the marker of an unclosed [* to the next ] or to the end of the
// document, so only the marker's own tokens are coloured.
func (c *tokenCategories) addSplatMarker(marker hcl.Range) {
	for i := c.indexAt(marker.Start.Byte); c.lexed[i].Range.Start.Byte < marker.End.Byte && c.isSplatMarker(c.lexed[i]); i++ {
		c.byStart[c.lexed[i].Range.Start.Byte] = categoryPunctuationDelimiter
	}
}

func (c *tokenCategories) isSplatMarker(token hclsyntax.Token) bool {
	switch token.Type {
	case hclsyntax.TokenOBrack, hclsyntax.TokenStar, hclsyntax.TokenCBrack, hclsyntax.TokenDot:
		return true
	}
	return false
}

// In a template the expression opens with the whole %{ for … } directive, so
// for is the token after the one that opens it in both forms.
func (c *tokenCategories) addForKeywords(e *hclsyntax.ForExpr) {
	c.byStart[c.tokenAfter(e.OpenRange.Start.Byte).Range.Start.Byte] = categoryKeywordRepeat
	c.byStart[c.tokenBefore(e.CollExpr.Range().Start.Byte).Range.Start.Byte] = categoryKeywordRepeat
	if e.CondExpr != nil {
		c.byStart[c.tokenBefore(e.CondExpr.Range().Start.Byte).Range.Start.Byte] = categoryKeywordConditional
	}
}

// addTemplateKeywords colours the endfor, if, else and endif that open a
// template's %{ … } directives, which the parser keeps no range for.
func (c *tokenCategories) addTemplateKeywords() {
	previous := hclsyntax.TokenNil
	for _, token := range c.lexed {
		if previous == hclsyntax.TokenTemplateControl {
			switch string(token.Bytes) {
			case "endfor":
				c.byStart[token.Range.Start.Byte] = categoryKeywordRepeat
			case "if", "else", "endif":
				c.byStart[token.Range.Start.Byte] = categoryKeywordConditional
			}
		}
		previous = token.Type
	}
}

func (c *tokenCategories) addSemanticDecls(decls []semanticDecl) {
	byPosition := map[[2]int]string{}
	for _, decl := range decls {
		byPosition[[2]int{decl.nameStart.Line, decl.nameStart.Column}] = categorySemanticPrefix + string(decl.tokenType)
	}
	for _, token := range c.lexed {
		category, ok := byPosition[[2]int{token.Range.Start.Line, token.Range.Start.Column}]
		if ok && token.Type == hclsyntax.TokenQuotedLit {
			c.byStart[token.Range.Start.Byte] = category
		}
	}
}

func (c *tokenCategories) indexAt(at int) int {
	return sort.Search(len(c.lexed), func(i int) bool {
		return c.lexed[i].Range.Start.Byte >= at
	})
}

func (c *tokenCategories) tokenAfter(at int) hclsyntax.Token {
	i := c.indexAt(at) + 1
	for c.isNewlineOrComment(c.lexed[i]) {
		i++
	}
	return c.lexed[i]
}

func (c *tokenCategories) tokenBefore(at int) hclsyntax.Token {
	i := c.indexAt(at) - 1
	for c.isNewlineOrComment(c.lexed[i]) {
		i--
	}
	return c.lexed[i]
}

func (c *tokenCategories) isNewlineOrComment(token hclsyntax.Token) bool {
	return token.Type == hclsyntax.TokenNewline || token.Type == hclsyntax.TokenComment
}

func (c *tokenCategories) rootOf(expr hclsyntax.Expression) (*hclsyntax.ScopeTraversalExpr, bool) {
	for {
		switch e := expr.(type) {
		case *hclsyntax.ScopeTraversalExpr:
			return e, true
		case *hclsyntax.IndexExpr:
			expr = e.Collection
		case *hclsyntax.RelativeTraversalExpr:
			expr = e.Source
		default:
			return nil, false
		}
	}
}

func (c *tokenCategories) of(token hclsyntax.Token) (string, bool) {
	category, ok := c.byStart[token.Range.Start.Byte]
	if !ok {
		category = lexicalCategories[token.Type]
	}

	return category, category != ""
}

func colouredBytes(token hclsyntax.Token) (int, int) {
	start, end := token.Range.Start.Byte, token.Range.End.Byte
	switch token.Type {
	case hclsyntax.TokenComment, hclsyntax.TokenOHeredoc:
		end = start + len(bytes.TrimRight(token.Bytes, "\r\n"))
	case hclsyntax.TokenCHeredoc:
		start = end - len(bytes.TrimLeft(token.Bytes, " \t"))
	}

	return start, end
}

// An invalid byte counts as one UTF-16 unit, as its replacement character does.
func utf16Offsets(src []byte) []int {
	offsets := make([]int, len(src)+1)
	unit := 0
	for i := 0; i < len(src); {
		r, width := utf8.DecodeRune(src[i:])
		for k := range width {
			offsets[i+k] = unit
		}
		unit += utf16.RuneLen(r)
		i += width
	}
	offsets[len(src)] = unit
	return offsets
}

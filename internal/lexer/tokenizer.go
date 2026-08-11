package lexer

import (
	"strings"

	"github.com/hpcsc/emod/internal/diagnostic"
)

type cursor struct {
	input    string
	filename string
	pos      int
	line     int
	column   int
}

func Scan(input, filename string) ([]*Token, []*diagnostic.Entry) {
	c := cursor{input: input, filename: filename, pos: 0, line: 1, column: 1}
	var tokens []*Token
	var diags []*diagnostic.Entry

	for c.pos < len(c.input) {
		c = skipWhitespace(c)
		if c.pos >= len(c.input) {
			break
		}

		switch {
		case peek(c) == '#':
			var tok *Token
			c, tok = readComment(c)
			tokens = append(tokens, tok)
		case peek(c) == '"':
			var tok *Token
			var d *diagnostic.Entry
			c, tok, d = readString(c)
			if d != nil {
				diags = append(diags, d)
			}
			if tok != nil {
				tokens = append(tokens, tok)
			}
		case peek(c) == '{':
			tokens = append(tokens, newToken(OpenBrace, "{", c))
			c = advance(c)
		case peek(c) == '}':
			tokens = append(tokens, newToken(CloseBrace, "}", c))
			c = advance(c)
		case peek(c) == '[':
			tokens = append(tokens, newToken(OpenBracket, "[", c))
			c = advance(c)
		case peek(c) == ']':
			tokens = append(tokens, newToken(CloseBracket, "]", c))
			c = advance(c)
		case peek(c) == ',':
			tokens = append(tokens, newToken(Comma, ",", c))
			c = advance(c)
		case peek(c) == '=':
			tokens = append(tokens, newToken(Equals, "=", c))
			c = advance(c)
		case peek(c) == '(':
			tokens = append(tokens, newToken(OpenParen, "(", c))
			c = advance(c)
		case peek(c) == ')':
			tokens = append(tokens, newToken(CloseParen, ")", c))
			c = advance(c)
		case peek(c) == ':':
			tokens = append(tokens, newToken(Colon, ":", c))
			c = advance(c)
		case peek(c) == '-' && peekAhead(c, 1) == '>':
			tokens = append(tokens, newToken(Arrow, "->", c))
			c = advance(advance(c))
		case isIdentifierStart(peek(c)):
			var tok *Token
			c, tok = readIdentifierOrKeyword(c)
			tokens = append(tokens, tok)
		case isDigit(peek(c)):
			var tok *Token
			c, tok = readNumber(c)
			tokens = append(tokens, tok)
		default:
			diags = append(diags, newDiag("unrecognized character: "+string(peek(c)), c))
			c = advance(c)
		}
	}

	tokens = append(tokens, newToken(EOF, "", c))
	return tokens, diags
}

func newToken(typ Kind, value string, c cursor) *Token {
	return &Token{Type: typ, Value: value, Line: c.line, Column: c.column}
}

func newDiag(msg string, c cursor) *diagnostic.Entry {
	return &diagnostic.Entry{Filename: c.filename, Line: c.line, Column: c.column, Message: msg}
}

func skipWhitespace(c cursor) cursor {
	for c.pos < len(c.input) {
		if isWhitespace(peek(c)) && peek(c) != '\n' {
			c = advance(c)
			continue
		}

		if peek(c) == '\n' {
			c = advance(c)
			c.line++
			c.column = 1
			continue
		}

		break
	}
	return c
}

func readComment(c cursor) (cursor, *Token) {
	startLine := c.line
	startCol := c.column

	start := c.pos
	for c.pos < len(c.input) && peek(c) != '\n' {
		c = advance(c)
	}

	return c, &Token{Type: Comment, Value: c.input[start:c.pos], Line: startLine, Column: startCol}
}

func readString(c cursor) (cursor, *Token, *diagnostic.Entry) {
	startLine := c.line
	startCol := c.column
	c = advance(c)

	var sb strings.Builder
	for c.pos < len(c.input) && peek(c) != '"' {
		if peek(c) == '\n' {
			c.line++
			c.column = 0
		}
		sb.WriteByte(c.input[c.pos])
		c = advance(c)
	}

	if c.pos >= len(c.input) {
		return c, nil, &diagnostic.Entry{
			Filename: c.filename, Line: startLine, Column: startCol, Message: "unterminated string",
		}
	}

	c = advance(c)
	return c, &Token{Type: String, Value: sb.String(), Line: startLine, Column: startCol}, nil
}

func readIdentifierOrKeyword(c cursor) (cursor, *Token) {
	startCol := c.column
	var sb strings.Builder

	for c.pos < len(c.input) && isIdentifierChar(peek(c)) {
		sb.WriteByte(c.input[c.pos])
		c = advance(c)
	}

	value := sb.String()
	return c, &Token{Type: keywordOrIdentifier(value), Value: value, Line: c.line, Column: startCol}
}

// readNumber produces Integer for digits alone and Decimal once a fractional
// part follows, keeping Integer as narrow as the version header expects it.
func readNumber(c cursor) (cursor, *Token) {
	startCol := c.column

	start := c.pos
	c = readDigits(c)

	kind := Integer
	if peek(c) == '.' && isDigit(peekAhead(c, 1)) {
		kind = Decimal
		c = readDigits(advance(c))
	}

	return c, &Token{Type: kind, Value: c.input[start:c.pos], Line: c.line, Column: startCol}
}

func readDigits(c cursor) cursor {
	for c.pos < len(c.input) && isDigit(peek(c)) {
		c = advance(c)
	}
	return c
}

func peek(c cursor) byte {
	if c.pos >= len(c.input) {
		return 0
	}
	return c.input[c.pos]
}

func peekAhead(c cursor, n int) byte {
	pos := c.pos + n
	if pos >= len(c.input) {
		return 0
	}
	return c.input[pos]
}

func advance(c cursor) cursor {
	if c.pos < len(c.input) {
		c.pos++
		c.column++
	}
	return c
}

func keywordOrIdentifier(word string) Kind {
	if kind, ok := keywords[word]; ok {
		return kind
	}
	return Identifier
}

func isIdentifierStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isIdentifierChar(ch byte) bool {
	return isIdentifierStart(ch) || isDigit(ch)
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\r'
}

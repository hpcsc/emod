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
		c = skipWhitespaceAndComments(c)
		if c.pos >= len(c.input) {
			break
		}

		switch {
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

func skipWhitespaceAndComments(c cursor) cursor {
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

		if peek(c) == '#' {
			for c.pos < len(c.input) && peek(c) != '\n' {
				c = advance(c)
			}
			continue
		}

		break
	}
	return c
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
	return c, &Token{Type: getKeywordKind(value), Value: value, Line: c.line, Column: startCol}
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

func getKeywordKind(s string) Kind {
	switch s {
	case "model":
		return KeywordModel
	case "actor":
		return KeywordActor
	case "context":
		return KeywordContext
	case "aggregate":
		return KeywordAggregate
	case "slice":
		return KeywordSlice
	case "command":
		return KeywordCommand
	case "event":
		return KeywordEvent
	case "fields":
		return KeywordFields
	case "flow":
		return KeywordFlow
	case "trigger":
		return KeywordTrigger
	case "view":
		return KeywordView
	case "automation":
		return KeywordAutomation
	case "translation":
		return KeywordTranslation
	case "subscribes":
		return KeywordSubscribes
	case "target":
		return KeywordTarget
	case "external_system":
		return KeywordExternalSystem
	case "reads":
		return KeywordReads
	default:
		return Identifier
	}
}

func isIdentifierStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isIdentifierChar(ch byte) bool {
	return isIdentifierStart(ch) || (ch >= '0' && ch <= '9')
}

func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\r'
}

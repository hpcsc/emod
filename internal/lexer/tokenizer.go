package lexer

import (
	"strings"
)

type Tokenizer struct {
	input  string
	pos    int
	line   int
	column int
	tokens []*Token
	errs   []*Token
}

func New(input string) *Tokenizer {
	return &Tokenizer{
		input: input,
		pos:   0,
		line:  1,
		column: 1,
	}
}

func (l *Tokenizer) Scan() ([]*Token, []*Token) {
	for l.pos < len(l.input) {
		l.skipWhitespaceAndComments()
		if l.pos >= len(l.input) {
			break
		}

		switch {
		case l.peek() == '"':
			l.readString()
		case l.peek() == '{':
			l.addToken(OpenBrace, "{")
			l.advance()
		case l.peek() == '}':
			l.addToken(CloseBrace, "}")
			l.advance()
		case l.peek() == ':':
			l.addToken(Colon, ":")
			l.advance()
		case l.peek() == '-' && l.peekAhead(1) == '>':
			l.addToken(Arrow, "->")
			l.advance()
			l.advance()
		case isIdentifierStart(l.peek()):
			l.readIdentifierOrKeyword()
		default:
			l.addError("unrecognized character: " + string(l.peek()))
			l.advance()
		}
	}

	l.addToken(EOF, "")
	return l.tokens, l.errs
}

func (l *Tokenizer) skipWhitespaceAndComments() {
	for l.pos < len(l.input) {
		if isWhitespace(l.peek()) && l.peek() != '\n' {
			l.advance()
			continue
		}

		if l.peek() == '\n' {
			l.advance()
			l.line++
			l.column = 1
			continue
		}

		if l.peek() == '#' {
			for l.pos < len(l.input) && l.peek() != '\n' {
				l.advance()
			}
			continue
		}

		break
	}
}

func (l *Tokenizer) readString() {
	startLine := l.line
	startCol := l.column
	l.advance()

	var sb strings.Builder
	for l.pos < len(l.input) && l.peek() != '"' {
		if l.peek() == '\n' {
			l.line++
			l.column = 0
		}
		sb.WriteByte(l.input[l.pos])
		l.advance()
	}

	if l.pos >= len(l.input) {
		l.errs = append(l.errs, &Token{
			Type:   Error,
			Value:  "unterminated string",
			Line:   startLine,
			Column: startCol,
		})
		return
	}

	l.advance()
	l.tokens = append(l.tokens, &Token{
		Type:   String,
		Value:  sb.String(),
		Line:   startLine,
		Column: startCol,
	})
}

func (l *Tokenizer) readIdentifierOrKeyword() {
	startCol := l.column
	var sb strings.Builder

	for l.pos < len(l.input) && isIdentifierChar(l.peek()) {
		sb.WriteByte(l.input[l.pos])
		l.advance()
	}

	value := sb.String()
	kind := getKeywordKind(value)
	if kind == Identifier {
		l.tokens = append(l.tokens, &Token{
			Type:   Identifier,
			Value:  value,
			Line:   l.line,
			Column: startCol,
		})
	} else {
		l.tokens = append(l.tokens, &Token{
			Type:   kind,
			Value:  value,
			Line:   l.line,
			Column: startCol,
		})
	}
}

func (l *Tokenizer) addToken(typ Kind, value string) {
	l.tokens = append(l.tokens, &Token{
		Type:   typ,
		Value:  value,
		Line:   l.line,
		Column: l.column,
	})
}

func (l *Tokenizer) addError(msg string) {
	l.errs = append(l.errs, &Token{
		Type:   Error,
		Value:  msg,
		Line:   l.line,
		Column: l.column,
	})
}

func (l *Tokenizer) peek() byte {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *Tokenizer) peekAhead(n int) byte {
	pos := l.pos + n
	if pos >= len(l.input) {
		return 0
	}
	return l.input[pos]
}

func (l *Tokenizer) advance() {
	if l.pos < len(l.input) {
		l.pos++
		l.column++
	}
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

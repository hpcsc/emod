package lexer

import (
	"strings"
)

// Lexer tokenizes .emod input.
type Lexer struct {
	input  string
	pos    int
	line   int
	column int
	tokens []*Token
	errs   []*Token
}

// New creates a new lexer for the given input.
func New(input string) *Lexer {
	return &Lexer{
		input: input,
		pos:   0,
		line:  1,
		column: 1,
	}
}

// Tokenize lexes the entire input and returns all tokens.
func (l *Lexer) Tokenize() []*Token {
	for l.pos < len(l.input) {
		l.skipWhitespaceAndComments()
		if l.pos >= len(l.input) {
			break
		}

		switch {
		case l.peek() == '"':
			l.readString()
		case l.peek() == '{':
			l.addToken(TokenOpenBrace, "{")
			l.advance()
		case l.peek() == '}':
			l.addToken(TokenCloseBrace, "}")
			l.advance()
		case l.peek() == ':':
			l.addToken(TokenColon, ":")
			l.advance()
		case l.peek() == '-' && l.peekAhead(1) == '>':
			l.addToken(TokenArrow, "->")
			l.advance()
			l.advance()
		case isIdentifierStart(l.peek()):
			l.readIdentifierOrKeyword()
		default:
			l.addError("unrecognized character: " + string(l.peek()))
			l.advance()
		}
	}

	l.addToken(TokenEOF, "")
	return l.tokens
}

// Errors returns all error tokens encountered.
func (l *Lexer) Errors() []*Token {
	return l.errs
}

func (l *Lexer) skipWhitespaceAndComments() {
	for l.pos < len(l.input) {
		// Skip whitespace
		if isWhitespace(l.peek()) && l.peek() != '\n' {
			l.advance()
			continue
		}

		// Skip newlines (but increment line counter)
		if l.peek() == '\n' {
			l.advance()
			l.line++
			l.column = 1
			continue
		}

		// Skip comments
		if l.peek() == '#' {
			for l.pos < len(l.input) && l.peek() != '\n' {
				l.advance()
			}
			continue
		}

		break
	}
}

func (l *Lexer) readString() {
	startLine := l.line
	startCol := l.column
	l.advance() // skip opening quote

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
			Type:   TokenError,
			Value:  "unterminated string",
			Line:   startLine,
			Column: startCol,
		})
		return
	}

	l.advance() // skip closing quote
	l.tokens = append(l.tokens, &Token{
		Type:   TokenString,
		Value:  sb.String(),
		Line:   startLine,
		Column: startCol,
	})
}

func (l *Lexer) readIdentifierOrKeyword() {
	startCol := l.column
	var sb strings.Builder

	for l.pos < len(l.input) && isIdentifierChar(l.peek()) {
		sb.WriteByte(l.input[l.pos])
		l.advance()
	}

	value := sb.String()
	tokenType := getKeywordType(value)
	if tokenType == TokenIdentifier {
		l.tokens = append(l.tokens, &Token{
			Type:   TokenIdentifier,
			Value:  value,
			Line:   l.line,
			Column: startCol,
		})
	} else {
		l.tokens = append(l.tokens, &Token{
			Type:   tokenType,
			Value:  value,
			Line:   l.line,
			Column: startCol,
		})
	}
}

func (l *Lexer) addToken(typ TokenType, value string) {
	l.tokens = append(l.tokens, &Token{
		Type:   typ,
		Value:  value,
		Line:   l.line,
		Column: l.column,
	})
}

func (l *Lexer) addError(msg string) {
	l.errs = append(l.errs, &Token{
		Type:   TokenError,
		Value:  msg,
		Line:   l.line,
		Column: l.column,
	})
}

func (l *Lexer) peek() byte {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *Lexer) peekAhead(n int) byte {
	pos := l.pos + n
	if pos >= len(l.input) {
		return 0
	}
	return l.input[pos]
}

func (l *Lexer) advance() {
	if l.pos < len(l.input) {
		l.pos++
		l.column++
	}
}

func getKeywordType(s string) TokenType {
	switch s {
	case "model":
		return TokenKeywordModel
	case "actor":
		return TokenKeywordActor
	case "context":
		return TokenKeywordContext
	case "aggregate":
		return TokenKeywordAggregate
	case "slice":
		return TokenKeywordSlice
	case "command":
		return TokenKeywordCommand
	case "event":
		return TokenKeywordEvent
	case "fields":
		return TokenKeywordFields
	case "flow":
		return TokenKeywordFlow
	default:
		return TokenIdentifier
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

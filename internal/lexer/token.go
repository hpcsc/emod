package lexer

// TokenType represents the type of a lexical token.
type TokenType int

const (
	// Keywords
	TokenKeywordModel TokenType = iota
	TokenKeywordActor
	TokenKeywordContext
	TokenKeywordAggregate
	TokenKeywordSlice
	TokenKeywordCommand
	TokenKeywordEvent
	TokenKeywordFields
	TokenKeywordFlow

	// Literals and identifiers
	TokenIdentifier
	TokenString

	// Operators and punctuation
	TokenOpenBrace
	TokenCloseBrace
	TokenArrow
	TokenColon

	// Special
	TokenEOF
	TokenError
)

// Token represents a single lexical token.
type Token struct {
	Type   TokenType
	Value  string
	Line   int
	Column int
}

// String returns a human-readable string for the token type.
func (t TokenType) String() string {
	switch t {
	case TokenKeywordModel:
		return "model"
	case TokenKeywordActor:
		return "actor"
	case TokenKeywordContext:
		return "context"
	case TokenKeywordAggregate:
		return "aggregate"
	case TokenKeywordSlice:
		return "slice"
	case TokenKeywordCommand:
		return "command"
	case TokenKeywordEvent:
		return "event"
	case TokenKeywordFields:
		return "fields"
	case TokenKeywordFlow:
		return "flow"
	case TokenIdentifier:
		return "identifier"
	case TokenString:
		return "string"
	case TokenOpenBrace:
		return "{"
	case TokenCloseBrace:
		return "}"
	case TokenArrow:
		return "->"
	case TokenColon:
		return ":"
	case TokenEOF:
		return "EOF"
	case TokenError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

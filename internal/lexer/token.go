package lexer

type Kind int

const (
	// Keywords
	KeywordModel Kind = iota
	KeywordActor
	KeywordContext
	KeywordAggregate
	KeywordSlice
	KeywordCommand
	KeywordEvent
	KeywordFields
	KeywordFlow

	// Literals and identifiers
	Identifier
	String

	// Operators and punctuation
	OpenBrace
	CloseBrace
	Arrow
	Colon

	// Special
	EOF
	Error
)

type Token struct {
	Type   Kind
	Value  string
	Line   int
	Column int
}

func (k Kind) String() string {
	switch k {
	case KeywordModel:
		return "model"
	case KeywordActor:
		return "actor"
	case KeywordContext:
		return "context"
	case KeywordAggregate:
		return "aggregate"
	case KeywordSlice:
		return "slice"
	case KeywordCommand:
		return "command"
	case KeywordEvent:
		return "event"
	case KeywordFields:
		return "fields"
	case KeywordFlow:
		return "flow"
	case Identifier:
		return "identifier"
	case String:
		return "string"
	case OpenBrace:
		return "{"
	case CloseBrace:
		return "}"
	case Arrow:
		return "->"
	case Colon:
		return ":"
	case EOF:
		return "EOF"
	case Error:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

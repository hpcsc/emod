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
	KeywordTrigger
	KeywordView
	KeywordAutomation
	KeywordTranslation
	KeywordSubscribes
	KeywordTarget
	KeywordExternalSystem
	KeywordReads
	KeywordSource
	KeywordExternal
	KeywordMode
	KeywordTags
	KeywordDecidesOn
	KeywordWhere
	KeywordAnd
	KeywordOr
	KeywordNot
	KeywordTag
	KeywordEvents

	// Literals and identifiers
	Identifier
	String

	// Operators and punctuation
	OpenBrace
	CloseBrace
	OpenBracket
	CloseBracket
	Arrow
	Colon
	Comma
	Equals
	OpenParen
	CloseParen

	// Special
	Comment
	EOF
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
	case KeywordTrigger:
		return "trigger"
	case KeywordView:
		return "view"
	case KeywordAutomation:
		return "automation"
	case KeywordTranslation:
		return "translation"
	case KeywordSubscribes:
		return "subscribes"
	case KeywordTarget:
		return "target"
	case KeywordExternalSystem:
		return "external_system"
	case KeywordReads:
		return "reads"
	case KeywordSource:
		return "source"
	case KeywordExternal:
		return "external"
	case KeywordMode:
		return "mode"
	case KeywordTags:
		return "tags"
	case KeywordDecidesOn:
		return "decides_on"
	case KeywordWhere:
		return "where"
	case KeywordAnd:
		return "and"
	case KeywordOr:
		return "or"
	case KeywordNot:
		return "not"
	case KeywordTag:
		return "tag"
	case KeywordEvents:
		return "events"
	case Identifier:
		return "identifier"
	case String:
		return "string"
	case OpenBrace:
		return "{"
	case CloseBrace:
		return "}"
	case OpenBracket:
		return "["
	case CloseBracket:
		return "]"
	case Arrow:
		return "->"
	case Colon:
		return ":"
	case Comma:
		return ","
	case Equals:
		return "="
	case OpenParen:
		return "("
	case CloseParen:
		return ")"
	case Comment:
		return "comment"
	case EOF:
		return "EOF"
	default:
		return "UNKNOWN"
	}
}

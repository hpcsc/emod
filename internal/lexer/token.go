package lexer

import (
	"maps"
	"slices"
)

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
	KeywordEmod
	KeywordDescription
	KeywordInvariant
	KeywordSpec
	KeywordGiven
	KeywordWhen
	KeywordThen
	KeywordRejected
	KeywordOn

	// Literals and identifiers
	Identifier
	String
	Integer

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

var keywords = map[string]Kind{
	"model":           KeywordModel,
	"actor":           KeywordActor,
	"context":         KeywordContext,
	"aggregate":       KeywordAggregate,
	"slice":           KeywordSlice,
	"command":         KeywordCommand,
	"event":           KeywordEvent,
	"fields":          KeywordFields,
	"flow":            KeywordFlow,
	"trigger":         KeywordTrigger,
	"view":            KeywordView,
	"automation":      KeywordAutomation,
	"translation":     KeywordTranslation,
	"subscribes":      KeywordSubscribes,
	"target":          KeywordTarget,
	"external_system": KeywordExternalSystem,
	"reads":           KeywordReads,
	"source":          KeywordSource,
	"external":        KeywordExternal,
	"mode":            KeywordMode,
	"tags":            KeywordTags,
	"decides_on":      KeywordDecidesOn,
	"where":           KeywordWhere,
	"and":             KeywordAnd,
	"or":              KeywordOr,
	"not":             KeywordNot,
	"tag":             KeywordTag,
	"events":          KeywordEvents,
	"emod":            KeywordEmod,
	"description":     KeywordDescription,
	"invariant":       KeywordInvariant,
	"spec":            KeywordSpec,
	"given":           KeywordGiven,
	"when":            KeywordWhen,
	"then":            KeywordThen,
	"rejected":        KeywordRejected,
	"on":              KeywordOn,
}

var keywordNames = invertKeywords()

func invertKeywords() map[Kind]string {
	names := make(map[Kind]string, len(keywords))
	for keyword, kind := range keywords {
		names[kind] = keyword
	}
	return names
}

func Keywords() []string {
	return slices.Sorted(maps.Keys(keywords))
}

type Token struct {
	Type   Kind
	Value  string
	Line   int
	Column int
}

func (k Kind) IsKeyword() bool {
	_, ok := keywordNames[k]
	return ok
}

func (k Kind) String() string {
	if keyword, ok := keywordNames[k]; ok {
		return keyword
	}

	switch k {
	case Identifier:
		return "identifier"
	case String:
		return "string"
	case Integer:
		return "integer"
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

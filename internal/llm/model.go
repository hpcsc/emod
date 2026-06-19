package llm

import (
	"context"
	"encoding/json"
)

type Effort int

const (
	EffortUnset Effort = iota
	EffortLow
	EffortMedium
	EffortHigh
	EffortXHigh
)

func (e Effort) String() string {
	switch e {
	case EffortLow:
		return "low"
	case EffortHigh:
		return "high"
	case EffortXHigh:
		return "xhigh"
	default:
		return "medium"
	}
}

type Message struct {
	Role    string
	Content string
}

type GenerateRequest struct {
	System   string
	Messages []Message
	Schema   json.RawMessage
	Effort   Effort
}

type Usage struct {
	InputTokens  int
	OutputTokens int
}

type Response struct {
	Text  string
	Usage Usage
}

type Model interface {
	Generate(ctx context.Context, request GenerateRequest) (Response, error)
}

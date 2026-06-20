//go:build unit

package llm

import (
	"context"
	"errors"
)

// ErrNoMoreResponses is returned once the queued responses are exhausted.
// The fake reports this sentinel rather than repeating the last response or
// panicking: tests drive repair-loop behaviour by the number of queued
// responses, so over-consuming must surface as a deterministic, assertable
// error instead of silently masking the call count or aborting the run.
var ErrNoMoreResponses = errors.New("llm: fake has no more queued responses")

type Fake struct {
	responses []Response
	requests  []GenerateRequest
	next      int
}

func NewFake(responses ...Response) *Fake {
	return &Fake{responses: responses}
}

func (f *Fake) Generate(_ context.Context, request GenerateRequest) (Response, error) {
	f.requests = append(f.requests, request)

	if f.next >= len(f.responses) {
		return Response{}, ErrNoMoreResponses
	}

	response := f.responses[f.next]
	f.next++

	return response, nil
}

func (f *Fake) Requests() []GenerateRequest {
	return f.requests
}

var _ Model = (*Fake)(nil)

//go:build unit

package llm

import (
	"context"
	"errors"
)

// ErrBroken is the failure NewBroken uses when WithError is not called, so a
// bare broken model still fails deterministically.
var ErrBroken = errors.New("llm: broken model")

type Broken struct {
	err   error
	usage Usage
}

func NewBroken() *Broken {
	return &Broken{err: ErrBroken}
}

func (b *Broken) WithError(err error) *Broken {
	b.err = err
	return b
}

func (b *Broken) WithUsage(usage Usage) *Broken {
	b.usage = usage
	return b
}

func (b *Broken) Generate(_ context.Context, _ GenerateRequest) (Response, error) {
	return Response{Usage: b.usage}, b.err
}

var _ Model = (*Broken)(nil)

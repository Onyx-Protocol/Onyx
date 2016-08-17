package span

import (
	"context"

	"github.com/resonancelabs/go-pub/instrument"
)

type key int

const (
	keySpan key = iota
)

// NewContextWithSpan returns a new context containing span s.
func NewContextWithSpan(ctx context.Context, s instrument.ActiveSpan) context.Context {
	return context.WithValue(ctx, keySpan, s)
}

func fromContext(ctx context.Context) instrument.ActiveSpan {
	sp := ctx.Value(keySpan)
	if sp == nil {
		return nil
	}
	return sp.(instrument.ActiveSpan)
}

package prottest

import (
	"context"
	"testing"
)

func TestMakeBlock(t *testing.T) {
	ctx := context.Background()
	c := NewChain(t)
	MakeBlock(ctx, t, c)
	MakeBlock(ctx, t, c)
	MakeBlock(ctx, t, c)

	var want uint64 = 4
	if got := c.Height(); got != want {
		t.Errorf("c.Height() = %d want %d", got, want)
	}
}

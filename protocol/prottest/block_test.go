package prottest

import "testing"

func TestMakeBlock(t *testing.T) {
	c := NewChain(t)
	MakeBlock(t, c)
	MakeBlock(t, c)
	MakeBlock(t, c)

	var want uint64 = 4
	if got := c.Height(); got != want {
		t.Errorf("c.Height() = %d want %d", got, want)
	}
}

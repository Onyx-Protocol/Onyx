package utxodb

import (
	"container/heap"
	"testing"
	"time"
)

func TestPoolReserveErr(t *testing.T) {
	p := &pool{
		ready: true,
	}
	u := &UTXO{Amount: 1}
	heap.Push(&p.outputs, u)

	now := time.Now()
	p.reserve(2, now, now.Add(time.Minute))
	if g := len(p.outputs); g != 1 {
		t.Errorf("len(p.outputs) = %d want 1", g)
	}
}

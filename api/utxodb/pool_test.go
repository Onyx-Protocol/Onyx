package utxodb

import (
	"container/heap"
	"testing"
	"time"

	"chain/fedchain-sandbox/wire"
)

func TestPoolReserveErr(t *testing.T) {
	p := &pool{
		byOutpoint: map[wire.OutPoint]*UTXO{},
		ready:      true,
	}
	u := &UTXO{Amount: 1}
	heap.Push(&p.outputs, u)
	p.byOutpoint[u.Outpoint] = u

	now := time.Now()
	p.reserve(Input{Amount: 2}, now, now.Add(time.Minute))
	if g := len(p.outputs); g != 1 {
		t.Errorf("len(p.outputs) = %d want 1", g)
	}
}

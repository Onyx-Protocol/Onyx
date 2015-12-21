package utxodb

import (
	"testing"

	"chain/fedchain/bc"
)

func TestUnreserveDeleted(t *testing.T) {
	rs := New(nil)
	p := rs.pool("acc1", bc.AssetID{})
	p.ready = true
	u1 := &UTXO{
		AccountID: "acc1",
		AssetID:   bc.AssetID{},
		Amount:    1,
		Outpoint:  bc.Outpoint{Index: 0},
	}
	u2 := &UTXO{
		AccountID: "acc1",
		AssetID:   bc.AssetID{},
		Amount:    1,
		Outpoint:  bc.Outpoint{Index: 1},
	}
	rs.insert([]*UTXO{u1, u2})
	rs.delete([]*UTXO{u1})
	rs.unreserve([]*UTXO{u1}) // should ignore u1
}

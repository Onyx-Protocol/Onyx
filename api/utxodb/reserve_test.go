package utxodb

import (
	"testing"

	"chain/fedchain/bc"
)

func TestUnreserveDeleted(t *testing.T) {
	rs := New(nil)
	p := rs.pool("b1", "a1")
	p.ready = true
	u1 := &UTXO{
		BucketID: "b1",
		AssetID:  "a1",
		Amount:   1,
		Outpoint: bc.Outpoint{Index: 0},
	}
	u2 := &UTXO{
		BucketID: "b1",
		AssetID:  "a1",
		Amount:   1,
		Outpoint: bc.Outpoint{Index: 1},
	}
	rs.insert([]*UTXO{u1, u2})
	rs.delete([]*UTXO{u1})
	rs.unreserve([]*UTXO{u1}) // should ignore u1
}

package tx

import (
	"chain/protocol/bc"
	"reflect"
	"testing"
	"time"
)

func BenchmarkEntryID(b *testing.B) {
	m := newMux(program{Code: []byte{1}, VMVersion: 1})
	m.addSourceID(bc.Hash{}, bc.AssetAmount{}, 1)

	entries := []entry{
		newIssuance(nil, bc.AssetAmount{}, bc.Hash{}, 0),
		newHeader(1, bc.Hash{}, uint64(time.Now().Unix()), uint64(time.Now().Unix())),
		m,
		newNonce(program{Code: []byte{1}, VMVersion: 1}, nil),
		newOutput(program{Code: []byte{1}, VMVersion: 1}, bc.Hash{}, 0),
		newRetirement(bc.Hash{}, 1),
		newSpend(newOutput(program{Code: []byte{1}, VMVersion: 1}, bc.Hash{}, 0), bc.Hash{}, 0),
	}

	for _, e := range entries {
		name := reflect.TypeOf(e).Elem().Name()
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				entryID(e)
			}
		})
	}
}

package tx

import (
	"chain/protocol/bc"
	"reflect"
	"testing"
	"time"
)

func BenchmarkEntryID(b *testing.B) {
	entries := []entry{
		newIssuance(bc.Hash{}, bc.AssetAmount{}, bc.Hash{}, 0),
		newHeader(1, []bc.Hash{{}}, bc.Hash{}, uint64(time.Now().Unix()), uint64(time.Now().Unix())),
		newMux([]valueSource{{
			Ref:      bc.Hash{},
			Value:    bc.AssetAmount{},
			Position: 1,
		}}, program{Code: []byte{1}, VMVersion: 1}),
		newNonce(program{Code: []byte{1}, VMVersion: 1}, bc.Hash{}),
		newOutput(valueSource{
			Ref:      bc.Hash{},
			Value:    bc.AssetAmount{},
			Position: 1,
		}, program{Code: []byte{1}, VMVersion: 1}, bc.Hash{}, 0),
		newRetirement(valueSource{
			Ref:      bc.Hash{},
			Value:    bc.AssetAmount{},
			Position: 1,
		}, bc.Hash{}, 1),
		newSpend(bc.Hash{}, bc.Hash{}, 0),
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

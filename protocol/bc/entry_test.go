package bc

import (
	"reflect"
	"testing"
	"time"
)

func BenchmarkEntryID(b *testing.B) {
	m := newMux(program{Code: []byte{1}, VMVersion: 1})
	m.addSourceID(Hash{}, AssetAmount{}, 1)

	entries := []entry{
		newIssuance(nil, AssetAmount{}, Hash{}, 0),
		newHeader(1, nil, Hash{}, uint64(time.Now().Unix()), uint64(time.Now().Unix())),
		m,
		newNonce(program{Code: []byte{1}, VMVersion: 1}, nil),
		newOutput(program{Code: []byte{1}, VMVersion: 1}, Hash{}, 0),
		newRetirement(Hash{}, 1),
		newSpend(newOutput(program{Code: []byte{1}, VMVersion: 1}, Hash{}, 0), Hash{}, 0),
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

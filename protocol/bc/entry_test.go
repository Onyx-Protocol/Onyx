package bc

import (
	"reflect"
	"testing"
	"time"
)

func BenchmarkEntryID(b *testing.B) {
	m := NewMux([]ValueSource{{Position: 1}}, Program{Code: []byte{1}, VMVersion: 1})

	entries := []Entry{
		NewIssuance(nil, AssetAmount{}, nil, 0),
		NewTxHeader(1, nil, nil, uint64(time.Now().Unix()), uint64(time.Now().Unix())),
		m,
		NewNonce(Program{Code: []byte{1}, VMVersion: 1}, nil),
		NewOutput(ValueSource{}, Program{Code: []byte{1}, VMVersion: 1}, nil, 0),
		NewRetirement(ValueSource{}, nil, 1),
		NewSpend(NewOutput(ValueSource{}, Program{Code: []byte{1}, VMVersion: 1}, nil, 0), nil, 0),
	}

	for _, e := range entries {
		name := reflect.TypeOf(e).Elem().Name()
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				EntryID(e)
			}
		})
	}
}

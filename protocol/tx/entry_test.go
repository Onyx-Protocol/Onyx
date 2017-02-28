package tx

import (
	"chain/protocol/bc"
	"testing"
	"time"
)

func BenchmarkEntryIDIssuance(b *testing.B) {
	entry := newIssuance(bc.Hash{}, bc.AssetAmount{}, bc.Hash{}, 0)
	for i := 0; i < b.N; i++ {
		_ = entryID(entry)
	}
}

func BenchmarkEntryIDHeader(b *testing.B) {
	entry := newHeader(1, []bc.Hash{{}}, bc.Hash{}, uint64(time.Now().Unix()), uint64(time.Now().Unix()))
	for i := 0; i < b.N; i++ {
		_ = entryID(entry)
	}
}

func BenchmarkEntryIDMux(b *testing.B) {
	entry := newMux([]valueSource{{
		Ref:      bc.Hash{},
		Value:    bc.AssetAmount{},
		Position: 1,
	}}, program{Code: []byte{1}, VMVersion: 1})
	for i := 0; i < b.N; i++ {
		_ = entryID(entry)
	}
}

func BenchmarkEntryIDNonce(b *testing.B) {
	entry := newNonce(program{Code: []byte{1}, VMVersion: 1}, bc.Hash{})
	for i := 0; i < b.N; i++ {
		_ = entryID(entry)
	}
}

func BenchmarkEntryIDOutput(b *testing.B) {
	entry := newOutput(valueSource{
		Ref:      bc.Hash{},
		Value:    bc.AssetAmount{},
		Position: 1,
	}, program{Code: []byte{1}, VMVersion: 1}, bc.Hash{}, 0)
	for i := 0; i < b.N; i++ {
		_ = entryID(entry)
	}
}

func BenchmarkEntryIDRetirement(b *testing.B) {
	entry := newRetirement(valueSource{
		Ref:      bc.Hash{},
		Value:    bc.AssetAmount{},
		Position: 1,
	}, bc.Hash{}, 1)
	for i := 0; i < b.N; i++ {
		_ = entryID(entry)
	}
}

func BenchmarkEntryIDSpend(b *testing.B) {
	entry := newSpend(bc.Hash{}, bc.Hash{}, 0)
	for i := 0; i < b.N; i++ {
		_ = entryID(entry)
	}
}

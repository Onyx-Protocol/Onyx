package tx

import "chain/protocol/bc"

type valueSource struct {
	Ref      entryRef
	Value    bc.AssetAmount
	Position uint64 // what int do we actually want?
}

type valueDestination struct {
	Ref      entryRef
	Position uint64
}

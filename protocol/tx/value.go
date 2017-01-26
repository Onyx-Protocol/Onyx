package tx

import "chain/protocol/bc"

type valueSource struct {
	Ref      entryRef
	Value    bc.AssetAmount
	Position int // what int do we actually want?
}

type valueDestination struct {
	Ref      entryRef
	Position int
}

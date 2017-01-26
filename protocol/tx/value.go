package tx

import "chain/protocol/bc"

type valueSource struct {
	ref      entryRef
	value    bc.AssetAmount
	position int // what int do we actually want?
}

type valueDestination struct {
	ref      entryRef
	position int
}

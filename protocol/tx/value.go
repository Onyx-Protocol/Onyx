package tx

import "chain/protocol/bc"

type valueSource struct {
	Ref      *EntryRef
	Value    bc.AssetAmount
	Position uint64 // what int do we actually want?
}

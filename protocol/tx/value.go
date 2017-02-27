package tx

import "chain/protocol/bc"

type valueSource struct {
	Ref      bc.Hash
	Value    bc.AssetAmount
	Position uint64 // what int do we actually want?
}

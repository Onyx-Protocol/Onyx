package legacy

import "chain/protocol/bc"

type IssuanceWitness struct {
	InitialBlock    bc.Hash
	AssetDefinition []byte
	VMVersion       uint64
	IssuanceProgram []byte
	Arguments       [][]byte
}

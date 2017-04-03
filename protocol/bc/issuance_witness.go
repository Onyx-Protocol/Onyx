package bc

type IssuanceWitness struct {
	InitialBlock    Hash
	AssetDefinition []byte
	VMVersion       uint64
	IssuanceProgram []byte
	Arguments       [][]byte
}

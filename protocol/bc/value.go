package bc

type valueSource struct {
	Ref      Hash
	Value    AssetAmount
	Position uint64 // what int do we actually want?
}

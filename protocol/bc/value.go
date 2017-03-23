package bc

type ValueSource struct {
	Ref      Hash
	Value    AssetAmount
	Position uint64 // what int do we actually want?
}

package bc

type ValueSource struct {
	Ref      Hash
	Value    AssetAmount
	Position uint64 // what int do we actually want?

	// The Entry corresponding to Ref, if available
	// The struct tag excludes the field from hashing
	Entry `entry:"-"`
}

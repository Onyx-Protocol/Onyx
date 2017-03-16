package bc

type valueSource struct {
	Ref      Hash
	Value    AssetAmount
	Position uint64

	// The Entry corresponding to Ref, if available
	// The struct tag excludes the field from hashing
	Entry `entry:"-"`
}

type ValueDestination struct {
	Ref      Hash
	Position uint64

	// The Entry corresponding to Ref, if available
	// The struct tag excludes the field from hashing
	Entry `entry:"-"`
}

package bc

type ValueSource struct {
	Ref      Hash
	Value    AssetAmount
	Position uint64

	// The Entry corresponding to Ref, if available
	// The struct tag excludes the field from hashing
	Entry `entry:"-"`
}

type ValueDestination struct {
	Ref      Hash
	Value    AssetAmount
	Position uint64

	// The Entry corresponding to Ref, if available.  Under normal
	// conditions, this field is always set except when the containing
	// entry is an Output inside of a Spend (i.e., a prevout).
	// The struct tag excludes the field from hashing.
	Entry `entry:"-"`
}

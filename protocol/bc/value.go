package bc

type ValueSource struct {
	Ref      Hash
	Value    AssetAmount
	Position uint64 // what int do we actually want?

	// The Entry corresponding to Ref, if available.  Under normal
	// conditions, this field is always set except when the containing
	// entry is an Output inside of a Spend (i.e., a prevout).
	// The struct tag excludes the field from hashing.
	Entry `entry:"-"`
}

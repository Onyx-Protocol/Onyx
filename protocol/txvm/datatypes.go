package txvm

const (
	TypeInt64  = 0
	TypeString = 1
	TypeTuple  = 2

	ValueTuple           = "value"
	OutputTuple          = "output"
	NonceTuple           = "nonce"
	RetirementTuple      = "retirement"
	AnchorTuple          = "anchor"
	AssetDefinitionTuple = "assetdefinition"
	TxHeaderTuple        = "txheader"
)

var tupleContents = map[string][]int{
	ValueTuple:           []int{TypeString, TypeTuple, TypeString, TypeInt64, TypeString},
	OutputTuple:          []int{TypeString, TypeTuple, TypeString, TypeTuple, TypeString},
	NonceTuple:           []int{TypeString, TypeTuple, TypeString, TypeInt64, TypeInt64},
	RetirementTuple:      []int{TypeString, TypeTuple, TypeString},
	AnchorTuple:          []int{TypeString, TypeTuple, TypeString},
	AssetDefinitionTuple: []int{TypeString, TypeTuple, TypeString},
	TxHeaderTuple:        []int{TypeString, TypeTuple, TypeString, TypeTuple, TypeTuple, TypeInt64, TypeInt64},
}

type Value interface {
	value()
	typ() int
}

// Bool converts x to a Value (either 0 or 1).
func Bool(x bool) Value {
	if x {
		return Int64(1)
	}
	return Int64(0)
}

// toBool converts v from a Value to a bool
func toBool(v Value) bool {
	n, ok := v.(Int64)
	return !ok || n != 0
}

type Bytes []byte

type Int64 int64

type VMTuple []Value

func (Bytes) value()   {}
func (Int64) value()   {}
func (VMTuple) value() {}

func (Bytes) typ() int   { return TypeString }
func (Int64) typ() int   { return TypeInt64 }
func (VMTuple) typ() int { return TypeTuple }

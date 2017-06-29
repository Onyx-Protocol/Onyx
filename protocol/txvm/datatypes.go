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
	MaxTimeTuple         = "maxtime"
	MinTimeTuple         = "mintime"
	AnnotationTuple      = "annotation"
	SummaryTuple         = "summary"
	AssetAmountTuple     = "assetamount"
)

var tupleContents = map[string][]int{
	ValueTuple:           []int{TypeString, TypeInt64, TypeString},
	OutputTuple:          []int{TypeString, TypeTuple, TypeString, TypeString},
	NonceTuple:           []int{TypeString, TypeString, TypeInt64, TypeInt64},
	RetirementTuple:      []int{TypeString, TypeTuple},
	AnchorTuple:          []int{TypeString, TypeString},
	AssetDefinitionTuple: []int{TypeString, TypeString},
	MaxTimeTuple:         []int{TypeString, TypeInt64},
	MinTimeTuple:         []int{TypeString, TypeInt64},
	AnnotationTuple:      []int{TypeString, TypeString},
	SummaryTuple:         []int{TypeString, TypeTuple, TypeTuple, TypeTuple, TypeTuple, TypeTuple},
	AssetAmountTuple:     []int{TypeInt64, TypeString},
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

type Tuple struct {
	a []Value
}

func (t *Tuple) Len() int {
	return len(t.a)
}

func (t *Tuple) Field(n int) Value {
	return t.a[n]
}

func newTuple(vals ...Value) Tuple {
	return Tuple{a: vals}
}

func (Bytes) value() {}
func (Int64) value() {}
func (Tuple) value() {}

func (Bytes) typ() int { return TypeString }
func (Int64) typ() int { return TypeInt64 }
func (Tuple) typ() int { return TypeTuple }

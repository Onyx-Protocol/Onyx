package txvm2

type value interface {
	typ() int
}

type (
	vint64  int64
	vstring []byte
	tuple   []value
)

const (
	int64type  = 33
	stringtype = 34
	tupletype  = 35
)

func (i vint64) typ()  { return int64type }
func (b vstring) typ() { return stringtype }
func (t tuple) typ()   { return tupletype }

func isNamed(v value, s string) bool {
	t, ok := v.(tuple)
	if !ok {
		return false
	}
	n, ok := t.name()
	if !ok {
		return false
	}
	if s != n {
		return false
	}
	if len(t) != len(namedTuples[n])+1 {
		return false
	}
	for i, typ := range namedTuples[n] {
		if t[i+1].typ() != typ {
			return false
		}
	}
	return true
}

func getTxID(v value) (txid [32]byte, ok bool) {
	if !isNamed(v, transactionIDTuple) {
		return txid, false
	}
	t := v.(tuple)
	// xxx check that len(t[1]) == len(txid)?
	b := t[1].(vstring)
	copy(txid[:], b)
	return txid, true
}

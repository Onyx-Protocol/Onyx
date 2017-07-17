package txvm2

type value interface {
	typ() int
}

type (
	vint64 int64
	vbytes []byte
	tuple  []value
)

const (
	int64type = 33
	bytestype = 34
	tupletype = 35
)

func (i vint64) typ() { return int64type }
func (b vbytes) typ() { return bytestype }
func (t tuple) typ()  { return tupletype }

func getTxID(v value) (txid [32]byte, ok bool) {
	if !isNamed(v, transactionIDTuple) {
		return txid, false
	}
	t := v.(tuple)
	// xxx check that len(t[1]) == len(txid)?
	b := t[1].(vbytes)
	copy(txid[:], b)
	return txid, true
}

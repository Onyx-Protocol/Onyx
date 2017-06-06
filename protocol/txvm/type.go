package txvm

import (
	"encoding/binary"
	"errors"
	"io"
	"math"

	"chain/crypto/sha3pool"
)

type VmID [32]byte

func makeID(x []byte) VmID {
	var id VmID
	if len(x) != len(id) {
		panic("bad id len")
	}
	copy(id[:], x)
	return id
}

// asset type and quantity data tuple
type aval struct {
	asset VmID
	n     int64
}

// linear asset amount
type value struct {
}

func makeValue(asset VmID, amount int64) *value {
	//avalID := ID{}
	//return value{avalID, asset, amount}
	return &value{}
}

// linear predicate
type pval struct {
}

// linear contract
type cval struct {
	src  valsrc
	prog []byte
	data VmID
	exth VmID
}

func (c *cval) typ() string { return "output1" }
func (c *cval) writeTo(w io.Writer) {
	w.Write(c.src.ref[:])
	w.Write(c.src.aval.asset[:])
	writeVarint63(w, uint64(c.src.aval.n))
	writeVarint63(w, uint64(c.src.pos))
	writeVarstr31(w, c.prog)
	w.Write(c.data[:])
	w.Write(c.exth[:])
}

type valsrc struct {
	ref  VmID
	aval aval
	pos  int64
}

type entry interface {
	typ() string
	writeTo(w io.Writer)
}

func entryID(v entry) VmID {
	h := sha3pool.Get256()
	defer sha3pool.Put256(h)

	io.WriteString(h, "entryid:")
	io.WriteString(h, v.typ())
	h.Write([]byte{':'})

	bh := sha3pool.Get256()
	defer sha3pool.Put256(bh)
	v.writeTo(bh)
	io.CopyN(h, bh, 32)

	var id VmID
	h.Read(id[:])
	return id
}

func writeVarint31(w io.Writer, val uint64) {
	if val > math.MaxInt32 {
		panic(errors.New("range"))
	}
	b := make([]byte, 9)
	n := binary.PutUvarint(b, val)
	w.Write(b[:n])
}

func writeVarint63(w io.Writer, val uint64) {
	if val > math.MaxInt64 {
		panic(errors.New("range"))
	}
	b := make([]byte, 9)
	n := binary.PutUvarint(b, val)
	w.Write(b[:n])
}

func writeVarstr31(w io.Writer, s []byte) {
	writeVarint31(w, uint64(len(s)))
	w.Write(s)
}

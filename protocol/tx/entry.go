package tx

import (
	"fmt"
	"io"
	"reflect"

	"chain/crypto/sha3pool"
	"chain/encoding/blockchain"
	"chain/errors"
	"chain/protocol/bc"
)

type entry interface {
	Type() string
	Body() interface{}

	// When an entry is created from a bc.TxInput or a bc.TxOutput, this
	// reports the position of that antecedent object within its
	// transaction. Both inputs (spends and issuances) and outputs
	// (including retirements) are numbered beginning at zero. Entries
	// not originating in this way report -1.
	Ordinal() int
}

var errInvalidValue = errors.New("invalid value")

func entryID(e entry) (hash bc.Hash) {
	// This nil test only handles the case where e is the zero value of
	// the entry interface.
	if e == nil {
		return hash
	}

	// This nil test handles the case where e is a nil pointer.
	if reflect.ValueOf(e).IsNil() {
		return hash
	}

	hasher := sha3pool.Get256()
	defer sha3pool.Put256(hasher)

	hasher.Write([]byte("entryid:"))
	hasher.Write([]byte(e.Type()))
	hasher.Write([]byte{':'})

	bh := sha3pool.Get256()
	defer sha3pool.Put256(bh)
	err := writeForHash(bh, e.Body())
	if err != nil {
		panic(err)
	}
	var innerHash bc.Hash
	bh.Read(innerHash[:])
	hasher.Write(innerHash[:])

	hasher.Read(hash[:])
	return hash
}

func writeForHash(w io.Writer, c interface{}) error {
	switch v := c.(type) {
	case byte:
		_, err := w.Write([]byte{v})
		return errors.Wrap(err, "writing byte for hash")
	case uint64:
		_, err := blockchain.WriteVarint63(w, v)
		return errors.Wrapf(err, "writing uint64 (%d) for hash", v)
	case []byte:
		_, err := blockchain.WriteVarstr31(w, v)
		return errors.Wrapf(err, "writing []byte (len %d) for hash", len(v))
	case string:
		_, err := blockchain.WriteVarstr31(w, []byte(v))
		return errors.Wrapf(err, "writing string (len %d) for hash", len(v))

		// TODO: The rest of these are all aliases for [32]byte. Do we
		// really need them all?

	case bc.Hash:
		_, err := w.Write(v[:])
		return errors.Wrap(err, "writing bc.Hash for hash")
	case bc.AssetID:
		_, err := w.Write(v[:])
		return errors.Wrap(err, "writing bc.AssetID for hash")
	}

	// The two container types in the spec (List and Struct)
	// correspond to slices and structs in Go. They can't be
	// handled with type assertions, so we must use reflect.
	switch v := reflect.ValueOf(c); v.Kind() {
	case reflect.Slice:
		l := v.Len()
		_, err := blockchain.WriteVarint31(w, uint64(l))
		if err != nil {
			return errors.Wrapf(err, "writing slice (len %d) for hash", l)
		}
		for i := 0; i < l; i++ {
			c := v.Index(i)
			if !c.CanInterface() {
				return errInvalidValue
			}
			err := writeForHash(w, c.Interface())
			if err != nil {
				return errors.Wrapf(err, "writing slice element %d for hash", i)
			}
		}
		return nil

	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			c := v.Field(i)
			if !c.CanInterface() {
				return errInvalidValue
			}
			err := writeForHash(w, c.Interface())
			if err != nil {
				t := v.Type()
				f := t.Field(i)
				return errors.Wrapf(err, "writing struct field %d (%s.%s) for hash", i, t.Name(), f.Name)
			}
		}
		return nil
	}

	return errors.Wrap(fmt.Errorf("bad type %T", c))
}

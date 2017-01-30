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

type entryRef bc.Hash

type extHash bc.Hash

var errInvalidValue = errors.New("invalid value")

func entryID(e entry) (entryRef, error) {
	h := sha3pool.Get256()
	defer sha3pool.Put256(h)

	h.Write([]byte("entryid:"))
	h.Write([]byte(e.Type()))
	h.Write([]byte{':'})

	bh := sha3pool.Get256()
	defer sha3pool.Put256(bh)
	err := writeForHash(bh, e.Body())
	if err != nil {
		return entryRef{}, err
	}
	io.CopyN(h, bh, 32)

	var hash entryRef
	h.Read(hash[:])

	return hash, nil
}

func writeForHash(w io.Writer, c interface{}) error {
	switch v := c.(type) {
	case byte:
		_, err := w.Write([]byte{v})
		return err
	case bc.Hash:
		_, err := w.Write(v[:])
		return err
	case entryRef: // xxx do we need so many [32]byte types?
		_, err := w.Write(v[:])
		return err
	case extHash: // xxx do we need so many [32]byte types?
		_, err := w.Write(v[:])
		return err
	case bc.AssetID: // xxx do we need so many [32]byte types?
		_, err := w.Write(v[:])
		return err
	case uint64:
		_, err := blockchain.WriteVarint63(w, v)
		return err
	case []byte:
		_, err := blockchain.WriteVarstr31(w, v)
		return err
	case string:
		_, err := blockchain.WriteVarstr31(w, []byte(v))
		return err
	}

	// The two container types in the spec (List and Struct)
	// correspond to slices and structs in Go. They can't be
	// handled with type assertions, so we must use reflect.
	switch v := reflect.ValueOf(c); v.Kind() {
	case reflect.Slice:
		l := v.Len()
		_, err := blockchain.WriteVarint31(w, uint64(l))
		if err != nil {
			return err
		}
		for i := 0; i < l; i++ {
			c := v.Index(i)
			if !c.CanInterface() {
				return errInvalidValue
			}
			err := writeForHash(w, c.Interface())
			if err != nil {
				return err
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
				return err
			}
		}
		return nil
	}

	return errors.Wrap(fmt.Errorf("bad type %T", c))
}

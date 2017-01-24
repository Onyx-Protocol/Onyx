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
<<<<<<< e69df8246b33bf70f67c8e8586e5e5d62ec94666
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
=======
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
>>>>>>> wip: begin txgraph outline

		// TODO: The rest of these are all aliases for [32]byte. Do we
		// really need them all?

	case bc.Hash:
		_, err := w.Write(v[:])
<<<<<<< e69df8246b33bf70f67c8e8586e5e5d62ec94666
		return err
	case entryRef:
		_, err := w.Write(v[:])
		return err
	case extHash:
		_, err := w.Write(v[:])
		return err
	case bc.AssetID:
		_, err := w.Write(v[:])
		return err
=======
		return errors.Wrap(err, "writing bc.Hash for hash")
	case entryRef:
		_, err := w.Write(v[:])
		return errors.Wrap(err, "writing entryRef for hash")
	case extHash:
		_, err := w.Write(v[:])
		return errors.Wrap(err, "writing extHash for hash")
	case bc.AssetID:
		_, err := w.Write(v[:])
		return errors.Wrap(err, "writing bc.AssetID for hash")
>>>>>>> wip: begin txgraph outline
	}

	// The two container types in the spec (List and Struct)
	// correspond to slices and structs in Go. They can't be
	// handled with type assertions, so we must use reflect.
	switch v := reflect.ValueOf(c); v.Kind() {
	case reflect.Slice:
		l := v.Len()
		_, err := blockchain.WriteVarint31(w, uint64(l))
		if err != nil {
<<<<<<< e69df8246b33bf70f67c8e8586e5e5d62ec94666
			return err
=======
			return errors.Wrapf(err, "writing slice (len %d) for hash", l)
>>>>>>> wip: begin txgraph outline
		}
		for i := 0; i < l; i++ {
			c := v.Index(i)
			if !c.CanInterface() {
				return errInvalidValue
			}
			err := writeForHash(w, c.Interface())
			if err != nil {
<<<<<<< e69df8246b33bf70f67c8e8586e5e5d62ec94666
				return err
=======
				return errors.Wrapf(err, "writing slice element %d for hash", i)
>>>>>>> wip: begin txgraph outline
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
<<<<<<< e69df8246b33bf70f67c8e8586e5e5d62ec94666
				return err
=======
				t := v.Type()
				f := t.Field(i)
				return errors.Wrapf(err, "writing struct field %d (%s.%s) for hash", i, t.Name(), f.Name)
>>>>>>> wip: begin txgraph outline
			}
		}
		return nil
	}

	return errors.Wrap(fmt.Errorf("bad type %T", c))
}

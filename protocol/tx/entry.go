package tx

import (
	"errors"
	"io"
	"reflect"

	"chain/crypto/sha3pool"
	"chain/encoding/blockchain"
	"chain/protocol/bc"
)

type entry interface {
	Type() string
	Body() interface{}
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
	case int:
		// TODO: Revisit this type--should this be a uint64?
		_, err := blockchain.WriteVarint63(w, uint64(v))
		return err
	case string:
		_, err := blockchain.WriteVarstr31(w, []byte(v))
		return err
	default:
		return writeForHashReflect(w, reflect.ValueOf(c))
	}
}

func writeForHashReflect(w io.Writer, v reflect.Value) error {
	// the only cases handled by writeForHashReflect are Lists and Structs
	switch v.Kind() {
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
	case reflect.Struct:
		return extStructWriteForHash(w, 0, v)
	}
	return errors.New("bad type")
}

func extStructWriteForHash(w io.Writer, i int, v reflect.Value) error {
	if v.Kind() != reflect.Struct {
		return errors.New("bad type: not an ExtHash")
	}

	l := v.NumField()
	for ; i < l; i++ {
		c := v.Field(i)

		// if c is an exthash and if i < l-1
		if c.Type() == reflect.TypeOf(extHash{}) && i < l-1 {
			h := sha3pool.Get256()
			defer sha3pool.Put256(h)

			err := extStructWriteForHash(h, i+1, v) // takes the "rest" of v
			if err != nil {
				return err
			}
			_, err = io.CopyN(w, h, 32)
			return err
		}

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

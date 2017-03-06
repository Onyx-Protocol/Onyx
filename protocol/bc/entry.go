package bc

import (
	"fmt"
	"io"
	"reflect"

	"chain/crypto/sha3pool"
	"chain/encoding/blockchain"
	"chain/errors"
)

type (
	// ValidChecker can check its validity with respect to a given
	// validation state.
	ValidChecker interface {
		// CheckValid checks the entry for validity w.r.t. the given
		// validation state.
		CheckValid(*validationState) error
	}

	validationState struct {
		// The ID of the blockchain
		blockchainID Hash

		// The enclosing transaction object
		tx *TxEntries

		// The ID of the nearest enclosing entry
		entryID Hash

		// The source position, for validating ValueSources
		sourcePos uint64

		// The destination position, for validating ValueDestinations
		destPos uint64
	}
)

// Entry is the interface implemented by each addressable unit in a
// blockchain: transaction components such as spends, issuances,
// outputs, and retirements (among others), plus blockheaders.
type Entry interface {
	ValidChecker

	// Type produces a short human-readable string uniquely identifying
	// the type of this entry.
	Type() string

	// Body produces the entry's body, which is used as input to
	// EntryID.
	body() interface{}

	// Ordinal reports the position of the TxInput or TxOutput within
	// its transaction, when this entry was created from such an
	// object. (See mapTx.) Both inputs (spends and issuances) and
	// outputs (including retirements) are numbered beginning at
	// zero. Entries not originating in this way report -1.
	Ordinal() int
}

var errInvalidValue = errors.New("invalid value")

// EntryID computes the identifier of an entry, as the hash of its
// body plus some metadata.
func EntryID(e Entry) (hash Hash) {
	if e == nil {
		return hash
	}

	// Nil pointer; not the same as nil interface above. (See
	// https://golang.org/doc/faq#nil_error.)
	if v := reflect.ValueOf(e); v.Kind() == reflect.Ptr && v.IsNil() {
		return hash
	}

	hasher := sha3pool.Get256()
	defer sha3pool.Put256(hasher)

	hasher.Write([]byte("entryid:"))
	hasher.Write([]byte(e.Type()))
	hasher.Write([]byte{':'})

	bh := sha3pool.Get256()
	defer sha3pool.Put256(bh)
	err := writeForHash(bh, e.body())
	if err != nil {
		panic(err)
	}
	var innerHash Hash
	bh.Read(innerHash[:])
	hasher.Write(innerHash[:])

	hasher.Read(hash[:])
	return hash
}

// writeForHash serializes the object c to the writer w, from which
// presumably a hash can be extracted.
//
// This function may propagate an error from the underlying writer,
// and may produce errors of its own if passed objects whose
// hash-serialization formats are not specified. It MUST NOT produce
// errors in other cases.
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
	case [][]byte:
		_, err := blockchain.WriteVarstrList(w, v)
		return errors.Wrapf(err, "writing [][]byte (len %d) for hash", len(v))
	case string:
		_, err := blockchain.WriteVarstr31(w, []byte(v))
		return errors.Wrapf(err, "writing string (len %d) for hash", len(v))
	case Hash:
		_, err := w.Write(v[:])
		return errors.Wrap(err, "writing Hash for hash")
	case AssetID:
		_, err := w.Write(v[:])
		return errors.Wrap(err, "writing AssetID for hash")
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
		typ := v.Type()
		for i := 0; i < typ.NumField(); i++ {
			sf := typ.Field(i)
			if sf.Tag.Get("entry") == "-" {
				// exclude this field from hashing
				continue
			}
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

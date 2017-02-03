package bc

import (
	"bytes"
	"database/sql/driver"
	"encoding/hex"
	"fmt"
	"io"

	"chain/crypto/sha3pool"
	"chain/encoding/blockchain"
	"chain/errors"
)

// CurrentTransactionVersion is the current latest
// supported transaction version.
const CurrentTransactionVersion = 1

// Tx holds a transaction along with its hash.
type Tx struct {
	TxData
	Hash Hash
}

// TxHashes holds data needed for validation and state updates.
type TxHashes struct {
	ID        Hash
	OutputIDs []Hash // each OutputID is also the corresponding UnspentID
	Issuances []struct {
		ID           Hash
		ExpirationMS uint64
	}
	VMContexts []*VMContext // one per old-style Input
}

type VMContext struct {
	TxRefDataHash Hash
	RefDataHash   Hash
	TxSigHash     Hash
	OutputID      *Hash
	EntryID       Hash
	NonceID       *Hash
}

func (tx *Tx) UnmarshalText(p []byte) error {
	if err := tx.TxData.UnmarshalText(p); err != nil {
		return err
	}

	tx.Hash = tx.TxData.Hash()
	return nil
}

// NewTx returns a new Tx containing data and its hash.
// If you have already computed the hash, use struct literal
// notation to make a Tx object directly.
func NewTx(data TxData) *Tx {
	return &Tx{
		TxData: data,
		Hash:   data.Hash(),
	}
}

// These flags are part of the wire protocol;
// they must not change.
const (
	SerWitness uint8 = 1 << iota
	SerPrevout
	SerMetadata

	// Bit mask for accepted serialization flags.
	// All other flag bits must be 0.
	SerTxHash   = 0x0 // this is used only for computing transaction hash - prevout and refdata are replaced with their hashes
	SerValid    = 0x7
	serRequired = 0x7 // we support only this combination of flags
)

// TxData encodes a transaction in the blockchain.
// Most users will want to use Tx instead;
// it includes the hash.
type TxData struct {
	Version uint64
	Inputs  []*TxInput
	Outputs []*TxOutput

	// Common fields
	MinTime uint64
	MaxTime uint64

	// The unconsumed suffix of the common fields extensible string
	CommonFieldsSuffix []byte

	// The unconsumed suffix of the common witness extensible string
	CommonWitnessSuffix []byte

	ReferenceData []byte
}

// HasIssuance returns true if this transaction has an issuance input.
func (tx *TxData) HasIssuance() bool {
	for _, in := range tx.Inputs {
		if in.IsIssuance() {
			return true
		}
	}
	return false
}

func (tx *TxData) UnmarshalText(p []byte) error {
	b := make([]byte, hex.DecodedLen(len(p)))
	_, err := hex.Decode(b, p)
	if err != nil {
		return err
	}
	return tx.readFrom(bytes.NewReader(b))
}

func (tx *TxData) Scan(val interface{}) error {
	b, ok := val.([]byte)
	if !ok {
		return errors.New("Scan must receive a byte slice")
	}
	return tx.readFrom(bytes.NewReader(b))
}

func (tx *TxData) Value() (driver.Value, error) {
	b := new(bytes.Buffer)
	_, err := tx.WriteTo(b)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (tx *TxData) readFrom(r io.Reader) error {
	var serflags [1]byte
	_, err := io.ReadFull(r, serflags[:])
	if err != nil {
		return errors.Wrap(err, "reading serialization flags")
	}
	if err == nil && serflags[0] != serRequired {
		return fmt.Errorf("unsupported serflags %#x", serflags[0])
	}

	tx.Version, _, err = blockchain.ReadVarint63(r)
	if err != nil {
		return errors.Wrap(err, "reading transaction version")
	}

	// Common fields
	tx.CommonFieldsSuffix, _, err = blockchain.ReadExtensibleString(r, func(r io.Reader) error {
		tx.MinTime, _, err = blockchain.ReadVarint63(r)
		if err != nil {
			return errors.Wrap(err, "reading transaction mintime")
		}
		tx.MaxTime, _, err = blockchain.ReadVarint63(r)
		return errors.Wrap(err, "reading transaction maxtime")
	})
	if err != nil {
		return errors.Wrap(err, "reading transaction common fields")
	}

	// Common witness
	tx.CommonWitnessSuffix, _, err = blockchain.ReadExtensibleString(r, tx.readCommonWitness)
	if err != nil {
		return errors.Wrap(err, "reading transaction common witness")
	}

	n, _, err := blockchain.ReadVarint31(r)
	if err != nil {
		return errors.Wrap(err, "reading number of transaction inputs")
	}
	for ; n > 0; n-- {
		ti := new(TxInput)
		err = ti.readFrom(r)
		if err != nil {
			return errors.Wrapf(err, "reading input %d", len(tx.Inputs))
		}
		tx.Inputs = append(tx.Inputs, ti)
	}

	n, _, err = blockchain.ReadVarint31(r)
	if err != nil {
		return errors.Wrap(err, "reading number of transaction outputs")
	}
	for ; n > 0; n-- {
		to := new(TxOutput)
		err = to.readFrom(r, tx.Version)
		if err != nil {
			return errors.Wrapf(err, "reading output %d", len(tx.Outputs))
		}
		tx.Outputs = append(tx.Outputs, to)
	}

	tx.ReferenceData, _, err = blockchain.ReadVarstr31(r)
	return errors.Wrap(err, "reading transaction reference data")
}

// does not read the enclosing extensible string
func (tx *TxData) readCommonWitness(r io.Reader) error {
	return nil
}

// Hash computes the hash of the transaction with reference data fields
// replaced by their hashes,
// and stores the result in Hash.
func (tx *TxData) Hash() Hash {
	h := sha3pool.Get256()
	tx.writeTo(h, SerTxHash) // error is impossible
	var v Hash
	h.Read(v[:])
	sha3pool.Put256(h)
	return v
}

func (tx *TxData) IssuanceHash(n int) (h Hash, err error) {
	if n < 0 || n >= len(tx.Inputs) {
		return h, fmt.Errorf("no input %d", n)
	}
	ii, ok := tx.Inputs[n].TypedInput.(*IssuanceInput)
	if !ok {
		return h, fmt.Errorf("not an issuance input")
	}
	buf := sha3pool.Get256()
	defer sha3pool.Put256(buf)

	_, err = blockchain.WriteVarstr31(buf, ii.Nonce)
	if err != nil {
		return h, err
	}
	assetID := ii.AssetID()
	buf.Write(assetID[:])
	_, err = blockchain.WriteVarint63(buf, tx.MinTime)
	if err != nil {
		return h, err
	}
	_, err = blockchain.WriteVarint63(buf, tx.MaxTime)
	if err != nil {
		return h, err
	}
	buf.Read(h[:])
	return h, nil
}

// HashForSig generates the hash required for the specified input's
// signature.
func (tx *TxData) HashForSig(idx uint32) Hash {
	return NewSigHasher(tx).Hash(idx)
}

func (tx *Tx) OutputID(outputIndex uint32) OutputID {
	return ComputeOutputID(tx.Hash, outputIndex)
}

func (tx *TxData) OutputID(outputIndex uint32) OutputID {
	return ComputeOutputID(tx.Hash(), outputIndex)
}

func (tx *TxData) MarshalText() ([]byte, error) {
	var buf bytes.Buffer
	tx.WriteTo(&buf) // error is impossible
	b := make([]byte, hex.EncodedLen(buf.Len()))
	hex.Encode(b, buf.Bytes())
	return b, nil
}

// WriteTo writes tx to w.
func (tx *TxData) WriteTo(w io.Writer) (int64, error) {
	ew := errors.NewWriter(w)
	err := tx.writeTo(ew, serRequired)
	if err != nil {
		return ew.Written(), ew.Err()
	}
	return ew.Written(), ew.Err()
}

func (tx *TxData) writeTo(w io.Writer, serflags byte) error {
	_, err := w.Write([]byte{serflags})
	if err != nil {
		return errors.Wrap(err, "writing serialization flags")
	}

	_, err = blockchain.WriteVarint63(w, tx.Version)
	if err != nil {
		return errors.Wrap(err, "writing transaction version")
	}

	// common fields
	_, err = blockchain.WriteExtensibleString(w, tx.CommonFieldsSuffix, func(w io.Writer) error {
		_, err := blockchain.WriteVarint63(w, tx.MinTime)
		if err != nil {
			return errors.Wrap(err, "writing transaction min time")
		}
		_, err = blockchain.WriteVarint63(w, tx.MaxTime)
		return errors.Wrap(err, "writing transaction max time")
	})
	if err != nil {
		return errors.Wrap(err, "writing common fields")
	}

	// common witness
	_, err = blockchain.WriteExtensibleString(w, tx.CommonWitnessSuffix, tx.writeCommonWitness)
	if err != nil {
		return errors.Wrap(err, "writing common witness")
	}

	_, err = blockchain.WriteVarint31(w, uint64(len(tx.Inputs)))
	if err != nil {
		return errors.Wrap(err, "writing tx input count")
	}
	for i, ti := range tx.Inputs {
		err = ti.writeTo(w, serflags)
		if err != nil {
			return errors.Wrapf(err, "writing tx input %d", i)
		}
	}

	_, err = blockchain.WriteVarint31(w, uint64(len(tx.Outputs)))
	if err != nil {
		return errors.Wrap(err, "writing tx output count")
	}
	for i, to := range tx.Outputs {
		err = to.writeTo(w, serflags)
		if err != nil {
			return errors.Wrapf(err, "writing tx output %d", i)
		}
	}

	return writeRefData(w, tx.ReferenceData, serflags)
}

// does not write the enclosing extensible string
func (tx *TxData) writeCommonWitness(w io.Writer) error {
	// Future protocol versions may add fields here.
	return nil
}

func writeRefData(w io.Writer, data []byte, serflags byte) error {
	if serflags&SerMetadata != 0 {
		_, err := blockchain.WriteVarstr31(w, data)
		return err
	}
	return writeFastHash(w, data)
}

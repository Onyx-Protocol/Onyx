package bc

import (
	"bytes"
	"database/sql/driver"
	"encoding/hex"
	"io"
	"strconv"

	"chain/crypto/hash256"
	"chain/errors"
)

const (
	// CurrentTransactionVersion is the current latest
	// supported transaction version.
	CurrentTransactionVersion = 1

	// InvalidOutputIndex indicates issuance transaction.
	InvalidOutputIndex uint32 = 0xffffffff
)

// Tx holds a transaction along with its hash.
type Tx struct {
	TxData
	Hash   Hash
	Stored bool // whether this tx is on durable storage
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

// TxData encodes a transaction in the blockchain.
// Most users will want to use Tx instead;
// it includes the hash.
type TxData struct {
	Version  uint32
	Inputs   []*TxInput
	Outputs  []*TxOutput
	LockTime uint64
	Metadata []byte
}

// TxInput encodes a single input in a transaction.
type TxInput struct {
	Previous        Outpoint
	SignatureScript []byte
	Metadata        []byte
	AssetDefinition []byte

	// Optional attributes for convenience during validation.
	// These are not serialized or hashed.
	Value   uint64
	AssetID AssetID
}

// TxOutput encodes a single output in a transaction.
type TxOutput struct {
	AssetID  AssetID
	Value    uint64
	Script   []byte
	Metadata []byte
}

// Outpoint defines a bitcoin data type that is used to track previous
// transaction outputs.
type Outpoint struct {
	Hash  Hash
	Index uint32
}

func NewOutpoint(b []byte, index uint32) *Outpoint {
	result := &Outpoint{Index: index}
	copy(result.Hash[:], b)
	return result
}

// Copy creates a deep copy of a transaction so that the original does not get
// modified when the copy is manipulated.
func (tx *TxData) Copy() *TxData {
	// Create new tx and start by copying primitive values and making space
	// for the transaction inputs and outputs.
	newTx := TxData{
		Version:  tx.Version,
		Inputs:   make([]*TxInput, 0, len(tx.Inputs)),
		Outputs:  make([]*TxOutput, 0, len(tx.Outputs)),
		LockTime: tx.LockTime,
		Metadata: copyBytes(tx.Metadata),
	}

	// Deep copy the old TxIn data.
	for _, oldIn := range tx.Inputs {
		newIn := new(TxInput)
		*newIn = *oldIn
		newIn.SignatureScript = copyBytes(oldIn.SignatureScript)
		newIn.Metadata = copyBytes(oldIn.Metadata)
		newIn.AssetDefinition = copyBytes(oldIn.AssetDefinition)
		newTx.Inputs = append(newTx.Inputs, newIn)
	}

	// Deep copy the old TxOut data.
	for _, oldOut := range tx.Outputs {
		newOut := new(TxOutput)
		*newOut = *oldOut
		newOut.Script = copyBytes(oldOut.Script)
		newOut.Metadata = copyBytes(oldOut.Metadata)
		newTx.Outputs = append(newTx.Outputs, newOut)
	}

	return &newTx
}

func copyBytes(b []byte) (n []byte) {
	if len(b) > 0 {
		n = make([]byte, len(b))
		copy(n, b)
	}
	return n
}

// IsIssuance returns true if this transaction is an issuance transaction.
// Issuance transaction is one with first input having
// Outpoint.Index == 0xffffffff.
func (tx *Tx) IsIssuance() bool {
	return len(tx.Inputs) > 0 && tx.Inputs[0].IsIssuance()
}

// IsIssuance returns true if input's index is 0xffffffff.
func (ti *TxInput) IsIssuance() bool {
	return ti.Previous.Index == InvalidOutputIndex
}

func (tx *TxData) UnmarshalText(p []byte) error {
	b := make([]byte, hex.DecodedLen(len(p)))
	_, err := hex.Decode(b, p)
	if err != nil {
		return err
	}
	r := &errors.Reader{R: bytes.NewReader(b)}
	tx.readFrom(r)
	return r.Err
}

func (tx *TxData) Scan(val interface{}) error {
	b, ok := val.([]byte)
	if !ok {
		return errors.New("Scan must receive a byte slice")
	}
	r := &errors.Reader{R: bytes.NewReader(b)}
	tx.readFrom(r)
	return r.Err
}

func (tx *TxData) Value() (driver.Value, error) {
	b := new(bytes.Buffer)
	_, err := tx.WriteTo(b)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (tx *TxData) readFrom(r *errors.Reader) {
	tx.Version = readUint32(r)

	for n := readUvarint(r); n > 0; n-- {
		ti := new(TxInput)
		ti.readFrom(r)
		tx.Inputs = append(tx.Inputs, ti)
	}

	for n := readUvarint(r); n > 0; n-- {
		to := new(TxOutput)
		to.readFrom(r)
		tx.Outputs = append(tx.Outputs, to)
	}

	tx.LockTime = readUint64(r)
	readBytes(r, &tx.Metadata)
}

func (ti *TxInput) readFrom(r *errors.Reader) {
	ti.Previous.readFrom(r)
	readBytes(r, (*[]byte)(&ti.SignatureScript))
	readBytes(r, &ti.Metadata)
	readBytes(r, &ti.AssetDefinition)
}

func (to *TxOutput) readFrom(r *errors.Reader) {
	io.ReadFull(r, to.AssetID[:])
	to.Value = readUint64(r)
	readBytes(r, (*[]byte)(&to.Script))
	readBytes(r, &to.Metadata)
}

func (p *Outpoint) readFrom(r *errors.Reader) (n int64, err error) {
	x, err := io.ReadFull(r, p.Hash[:])
	if err != nil {
		return int64(x), err
	}
	p.Index = readUint32(r)
	return 32 + 4, nil
}

// Hash computes the hash of the transaction with metadata fields
// replaced by their hashes,
// and stores the result in Hash.
func (tx *TxData) Hash() Hash {
	h := hash256.New()
	tx.writeTo(h, true) // error is impossible
	var v Hash
	h.Sum(v[:0])
	return v
}

// MarshalText satisfies encoding.TextMarshaller interface
func (tx *TxData) MarshalText() ([]byte, error) {
	var buf bytes.Buffer
	tx.WriteTo(&buf) // error is impossible
	b := make([]byte, hex.EncodedLen(buf.Len()))
	hex.Encode(b, buf.Bytes())
	return b, nil
}

// WriteTo writes tx to w.
func (tx *TxData) WriteTo(w io.Writer) (int64, error) {
	return tx.writeTo(w, false)
}

func (tx *TxData) writeTo(w io.Writer, forHashing bool) (n int64, err error) {
	ew := errors.NewWriter(w)
	writeUint32(ew, tx.Version)

	writeUvarint(ew, uint64(len(tx.Inputs)))
	for _, ti := range tx.Inputs {
		ti.writeTo(ew, forHashing)
	}

	writeUvarint(ew, uint64(len(tx.Outputs)))
	for _, to := range tx.Outputs {
		to.writeTo(ew, forHashing)
	}

	writeUint64(ew, tx.LockTime)
	if forHashing {
		h := hash256.Sum(tx.Metadata)
		ew.Write(h[:])
	} else {
		writeBytes(ew, tx.Metadata)
	}
	return ew.Written(), ew.Err()
}

func (ti *TxInput) writeTo(w *errors.Writer, forHashing bool) {
	ti.Previous.WriteTo(w)

	// Write the signature script or its hash depending on serialization mode.
	// Hashing the hash of the sigscript allows us to prune signatures,
	// redeem scripts and contracts to optimize memory/storage use.
	// Write the metadata or its hash depending on serialization mode.
	if forHashing {
		h := hash256.Sum(ti.SignatureScript)
		w.Write(h[:])
		h = fastHash(ti.Metadata)
		w.Write(h[:])
		h = fastHash(ti.AssetDefinition)
		w.Write(h[:])
	} else {
		writeBytes(w, ti.SignatureScript)
		writeBytes(w, ti.Metadata)
		writeBytes(w, ti.AssetDefinition)
	}
}

func (to *TxOutput) writeTo(w *errors.Writer, forHashing bool) {
	w.Write(to.AssetID[:])
	writeUint64(w, to.Value)
	writeBytes(w, to.Script)

	// Write the metadata or its hash depending on serialization mode.
	if forHashing {
		h := fastHash(to.Metadata)
		w.Write(h[:])
	} else {
		writeBytes(w, to.Metadata)
	}
}

// String returns the Outpoint in the human-readable form "hash:index".
func (p Outpoint) String() string {
	return p.Hash.String() + ":" + strconv.FormatUint(uint64(p.Index), 10)
}

// WriteTo writes p to w.
func (p Outpoint) WriteTo(w io.Writer) (n int64, err error) {
	_, err = w.Write(p.Hash[:])
	if err != nil {
		return 0, err
	}

	var buf [4]byte
	endianness.PutUint32(buf[:], p.Index)
	_, err = w.Write(buf[:])
	if err != nil {
		return 32, err
	}
	return 32 + 4, nil
}

package bc

import (
	"bytes"
	"database/sql/driver"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"

	"chain/crypto/hash256"
	"chain/encoding/blockchain"
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
	SerValid    = 0x7
	serRequired = 0x7 // we support only this combination of flags
)

// TxData encodes a transaction in the blockchain.
// Most users will want to use Tx instead;
// it includes the hash.
type TxData struct {
	SerFlags uint8
	Version  uint32
	Inputs   []*TxInput
	Outputs  []*TxOutput
	LockTime uint64
	Metadata []byte
}

// TxInput encodes a single input in a transaction.
type TxInput struct {
	Previous        Outpoint
	AssetAmount     AssetAmount
	PrevScript      []byte
	SignatureScript []byte
	Metadata        []byte
	AssetDefinition []byte
}

// TxOutput encodes a single output in a transaction.
type TxOutput struct {
	AssetAmount
	Script   []byte
	Metadata []byte
}

// Outpoint defines a bitcoin data type that is used to track previous
// transaction outputs.
type Outpoint struct {
	Hash  Hash   `json:"hash"`
	Index uint32 `json:"index"`
}

func NewOutpoint(b []byte, index uint32) *Outpoint {
	result := &Outpoint{Index: index}
	copy(result.Hash[:], b)
	return result
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
	var serflags [1]byte
	io.ReadFull(r, serflags[:])
	tx.SerFlags = serflags[0]
	if r.Err == nil && tx.SerFlags != serRequired {
		r.Err = fmt.Errorf("unsupported serflags %#x", tx.SerFlags)
		return
	}

	tx.Version = blockchain.ReadUint32(r)

	for n := blockchain.ReadUvarint(r); n > 0; n-- {
		ti := new(TxInput)
		ti.readFrom(r)
		tx.Inputs = append(tx.Inputs, ti)
	}

	for n := blockchain.ReadUvarint(r); n > 0; n-- {
		to := new(TxOutput)
		to.readFrom(r)
		tx.Outputs = append(tx.Outputs, to)
	}

	tx.LockTime = blockchain.ReadUint64(r)
	blockchain.ReadBytes(r, &tx.Metadata)
}

func (ti *TxInput) readFrom(r *errors.Reader) {
	ti.Previous.readFrom(r)
	ti.AssetAmount.readFrom(r)
	blockchain.ReadBytes(r, &ti.PrevScript)
	blockchain.ReadBytes(r, (*[]byte)(&ti.SignatureScript))
	blockchain.ReadBytes(r, &ti.Metadata)
	blockchain.ReadBytes(r, &ti.AssetDefinition)
}

func (to *TxOutput) readFrom(r *errors.Reader) {
	to.AssetAmount.readFrom(r)
	blockchain.ReadBytes(r, (*[]byte)(&to.Script))
	blockchain.ReadBytes(r, &to.Metadata)
}

func (p *Outpoint) readFrom(r *errors.Reader) (n int64, err error) {
	x, err := io.ReadFull(r, p.Hash[:])
	if err != nil {
		return int64(x), err
	}
	p.Index = blockchain.ReadUint32(r)
	return 32 + 4, nil
}

// Hash computes the hash of the transaction with metadata fields
// replaced by their hashes,
// and stores the result in Hash.
func (tx *TxData) Hash() Hash {
	h := hash256.New()
	tx.writeTo(h, 0) // error is impossible
	var v Hash
	h.Sum(v[:0])
	return v
}

// WitnessHash is the combined hash of the
// transactions hash and signature data hash.
// It is used to compute the TxRoot of a block.
func (tx *TxData) WitnessHash() Hash {
	var data []byte

	var lenBytes [9]byte
	n := binary.PutUvarint(lenBytes[:], uint64(len(tx.Inputs)))
	data = append(data, lenBytes[:n]...)

	for _, in := range tx.Inputs {
		sigHash := hash256.Sum(in.SignatureScript)
		data = append(data, sigHash[:]...)
	}

	txHash := tx.Hash()
	dataHash := hash256.Sum(data)

	return hash256.Sum(append(txHash[:], dataHash[:]...))
}

// HashForSig generates the hash required for the
// specified input's signature, given the AssetAmount
// of its matching previous output.
func (tx *TxData) HashForSig(idx int, assetAmount AssetAmount, hashType SigHashType) Hash {
	return tx.HashForSigCached(idx, assetAmount, hashType, nil)
}

// SigHashCache is used to reduce redundant work
// in consecutive HashForSigCached calls.
type SigHashCache struct {
	inputsHash, outputsHash *Hash
}

// HashForSigCached is the same operation as HashForSig,
// but it also stores some of the intermediate hashes
// in order to reduce work in consecutive calls for the
// same transaction.
func (tx *TxData) HashForSigCached(idx int, assetAmount AssetAmount, hashType SigHashType, cache *SigHashCache) Hash {
	var hash, inputsHash, outputsHash Hash

	if (hashType & SigHashAnyOneCanPay) == 0 {
		if cache != nil && cache.inputsHash != nil {
			inputsHash = *cache.inputsHash
		}
		h := hash256.New()
		w := errors.NewWriter(h)
		blockchain.WriteUvarint(w, uint64(len(tx.Inputs)))
		for _, in := range tx.Inputs {
			in.writeTo(w, 0)
		}
		h.Sum(inputsHash[:0])
		if cache != nil {
			cache.inputsHash = &inputsHash
		}
	}

	switch hashType & sigHashMask {
	case SigHashSingle:
		if idx >= len(tx.Outputs) {
			break
		}
		h := hash256.New()
		w := errors.NewWriter(h)
		blockchain.WriteUvarint(w, 1)
		tx.Outputs[idx].writeTo(w, 0)
		h.Sum(outputsHash[:0])
	case SigHashNone:
		break
	default:
		if cache != nil && cache.outputsHash != nil {
			outputsHash = *cache.outputsHash
		} else {
			h := hash256.New()
			w := errors.NewWriter(h)
			blockchain.WriteUvarint(w, uint64(len(tx.Outputs)))
			for _, out := range tx.Outputs {
				out.writeTo(w, 0)
			}
			h.Sum(outputsHash[:0])
			if cache != nil {
				cache.outputsHash = &outputsHash
			}
		}
	}

	h := hash256.New()
	w := errors.NewWriter(h)

	blockchain.WriteUint32(w, tx.Version)

	w.Write(inputsHash[:])

	var buf bytes.Buffer
	tx.Inputs[idx].writeTo(errors.NewWriter(&buf), 0)
	blockchain.WriteBytes(w, buf.Bytes())

	assetAmount.writeTo(w)

	w.Write(outputsHash[:])

	blockchain.WriteUint64(w, tx.LockTime)

	writeMetadata(w, tx.Metadata, 0)

	w.Write([]byte{byte(hashType)})

	h.Sum(hash[:0])

	return hash
}

// MarshalText satisfies blockchain.TextMarshaller interface
func (tx *TxData) MarshalText() ([]byte, error) {
	var buf bytes.Buffer
	tx.WriteTo(&buf) // error is impossible
	b := make([]byte, hex.EncodedLen(buf.Len()))
	hex.Encode(b, buf.Bytes())
	return b, nil
}

// WriteTo writes tx to w.
func (tx *TxData) WriteTo(w io.Writer) (int64, error) {
	return tx.writeTo(w, serRequired)
}

func (tx *TxData) writeTo(w io.Writer, serflags byte) (n int64, err error) {
	ew := errors.NewWriter(w)
	ew.Write([]byte{serflags})
	blockchain.WriteUint32(ew, tx.Version)

	blockchain.WriteUvarint(ew, uint64(len(tx.Inputs)))
	for _, ti := range tx.Inputs {
		ti.writeTo(ew, serflags)
	}

	blockchain.WriteUvarint(ew, uint64(len(tx.Outputs)))
	for _, to := range tx.Outputs {
		to.writeTo(ew, serflags)
	}

	blockchain.WriteUint64(ew, tx.LockTime)
	writeMetadata(ew, tx.Metadata, serflags)
	return ew.Written(), ew.Err()
}

func (ti *TxInput) writeTo(w *errors.Writer, serflags byte) {
	ti.Previous.WriteTo(w)

	if serflags&SerPrevout != 0 {
		ti.AssetAmount.writeTo(w)
		blockchain.WriteBytes(w, ti.PrevScript)
	}

	// Write the signature script or its hash depending on serialization mode.
	// Hashing the hash of the sigscript allows us to prune signatures,
	// redeem scripts and contracts to optimize memory/storage use.
	// Write the metadata or its hash depending on serialization mode.
	if serflags&SerWitness != 0 {
		blockchain.WriteBytes(w, ti.SignatureScript)
	} else {
		blockchain.WriteBytes(w, nil)
	}

	writeMetadata(w, ti.Metadata, serflags)
	writeMetadata(w, ti.AssetDefinition, serflags)
}

func (to *TxOutput) writeTo(w *errors.Writer, serflags byte) {
	to.AssetAmount.writeTo(w)
	blockchain.WriteBytes(w, to.Script)

	// Write the metadata or its hash depending on serialization mode.
	writeMetadata(w, to.Metadata, serflags)
}

// String returns the Outpoint in the human-readable form "hash:index".
func (p Outpoint) String() string {
	return p.Hash.String() + ":" + strconv.FormatUint(uint64(p.Index), 10)
}

// WriteTo writes p to w.
func (p Outpoint) WriteTo(ew *errors.Writer) {
	ew.Write(p.Hash[:])
	blockchain.WriteUint32(ew, p.Index)
}

type AssetAmount struct {
	AssetID AssetID `json:"asset_id"`
	Amount  uint64  `json:"amount"`
}

func (a *AssetAmount) readFrom(r *errors.Reader) {
	io.ReadFull(r, a.AssetID[:])
	a.Amount = blockchain.ReadUint64(r)
}

func (a AssetAmount) writeTo(w *errors.Writer) {
	w.Write(a.AssetID[:])
	blockchain.WriteUint64(w, a.Amount)
}

func writeMetadata(w *errors.Writer, data []byte, serflags byte) {
	if serflags&SerMetadata != 0 {
		blockchain.WriteBytes(w, data)
	} else {
		h := fastHash(data)
		blockchain.WriteBytes(w, h)
	}
}

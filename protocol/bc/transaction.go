package bc

import (
	"bytes"
	"database/sql/driver"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"

	"golang.org/x/crypto/sha3"

	"chain/encoding/blockchain"
	"chain/errors"
)

const (
	// CurrentTransactionVersion is the current latest
	// supported transaction version.
	CurrentTransactionVersion = 1

	VMVersion = 1
)

const (
	refDataMaxByteLength      = 500000 // 500 kb
	witnessMaxByteLength      = 500000 // 500 kb TODO(bobg): move this where it makes sense for block.go to share it
	commonFieldsMaxByteLength = 500000 // 500 kb
)

// Tx holds a transaction along with its hash.
type Tx struct {
	TxData
	Hash Hash
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
	Version       uint32
	Inputs        []*TxInput
	Outputs       []*TxOutput
	MinTime       uint64
	MaxTime       uint64
	ReferenceData []byte
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
		return err
	}
	if err == nil && serflags[0] != serRequired {
		return fmt.Errorf("unsupported serflags %#x", serflags[0])
	}

	v, err := blockchain.ReadUvarint(r)
	if err != nil {
		return err
	}
	tx.Version = uint32(v)

	commonFields, err := blockchain.ReadBytes(r, commonFieldsMaxByteLength)
	if err != nil {
		return err
	}
	buf := bytes.NewReader(commonFields)
	tx.MinTime, err = blockchain.ReadUvarint(buf)
	if err != nil {
		return err
	}
	tx.MaxTime, err = blockchain.ReadUvarint(buf)
	if err != nil {
		return err
	}

	// Common witness, empty in v1
	_, err = blockchain.ReadBytes(r, witnessMaxByteLength)
	if err != nil {
		return err
	}

	n, err := blockchain.ReadUvarint(r)
	if err != nil {
		return err
	}
	for ; n > 0; n-- {
		ti := new(TxInput)
		err = ti.readFrom(r)
		if err != nil {
			return err
		}
		tx.Inputs = append(tx.Inputs, ti)
	}

	n, err = blockchain.ReadUvarint(r)
	if err != nil {
		return err
	}
	for ; n > 0; n-- {
		to := new(TxOutput)
		err = to.readFrom(r)
		if err != nil {
			return err
		}
		tx.Outputs = append(tx.Outputs, to)
	}

	tx.ReferenceData, err = blockchain.ReadBytes(r, refDataMaxByteLength)
	return err
}

func (p *Outpoint) readFrom(r io.Reader) error {
	_, err := io.ReadFull(r, p.Hash[:])
	if err != nil {
		return err
	}
	index, err := blockchain.ReadUvarint(r)
	if err != nil {
		return err
	}
	// TODO(bobg): range check index
	p.Index = uint32(index)
	return nil
}

// Hash computes the hash of the transaction with reference data fields
// replaced by their hashes,
// and stores the result in Hash.
func (tx *TxData) Hash() Hash {
	h := sha3.New256()
	tx.writeTo(h, 0) // error is impossible
	var v Hash
	h.Sum(v[:0])
	return v
}

// WitnessHash is the combined hash of the
// transactions hash and signature data hash.
// It is used to compute the TxRoot of a block.
func (tx *TxData) WitnessHash() Hash {
	var b bytes.Buffer

	txhash := tx.Hash()
	b.Write(txhash[:])

	blockchain.WriteUvarint(&b, uint64(len(tx.Inputs)))
	for _, txin := range tx.Inputs {
		h := txin.WitnessHash()
		b.Write(h[:])
	}

	blockchain.WriteUvarint(&b, uint64(len(tx.Outputs)))
	for _, txout := range tx.Outputs {
		h := txout.WitnessHash()
		b.Write(h[:])
	}

	return sha3.Sum256(b.Bytes())
}

func (tx *TxData) IssuanceHash(n int) (h Hash, err error) {
	if n < 0 || n >= len(tx.Inputs) {
		return h, fmt.Errorf("no input %d", n)
	}
	ii, ok := tx.Inputs[n].TypedInput.(*IssuanceInput)
	if !ok {
		return h, fmt.Errorf("not an issuance input")
	}
	buf := sha3.New256()
	blockchain.WriteBytes(buf, ii.Nonce)
	assetID := ii.AssetID()
	buf.Write(assetID[:])
	blockchain.WriteUvarint(buf, tx.MinTime)
	blockchain.WriteUvarint(buf, tx.MaxTime)
	copy(h[:], buf.Sum(nil))
	return h, nil
}

// HashForSig generates the hash required for the specified input's
// signature.
func (tx *TxData) HashForSig(idx int) Hash {
	return NewSigHasher(tx).Hash(idx)
}

// SigHasher caches a txhash for reuse with multiple inputs.
type SigHasher struct {
	txData *TxData
	txHash *Hash // not computed until needed
}

func NewSigHasher(txData *TxData) *SigHasher {
	return &SigHasher{txData: txData}
}

func (s *SigHasher) Hash(idx int) Hash {
	if s.txHash == nil {
		h := s.txData.Hash()
		s.txHash = &h
	}
	var buf bytes.Buffer
	buf.Write((*s.txHash)[:])
	blockchain.WriteUvarint(&buf, uint64(idx))

	var h Hash
	inp := s.txData.Inputs[idx]
	si, ok := inp.TypedInput.(*SpendInput)
	if ok {
		// inp is a spend
		var ocBuf bytes.Buffer
		si.OutputCommitment.writeTo(&ocBuf, inp.AssetVersion)
		h = sha3.Sum256(ocBuf.Bytes())
	} else {
		// inp is an issuance
		h = emptyHash
	}

	buf.Write(h[:])
	return sha3.Sum256(buf.Bytes())
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
	ew := errors.NewWriter(w)
	tx.writeTo(ew, serRequired)
	return ew.Written(), ew.Err()
}

// assumes w has sticky errors
func (tx *TxData) writeTo(w io.Writer, serflags byte) {
	w.Write([]byte{serflags})
	blockchain.WriteUvarint(w, uint64(tx.Version))

	// common fields
	var buf bytes.Buffer
	blockchain.WriteUvarint(&buf, tx.MinTime)
	blockchain.WriteUvarint(&buf, tx.MaxTime)
	blockchain.WriteBytes(w, buf.Bytes())

	// common witness
	blockchain.WriteBytes(w, []byte{})

	blockchain.WriteUvarint(w, uint64(len(tx.Inputs)))
	for _, ti := range tx.Inputs {
		ti.writeTo(w, serflags)
	}

	blockchain.WriteUvarint(w, uint64(len(tx.Outputs)))
	for _, to := range tx.Outputs {
		to.writeTo(w, serflags)
	}

	writeRefData(w, tx.ReferenceData, serflags)
}

// String returns the Outpoint in the human-readable form "hash:index".
func (p Outpoint) String() string {
	return p.Hash.String() + ":" + strconv.FormatUint(uint64(p.Index), 10)
}

// WriteTo writes p to w.
// It assumes w has sticky errors.
func (p Outpoint) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(p.Hash[:])
	if err != nil {
		return int64(n), err
	}
	u, err := blockchain.WriteUvarint(w, uint64(p.Index))
	return int64(n + u), err
}

type AssetAmount struct {
	AssetID AssetID `json:"asset_id"`
	Amount  uint64  `json:"amount"`
}

// assumes r has sticky errors
func (a *AssetAmount) readFrom(r io.Reader) {
	io.ReadFull(r, a.AssetID[:])
	a.Amount, _ = blockchain.ReadUvarint(r)
}

// assumes w has sticky errors
func (a AssetAmount) writeTo(w io.Writer) {
	w.Write(a.AssetID[:])
	blockchain.WriteUvarint(w, a.Amount)
}

// assumes w has sticky errors
func writeRefData(w io.Writer, data []byte, serflags byte) {
	if serflags&SerMetadata != 0 {
		blockchain.WriteBytes(w, data)
	} else {
		h := fastHash(data)
		blockchain.WriteBytes(w, h)
	}
}

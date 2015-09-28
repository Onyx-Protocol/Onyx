package bc

import (
	"encoding/binary"
	"fmt"
	"io"

	"chain/crypto/hash256"
	"chain/errors"
	"chain/fedchain/script"
)

const (
	// CurrentTransactionVersion is the current latest
	// supported transaction version.
	CurrentTransactionVersion = 1

	// InvalidOutputIndex indicates issuance transaction.
	InvalidOutputIndex uint32 = 0xffffffff
)

// Tx encodes a transaction in the blockchain.
type Tx struct {
	Version  uint32
	Inputs   []TxInput
	Outputs  []TxOutput
	LockTime uint64
	Metadata []byte
}

// TxInput encodes a single input in a transaction.
type TxInput struct {
	Previous        Outpoint
	SignatureScript script.Script
	Metadata        []byte

	// Optional attributes for convenience during validation.
	// These are not serialized or hashed.
	Value      uint64
	AssetID    AssetID
	IssuanceID IssuanceID
}

// TxOutput encodes a single output in a transaction.
type TxOutput struct {
	AssetID  AssetID
	Value    uint64
	Script   script.Script
	Metadata []byte

	// Optional attributes for convenience during validation.
	// These are not serialized or hashed.
	Outpoint   Outpoint
	IssuanceID IssuanceID
}

// Outpoint defines a bitcoin data type that is used to track previous
// transaction outputs.
type Outpoint struct {
	Hash  [32]byte
	Index uint32
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

// Hash returns hash of the transaction with metadata fields
// replaced by their hashes.
func (tx *Tx) Hash() [32]byte {
	h := hash256.New()
	tx.WriteForHashTo(h) // error is impossible
	var v [32]byte
	h.Sum(v[:0])
	return v
}

// WriteTo writes tx to w.
func (tx *Tx) WriteTo(w io.Writer) (int64, error) {
	return tx.writeTo(w, false)
}

// WriteForHashTo writes tx to w, substituting the Hash256
// of the signature scripts and metadata in place of
// those fields.
func (tx *Tx) WriteForHashTo(w io.Writer) (int64, error) {
	return tx.writeTo(w, true)
}

func (tx *Tx) writeTo(w io.Writer, forHashing bool) (n int64, err error) {
	ew := errors.NewWriter(w)
	binary.Write(ew, endianness, tx.Version)

	writeUvarint(ew, uint64(len(tx.Inputs)))
	for i := range tx.Inputs {
		ti := &tx.Inputs[i]
		ti.writeTo(ew, forHashing)
	}

	writeUvarint(ew, uint64(len(tx.Outputs)))
	for i := range tx.Outputs {
		to := &tx.Outputs[i]
		to.writeTo(ew, forHashing)
	}

	binary.Write(ew, endianness, tx.LockTime)
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
		h = hash256.Sum([]byte(ti.Metadata))
		w.Write(h[:])
	} else {
		writeBytes(w, ti.SignatureScript)
		writeBytes(w, ti.Metadata)
	}
}

func (to *TxOutput) writeTo(w *errors.Writer, forHashing bool) {
	w.Write(to.AssetID[:])
	binary.Write(w, endianness, to.Value)
	writeBytes(w, to.Script)

	// Write the metadata or its hash depending on serialization mode.
	if forHashing {
		h := hash256.Sum([]byte(to.Metadata))
		w.Write(h[:])
	} else {
		writeBytes(w, to.Metadata)
	}
}

// String returns the Outpoint in the human-readable form "hash:index".
func (p Outpoint) String() string {
	return fmt.Sprintf("%x:%d", p.Hash, p.Index)
}

// WriteTo writes p to w.
func (p Outpoint) WriteTo(w io.Writer) (n int64, err error) {
	err = binary.Write(w, endianness, p)
	if err != nil {
		return 0, err
	}
	return 32 + 4, nil
}

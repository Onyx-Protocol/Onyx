package bc

import (
	"bytes"
	"database/sql/driver"
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/sha3"

	"chain/encoding/blockchain"
	"chain/errors"
)

const (
	commitmentMaxByteLength = 1000000 // 1mb

	// MaxProgramByteLength is the maximum length in bytes of a COS program.
	MaxProgramByteLength = 1000000
)

const (
	SerBlockWitness      = 1
	SerBlockTransactions = 2

	SerBlockSigHash = 0
	SerBlockHeader  = SerBlockWitness
	SerBlockFull    = SerBlockWitness | SerBlockTransactions
)

// Block describes a complete block, including its header
// and the transactions it contains.
type Block struct {
	BlockHeader
	Transactions []*Tx
}

func (b *Block) Scan(val interface{}) error {
	buf, ok := val.([]byte)
	if !ok {
		return errors.New("Scan must receive a byte slice")
	}
	return b.readFrom(bytes.NewReader(buf))
}

func (b *Block) Value() (driver.Value, error) {
	buf := new(bytes.Buffer)
	_, err := b.WriteTo(buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (b *Block) readFrom(r io.Reader) error {
	serflags, err := b.BlockHeader.readFrom(r)
	if err != nil {
		return err
	}
	if serflags&SerBlockTransactions == SerBlockTransactions {
		n, err := blockchain.ReadUvarint(r)
		if err != nil {
			return err
		}
		for ; n > 0; n-- {
			var data TxData
			err = data.readFrom(r)
			if err != nil {
				return err
			}
			// TODO(kr): store/reload hashes;
			// don't compute here if not necessary.
			tx := NewTx(data)
			b.Transactions = append(b.Transactions, tx)
		}
	}
	return nil
}

func (b *Block) WriteTo(w io.Writer) (int64, error) {
	ew := errors.NewWriter(w)
	b.writeTo(ew, SerBlockFull)
	return ew.Written(), ew.Err()
}

// assumes w has sticky errors
func (b *Block) writeTo(w io.Writer, serflags uint8) {
	b.BlockHeader.writeTo(w, serflags)
	if serflags&SerBlockTransactions == SerBlockTransactions {
		blockchain.WriteUvarint(w, uint64(len(b.Transactions)))
		for _, tx := range b.Transactions {
			tx.WriteTo(w)
		}
	}
}

// NewBlockVersion is the version to use when creating new blocks.
const NewBlockVersion = 1

// BlockHeader describes necessary data of the block.
type BlockHeader struct {
	// Version of the block.
	Version uint32

	// Height of the block in the block chain.
	// Genesis block has height 0.
	Height uint64

	// Hash of the previous block in the block chain.
	PreviousBlockHash Hash

	// Time of the block in milliseconds.
	// Must grow monotonically and can be equal
	// to the time in the previous block.
	TimestampMS uint64

	// The next three fields constitute the block's "commitment."

	// TransactionsMerkleRoot is the root hash of the Merkle binary hash
	// tree formed by the transaction witness hashes of all transactions
	// included in the block.
	TransactionsMerkleRoot Hash

	// AssetsMerkleRoot is the root hash of the Merkle Patricia Tree of
	// the set of unspent outputs with asset version 1 after applying
	// the block.
	AssetsMerkleRoot Hash

	// ConsensusProgram is the predicate for validating the next block.
	ConsensusProgram []byte

	// Witness is a vector of arguments to the previous block's
	// ConsensusProgram for validating this block.
	Witness [][]byte
}

// Time returns the time represented by the Timestamp in bh.
func (bh *BlockHeader) Time() time.Time {
	tsNano := bh.TimestampMS * uint64(time.Millisecond)
	return time.Unix(0, int64(tsNano)).UTC()
}

func (bh *BlockHeader) Scan(val interface{}) error {
	buf, ok := val.([]byte)
	if !ok {
		return errors.New("Scan must receive a byte slice")
	}
	_, err := bh.readFrom(bytes.NewReader(buf))
	return err
}

func (bh *BlockHeader) Value() (driver.Value, error) {
	buf := new(bytes.Buffer)
	_, err := bh.WriteTo(buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Hash returns complete hash of the block header.
func (bh *BlockHeader) Hash() Hash {
	h := sha3.New256()
	bh.WriteTo(h) // error is impossible
	var v [32]byte
	h.Sum(v[:0])
	return v
}

// HashForSig returns a hash of the block header without witness.
// This hash is used for signing the block and verifying the
// signature.
func (bh *BlockHeader) HashForSig() Hash {
	h := sha3.New256()
	bh.WriteForSigTo(h) // error is impossible
	var v [32]byte
	h.Sum(v[:0])
	return v
}

func (bh *BlockHeader) readFrom(r io.Reader) (uint8, error) {
	var serflags [1]byte
	io.ReadFull(r, serflags[:])
	switch serflags[0] {
	case SerBlockSigHash, SerBlockHeader, SerBlockFull:
	default:
		return 0, fmt.Errorf("unsupported serialization flags 0x%x", serflags)
	}

	v, err := blockchain.ReadUvarint(r)
	if err != nil {
		return 0, err
	}
	bh.Version = uint32(v)

	bh.Height, err = blockchain.ReadUvarint(r)
	if err != nil {
		return 0, err
	}
	_, err = io.ReadFull(r, bh.PreviousBlockHash[:])
	if err != nil {
		return 0, err
	}

	bh.TimestampMS, err = blockchain.ReadUvarint(r)
	if err != nil {
		return 0, err
	}

	commitment, err := blockchain.ReadBytes(r, commitmentMaxByteLength)
	if err != nil {
		return 0, err
	}
	if len(commitment) < 64 {
		return 0, fmt.Errorf("block commitment string too short")
	}
	copy(bh.TransactionsMerkleRoot[:], commitment[:32])
	copy(bh.AssetsMerkleRoot[:], commitment[32:64])
	progReader := bytes.NewReader(commitment[64:])
	bh.ConsensusProgram, err = blockchain.ReadBytes(progReader, MaxProgramByteLength)
	if err != nil {
		return 0, err
	}

	if serflags[0]&SerBlockWitness == SerBlockWitness {
		witness, err := blockchain.ReadBytes(r, witnessMaxByteLength)
		if err != nil {
			return 0, err
		}
		witnessReader := bytes.NewReader(witness)

		n, err := blockchain.ReadUvarint(witnessReader)
		if err != nil {
			return 0, err
		}

		for ; n > 0; n-- {
			w, err := blockchain.ReadBytes(witnessReader, witnessMaxByteLength)
			if err != nil {
				return 0, errors.Wrap(err, "reading block witness")
			}
			bh.Witness = append(bh.Witness, w)
		}
	}

	return serflags[0], nil
}

func (bh *BlockHeader) WriteTo(w io.Writer) (int64, error) {
	ew := errors.NewWriter(w)
	bh.writeTo(ew, SerBlockHeader)
	return ew.Written(), ew.Err()
}

// WriteForSigTo writes bh to w in a format suitable for signing.
func (bh *BlockHeader) WriteForSigTo(w io.Writer) (int64, error) {
	ew := errors.NewWriter(w)
	bh.writeTo(ew, SerBlockSigHash)
	return ew.Written(), ew.Err()
}

// writeTo writes bh to w.
func (bh *BlockHeader) writeTo(w io.Writer, serflags uint8) error {
	w.Write([]byte{serflags})
	blockchain.WriteUvarint(w, uint64(bh.Version))
	blockchain.WriteUvarint(w, bh.Height)
	w.Write(bh.PreviousBlockHash[:])
	blockchain.WriteUvarint(w, bh.TimestampMS)

	var commitment bytes.Buffer
	commitment.Write(bh.TransactionsMerkleRoot[:])
	commitment.Write(bh.AssetsMerkleRoot[:])
	blockchain.WriteBytes(&commitment, bh.ConsensusProgram)

	blockchain.WriteBytes(w, commitment.Bytes())

	if serflags&SerBlockWitness == SerBlockWitness {
		var witnessBuf bytes.Buffer

		blockchain.WriteUvarint(&witnessBuf, uint64(len(bh.Witness)))
		for _, witness := range bh.Witness {
			blockchain.WriteBytes(&witnessBuf, witness)
		}

		blockchain.WriteBytes(w, witnessBuf.Bytes())
	}

	return nil
}

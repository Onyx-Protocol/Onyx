package bc

import (
	"bytes"
	"database/sql/driver"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"chain/crypto/sha3pool"
	"chain/encoding/blockchain"
	"chain/encoding/bufpool"
	"chain/errors"
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

// MarshalText fulfills the json.Marshaler interface.
// This guarantees that blocks will get deserialized correctly
// when being parsed from HTTP requests.
func (b *Block) MarshalText() ([]byte, error) {
	buf := bufpool.Get()
	defer bufpool.Put(buf)
	_, err := b.WriteTo(buf)
	if err != nil {
		return nil, err
	}

	enc := make([]byte, hex.EncodedLen(buf.Len()))
	hex.Encode(enc, buf.Bytes())
	return enc, nil
}

// UnmarshalText fulfills the encoding.TextUnmarshaler interface.
func (b *Block) UnmarshalText(text []byte) error {
	decoded := make([]byte, hex.DecodedLen(len(text)))
	_, err := hex.Decode(decoded, text)
	if err != nil {
		return err
	}
	return b.readFrom(bytes.NewReader(decoded))
}

// Scan fulfills the sql.Scanner interface.
func (b *Block) Scan(val interface{}) error {
	buf, ok := val.([]byte)
	if !ok {
		return errors.New("Scan must receive a byte slice")
	}
	return b.readFrom(bytes.NewReader(buf))
}

// Value fulfills the sql.driver.Valuer interface.
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
		n, _, err := blockchain.ReadVarint31(r)
		if err != nil {
			return errors.Wrap(err, "reading number of transactions")
		}
		for ; n > 0; n-- {
			var data TxData
			err = data.readFrom(r)
			if err != nil {
				return errors.Wrapf(err, "reading transaction %d", len(b.Transactions))
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
		blockchain.WriteVarint31(w, uint64(len(b.Transactions))) // TODO(bobg): check and return error
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
	Version uint64

	// Height of the block in the block chain.
	// Initial block has height 1.
	Height uint64

	// Hash of the previous block in the block chain.
	PreviousBlockHash Hash

	// Time of the block in milliseconds.
	// Must grow monotonically and can be equal
	// to the time in the previous block.
	TimestampMS uint64

	BlockCommitment
	CommitmentSuffix []byte

	BlockWitness
	WitnessSuffix []byte
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
	h := sha3pool.Get256()
	bh.WriteTo(h) // error is impossible
	var v [32]byte
	h.Read(v[:])
	sha3pool.Put256(h)
	return v
}

// HashForSig returns a hash of the block header without witness.
// This hash is used for signing the block and verifying the
// signature.
func (bh *BlockHeader) HashForSig() Hash {
	h := sha3pool.Get256()
	bh.WriteForSigTo(h) // error is impossible
	var v [32]byte
	h.Read(v[:])
	sha3pool.Put256(h)
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

	var err error

	bh.Version, _, err = blockchain.ReadVarint63(r)
	if err != nil {
		return 0, err
	}

	bh.Height, _, err = blockchain.ReadVarint63(r)
	if err != nil {
		return 0, err
	}

	_, err = io.ReadFull(r, bh.PreviousBlockHash[:])
	if err != nil {
		return 0, err
	}

	bh.TimestampMS, _, err = blockchain.ReadVarint63(r)
	if err != nil {
		return 0, err
	}

	bh.CommitmentSuffix, _, err = blockchain.ReadExtensibleString(r, bh.BlockCommitment.readFrom)
	if err != nil {
		return 0, err
	}

	if serflags[0]&SerBlockWitness == SerBlockWitness {
		bh.WitnessSuffix, _, err = blockchain.ReadExtensibleString(r, func(r io.Reader) (err error) {
			bh.Witness, _, err = blockchain.ReadVarstrList(r)
			return err
		})
		if err != nil {
			return 0, err
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

	var err error

	_, err = blockchain.WriteVarint63(w, bh.Version)
	if err != nil {
		return err
	}
	_, err = blockchain.WriteVarint63(w, bh.Height)
	if err != nil {
		return err
	}
	_, err = w.Write(bh.PreviousBlockHash[:])
	if err != nil {
		return err
	}
	_, err = blockchain.WriteVarint63(w, bh.TimestampMS)
	if err != nil {
		return err
	}

	_, err = blockchain.WriteExtensibleString(w, bh.CommitmentSuffix, bh.BlockCommitment.writeTo)
	if err != nil {
		return err
	}

	if serflags&SerBlockWitness == SerBlockWitness {
		_, err = blockchain.WriteExtensibleString(w, bh.WitnessSuffix, bh.BlockWitness.writeTo)
		if err != nil {
			return err
		}
	}

	return nil
}

package legacy

import (
	"bytes"
	"database/sql/driver"
	"encoding/hex"
	"fmt"
	"io"

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

	r := bytes.NewReader(decoded)
	err = b.readFrom(r)
	if err != nil {
		return err
	}
	if trailing := r.Len(); trailing > 0 {
		return fmt.Errorf("trailing garbage (%d bytes)", trailing)
	}
	return nil
}

// Scan fulfills the sql.Scanner interface.
func (b *Block) Scan(val interface{}) error {
	driverBuf, ok := val.([]byte)
	if !ok {
		return errors.New("Scan must receive a byte slice")
	}
	buf := make([]byte, len(driverBuf))
	copy(buf[:], driverBuf)
	r := bytes.NewReader(buf)
	err := b.readFrom(r)
	if err != nil {
		return err
	}
	if trailing := r.Len(); trailing > 0 {
		return fmt.Errorf("trailing garbage (%d bytes)", trailing)
	}
	return nil
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

func (b *Block) readFrom(r blockchain.Reader) error {
	serflags, err := b.BlockHeader.readFrom(r)
	if err != nil {
		return err
	}
	if serflags&SerBlockTransactions == SerBlockTransactions {
		n, err := blockchain.ReadVarint31(r)
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

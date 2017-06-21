package legacy

import (
	"bytes"
	"database/sql/driver"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"chain/encoding/blockchain"
	"chain/encoding/bufpool"
	"chain/errors"
	"chain/protocol/bc"
)

// BlockHeader describes necessary data of the block.
type BlockHeader struct {
	// Version of the block.
	Version uint64

	// Height of the block in the block chain.
	// Initial block has height 1.
	Height uint64

	// Hash of the previous block in the block chain.
	PreviousBlockHash bc.Hash

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
	driverBuf, ok := val.([]byte)
	if !ok {
		return errors.New("Scan must receive a byte slice")
	}
	buf := make([]byte, len(driverBuf))
	copy(buf[:], driverBuf)
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
func (bh *BlockHeader) Hash() bc.Hash {
	h, _ := mapBlockHeader(bh)
	return h
}

// MarshalText fulfills the json.Marshaler interface.
// This guarantees that block headers will get deserialized correctly
// when being parsed from HTTP requests.
func (bh *BlockHeader) MarshalText() ([]byte, error) {
	buf := bufpool.Get()
	defer bufpool.Put(buf)
	_, err := bh.WriteTo(buf)
	if err != nil {
		return nil, err
	}

	enc := make([]byte, hex.EncodedLen(buf.Len()))
	hex.Encode(enc, buf.Bytes())
	return enc, nil
}

// UnmarshalText fulfills the encoding.TextUnmarshaler interface.
func (bh *BlockHeader) UnmarshalText(text []byte) error {
	decoded := make([]byte, hex.DecodedLen(len(text)))
	_, err := hex.Decode(decoded, text)
	if err != nil {
		return err
	}
	_, err = bh.readFrom(bytes.NewReader(decoded))
	return err
}

func (bh *BlockHeader) readFrom(r blockchain.Reader) (uint8, error) {
	var serflags [1]byte
	io.ReadFull(r, serflags[:])
	switch serflags[0] {
	case SerBlockSigHash, SerBlockHeader, SerBlockFull:
	default:
		return 0, fmt.Errorf("unsupported serialization flags 0x%x", serflags)
	}

	var err error

	bh.Version, err = blockchain.ReadVarint63(r)
	if err != nil {
		return 0, err
	}

	bh.Height, err = blockchain.ReadVarint63(r)
	if err != nil {
		return 0, err
	}

	_, err = bh.PreviousBlockHash.ReadFrom(r)
	if err != nil {
		return 0, err
	}

	bh.TimestampMS, err = blockchain.ReadVarint63(r)
	if err != nil {
		return 0, err
	}

	bh.CommitmentSuffix, err = blockchain.ReadExtensibleString(r, bh.BlockCommitment.readFrom)
	if err != nil {
		return 0, err
	}

	if serflags[0]&SerBlockWitness == SerBlockWitness {
		bh.WitnessSuffix, err = blockchain.ReadExtensibleString(r, func(r blockchain.Reader) (err error) {
			bh.Witness, err = blockchain.ReadVarstrList(r)
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
	_, err = bh.PreviousBlockHash.WriteTo(w)
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

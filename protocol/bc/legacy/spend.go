package legacy

import (
	"fmt"
	"io"

	"chain/crypto/sha3pool"
	"chain/encoding/blockchain"
	"chain/errors"
	"chain/protocol/bc"
)

// SpendInput satisfies the TypedInput interface and represents a spend transaction.
type SpendInput struct {
	// Commitment
	SpendCommitment

	// The unconsumed suffix of the output commitment
	SpendCommitmentSuffix []byte

	// Witness
	Arguments [][]byte
}

func (si *SpendInput) IsIssuance() bool { return false }

func NewSpendInput(arguments [][]byte, sourceID bc.Hash, assetID bc.AssetID, amount uint64, sourcePos uint64, controlProgram []byte, outRefDataHash bc.Hash, referenceData []byte) *TxInput {
	const (
		vmver    = 1
		assetver = 1
	)
	sc := SpendCommitment{
		AssetAmount: bc.AssetAmount{
			AssetId: &assetID,
			Amount:  amount,
		},
		SourceID:       sourceID,
		SourcePosition: sourcePos,
		VMVersion:      vmver,
		ControlProgram: controlProgram,
		RefDataHash:    outRefDataHash,
	}
	return &TxInput{
		AssetVersion:  assetver,
		ReferenceData: referenceData,
		TypedInput: &SpendInput{
			SpendCommitment: sc,
			Arguments:       arguments,
		},
	}
}

// SpendCommitment contains the commitment data for a transaction
// output (which also appears in the spend input of that output).
type SpendCommitment struct {
	bc.AssetAmount
	SourceID       bc.Hash
	SourcePosition uint64
	VMVersion      uint64
	ControlProgram []byte
	RefDataHash    bc.Hash
}

func (sc *SpendCommitment) writeExtensibleString(w io.Writer, suffix []byte, assetVersion uint64) error {
	_, err := blockchain.WriteExtensibleString(w, suffix, func(w io.Writer) error {
		return sc.writeContents(w, suffix, assetVersion)
	})
	return err
}

func (sc *SpendCommitment) writeContents(w io.Writer, suffix []byte, assetVersion uint64) (err error) {
	if assetVersion == 1 {
		_, err = sc.SourceID.WriteTo(w)
		if err != nil {
			return errors.Wrap(err, "writing source id")
		}
		_, err = sc.AssetAmount.WriteTo(w)
		if err != nil {
			return errors.Wrap(err, "writing asset amount")
		}
		_, err = blockchain.WriteVarint63(w, sc.SourcePosition)
		if err != nil {
			return errors.Wrap(err, "writing source position")
		}
		_, err = blockchain.WriteVarint63(w, sc.VMVersion)
		if err != nil {
			return errors.Wrap(err, "writing vm version")
		}
		_, err = blockchain.WriteVarstr31(w, sc.ControlProgram)
		if err != nil {
			return errors.Wrap(err, "writing control program")
		}
		_, err = sc.RefDataHash.WriteTo(w)
		if err != nil {
			return errors.Wrap(err, "writing reference data hash")
		}
	}
	if len(suffix) > 0 {
		_, err = w.Write(suffix)
		if err != nil {
			return errors.Wrap(err, "writing suffix")
		}
	}
	return nil
}

func (sc *SpendCommitment) readFrom(r blockchain.Reader, assetVersion uint64) (suffix []byte, err error) {
	return blockchain.ReadExtensibleString(r, func(r blockchain.Reader) error {
		if assetVersion == 1 {
			_, err := sc.SourceID.ReadFrom(r)
			if err != nil {
				return errors.Wrap(err, "reading source id")
			}
			err = sc.AssetAmount.ReadFrom(r)
			if err != nil {
				return errors.Wrap(err, "reading asset+amount")
			}
			sc.SourcePosition, err = blockchain.ReadVarint63(r)
			if err != nil {
				return errors.Wrap(err, "reading source position")
			}
			sc.VMVersion, err = blockchain.ReadVarint63(r)
			if err != nil {
				return errors.Wrap(err, "reading VM version")
			}
			if sc.VMVersion != 1 {
				return fmt.Errorf("unrecognized VM version %d for asset version 1", sc.VMVersion)
			}
			sc.ControlProgram, err = blockchain.ReadVarstr31(r)
			if err != nil {
				return errors.Wrap(err, "reading control program")
			}
			_, err = sc.RefDataHash.ReadFrom(r)
			if err != nil {
				return errors.Wrap(err, "reading reference data hash")
			}
			return nil
		}
		return nil
	})
}

func (sc *SpendCommitment) Hash(suffix []byte, assetVersion uint64) (spendhash bc.Hash) {
	h := sha3pool.Get256()
	defer sha3pool.Put256(h)
	sc.writeExtensibleString(h, suffix, assetVersion) // TODO(oleg): get rid of this assetVersion parameter to actually write all the bytes
	spendhash.ReadFrom(h)
	return spendhash
}

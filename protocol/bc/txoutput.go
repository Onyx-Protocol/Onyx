package bc

import (
	"fmt"
	"io"

	"chain/crypto/sha3pool"
	"chain/encoding/blockchain"
	"chain/errors"
)

// TODO(bobg): Review serialization/deserialization logic for
// assetVersions other than 1.

type TxOutput struct {
	AssetVersion uint64
	OutputCommitment
	ReferenceData []byte
}

func NewTxOutput(assetID AssetID, amount uint64, controlProgram, referenceData []byte) *TxOutput {
	return &TxOutput{
		AssetVersion: 1,
		OutputCommitment: OutputCommitment{
			AssetAmount: AssetAmount{
				AssetID: assetID,
				Amount:  amount,
			},
			VMVersion:      1,
			ControlProgram: controlProgram,
		},
		ReferenceData: referenceData,
	}
}

func (to *TxOutput) readFrom(r io.Reader, txVersion uint64) (err error) {
	to.AssetVersion, _, err = blockchain.ReadVarint63(r)
	if err != nil {
		return errors.Wrap(err, "reading asset version")
	}
	if txVersion == 1 && to.AssetVersion != 1 {
		return fmt.Errorf("unrecognized asset version %d for transaction version %d", to.AssetVersion, txVersion)
	}

	all := txVersion == 1
	_, err = blockchain.ReadExtensibleString(r, all, func(r io.Reader) error {
		_, err := to.OutputCommitment.ReadFrom(r)
		return err
	})
	if err != nil {
		return errors.Wrap(err, "reading output commitment")
	}

	to.ReferenceData, _, err = blockchain.ReadVarstr31(r)
	if err != nil {
		return errors.Wrap(err, "reading reference data")
	}

	// TODO(bobg): test that serialization flags include SerWitness, when we relax the serflags-must-be-0x7 rule
	_, err = blockchain.ReadExtensibleString(r, false, func(r io.Reader) error {
		return to.OutputCommitment.readWitness(r)
	})
	return err
}

func (to *TxOutput) writeTo(w io.Writer, serflags byte) error {
	_, err := blockchain.WriteVarint63(w, to.AssetVersion)
	if err != nil {
		return errors.Wrap(err, "writing asset version")
	}

	_, err = blockchain.WriteExtensibleString(w, func(w io.Writer) error {
		if to.AssetVersion == 1 {
			_, err := to.OutputCommitment.WriteTo(w)
			return err
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "writing output commitment")
	}

	err = writeRefData(w, to.ReferenceData, serflags)
	if err != nil {
		return errors.Wrap(err, "writing reference data")
	}

	if serflags&SerWitness != 0 {
		_, err = blockchain.WriteExtensibleString(w, func(w io.Writer) error {
			return to.OutputCommitment.writeWitness(w)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (to *TxOutput) WitnessHash() (hash Hash, err error) {
	hasher := sha3pool.Get256()
	defer sha3pool.Put256(hasher)

	err = to.OutputCommitment.writeWitness(hasher)
	if err != nil {
		return hash, err
	}

	hasher.Read(hash[:])
	return hash, err
}

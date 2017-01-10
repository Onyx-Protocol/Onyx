package bc

import (
	"io"

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

// assumes r has sticky errors
func (to *TxOutput) readFrom(r io.Reader, txVersion uint64) (err error) {
	to.AssetVersion, _, err = blockchain.ReadVarint63(r)
	if err != nil {
		return errors.Wrap(err, "reading asset version")
	}

	_, err = to.OutputCommitment.readFrom(r, txVersion, to.AssetVersion)
	if err != nil {
		return errors.Wrap(err, "reading output commitment")
	}

	to.ReferenceData, _, err = blockchain.ReadVarstr31(r)
	if err != nil {
		return errors.Wrap(err, "reading reference data")
	}

	// read and ignore the (empty) output witness
	_, _, err = blockchain.ReadVarstr31(r)

	return errors.Wrap(err, "reading output witness")
}

// assumes r has sticky errors
func (to *TxOutput) writeTo(w io.Writer, serflags byte) {
	blockchain.WriteVarint63(w, to.AssetVersion) // TODO(bobg): check and return error
	to.OutputCommitment.writeTo(w, to.AssetVersion)
	writeRefData(w, to.ReferenceData, serflags)
	blockchain.WriteVarstr31(w, nil)
}

func (to *TxOutput) witnessHash() Hash {
	return EmptyStringHash
}

func (to *TxOutput) WriteCommitment(w io.Writer) {
	to.OutputCommitment.writeTo(w, to.AssetVersion)
}

func (to *TxOutput) CommitmentHash() Hash {
	return to.OutputCommitment.Hash(to.AssetVersion)
}

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

func (to *TxOutput) writeTo(w io.Writer, serflags byte) error {
	_, err := blockchain.WriteVarint63(w, to.AssetVersion)
	if err != nil {
		return errors.Wrap(err, "writing asset version")
	}

	err = to.OutputCommitment.writeTo(w, to.AssetVersion)
	if err != nil {
		return errors.Wrap(err, "writing output commitment")
	}

	err = writeRefData(w, to.ReferenceData, serflags)
	if err != nil {
		return errors.Wrap(err, "writing reference data")
	}

	// write witness (empty in v1)
	_, err = blockchain.WriteVarstr31(w, nil)
	if err != nil {
		return errors.Wrap(err, "writing witness")
	}
	return nil
}

func (to *TxOutput) witnessHash() Hash {
	return EmptyStringHash
}

func (to *TxOutput) WriteCommitment(w io.Writer) {
	to.OutputCommitment.writeTo(w, to.AssetVersion)
}

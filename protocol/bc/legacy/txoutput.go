package legacy

import (
	"io"

	"chain/encoding/blockchain"
	"chain/errors"
	"chain/protocol/bc"
)

// TODO(bobg): Review serialization/deserialization logic for
// assetVersions other than 1.

type TxOutput struct {
	AssetVersion uint64
	OutputCommitment

	// Unconsumed suffixes of the commitment and witness extensible strings.
	CommitmentSuffix []byte
	WitnessSuffix    []byte

	ReferenceData []byte
}

func NewTxOutput(assetID bc.AssetID, amount uint64, controlProgram, referenceData []byte) *TxOutput {
	return &TxOutput{
		AssetVersion: 1,
		OutputCommitment: OutputCommitment{
			AssetAmount: bc.AssetAmount{
				AssetId: &assetID,
				Amount:  amount,
			},
			VMVersion:      1,
			ControlProgram: controlProgram,
		},
		ReferenceData: referenceData,
	}
}

func (to *TxOutput) readFrom(r blockchain.Reader, txVersion uint64) (err error) {
	to.AssetVersion, err = blockchain.ReadVarint63(r)
	if err != nil {
		return errors.Wrap(err, "reading asset version")
	}

	to.CommitmentSuffix, err = to.OutputCommitment.readFrom(r, to.AssetVersion)
	if err != nil {
		return errors.Wrap(err, "reading output commitment")
	}

	to.ReferenceData, err = blockchain.ReadVarstr31(r)
	if err != nil {
		return errors.Wrap(err, "reading reference data")
	}

	// read and ignore the (empty) output witness
	_, err = blockchain.ReadVarstr31(r)

	return errors.Wrap(err, "reading output witness")
}

func (to *TxOutput) writeTo(w io.Writer, serflags byte) error {
	_, err := blockchain.WriteVarint63(w, to.AssetVersion)
	if err != nil {
		return errors.Wrap(err, "writing asset version")
	}

	err = to.WriteCommitment(w)
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

func (to *TxOutput) WriteCommitment(w io.Writer) error {
	return to.OutputCommitment.writeExtensibleString(w, to.CommitmentSuffix, to.AssetVersion)
}

func (to *TxOutput) CommitmentHash() bc.Hash {
	return to.OutputCommitment.Hash(to.CommitmentSuffix, to.AssetVersion)
}

// ComputeOutputID assembles an output entry given a spend commitment
// and computes and returns its corresponding entry ID.
func ComputeOutputID(sc *SpendCommitment) (h bc.Hash, err error) {
	defer func() {
		if r, ok := recover().(error); ok {
			err = r
		}
	}()
	src := &bc.ValueSource{
		Ref:      &sc.SourceID,
		Value:    &sc.AssetAmount,
		Position: sc.SourcePosition,
	}
	o := bc.NewOutput(src, &bc.Program{VmVersion: sc.VMVersion, Code: sc.ControlProgram}, &sc.RefDataHash, 0)

	h = bc.EntryID(o)
	return h, nil
}

package bc

import (
	"bytes"
	"io"

	"chain/encoding/blockchain"
)

// TODO(bobg): Review serialization/deserialization logic for
// assetVersions other than 1.

type (
	TxOutput struct {
		AssetVersion uint64
		OutputCommitment
		ReferenceData []byte
	}

	OutputCommitment struct {
		AssetAmount
		VMVersion      uint64
		ControlProgram []byte
	}
)

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
func (to *TxOutput) readFrom(r io.Reader) (err error) {
	to.AssetVersion, _, err = blockchain.ReadVarint63(r)
	if err != nil {
		return err
	}

	err = to.OutputCommitment.readFrom(r, to.AssetVersion)
	if err != nil {
		return err
	}

	to.ReferenceData, _, err = blockchain.ReadVarstr31(r)
	if err != nil {
		return err
	}

	// read and ignore the (empty) output witness
	_, _, err = blockchain.ReadVarstr31(r)

	return err
}

func (oc *OutputCommitment) readFrom(r io.Reader, assetVersion uint64) (err error) {
	b, _, err := blockchain.ReadVarstr31(r)
	if err != nil {
		return err
	}

	if assetVersion != 1 {
		return nil
	}

	rb := bytes.NewBuffer(b)
	oc.AssetAmount.readFrom(rb)

	oc.VMVersion, _, err = blockchain.ReadVarint63(rb)
	if err != nil {
		return err
	}

	oc.ControlProgram, _, err = blockchain.ReadVarstr31(rb)
	return err
}

// assumes r has sticky errors
func (to *TxOutput) writeTo(w io.Writer, serflags byte) {
	blockchain.WriteVarint63(w, to.AssetVersion) // TODO(bobg): check and return error
	to.OutputCommitment.writeTo(w, to.AssetVersion)
	writeRefData(w, to.ReferenceData, serflags)
	blockchain.WriteVarstr31(w, nil)
}

func (to TxOutput) WitnessHash() Hash {
	return emptyHash
}

func (to TxOutput) Commitment() []byte {
	var buf bytes.Buffer
	to.OutputCommitment.writeTo(&buf, to.AssetVersion)
	return buf.Bytes()
}

func (oc OutputCommitment) writeTo(w io.Writer, assetVersion uint64) {
	b := new(bytes.Buffer)
	if assetVersion == 1 {
		oc.AssetAmount.writeTo(b)
		blockchain.WriteVarint63(b, oc.VMVersion) // TODO(bobg): check and return error
		blockchain.WriteVarstr31(b, oc.ControlProgram)
	}
	blockchain.WriteVarstr31(w, b.Bytes()) // TODO(bobg): check and return error
}

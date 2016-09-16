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
		AssetVersion uint32
		OutputCommitment
		ReferenceData []byte
	}

	OutputCommitment struct {
		AssetAmount
		VMVersion      uint32
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
	assetVersion, _ := blockchain.ReadUvarint(r)
	to.AssetVersion = uint32(assetVersion)
	err = to.OutputCommitment.readFrom(r, to.AssetVersion)
	if err != nil {
		return err
	}
	to.ReferenceData, err = blockchain.ReadBytes(r, refDataMaxByteLength)
	if err != nil {
		return err
	}
	// read and ignore the (empty) output witness
	_, err = blockchain.ReadBytes(r, commitmentMaxByteLength) // TODO(bobg): What's the right limit here?
	if err != nil {
		return err
	}
	return nil
}

func (oc *OutputCommitment) readFrom(r io.Reader, assetVersion uint32) (err error) {
	b, err := blockchain.ReadBytes(r, commitmentMaxByteLength) // TODO(bobg): Is this the right limit here?
	if err != nil {
		return err
	}
	if assetVersion != 1 {
		return nil
	}
	rb := bytes.NewBuffer(b)
	oc.AssetAmount.readFrom(rb)
	vmVersion, _ := blockchain.ReadUvarint(rb)
	oc.VMVersion = uint32(vmVersion)
	oc.ControlProgram, err = blockchain.ReadBytes(rb, MaxProgramByteLength)
	if err != nil {
		return err
	}
	return nil
}

// assumes r has sticky errors
func (to *TxOutput) writeTo(w io.Writer, serflags byte) {
	blockchain.WriteUvarint(w, uint64(to.AssetVersion))
	to.OutputCommitment.writeTo(w, to.AssetVersion)
	writeRefData(w, to.ReferenceData, serflags)
	blockchain.WriteBytes(w, nil) // empty output witness
}

func (to TxOutput) WitnessHash() Hash {
	return emptyHash
}

func (oc OutputCommitment) writeTo(w io.Writer, assetVersion uint32) {
	b := new(bytes.Buffer)
	if assetVersion == 1 {
		oc.AssetAmount.writeTo(b)
		blockchain.WriteUvarint(b, uint64(oc.VMVersion))
		blockchain.WriteBytes(b, oc.ControlProgram)
	}
	blockchain.WriteBytes(w, b.Bytes())
}

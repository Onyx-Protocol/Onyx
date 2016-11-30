package bc

import (
	"bytes"
	"fmt"
	"io"

	"chain/encoding/blockchain"
	"chain/encoding/bufpool"
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
func (to *TxOutput) readFrom(r io.Reader, txVersion uint64) (err error) {
	to.AssetVersion, _, err = blockchain.ReadVarint63(r)
	if err != nil {
		return err
	}

	_, err = to.OutputCommitment.readFrom(r, txVersion, to.AssetVersion)
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

func (oc *OutputCommitment) readFrom(r io.Reader, txVersion, assetVersion uint64) (n int, err error) {
	b, n, err := blockchain.ReadVarstr31(r)
	if err != nil {
		return n, err
	}

	if assetVersion != 1 {
		return n, nil
	}

	rb := bytes.NewBuffer(b)
	n1, err := oc.AssetAmount.readFrom(rb)
	if err != nil {
		return n, err
	}
	var n2 int
	oc.VMVersion, n2, err = blockchain.ReadVarint63(rb)
	if err != nil {
		return n, err
	}
	var n3 int
	oc.ControlProgram, n3, err = blockchain.ReadVarstr31(rb)
	if err != nil {
		return n, err
	}

	if txVersion == 1 && n1+n2+n3 < len(b) {
		return n, fmt.Errorf("unrecognized extra data in output commitment for transaction version 1")
	}

	return n, nil
}

// assumes r has sticky errors
func (to *TxOutput) writeTo(w io.Writer, serflags byte) {
	blockchain.WriteVarint63(w, to.AssetVersion) // TODO(bobg): check and return error
	to.OutputCommitment.writeTo(w, to.AssetVersion)
	writeRefData(w, to.ReferenceData, serflags)
	blockchain.WriteVarstr31(w, nil)
}

func (to *TxOutput) witnessHash() Hash {
	return emptyHash
}

func (to *TxOutput) WriteCommitment(w io.Writer) {
	to.OutputCommitment.writeTo(w, to.AssetVersion)
}

func (oc *OutputCommitment) writeTo(w io.Writer, assetVersion uint64) {
	b := bufpool.Get()
	defer bufpool.Put(b)
	if assetVersion == 1 {
		oc.AssetAmount.writeTo(b)
		blockchain.WriteVarint63(b, oc.VMVersion) // TODO(bobg): check and return error
		blockchain.WriteVarstr31(b, oc.ControlProgram)
	}
	blockchain.WriteVarstr31(w, b.Bytes()) // TODO(bobg): check and return error
}

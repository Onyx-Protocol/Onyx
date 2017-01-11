package bc

import (
	"fmt"
	"io"

	"chain/encoding/blockchain"
	"chain/encoding/bufpool"
	"chain/errors"
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
	if assetVersion != 1 {
		return n, fmt.Errorf("unrecognized asset version %d", assetVersion)
	}
	all := txVersion == 1
	return blockchain.ReadExtensibleString(r, all, func(r io.Reader) error {
		_, err := oc.AssetAmount.readFrom(r)
		if err != nil {
			return errors.Wrap(err, "reading asset+amount")
		}

		oc.VMVersion, _, err = blockchain.ReadVarint63(r)
		if err != nil {
			return errors.Wrap(err, "reading VM version")
		}

		if oc.VMVersion != 1 {
			return fmt.Errorf("unrecognized VM version %d for asset version 1", oc.VMVersion)
		}

		oc.ControlProgram, _, err = blockchain.ReadVarstr31(r)
		return errors.Wrap(err, "reading control program")
	})
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

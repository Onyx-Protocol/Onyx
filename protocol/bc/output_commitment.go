package bc

import (
	"fmt"
	"io"

	"chain/encoding/blockchain"
	"chain/errors"
)

// OutputCommitment contains the commitment data for a transaction
// output (which also appears in the spend input of that output).
type OutputCommitment struct {
	AssetAmount
	VMVersion      uint64
	ControlProgram []byte
}

func (oc *OutputCommitment) writeExtensibleString(w io.Writer, assetVersion uint64) error {
	_, err := blockchain.WriteExtensibleString(w, func(w io.Writer) error {
		return oc.writeContents(w, assetVersion)
	})
	return err
}

func (oc *OutputCommitment) writeContents(w io.Writer, assetVersion uint64) error {
	if assetVersion == 1 {
		err := oc.AssetAmount.writeTo(w)
		if err != nil {
			return errors.Wrap(err, "writing asset amount")
		}
		_, err = blockchain.WriteVarint63(w, oc.VMVersion)
		if err != nil {
			return errors.Wrap(err, "writing vm version")
		}
		_, err = blockchain.WriteVarstr31(w, oc.ControlProgram)
		if err != nil {
			return errors.Wrap(err, "writing control program")
		}
	}
	return nil
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

package bc

import (
	"fmt"
	"io"

	"chain/encoding/blockchain"
	"chain/errors"
)

type OutputCommitment struct {
	AssetAmount
	VMVersion      uint64
	ControlProgram []byte
}

func (oc *OutputCommitment) WriteTo(w io.Writer) error {
	err := oc.AssetAmount.writeTo(w)
	if err != nil {
		return err
	}
	_, err = blockchain.WriteVarint63(w, oc.VMVersion)
	if err != nil {
		return err
	}
	_, err = blockchain.WriteVarstr31(w, oc.ControlProgram)
	return err
}

// does not write the enclosing extensible string
func (oc *OutputCommitment) writeWitness(w io.Writer) error {
	return nil
}

func (oc *OutputCommitment) ReadFrom(r io.Reader) error {
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
}

// does not read the enclosing extensible string
func (oc *OutputCommitment) readWitness(r io.Reader) error {
	return nil
}

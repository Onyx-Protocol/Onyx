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

func (oc *OutputCommitment) WriteTo(w io.Writer) (int64, error) {
	n, err := oc.AssetAmount.writeTo(w)
	if err != nil {
		return n, err
	}
	n2, err := blockchain.WriteVarint63(w, oc.VMVersion)
	n += int64(n2)
	if err != nil {
		return n, err
	}
	n2, err = blockchain.WriteVarstr31(w, oc.ControlProgram)
	n += int64(n2)
	return n, err
}

// does not write the enclosing extensible string
func (oc *OutputCommitment) writeWitness(w io.Writer) error {
	return nil
}

func (oc *OutputCommitment) ReadFrom(r io.Reader) (int64, error) {
	n, err := oc.AssetAmount.readFrom(r)
	if err != nil {
		return int64(n), errors.Wrap(err, "reading asset+amount")
	}

	var n2 int
	oc.VMVersion, n2, err = blockchain.ReadVarint63(r)
	n += n2
	if err != nil {
		return int64(n), errors.Wrap(err, "reading VM version")
	}

	if oc.VMVersion != 1 {
		return int64(n), fmt.Errorf("unrecognized VM version %d for asset version 1", oc.VMVersion)
	}

	oc.ControlProgram, n2, err = blockchain.ReadVarstr31(r)
	n += n2
	return int64(n), errors.Wrap(err, "reading control program")
}

// does not read the enclosing extensible string
func (oc *OutputCommitment) readWitness(r io.Reader) error {
	return nil
}

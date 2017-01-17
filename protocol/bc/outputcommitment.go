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

func (oc *OutputCommitment) WriteTo(w io.Writer) error {
	err := oc.AssetAmount.writeTo(w)
	if err != nil {
		return errors.Wrap(err, "writing asset amount")
	}

	_, err = blockchain.WriteVarint63(w, oc.VMVersion)
	if err != nil {
		return errors.Wrap(err, "writing vm version")
	}

	_, err = blockchain.WriteVarstr31(w, oc.ControlProgram)
	return errors.Wrap(err, "writing control program")
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

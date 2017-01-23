package bc

import (
	"fmt"
	"io"

	"chain/crypto/sha3pool"
	"chain/encoding/blockchain"
	"chain/encoding/bufpool"
	"chain/errors"
)

// OutputCommitment contains the commitment data for a transaction
// output (which also appears in the spend input of that output).
type OutputCommitment struct {
	AssetAmount
	VMVersion      uint64
	ControlProgram []byte
}

// TODO(oleg): fix this implementation to respect all raw bytes in the OutputCommitment, irrespective of asset version.
func (oc *OutputCommitment) writeTo(w io.Writer, assetVersion uint64) (err error) {
	b := bufpool.Get()
	defer bufpool.Put(b)
	if assetVersion == 1 {
		err = oc.AssetAmount.writeTo(b)
		if err != nil {
			return errors.Wrap(err, "writing asset amount")
		}

		_, err = blockchain.WriteVarint63(b, oc.VMVersion)
		if err != nil {
			return errors.Wrap(err, "writing vm version")
		}
		_, err = blockchain.WriteVarstr31(b, oc.ControlProgram)
		if err != nil {
			return err
		}
	}

	_, err = blockchain.WriteVarstr31(w, b.Bytes())
	return errors.Wrap(err, "writing control program")
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

func (oc *OutputCommitment) Hash(assetVersion uint64) (outputhash Hash) {
	h := sha3pool.Get256()
	defer sha3pool.Put256(h)
	oc.writeTo(h, assetVersion) // TODO(oleg): get rid of this assetVersion parameter to actually write all the bytes
	h.Read(outputhash[:])
	return outputhash
}

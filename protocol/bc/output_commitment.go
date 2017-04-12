package bc

import (
	"fmt"
	"io"

	"chain/crypto/sha3pool"
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

func (oc *OutputCommitment) writeExtensibleString(w io.Writer, suffix []byte, assetVersion uint64) error {
	_, err := blockchain.WriteExtensibleString(w, suffix, func(w io.Writer) error {
		return oc.writeContents(w, suffix, assetVersion)
	})
	return err
}

func (oc *OutputCommitment) writeContents(w io.Writer, suffix []byte, assetVersion uint64) (err error) {
	if assetVersion == 1 {
		_, err = oc.AssetAmount.WriteTo(w)
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
	if len(suffix) > 0 {
		_, err = w.Write(suffix)
		if err != nil {
			return errors.Wrap(err, "writing suffix")
		}
	}
	return nil
}

func (oc *OutputCommitment) readFrom(r io.Reader, assetVersion uint64) (suffix []byte, n int, err error) {
	return blockchain.ReadExtensibleString(r, func(r io.Reader) error {
		if assetVersion == 1 {
			_, err := oc.AssetAmount.ReadFrom(r)
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
		return nil
	})
}

func (oc *OutputCommitment) Hash(suffix []byte, assetVersion uint64) (outputhash Hash) {
	h := sha3pool.Get256()
	defer sha3pool.Put256(h)
	oc.writeExtensibleString(h, suffix, assetVersion) // TODO(oleg): get rid of this assetVersion parameter to actually write all the bytes
	outputhash.ReadFrom(h)
	return outputhash
}

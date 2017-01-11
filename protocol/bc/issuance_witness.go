package bc

import (
	"fmt"
	"io"

	"chain-stealth/encoding/blockchain"
	"chain-stealth/errors"
)

type IssuanceWitness struct {
	InitialBlock    Hash
	VMVersion       uint64
	IssuanceProgram []byte
	Arguments       [][]byte
}

func (aw *IssuanceWitness) AssetID(assetVersion uint64) AssetID {
	return ComputeAssetID(aw.IssuanceProgram, aw.InitialBlock, assetVersion, aw.VMVersion)
}

func (aw *IssuanceWitness) writeTo(w io.Writer) error {
	_, err := w.Write(aw.InitialBlock[:])
	if err != nil {
		return err
	}
	_, err = blockchain.WriteVarint63(w, aw.VMVersion)
	if err != nil {
		return err
	}
	_, err = blockchain.WriteVarstr31(w, aw.IssuanceProgram)
	if err != nil {
		return err
	}
	_, err = blockchain.WriteVarstrList(w, aw.Arguments)
	return err
}

func (aw *IssuanceWitness) readFrom(r io.Reader, assetVersion uint64) error {
	_, err := io.ReadFull(r, aw.InitialBlock[:])
	if err != nil {
		return errors.Wrap(err, "reading initial block hash")
	}
	aw.VMVersion, _, err = blockchain.ReadVarint63(r)
	if err != nil {
		return errors.Wrap(err, "reading VM version")
	}
	if (assetVersion == 1 || assetVersion == 2) && aw.VMVersion != 1 {
		return fmt.Errorf("unrecognized VM version %d for asset version %d", aw.VMVersion, assetVersion)
	}
	aw.IssuanceProgram, _, err = blockchain.ReadVarstr31(r)
	if err != nil {
		return errors.Wrap(err, "reading issuance program")
	}
	aw.Arguments, _, err = blockchain.ReadVarstrList(r)
	return errors.Wrap(err, "reading arguments")
}

package bc

import (
	"fmt"
	"io"

	"chain/crypto/sha3pool"
	"chain/encoding/blockchain"
	"chain/errors"
)

type IssuanceWitness struct {
	InitialBlock    Hash
	AssetDefinition []byte
	VMVersion       uint64
	IssuanceProgram []byte
	Arguments       [][]byte
}

func (iw *IssuanceWitness) AssetID() AssetID {
	return ComputeAssetID(iw.IssuanceProgram, iw.InitialBlock, iw.VMVersion, iw.AssetDefinitionHash())
}

func (iw *IssuanceWitness) AssetDefinitionHash() (defhash Hash) {
	if len(iw.AssetDefinition) == 0 {
		return EmptyStringHash
	}
	sha := sha3pool.Get256()
	defer sha3pool.Put256(sha)
	sha.Write(iw.AssetDefinition)
	sha.Read(defhash[:])
	return
}

func (iw *IssuanceWitness) writeTo(w io.Writer) error {
	_, err := w.Write(iw.InitialBlock[:])
	if err != nil {
		return err
	}
	_, err = blockchain.WriteVarstr31(w, iw.AssetDefinition)
	if err != nil {
		return err
	}
	_, err = blockchain.WriteVarint63(w, iw.VMVersion)
	if err != nil {
		return err
	}
	_, err = blockchain.WriteVarstr31(w, iw.IssuanceProgram)
	if err != nil {
		return err
	}
	_, err = blockchain.WriteVarstrList(w, iw.Arguments)
	return err
}

func (iw *IssuanceWitness) readFrom(r io.Reader, assetVersion uint64) error {
	_, err := io.ReadFull(r, iw.InitialBlock[:])
	if err != nil {
		return errors.Wrap(err, "reading initial block hash")
	}
	iw.AssetDefinition, _, err = blockchain.ReadVarstr31(r)
	if err != nil {
		return errors.Wrap(err, "reading asset definition")
	}
	iw.VMVersion, _, err = blockchain.ReadVarint63(r)
	if err != nil {
		return errors.Wrap(err, "reading VM version")
	}
	if (assetVersion == 1 || assetVersion == 2) && iw.VMVersion != 1 {
		return fmt.Errorf("unrecognized VM version %d for asset version %d", iw.VMVersion, assetVersion)
	}
	iw.IssuanceProgram, _, err = blockchain.ReadVarstr31(r)
	if err != nil {
		return errors.Wrap(err, "reading issuance program")
	}
	iw.Arguments, _, err = blockchain.ReadVarstrList(r)
	return errors.Wrap(err, "reading arguments")
}

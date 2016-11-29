package bc

import (
	"fmt"
	"io"

	"chain-stealth/crypto/ca"
	"chain-stealth/encoding/blockchain"
	"chain-stealth/errors"
)

type Outputv1 struct {
	// commitment
	AssetAmount    AssetAmount
	VMVersion      uint64
	ControlProgram []byte
}

func (oc *Outputv1) WriteTo(w io.Writer) error {
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
func (oc *Outputv1) writeWitness(w io.Writer) error {
	return nil
}

func (oc *Outputv1) ReadFrom(r io.Reader) error {
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
func (oc *Outputv1) readWitness(r io.Reader) error {
	return nil
}

func (oc *Outputv1) GetAssetAmount() (AssetAmount, bool) {
	return oc.AssetAmount, true
}

func (oc *Outputv1) Amount() (uint64, bool) {
	return oc.AssetAmount.Amount, true
}

func (oc *Outputv1) AssetID() (AssetID, bool) {
	return oc.AssetAmount.AssetID, true
}

func (oc *Outputv1) AssetDescriptor() ca.AssetDescriptor {
	return nil
}

func (oc *Outputv1) ValueDescriptor() ca.ValueDescriptor {
	return nil

}

func (oc *Outputv1) VMVer() uint64 { return oc.VMVersion }

func (oc *Outputv1) Program() []byte { return oc.ControlProgram }

package bc

import (
	"bytes"
	"io"

	"chain-stealth/crypto/ca"
	"chain-stealth/encoding/blockchain"
	"chain-stealth/errors"
)

type Outputv2 struct {
	// commitment
	assetDescriptor ca.AssetDescriptor
	valueDescriptor ca.ValueDescriptor
	VMVersion       uint64
	ControlProgram  []byte

	// witness
	assetRangeProof *ca.AssetRangeProof
	valueRangeProof *ca.ValueRangeProof
}

func (oc *Outputv2) WriteTo(w io.Writer) error {
	err := oc.assetDescriptor.WriteTo(w)
	if err != nil {
		return err
	}
	err = oc.valueDescriptor.WriteTo(w)
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
func (o *Outputv2) writeWitness(w io.Writer) error {
	_, err := blockchain.WriteExtensibleString(w, func(w io.Writer) error {
		if o.assetRangeProof != nil {
			return o.assetRangeProof.WriteTo(w)
		}
		return nil
	})
	if err != nil {
		return err
	}

	_, err = blockchain.WriteExtensibleString(w, func(w io.Writer) error {
		if o.valueRangeProof != nil {
			return o.valueRangeProof.WriteTo(w)
		}
		return nil
	})

	return err
}

func (oc *Outputv2) ReadFrom(r io.Reader) (err error) {
	oc.assetDescriptor, err = ca.ReadAssetDescriptor(r)
	if err != nil {
		return errors.Wrap(err, "reading asset descriptor")
	}
	oc.valueDescriptor, err = ca.ReadValueDescriptor(r, oc.assetDescriptor)
	if err != nil {
		return errors.Wrap(err, "reading value descriptor")
	}
	oc.VMVersion, _, err = blockchain.ReadVarint63(r)
	if err != nil {
		return errors.Wrap(err, "reading VM version")
	}
	oc.ControlProgram, _, err = blockchain.ReadVarstr31(r)
	return errors.Wrap(err, "reading control program")
}

// does not read the enclosing extensible string
func (o *Outputv2) readWitness(r io.Reader) error {
	s, _, err := blockchain.ReadVarstr31(r)
	if err != nil {
		return errors.Wrap(err, "reading asset range proof optional string")
	}
	if len(s) > 0 {
		o.assetRangeProof = new(ca.AssetRangeProof)
		err = o.assetRangeProof.ReadFrom(bytes.NewReader(s))
		if err != nil {
			return errors.Wrap(err, "parsing asset range proof")
		}
	}

	s, _, err = blockchain.ReadVarstr31(r)
	if err != nil {
		return errors.Wrap(err, "reading value range proof optional string")
	}
	if len(s) > 0 {
		o.valueRangeProof = new(ca.ValueRangeProof)
		err = o.valueRangeProof.ReadFrom(bytes.NewReader(s))
		if err != nil {
			return errors.Wrap(err, "parsing value range proof")
		}
	}

	return nil
}

func (oc *Outputv2) GetAssetAmount() (aa AssetAmount, ok bool) {
	aa.Amount, ok = oc.Amount()
	if !ok {
		return aa, false
	}
	aa.AssetID, ok = oc.AssetID()
	return aa, ok
}

func (oc *Outputv2) Amount() (uint64, bool) {
	if bl, ok := oc.valueDescriptor.(*ca.NonblindedValueDescriptor); ok {
		return bl.Value, true
	}
	return 0, false
}

func (oc *Outputv2) AssetID() (assetID AssetID, ok bool) {
	if bl, ok := oc.assetDescriptor.(*ca.NonblindedAssetDescriptor); ok {
		return AssetID(bl.AssetID), true
	}
	return assetID, false
}

func (oc *Outputv2) VMVer() uint64 { return oc.VMVersion }

func (oc *Outputv2) Program() []byte { return oc.ControlProgram }

func (o *Outputv2) AssetDescriptor() ca.AssetDescriptor {
	return o.assetDescriptor
}

func (o *Outputv2) ValueDescriptor() ca.ValueDescriptor {
	return o.valueDescriptor
}

func (o *Outputv2) AssetRangeProof() *ca.AssetRangeProof {
	return o.assetRangeProof
}

func (o *Outputv2) ValueRangeProof() *ca.ValueRangeProof {
	return o.valueRangeProof
}

package ca

import (
	"fmt"
	"io"

	"chain-stealth/encoding/blockchain"
)

type ValueDescriptor interface {
	Commitment() ValueCommitment
	EncryptedValue() *EncryptedValue
	IsBlinded() bool
	WriteTo(w io.Writer) error
}

// TODO: handle descriptor type: 0 = nonblinded, 1 = blinded only, 3 = blinded+encrypted
// TODO: Serialize()
// TODO: Deserialize()

// Type=0x00 - unblinded
type NonblindedValueDescriptor struct {
	Value           uint64
	assetDescriptor AssetDescriptor
}

func (v *NonblindedValueDescriptor) Commitment() ValueCommitment {
	return CreateNonblindedValueCommitment(v.assetDescriptor.Commitment(), v.Value)
}

func (v *NonblindedValueDescriptor) EncryptedValue() *EncryptedValue {
	return nil
}

func (v *NonblindedValueDescriptor) IsBlinded() bool {
	return false
}

func (v *NonblindedValueDescriptor) WriteTo(w io.Writer) error {
	_, err := w.Write([]byte{0x00})
	if err != nil {
		return err
	}
	_, err = blockchain.WriteVarint63(w, v.Value)
	return err
}

// Type=0x01 - blinded, but not encrypted
// Type=0x03 - blinded and encrypted
type BlindedValueDescriptor struct {
	V    ValueCommitment
	evef *EncryptedValue // nil means not encrypted
}

func (v *BlindedValueDescriptor) Commitment() ValueCommitment {
	return v.V
}

func (v *BlindedValueDescriptor) EncryptedValue() *EncryptedValue {
	return v.evef
}

func (v *BlindedValueDescriptor) IsBlinded() bool {
	return true
}

func (v *BlindedValueDescriptor) WriteTo(w io.Writer) error {
	var t byte
	if v.evef == nil {
		t = 0x01
	} else {
		t = 0x03
	}
	_, err := w.Write([]byte{t})
	if err != nil {
		return err
	}
	_, err = w.Write(v.V.Bytes())
	if err != nil {
		return err
	}
	if v.evef != nil {
		err = v.evef.WriteTo(w)
		if err != nil {
			return err
		}
	}
	return nil
}

func ReadValueDescriptor(r io.Reader, ad AssetDescriptor) (ValueDescriptor, error) {
	var t [1]byte
	_, err := io.ReadFull(r, t[:])
	if err != nil {
		return nil, err
	}
	switch t[0] {
	case 0:
		amount, _, err := blockchain.ReadVarint63(r)
		return &NonblindedValueDescriptor{
			Value:           amount,
			assetDescriptor: ad,
		}, err
	case 1, 3:
		d := new(BlindedValueDescriptor)
		err := d.V.readFrom(r)
		if err != nil {
			return nil, err
		}
		if t[0] == 3 {
			d.evef = new(EncryptedValue)
			err = d.evef.readFrom(r)
			if err != nil {
				return nil, err
			}
		}
		return d, nil
	}
	return nil, fmt.Errorf("unrecognized value descriptor type %d", t[0])
}

func CreateBlindedValueDescriptor(V ValueCommitment, evef *EncryptedValue) *BlindedValueDescriptor {
	return &BlindedValueDescriptor{V: V, evef: evef}
}

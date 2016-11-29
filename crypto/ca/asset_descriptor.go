package ca

import (
	"fmt"
	"io"
)

type AssetDescriptor interface {
	Commitment() AssetCommitment
	IsBlinded() bool
	EncryptedAssetID() *EncryptedAssetID
	WriteTo(io.Writer) error
}

// Type=0x00 - unblinded
type NonblindedAssetDescriptor struct {
	AssetID
}

func (a *NonblindedAssetDescriptor) Commitment() AssetCommitment {
	return CreateNonblindedAssetCommitment(a.AssetID)
}

func (a *NonblindedAssetDescriptor) IsBlinded() bool {
	return false
}

func (a *NonblindedAssetDescriptor) EncryptedAssetID() *EncryptedAssetID {
	return nil
}

func (a *NonblindedAssetDescriptor) WriteTo(w io.Writer) error {
	_, err := w.Write([]byte{0x00})
	if err != nil {
		return err
	}
	_, err = w.Write(a.AssetID[:])
	return err
}

// Type=0x01 - blinded, but not encrypted
// Type=0x03 - blinded and encrypted
type BlindedAssetDescriptor struct {
	H    AssetCommitment
	eaec *EncryptedAssetID // nil means not encrypted
}

func (a *BlindedAssetDescriptor) Commitment() AssetCommitment {
	return a.H
}

func (a *BlindedAssetDescriptor) IsBlinded() bool {
	return true
}

func (a *BlindedAssetDescriptor) EncryptedAssetID() *EncryptedAssetID {
	return a.eaec
}

func (a *BlindedAssetDescriptor) WriteTo(w io.Writer) error {
	var t byte
	if a.eaec == nil {
		t = 0x01
	} else {
		t = 0x03
	}
	_, err := w.Write([]byte{t})
	if err != nil {
		return err
	}
	_, err = w.Write(a.H.Bytes())
	if err != nil {
		return err
	}
	if a.eaec != nil {
		err = a.eaec.WriteTo(w)
		if err != nil {
			return err
		}
	}
	return nil
}

func ReadAssetDescriptor(r io.Reader) (AssetDescriptor, error) {
	var t [1]byte
	_, err := io.ReadFull(r, t[:])
	if err != nil {
		return nil, err
	}
	switch t[0] {
	case 0:
		d := new(NonblindedAssetDescriptor)
		_, err = io.ReadFull(r, d.AssetID[:])
		if err != nil {
			return nil, err
		}
		return d, nil

	case 1, 3:
		d := new(BlindedAssetDescriptor)
		err = d.H.readFrom(r)
		if err != nil {
			return nil, err
		}
		if t[0] == 3 {
			d.eaec = new(EncryptedAssetID)
			err = d.eaec.readFrom(r)
			if err != nil {
				return nil, err
			}
		}
		return d, nil
	}
	return nil, fmt.Errorf("unrecognized asset descriptor type %d", t[0])
}

func CreateBlindedAssetDescriptor(H AssetCommitment, eaec *EncryptedAssetID) *BlindedAssetDescriptor {
	return &BlindedAssetDescriptor{H: H, eaec: eaec}
}

package legacy

import (
	"fmt"
	"io"

	"chain/encoding/blockchain"
	"chain/errors"
	"chain/protocol/bc"
)

type (
	TxInput struct {
		AssetVersion  uint64
		ReferenceData []byte
		TypedInput

		// Unconsumed suffixes of the commitment and witness extensible
		// strings.
		CommitmentSuffix []byte
		WitnessSuffix    []byte
	}

	TypedInput interface {
		IsIssuance() bool
	}
)

var errBadAssetID = errors.New("asset ID does not match other issuance parameters")

func (t *TxInput) AssetAmount() bc.AssetAmount {
	if ii, ok := t.TypedInput.(*IssuanceInput); ok {
		assetID := ii.AssetID()
		return bc.AssetAmount{
			AssetId: &assetID,
			Amount:  ii.Amount,
		}
	}
	si := t.TypedInput.(*SpendInput)
	return si.AssetAmount
}

func (t *TxInput) AssetID() bc.AssetID {
	if ii, ok := t.TypedInput.(*IssuanceInput); ok {
		return ii.AssetID()
	}
	si := t.TypedInput.(*SpendInput)
	return *si.AssetId
}

func (t *TxInput) Amount() uint64 {
	if ii, ok := t.TypedInput.(*IssuanceInput); ok {
		return ii.Amount
	}
	si := t.TypedInput.(*SpendInput)
	return si.Amount
}

func (t *TxInput) ControlProgram() []byte {
	if si, ok := t.TypedInput.(*SpendInput); ok {
		return si.ControlProgram
	}
	return nil
}

func (t *TxInput) IssuanceProgram() []byte {
	if ii, ok := t.TypedInput.(*IssuanceInput); ok {
		return ii.IssuanceProgram
	}
	return nil
}

func (t *TxInput) Arguments() [][]byte {
	switch inp := t.TypedInput.(type) {
	case *IssuanceInput:
		return inp.Arguments
	case *SpendInput:
		return inp.Arguments
	}
	return nil
}

func (t *TxInput) SetArguments(args [][]byte) {
	switch inp := t.TypedInput.(type) {
	case *IssuanceInput:
		inp.Arguments = args
	case *SpendInput:
		inp.Arguments = args
	}
}

func (t *TxInput) readFrom(r blockchain.Reader) (err error) {
	t.AssetVersion, err = blockchain.ReadVarint63(r)
	if err != nil {
		return err
	}

	var (
		ii      *IssuanceInput
		si      *SpendInput
		assetID bc.AssetID
	)

	t.CommitmentSuffix, err = blockchain.ReadExtensibleString(r, func(r blockchain.Reader) error {
		if t.AssetVersion != 1 {
			return nil
		}
		var icType [1]byte
		_, err = io.ReadFull(r, icType[:])
		if err != nil {
			return errors.Wrap(err, "reading input commitment type")
		}
		switch icType[0] {
		case 0:
			ii = new(IssuanceInput)

			ii.Nonce, err = blockchain.ReadVarstr31(r)
			if err != nil {
				return err
			}
			_, err = assetID.ReadFrom(r)
			if err != nil {
				return err
			}
			ii.Amount, err = blockchain.ReadVarint63(r)
			if err != nil {
				return err
			}

		case 1:
			si = new(SpendInput)
			si.SpendCommitmentSuffix, err = si.SpendCommitment.readFrom(r, 1)
			if err != nil {
				return err
			}

		default:
			return fmt.Errorf("unsupported input type %d", icType[0])
		}
		return nil
	})
	if err != nil {
		return err
	}

	t.ReferenceData, err = blockchain.ReadVarstr31(r)
	if err != nil {
		return err
	}

	t.WitnessSuffix, err = blockchain.ReadExtensibleString(r, func(r blockchain.Reader) error {
		// TODO(bobg): test that serialization flags include SerWitness, when we relax the serflags-must-be-0x7 rule
		if t.AssetVersion != 1 {
			return nil
		}

		if ii != nil {
			// read IssuanceInput witness
			_, err = ii.InitialBlock.ReadFrom(r)
			if err != nil {
				return err
			}

			ii.AssetDefinition, err = blockchain.ReadVarstr31(r)
			if err != nil {
				return err
			}

			ii.VMVersion, err = blockchain.ReadVarint63(r)
			if err != nil {
				return err
			}

			ii.IssuanceProgram, err = blockchain.ReadVarstr31(r)
			if err != nil {
				return err
			}

			if ii.AssetID() != assetID {
				return errBadAssetID
			}
		}
		args, err := blockchain.ReadVarstrList(r)
		if err != nil {
			return err
		}
		if ii != nil {
			ii.Arguments = args
		} else if si != nil {
			si.Arguments = args
		}
		return nil
	})
	if err != nil {
		return err
	}
	if ii != nil {
		t.TypedInput = ii
	} else if si != nil {
		t.TypedInput = si
	}
	return nil
}

func (t *TxInput) writeTo(w io.Writer, serflags uint8) error {
	_, err := blockchain.WriteVarint63(w, t.AssetVersion)
	if err != nil {
		return errors.Wrap(err, "writing asset version")
	}

	_, err = blockchain.WriteExtensibleString(w, t.CommitmentSuffix, func(w io.Writer) error {
		return t.WriteInputCommitment(w, serflags)
	})

	if err != nil {
		return errors.Wrap(err, "writing input commitment")
	}

	_, err = blockchain.WriteVarstr31(w, t.ReferenceData)
	if err != nil {
		return errors.Wrap(err, "writing reference data")
	}

	if serflags&SerWitness != 0 {
		_, err = blockchain.WriteExtensibleString(w, t.WitnessSuffix, t.writeInputWitness)
		if err != nil {
			return errors.Wrap(err, "writing input witness")
		}
	}

	return nil
}

func (t *TxInput) WriteInputCommitment(w io.Writer, serflags uint8) error {
	if t.AssetVersion != 1 {
		return nil
	}
	switch inp := t.TypedInput.(type) {
	case *IssuanceInput:
		_, err := w.Write([]byte{0}) // issuance type
		if err != nil {
			return err
		}
		_, err = blockchain.WriteVarstr31(w, inp.Nonce)
		if err != nil {
			return err
		}
		assetID := t.AssetID()
		_, err = assetID.WriteTo(w)
		if err != nil {
			return err
		}
		_, err = blockchain.WriteVarint63(w, inp.Amount)
		return err

	case *SpendInput:
		_, err := w.Write([]byte{1}) // spend type
		if err != nil {
			return err
		}
		if serflags&SerPrevout != 0 {
			err = inp.SpendCommitment.writeExtensibleString(w, inp.SpendCommitmentSuffix, t.AssetVersion)
		} else {
			prevouthash := inp.SpendCommitment.Hash(inp.SpendCommitmentSuffix, t.AssetVersion)
			_, err = prevouthash.WriteTo(w)
		}
		return err
	}
	return nil
}

func (t *TxInput) writeInputWitness(w io.Writer) error {
	if t.AssetVersion != 1 {
		return nil
	}
	switch inp := t.TypedInput.(type) {
	case *IssuanceInput:
		_, err := inp.InitialBlock.WriteTo(w)
		if err != nil {
			return err
		}
		_, err = blockchain.WriteVarstr31(w, inp.AssetDefinition)
		if err != nil {
			return err
		}
		_, err = blockchain.WriteVarint63(w, inp.VMVersion)
		if err != nil {
			return err
		}
		_, err = blockchain.WriteVarstr31(w, inp.IssuanceProgram)
		if err != nil {
			return err
		}
		_, err = blockchain.WriteVarstrList(w, inp.Arguments)
		return err

	case *SpendInput:
		_, err := blockchain.WriteVarstrList(w, inp.Arguments)
		return err
	}
	return nil
}

func (t *TxInput) SpentOutputID() (o bc.Hash, err error) {
	if si, ok := t.TypedInput.(*SpendInput); ok {
		o, err = ComputeOutputID(&si.SpendCommitment)
	}
	return o, err
}

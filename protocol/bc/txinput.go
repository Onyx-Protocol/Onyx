package bc

import (
	"fmt"
	"io"

	"chain/crypto/sha3pool"
	"chain/encoding/blockchain"
	"chain/errors"
)

type (
	TxInput struct {
		AssetVersion  uint64
		ReferenceData []byte
		TypedInput
	}

	TypedInput interface {
		IsIssuance() bool
	}

	SpendInput struct {
		// Commitment
		Outpoint
		OutputCommitment

		// Witness
		Arguments [][]byte
	}

	IssuanceInput struct {
		// Commitment
		Nonce  []byte
		Amount uint64
		// Note: as long as we require serflags=0x7, we don't need to
		// explicitly store the asset ID here even though it's technically
		// part of the input commitment. We can compute it instead from
		// values in the witness (which, with serflags other than 0x7,
		// might not be present).

		// Witness
		InitialBlock    Hash
		AssetDefinition []byte
		VMVersion       uint64
		IssuanceProgram []byte
		Arguments       [][]byte
	}
)

var errBadAssetID = errors.New("asset ID does not match other issuance parameters")

func NewSpendInput(txhash Hash, index uint32, arguments [][]byte, assetID AssetID, amount uint64, controlProgram, referenceData []byte) *TxInput {
	return &TxInput{
		AssetVersion:  1,
		ReferenceData: referenceData,
		TypedInput: &SpendInput{
			Outpoint: Outpoint{
				Hash:  txhash,
				Index: index,
			},
			OutputCommitment: OutputCommitment{
				AssetAmount: AssetAmount{
					AssetID: assetID,
					Amount:  amount,
				},
				VMVersion:      1,
				ControlProgram: controlProgram,
			},
			Arguments: arguments,
		},
	}
}

func NewIssuanceInput(
	nonce []byte,
	amount uint64,
	referenceData []byte,
	initialBlock Hash,
	issuanceProgram []byte,
	arguments [][]byte,
	assetDefinition []byte,
) *TxInput {
	return &TxInput{
		AssetVersion:  1,
		ReferenceData: referenceData,
		TypedInput: &IssuanceInput{
			Nonce:           nonce,
			Amount:          amount,
			InitialBlock:    initialBlock,
			AssetDefinition: assetDefinition,
			VMVersion:       1,
			IssuanceProgram: issuanceProgram,
			Arguments:       arguments,
		},
	}
}

func (t *TxInput) AssetAmount() AssetAmount {
	if ii, ok := t.TypedInput.(*IssuanceInput); ok {
		return AssetAmount{
			AssetID: ii.AssetID(),
			Amount:  ii.Amount,
		}
	}
	si := t.TypedInput.(*SpendInput)
	return si.AssetAmount
}

func (t *TxInput) AssetID() AssetID {
	if ii, ok := t.TypedInput.(*IssuanceInput); ok {
		return ii.AssetID()
	}
	si := t.TypedInput.(*SpendInput)
	return si.AssetID
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

func (t *TxInput) readFrom(r io.Reader, txVersion uint64) (err error) {
	t.AssetVersion, _, err = blockchain.ReadVarint63(r)
	if err != nil {
		return err
	}

	var (
		ii      *IssuanceInput
		si      *SpendInput
		assetID AssetID
	)

	all := txVersion == 1
	_, err = blockchain.ReadExtensibleString(r, all, func(r io.Reader) error {
		if t.AssetVersion == 1 {
			var icType [1]byte
			_, err = io.ReadFull(r, icType[:])
			if err != nil {
				return errors.Wrap(err, "reading input commitment type")
			}
			switch icType[0] {
			case 0:
				ii = new(IssuanceInput)

				ii.Nonce, _, err = blockchain.ReadVarstr31(r)
				if err != nil {
					return err
				}
				_, err = io.ReadFull(r, assetID[:])
				if err != nil {
					return err
				}
				ii.Amount, _, err = blockchain.ReadVarint63(r)
				if err != nil {
					return err
				}

			case 1:
				si = new(SpendInput)

				_, err = si.Outpoint.readFrom(r)
				if err != nil {
					return err
				}
				_, err = si.OutputCommitment.readFrom(r, txVersion, 1)
				if err != nil {
					return err
				}

			default:
				return fmt.Errorf("unsupported input type %d", icType[0])
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	t.ReferenceData, _, err = blockchain.ReadVarstr31(r)
	if err != nil {
		return err
	}

	_, err = blockchain.ReadExtensibleString(r, false, func(r io.Reader) error {
		// TODO(bobg): test that serialization flags include SerWitness, when we relax the serflags-must-be-0x7 rule
		if ii != nil {
			// read IssuanceInput witness
			_, err = io.ReadFull(r, ii.InitialBlock[:])
			if err != nil {
				return err
			}

			ii.AssetDefinition, _, err = blockchain.ReadVarstr31(r)
			if err != nil {
				return err
			}

			ii.VMVersion, _, err = blockchain.ReadVarint63(r)
			if err != nil {
				return err
			}

			ii.IssuanceProgram, _, err = blockchain.ReadVarstr31(r)
			if err != nil {
				return err
			}

			computedAssetID := ComputeAssetID(ii.IssuanceProgram, ii.InitialBlock, ii.VMVersion, ii.AssetDefinitionHash())
			if computedAssetID != assetID {
				return errBadAssetID
			}
		}
		args, _, err := blockchain.ReadVarstrList(r)
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

// assumes w has sticky errors
func (t *TxInput) writeTo(w io.Writer, serflags uint8) {
	blockchain.WriteVarint63(w, t.AssetVersion) // TODO(bobg): check and return error
	blockchain.WriteExtensibleString(w, t.WriteInputCommitment)
	blockchain.WriteVarstr31(w, t.ReferenceData)
	if serflags&SerWitness != 0 {
		blockchain.WriteExtensibleString(w, t.writeInputWitness)
	}
}

func (t *TxInput) WriteInputCommitment(w io.Writer) error {
	if t.AssetVersion == 1 {
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
			_, err = w.Write(assetID[:])
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
			_, err = inp.Outpoint.WriteTo(w)
			if err != nil {
				return err
			}
			err = inp.OutputCommitment.writeTo(w, t.AssetVersion)
			return err
		}
	}
	return nil
}

func (t *TxInput) writeInputWitness(w io.Writer) error {
	if t.AssetVersion == 1 {
		switch inp := t.TypedInput.(type) {
		case *IssuanceInput:
			_, err := w.Write(inp.InitialBlock[:])
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
	}
	return nil
}

func (t *TxInput) witnessHash() Hash {
	var h Hash
	sha := sha3pool.Get256()
	defer sha3pool.Put256(sha)
	t.writeInputWitness(sha)
	sha.Read(h[:])
	return h
}

func (t *TxInput) Outpoint() (o Outpoint) {
	if si, ok := t.TypedInput.(*SpendInput); ok {
		o = si.Outpoint
	}
	return o
}

func (si *SpendInput) IsIssuance() bool { return false }

func (ii *IssuanceInput) IsIssuance() bool { return true }

func (ii *IssuanceInput) AssetID() AssetID {
	return ComputeAssetID(ii.IssuanceProgram, ii.InitialBlock, ii.VMVersion, ii.AssetDefinitionHash())
}

func (ii *IssuanceInput) AssetDefinitionHash() (defhash Hash) {
	sha := sha3pool.Get256()
	defer sha3pool.Put256(sha)
	sha.Write(ii.AssetDefinition)
	sha.Read(defhash[:])
	return
}

package bc

import (
	"bytes"
	"fmt"
	"io"

	"chain/crypto/sha3pool"
	"chain/encoding/blockchain"
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
		VMVersion       uint64
		IssuanceProgram []byte
		Arguments       [][]byte
	}
)

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

func NewIssuanceInput(nonce []byte, amount uint64, referenceData []byte, initialBlock Hash, issuanceProgram []byte, arguments [][]byte) *TxInput {
	return &TxInput{
		AssetVersion:  1,
		ReferenceData: referenceData,
		TypedInput: &IssuanceInput{
			Nonce:           nonce,
			Amount:          amount,
			InitialBlock:    initialBlock,
			VMVersion:       1,
			IssuanceProgram: issuanceProgram,
			Arguments:       arguments,
		},
	}
}

func (t TxInput) AssetAmount() AssetAmount {
	if ii, ok := t.TypedInput.(*IssuanceInput); ok {
		return AssetAmount{
			AssetID: ii.AssetID(),
			Amount:  ii.Amount,
		}
	}
	si := t.TypedInput.(*SpendInput)
	return si.AssetAmount
}

func (t TxInput) AssetID() AssetID {
	if ii, ok := t.TypedInput.(*IssuanceInput); ok {
		return ii.AssetID()
	}
	si := t.TypedInput.(*SpendInput)
	return si.AssetID
}

func (t TxInput) Amount() uint64 {
	if ii, ok := t.TypedInput.(*IssuanceInput); ok {
		return ii.Amount
	}
	si := t.TypedInput.(*SpendInput)
	return si.Amount
}

func (t TxInput) ControlProgram() []byte {
	if si, ok := t.TypedInput.(*SpendInput); ok {
		return si.ControlProgram
	}
	return nil
}

func (t TxInput) IssuanceProgram() []byte {
	if ii, ok := t.TypedInput.(*IssuanceInput); ok {
		return ii.IssuanceProgram
	}
	return nil
}

func (t TxInput) Arguments() [][]byte {
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

	inputCommitment, _, err := blockchain.ReadVarstr31(r)
	if err != nil {
		return err
	}

	var (
		ii *IssuanceInput
		si *SpendInput
	)
	if t.AssetVersion == 1 {
		icBuf := bytes.NewBuffer(inputCommitment)
		var icType [1]byte
		_, err = io.ReadFull(icBuf, icType[:])
		if err != nil {
			return err
		}
		bytesRead := 1
		var n int
		switch icType[0] {
		case 0:
			ii = new(IssuanceInput)

			ii.Nonce, n, err = blockchain.ReadVarstr31(icBuf)
			if err != nil {
				return err
			}
			bytesRead += n

			var assetID Hash
			n, err = io.ReadFull(icBuf, assetID[:])
			if err != nil {
				return err
			}
			bytesRead += n

			ii.Amount, n, err = blockchain.ReadVarint63(icBuf)
			if err != nil {
				return err
			}
			bytesRead += n

		case 1:
			si = new(SpendInput)
			n, err = si.Outpoint.readFrom(icBuf)
			if err != nil {
				return err
			}
			bytesRead += n
			n, err = si.OutputCommitment.readFrom(icBuf, txVersion, 1)
			if err != nil {
				return err
			}
			bytesRead += n

		default:
			return fmt.Errorf("unsupported input type %d", icType[0])
		}

		if txVersion == 1 && bytesRead < len(inputCommitment) {
			return fmt.Errorf("unrecognized extra data in input commitment for transaction version 1")
		}
	}

	t.ReferenceData, _, err = blockchain.ReadVarstr31(r)
	if err != nil {
		return err
	}

	inputWitness, _, err := blockchain.ReadVarstr31(r)
	if err != nil {
		return err
	}

	if assetVersion == 1 { // TODO(bobg): also test serialization flags include SerWitness, when we relax the serflags-must-be-0x7 rule
		iwBuf := bytes.NewBuffer(inputWitness)
		if ii != nil {
			// read IssuanceInput witness
			_, err = io.ReadFull(iwBuf, ii.InitialBlock[:])
			if err != nil {
				return err
			}

			ii.VMVersion, _, err = blockchain.ReadVarint63(iwBuf)
			if err != nil {
				return err
			}

			ii.IssuanceProgram, _, err = blockchain.ReadVarstr31(iwBuf)
			if err != nil {
				return err
			}
		}

		// The following is shared in common by spendinputs and issuanceinputs
		n, _, err := blockchain.ReadVarint31(iwBuf)
		if err != nil {
			return err
		}
		var args [][]byte
		for ; n > 0; n-- {
			arg, _, err := blockchain.ReadVarstr31(iwBuf)
			if err != nil {
				return err
			}
			args = append(args, arg)
		}
		if ii != nil {
			ii.Arguments = args
		} else if si != nil {
			si.Arguments = args
		}
	}
	if ii != nil {
		t.TypedInput = ii
	} else if si != nil {
		t.TypedInput = si
	}
	return nil
}

// assumes w has sticky errors
func (t TxInput) writeTo(w io.Writer, serflags uint8) {
	blockchain.WriteVarint63(w, t.AssetVersion) // TODO(bobg): check and return error
	blockchain.WriteVarstr31(w, t.InputCommitmentBytes())
	blockchain.WriteVarstr31(w, t.ReferenceData)
	if serflags&SerWitness != 0 {
		blockchain.WriteVarstr31(w, t.inputWitnessBytes())
	}
}

func (t TxInput) InputCommitmentBytes() []byte {
	inputCommitment := new(bytes.Buffer)
	if t.AssetVersion == 1 {
		switch inp := t.TypedInput.(type) {
		case *IssuanceInput:
			inputCommitment.Write([]byte{0})                     // issuance type
			blockchain.WriteVarstr31(inputCommitment, inp.Nonce) // TODO(bobg): check and return error
			assetID := t.AssetID()
			inputCommitment.Write(assetID[:])
			blockchain.WriteVarint63(inputCommitment, inp.Amount) // TODO(bobg): check and return error

		case *SpendInput:
			inputCommitment.Write([]byte{1}) // spend type
			inp.Outpoint.WriteTo(inputCommitment)
			inp.OutputCommitment.writeTo(inputCommitment, t.AssetVersion)
		}
	}
	return inputCommitment.Bytes()
}

func (t TxInput) inputWitnessBytes() []byte {
	inputWitness := new(bytes.Buffer)
	if t.AssetVersion == 1 {
		var arguments [][]byte
		switch inp := t.TypedInput.(type) {
		case *IssuanceInput:
			inputWitness.Write(inp.InitialBlock[:])
			blockchain.WriteVarint63(inputWitness, inp.VMVersion)       // TODO(bobg): check and return error
			blockchain.WriteVarstr31(inputWitness, inp.IssuanceProgram) // TODO(bobg): check and return error
			arguments = inp.Arguments
		case *SpendInput:
			arguments = inp.Arguments
		}
		blockchain.WriteVarint31(inputWitness, uint64(len(arguments))) // TODO(bobg): check and return error
		for _, arg := range arguments {
			blockchain.WriteVarstr31(inputWitness, arg) // TODO(bobg): check and return error
		}
	}
	return inputWitness.Bytes()
}

func (t TxInput) WitnessHash() Hash {
	var h Hash
	sha3pool.Sum256(h[:], t.inputWitnessBytes())
	return h
}

func (t TxInput) Outpoint() (o Outpoint) {
	if si, ok := t.TypedInput.(*SpendInput); ok {
		o = si.Outpoint
	}
	return o
}

func (si SpendInput) IsIssuance() bool { return false }

func (ii IssuanceInput) IsIssuance() bool { return true }

func (ii IssuanceInput) AssetID() AssetID {
	return ComputeAssetID(ii.IssuanceProgram, ii.InitialBlock, ii.VMVersion)
}

package bc

import (
	"bytes"
	"fmt"
	"io"

	"golang.org/x/crypto/sha3"

	"chain/encoding/blockchain"
)

// TODO(bobg): Review serialization/deserialization logic for
// assetVersions other than 1.

type (
	TxInput struct {
		AssetVersion  uint32
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
		VMVersion       uint32
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

func (t *TxInput) readFrom(r io.Reader) (err error) {
	assetVersion, _ := blockchain.ReadUvarint(r)
	t.AssetVersion = uint32(assetVersion)
	inputCommitment, err := blockchain.ReadBytes(r, commitmentMaxByteLength)
	if err != nil {
		return err
	}
	var (
		ii *IssuanceInput
		si *SpendInput
	)
	if assetVersion == 1 {
		icBuf := bytes.NewBuffer(inputCommitment)
		var icType [1]byte
		_, err = io.ReadFull(icBuf, icType[:])
		if err != nil {
			return err
		}
		switch icType[0] {
		case 0:
			ii = new(IssuanceInput)
			ii.Nonce, err = blockchain.ReadBytes(icBuf, commitmentMaxByteLength)
			if err != nil {
				return err
			}
			var assetID Hash
			_, err = io.ReadFull(icBuf, assetID[:])
			if err != nil {
				return err
			}
			ii.Amount, err = blockchain.ReadUvarint(icBuf)
			if err != nil {
				return err
			}
		case 1:
			si = new(SpendInput)
			si.Outpoint.readFrom(icBuf)
			err = si.OutputCommitment.readFrom(icBuf, 1)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported input type %d", icType[0])
		}
	}
	t.ReferenceData, err = blockchain.ReadBytes(r, refDataMaxByteLength)
	if err != nil {
		return err
	}
	inputWitness, err := blockchain.ReadBytes(r, commitmentMaxByteLength)
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
			vmVersion, err := blockchain.ReadUvarint(iwBuf)
			if err != nil {
				return err
			}
			// TODO(bobg): check range
			ii.VMVersion = uint32(vmVersion)
			ii.IssuanceProgram, err = blockchain.ReadBytes(iwBuf, commitmentMaxByteLength)
			if err != nil {
				return err
			}
		}
		// The following is shared in common by spendinputs and issuanceinputs
		nArgs, err := blockchain.ReadUvarint(iwBuf)
		if err != nil {
			return err
		}
		var args [][]byte
		for i := uint64(0); i < nArgs; i++ {
			arg, err := blockchain.ReadBytes(iwBuf, commitmentMaxByteLength)
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
	blockchain.WriteUvarint(w, uint64(t.AssetVersion))
	blockchain.WriteBytes(w, t.InputCommitmentBytes())
	blockchain.WriteBytes(w, t.ReferenceData)
	if serflags&SerWitness != 0 {
		blockchain.WriteBytes(w, t.inputWitnessBytes())
	}
}

func (t TxInput) InputCommitmentBytes() []byte {
	inputCommitment := new(bytes.Buffer)
	if t.AssetVersion == 1 {
		switch inp := t.TypedInput.(type) {
		case *IssuanceInput:
			inputCommitment.Write([]byte{0}) // issuance type
			blockchain.WriteBytes(inputCommitment, inp.Nonce)
			assetID := t.AssetID()
			inputCommitment.Write(assetID[:])
			blockchain.WriteUvarint(inputCommitment, inp.Amount)
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
			blockchain.WriteUvarint(inputWitness, uint64(inp.VMVersion))
			blockchain.WriteBytes(inputWitness, inp.IssuanceProgram)
			arguments = inp.Arguments
		case *SpendInput:
			arguments = inp.Arguments
		}
		blockchain.WriteUvarint(inputWitness, uint64(len(arguments)))
		for _, arg := range arguments {
			blockchain.WriteBytes(inputWitness, arg)
		}
	}
	return inputWitness.Bytes()
}

func (t TxInput) WitnessHash() Hash {
	return sha3.Sum256(t.inputWitnessBytes())
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

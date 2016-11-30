package bc

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"chain/crypto/sha3pool"
	"chain/encoding/blockchain"
	"chain/encoding/bufpool"
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

	inputCommitment, _, err := blockchain.ReadVarstr31(r)
	if err != nil {
		return err
	}

	var (
		ii      *IssuanceInput
		si      *SpendInput
		assetID AssetID
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

			computedAssetID := ComputeAssetID(ii.IssuanceProgram, ii.InitialBlock, ii.VMVersion)
			if computedAssetID != assetID {
				return errBadAssetID
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
func (t *TxInput) writeTo(w io.Writer, serflags uint8) {
	blockchain.WriteVarint63(w, t.AssetVersion) // TODO(bobg): check and return error
	buf := bufpool.Get()
	defer bufpool.Put(buf)
	t.WriteInputCommitment(buf)
	blockchain.WriteVarstr31(w, buf.Bytes())
	blockchain.WriteVarstr31(w, t.ReferenceData)
	if serflags&SerWitness != 0 {
		buf.Reset()
		t.writeInputWitness(buf)
		blockchain.WriteVarstr31(w, buf.Bytes())
	}
}

func (t *TxInput) WriteInputCommitment(w io.Writer) {
	if t.AssetVersion == 1 {
		switch inp := t.TypedInput.(type) {
		case *IssuanceInput:
			w.Write([]byte{0})                     // issuance type
			blockchain.WriteVarstr31(w, inp.Nonce) // TODO(bobg): check and return error
			assetID := t.AssetID()
			w.Write(assetID[:])
			blockchain.WriteVarint63(w, inp.Amount) // TODO(bobg): check and return error

		case *SpendInput:
			w.Write([]byte{1}) // spend type
			inp.Outpoint.WriteTo(w)
			inp.OutputCommitment.writeTo(w, t.AssetVersion)
		}
	}
}

func (t *TxInput) writeInputWitness(w io.Writer) {
	if t.AssetVersion == 1 {
		var arguments [][]byte
		switch inp := t.TypedInput.(type) {
		case *IssuanceInput:
			w.Write(inp.InitialBlock[:])
			blockchain.WriteVarint63(w, inp.VMVersion)       // TODO(bobg): check and return error
			blockchain.WriteVarstr31(w, inp.IssuanceProgram) // TODO(bobg): check and return error
			arguments = inp.Arguments
		case *SpendInput:
			arguments = inp.Arguments
		}
		blockchain.WriteVarint31(w, uint64(len(arguments))) // TODO(bobg): check and return error
		for _, arg := range arguments {
			blockchain.WriteVarstr31(w, arg) // TODO(bobg): check and return error
		}
	}
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
	return ComputeAssetID(ii.IssuanceProgram, ii.InitialBlock, ii.VMVersion)
}

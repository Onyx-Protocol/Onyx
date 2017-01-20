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
		readWitness(r io.Reader, assetVersion uint64) error
	}
)

var errBadAssetID = errors.New("asset ID does not match other issuance parameters")

func (t *TxInput) writeTo(w io.Writer, serflags uint8) error {
	_, err := blockchain.WriteVarint63(w, t.AssetVersion)
	if err != nil {
		return err
	}
	_, err = blockchain.WriteExtensibleString(w, func(w io.Writer) error {
		return t.WriteInputCommitment(w)
	})
	if err != nil {
		return err
	}
	_, err = blockchain.WriteVarstr31(w, t.ReferenceData)
	if err != nil {
		return err
	}
	if serflags&SerWitness != 0 {
		_, err = blockchain.WriteExtensibleString(w, func(w io.Writer) error {
			return t.writeInputWitness(w)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// WriteInputCommitment writes the bare input commitment to w. It's up
// to the caller to wrap it in an "extensible string" for use in
// transaction serialization.
func (t *TxInput) WriteInputCommitment(w io.Writer) (err error) {
	if t.AssetVersion == 1 {
		switch inp := t.TypedInput.(type) {
		case *IssuanceInput:
			_, err = w.Write([]byte{0}) // issuance type
			if err != nil {
				return err
			}
			_, err = blockchain.WriteVarstr31(w, inp.Nonce)
			if err != nil {
				return err
			}
			assetID := inp.AssetID()
			_, err = w.Write(assetID[:])
			if err != nil {
				return err
			}
			_, err = blockchain.WriteVarint63(w, inp.Amount)
			return err

		case *SpendInput:
			_, err = w.Write([]byte{1}) // spend type
			if err != nil {
				return err
			}
			_, err = inp.Outpoint.WriteTo(w)
			if err != nil {
				return err
			}
			// Nested extensible string
			_, err = blockchain.WriteExtensibleString(w, func(w io.Writer) error {
				return inp.OutputCommitment.writeTo(w, t.AssetVersion)
			})
			return err
		}
		return fmt.Errorf("unknown input type %T", t.TypedInput)
	}
	return nil
}

func (t *TxInput) writeInputWitness(w io.Writer) (err error) {
	if t.AssetVersion == 1 {
		switch inp := t.TypedInput.(type) {
		case *IssuanceInput:
			return inp.IssuanceWitness.writeTo(w)

		case *SpendInput:
			_, err = blockchain.WriteVarstrList(w, inp.Arguments)
			return err
		}
		return fmt.Errorf("unknown input type %T", t.TypedInput)
	}
	return nil
}

func (t *TxInput) readFrom(r io.Reader, txVersion uint64) (err error) {
	t.AssetVersion, _, err = blockchain.ReadVarint63(r)
	if err != nil {
		return errors.Wrap(err, "reading asset version")
	}
	if txVersion == 1 && t.AssetVersion != 1 {
		return fmt.Errorf("unrecognized asset version %d for transaction version %d", t.AssetVersion, txVersion)
	}

	var (
		assetID AssetID
		ii      *IssuanceInput
	)

	all := txVersion == 1
	_, err = blockchain.ReadExtensibleString(r, all, func(r io.Reader) error {
		var icType [1]byte
		_, err = io.ReadFull(r, icType[:])
		if err != nil {
			return errors.Wrap(err, "reading input commitment type")
		}
		switch icType[0] {
		case 0:
			if t.AssetVersion == 1 {
				ii = new(IssuanceInput)
				assetID, err = ii.readCommitment(r)
				if err != nil {
					return errors.Wrap(err, "reading issuance input commitment (v1)")
				}
				t.TypedInput = ii
			}
		case 1:
			inp := new(SpendInput)
			err = inp.readCommitment(r, txVersion, t.AssetVersion)
			if err != nil {
				return errors.Wrap(err, "reading spend input commitment")
			}
			t.TypedInput = inp
		default:
			return fmt.Errorf("unsupported input type %d", icType[0])
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "reading input commitment")
	}

	t.ReferenceData, _, err = blockchain.ReadVarstr31(r)
	if err != nil {
		return errors.Wrap(err, "reading input reference data")
	}

	_, err = blockchain.ReadExtensibleString(r, false, func(r io.Reader) error {
		// TODO(bobg): test that serialization flags include SerWitness, when we relax the serflags-must-be-0x7 rule
		return t.TypedInput.readWitness(r, t.AssetVersion)
	})
	if err != nil {
		return errors.Wrap(err, "reading input witness")
	}

	if ii != nil {
		if assetID != ii.AssetID() {
			return errBadAssetID
		}
	}

	return nil
}

func (t *TxInput) AssetAmount() (assetAmount AssetAmount) {
	switch inp := t.TypedInput.(type) {
	case *IssuanceInput:
		assetAmount.AssetID = inp.AssetID()
		assetAmount.Amount = inp.Amount
		return assetAmount

	case *SpendInput:
		return inp.AssetAmount
	}
	return assetAmount
}

func (t *TxInput) AssetID() (assetID AssetID) {
	switch inp := t.TypedInput.(type) {
	case *IssuanceInput:
		return inp.AssetID()

	case *SpendInput:
		return inp.AssetID
	}
	return assetID
}

func (t *TxInput) Amount() uint64 {
	switch inp := t.TypedInput.(type) {
	case *IssuanceInput:
		return inp.Amount

	case *SpendInput:
		return inp.Amount
	}
	return 0
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

func (t *TxInput) witnessHash() (h Hash, err error) {
	sha := sha3pool.Get256()
	defer sha3pool.Put256(sha)
	err = t.writeInputWitness(sha)
	if err != nil {
		return h, err
	}
	sha.Read(h[:])
	return h, nil
}

func (t *TxInput) Outpoint() (o Outpoint) {
	if si, ok := t.TypedInput.(*SpendInput); ok {
		return si.Outpoint
	}
	return o
}

func (t *TxInput) InitialBlock() (blockID Hash, ok bool) {
	switch inp := t.TypedInput.(type) {
	case *IssuanceInput:
		return inp.InitialBlock, true
	}
	return blockID, false
}

func (t *TxInput) Nonce() ([]byte, bool) {
	switch inp := t.TypedInput.(type) {
	case *IssuanceInput:
		return inp.Nonce, true
	}
	return nil, false
}

func (t *TxInput) VMVer() uint64 {
	switch inp := t.TypedInput.(type) {
	case *IssuanceInput:
		return inp.VMVersion
	case *SpendInput:
		return inp.VMVersion
	}
	return 0
}

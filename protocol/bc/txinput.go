package bc

import (
	"fmt"
	"io"

	"chain-stealth/crypto/ca"
	"chain-stealth/crypto/sha3pool"
	"chain-stealth/encoding/blockchain"
	"chain-stealth/errors"
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

	CAValues struct {
		Value                    uint64
		AssetCommitment          ca.AssetCommitment
		CumulativeBlindingFactor ca.Scalar
		ValueCommitment          ca.ValueCommitment
		ValueBlindingFactor      ca.Scalar
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
			TypedOutput: &Outputv1{
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

func NewConfidentialSpend(outpoint Outpoint, out *Outputv2, referenceData []byte) *TxInput {
	return &TxInput{
		AssetVersion:  2,
		ReferenceData: referenceData,
		TypedInput: &SpendInput{
			Outpoint:    outpoint,
			TypedOutput: out,
		},
	}
}

func NewIssuanceInput(nonce []byte, amount uint64, referenceData []byte, initialBlock Hash, issuanceProgram []byte, arguments [][]byte) *TxInput {
	return &TxInput{
		AssetVersion:  1,
		ReferenceData: referenceData,
		TypedInput: &IssuanceInput1{
			Nonce:  nonce,
			Amount: amount,
			AssetWitness: AssetWitness{
				InitialBlock:    initialBlock,
				VMVersion:       1,
				IssuanceProgram: issuanceProgram,
				Arguments:       arguments,
			},
		},
	}
}

func NewConfidentialIssuanceInput(nonce []byte, amount uint64, referenceData []byte, initialBlock Hash, issuanceProgram []byte, arguments [][]byte, rek ca.RecordKey) (*TxInput, *CAValues, error) {
	assetID := ComputeAssetID(issuanceProgram, initialBlock, 2, 1)
	iek := ca.DeriveIntermediateKey(rek)
	aek := ca.DeriveAssetKey(iek)
	y, Y0 := ca.CreateTransientIssuanceKey(ca.AssetID(assetID), aek)

	// TODO(jackson): In a real system, we'd probably want to use a IARP
	// program that commits to the TXSIGHASH instead of just OP_TRUE.
	ad, vd, iarp, vrp, c, f, err := ca.EncryptIssuance(
		rek,
		ca.AssetID(assetID),
		amount,
		64,
		[]ca.AssetID{ca.AssetID(assetID)},
		[]ca.Point{Y0},
		y,
		1,            // current vm version
		[]byte{0x51}, // 0x51 is OP_TRUE, minus the circular dependency on protocol/vm
	)
	if err != nil {
		return nil, nil, err
	}

	return &TxInput{
			AssetVersion:  2,
			ReferenceData: referenceData,
			TypedInput: &IssuanceInput2{
				Nonce:           nonce,
				assetDescriptor: ad,
				valueDescriptor: vd,
				AssetChoices: []AssetWitness{
					{
						InitialBlock:    initialBlock,
						VMVersion:       1,
						IssuanceProgram: issuanceProgram,
						Arguments:       arguments,
					},
				},
				issuanceAssetRangeProof: iarp,
				valueRangeProof:         vrp,
			},
		}, &CAValues{
			Value:                    amount,
			AssetCommitment:          ad.Commitment(),
			CumulativeBlindingFactor: c,
			ValueCommitment:          vd.Commitment(),
			ValueBlindingFactor:      f,
		}, nil
}

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
	if t.AssetVersion == 1 || t.AssetVersion == 2 {
		switch inp := t.TypedInput.(type) {
		case *IssuanceInput1:
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

		case *IssuanceInput2:
			_, err = w.Write([]byte{0})
			if err != nil {
				return err
			}
			_, err = blockchain.WriteVarstr31(w, inp.Nonce)
			if err != nil {
				return err
			}
			err = inp.assetDescriptor.WriteTo(w)
			if err != nil {
				return err
			}
			return inp.valueDescriptor.WriteTo(w)

		case *SpendInput:
			_, err = w.Write([]byte{1}) // spend type
			if err != nil {
				return err
			}
			_, err = inp.Outpoint.WriteTo(w)
			if err != nil {
				return err
			}

			_, err = blockchain.WriteExtensibleString(w, func(w io.Writer) error {
				return inp.TypedOutput.WriteTo(w)
			})
			return err
		}
		return fmt.Errorf("unknown input type %T", t.TypedInput)
	}
	return nil
}

func (t *TxInput) writeInputWitness(w io.Writer) (err error) {
	if t.AssetVersion == 1 || t.AssetVersion == 2 {
		switch inp := t.TypedInput.(type) {
		case *IssuanceInput1:
			return inp.AssetWitness.writeTo(w)

		case *IssuanceInput2:
			_, err = blockchain.WriteVarint31(w, uint64(len(inp.AssetChoices)))
			if err != nil {
				return err
			}
			for _, c := range inp.AssetChoices {
				err = c.writeTo(w)
				if err != nil {
					return err
				}
			}

			_, err = blockchain.WriteExtensibleString(w, func(w io.Writer) error {
				if inp.issuanceAssetRangeProof != nil {
					err = inp.issuanceAssetRangeProof.WriteTo(w)
					if err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				return err
			}

			_, err = blockchain.WriteExtensibleString(w, func(w io.Writer) error {
				if inp.valueRangeProof != nil {
					err = inp.valueRangeProof.WriteTo(w)
					if err != nil {
						return err
					}
				}
				return nil
			})
			return err

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
	if (txVersion == 1 && t.AssetVersion != 1) ||
		(txVersion == 2 && t.AssetVersion != 1 && t.AssetVersion != 2) {
		return fmt.Errorf("unrecognized asset version %d for transaction version %d", t.AssetVersion, txVersion)
	}

	var (
		assetID AssetID
		ii1     *IssuanceInput1
	)

	all := txVersion == 1 || txVersion == 2
	_, err = blockchain.ReadExtensibleString(r, all, func(r io.Reader) error {
		var icType [1]byte
		_, err = io.ReadFull(r, icType[:])
		if err != nil {
			return errors.Wrap(err, "reading input commitment type")
		}
		switch icType[0] {
		case 0:
			if t.AssetVersion == 1 {
				ii1 = new(IssuanceInput1)
				assetID, err = ii1.readCommitment(r)
				if err != nil {
					return errors.Wrap(err, "reading issuance input commitment (v1)")
				}
				t.TypedInput = ii1
			} else {
				inp := new(IssuanceInput2)
				err = inp.readCommitment(r)
				if err != nil {
					return errors.Wrap(err, "reading issuance input commitment (v2)")
				}
				t.TypedInput = inp
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

	if ii1 != nil {
		if assetID != ii1.AssetID() {
			return errBadAssetID
		}
	}

	return nil
}

func (t *TxInput) AssetAmount() (assetAmount AssetAmount, ok bool) {
	switch inp := t.TypedInput.(type) {
	case *IssuanceInput1:
		assetAmount.AssetID = inp.AssetID()
		assetAmount.Amount = inp.Amount
		return assetAmount, true

	case *IssuanceInput2:
		ad, ok := inp.assetDescriptor.(*ca.NonblindedAssetDescriptor)
		if !ok {
			return assetAmount, false
		}
		vd, ok := inp.valueDescriptor.(*ca.NonblindedValueDescriptor)
		if !ok {
			return assetAmount, false
		}
		assetAmount.AssetID = AssetID(ad.AssetID)
		assetAmount.Amount = vd.Value
		return assetAmount, true

	case *SpendInput:
		switch oc := inp.TypedOutput.(type) {
		case *Outputv1:
			return oc.AssetAmount, true
		case *Outputv2:
			assetAmount.AssetID, ok = oc.AssetID()
			if !ok {
				return assetAmount, false
			}
			assetAmount.Amount, ok = oc.Amount()
			return assetAmount, ok
		}
	}
	return assetAmount, false
}

func (t *TxInput) AssetID() (assetID AssetID, ok bool) {
	switch inp := t.TypedInput.(type) {
	case *IssuanceInput1:
		return inp.AssetID(), true

	case *IssuanceInput2:
		switch ad := inp.assetDescriptor.(type) {
		case *ca.NonblindedAssetDescriptor:
			return AssetID(ad.AssetID), true
		}

	case *SpendInput:
		switch oc := inp.TypedOutput.(type) {
		case *Outputv1:
			return oc.AssetAmount.AssetID, true
		}
	}
	return assetID, false
}

func (t *TxInput) Amount() (uint64, bool) {
	switch inp := t.TypedInput.(type) {
	case *IssuanceInput1:
		return inp.Amount, true

	case *IssuanceInput2:
		if vd, ok := inp.valueDescriptor.(*ca.NonblindedValueDescriptor); ok {
			return vd.Value, true
		}

	case *SpendInput:
		switch oc := inp.TypedOutput.(type) {
		case *Outputv1:
			return oc.AssetAmount.Amount, true
		}
	}
	return 0, false
}

func (t *TxInput) ControlProgram() []byte {
	if si, ok := t.TypedInput.(*SpendInput); ok {
		return si.Program()
	}
	return nil
}

func (t *TxInput) IssuanceProgram() ([]byte, bool) {
	if ii, ok := t.TypedInput.(*IssuanceInput1); ok {
		return ii.IssuanceProgram, true
	}
	return nil, false
}

func (t *TxInput) Arguments() ([][]byte, bool) {
	switch inp := t.TypedInput.(type) {
	case *IssuanceInput1:
		return inp.Arguments, true
	case *SpendInput:
		return inp.Arguments, true
	}
	return nil, false
}

func (t *TxInput) AssetDescriptor() ca.AssetDescriptor {
	switch inp := t.TypedInput.(type) {
	case *SpendInput:
		return inp.AssetDescriptor()
	case *IssuanceInput2:
		return inp.AssetDescriptor()
	}
	return nil
}

func (t *TxInput) ValueDescriptor() ca.ValueDescriptor {
	switch inp := t.TypedInput.(type) {
	case *SpendInput:
		return inp.ValueDescriptor()
	case *IssuanceInput2:
		return inp.ValueDescriptor()
	}
	return nil
}

func (t *TxInput) SetArguments(args [][]byte) {
	switch inp := t.TypedInput.(type) {
	case *IssuanceInput1:
		inp.Arguments = args
	case *IssuanceInput2:
		// TODO(jackson): Support confidential assets in issuance. This
		// will require some refactoring of the interfaces surrounding
		// witnesses.
		if len(inp.AssetChoices) > 1 {
			panic("issuance with multiple asset choices not implemented")
		}
		for i, ac := range inp.AssetChoices {
			ac.Arguments = args
			inp.AssetChoices[i] = ac
		}
	case *SpendInput:
		inp.Arguments = args
	}
}

func (t *TxInput) AssetCommitment() ca.AssetCommitment {
	ad := t.AssetDescriptor()
	if ad != nil {
		return ad.Commitment()
	}
	assetID, _ := t.AssetID()
	return ca.CreateNonblindedAssetCommitment(ca.AssetID(assetID))
}

func (t *TxInput) ValueCommitment() ca.ValueCommitment {
	vd := t.ValueDescriptor()
	if vd != nil {
		return vd.Commitment()
	}
	ac := t.AssetCommitment()
	value, _ := t.Amount()
	return ca.CreateNonblindedValueCommitment(ac, value)
}

func (t *TxInput) WitnessHash() (h Hash, err error) {
	sha := sha3pool.Get256()
	defer sha3pool.Put256(sha)
	err = t.writeInputWitness(sha)
	if err != nil {
		return h, err
	}
	sha.Read(h[:])
	return h, nil
}

func (t *TxInput) Outpoint() (o Outpoint, ok bool) {
	if si, ok := t.TypedInput.(*SpendInput); ok {
		return si.Outpoint, true
	}
	return o, false
}

func (t *TxInput) InitialBlock() (blockID Hash, ok bool) {
	switch inp := t.TypedInput.(type) {
	case *IssuanceInput1:
		return inp.InitialBlock, true
	case *IssuanceInput2:
		if len(inp.AssetChoices) == 1 {
			return inp.AssetChoices[0].InitialBlock, true
		}
	}
	return blockID, false
}

func (t *TxInput) Nonce() ([]byte, bool) {
	switch inp := t.TypedInput.(type) {
	case *IssuanceInput1:
		return inp.Nonce, true
	case *IssuanceInput2:
		return inp.Nonce, true
	}
	return nil, false
}

func (t *TxInput) VMVer() (uint64, bool) {
	switch inp := t.TypedInput.(type) {
	case *IssuanceInput1:
		return inp.VMVersion, true
	case *IssuanceInput2:
		if len(inp.AssetChoices) == 1 {
			return inp.AssetChoices[0].VMVersion, true
		}
	case *SpendInput:
		return inp.VMVer(), true
	}
	return 0, false
}

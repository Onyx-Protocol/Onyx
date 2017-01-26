package bc

// SpendInput satisfies the TypedInput interface and represents a spend transaction.
type SpendInput struct {
	// Commitment
	SpentOutputID OutputID
	OutputCommitment

	// The unconsumed suffix of the output commitment
	OutputCommitmentSuffix []byte

	// Witness
	Arguments [][]byte
}

func (si *SpendInput) IsIssuance() bool { return false }

func NewSpendInput(prevoutID OutputID, arguments [][]byte, assetID AssetID, amount uint64, controlProgram, referenceData []byte) *TxInput {
	const (
		vmver    = 1
		assetver = 1
	)
	oc := OutputCommitment{
		AssetAmount: AssetAmount{
			AssetID: assetID,
			Amount:  amount,
		},
		VMVersion:      vmver,
		ControlProgram: controlProgram,
	}
	return &TxInput{
		AssetVersion:  assetver,
		ReferenceData: referenceData,
		TypedInput: &SpendInput{
			SpentOutputID:    prevoutID,
			OutputCommitment: oc,
			Arguments:        arguments,
		},
	}
}

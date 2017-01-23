package bc

// SpendInput satisfies the TypedInput interface and represents a spend transaction.
type SpendInput struct {
	// Commitment
	OutputID
	OutputCommitment

	// Witness
	Arguments [][]byte
}

func (si *SpendInput) IsIssuance() bool { return false }

func NewSpendInput(outid OutputID, arguments [][]byte, assetID AssetID, amount uint64, controlProgram, referenceData []byte) *TxInput {
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
			OutputID:         outid,
			OutputCommitment: oc,
			Arguments:        arguments,
		},
	}
}

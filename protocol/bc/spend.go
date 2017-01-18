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

func NewSpendInput(txhash Hash, index uint32, arguments [][]byte, assetID AssetID, amount uint64, controlProgram, referenceData []byte) *TxInput {
	av := uint64(1)
	oc := OutputCommitment{
		AssetAmount: AssetAmount{
			AssetID: assetID,
			Amount:  amount,
		},
		VMVersion:      1,
		ControlProgram: controlProgram,
	}
	return &TxInput{
		AssetVersion:  av,
		ReferenceData: referenceData,
		TypedInput: &SpendInput{
			OutputID: ComputeOutputID(txhash, index, oc.Hash(av)),
			OutputCommitment: oc,
			Arguments: arguments,
		},
	}
}

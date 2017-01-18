package bc

// SpendInput satisfies the TypedInput interface and represents a spend transaction.
type SpendInput struct {
	// Commitment
	Outpoint
	OutputCommitment

	// Witness
	Arguments [][]byte
}

func (si *SpendInput) IsIssuance() bool { return false }

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

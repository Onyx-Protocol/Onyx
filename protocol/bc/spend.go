package bc

import (
	"io"

	"chain/encoding/blockchain"
	"chain/errors"
)

// SpendInput satisfies the TypedInput interface and represents a spend transaction.
type SpendInput struct {
	// Commitment
	Outpoint
	OutputCommitment

	// Witness
	Arguments [][]byte
}

func (si *SpendInput) IsIssuance() bool { return false }

func (si *SpendInput) readCommitment(r io.Reader, txVersion, assetVersion uint64) (err error) {
	_, err = si.Outpoint.readFrom(r)
	if err != nil {
		return errors.Wrap(err, "reading outpoint")
	}
	all := txVersion == 1
	_, err = blockchain.ReadExtensibleString(r, all, func(r io.Reader) error {
		_, err := si.OutputCommitment.ReadFrom(r)
		return err
	})
	return errors.Wrap(err, "reading output commitment")
}

func (si *SpendInput) readWitness(r io.Reader, _ uint64) (err error) {
	si.Arguments, _, err = blockchain.ReadVarstrList(r)
	return errors.Wrap(err, "reading input witness")
}

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

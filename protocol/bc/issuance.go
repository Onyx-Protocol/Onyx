package bc

import (
	"io"

	"chain/encoding/blockchain"
	"chain/errors"
)

type IssuanceInput struct {
	// Commitment
	Nonce  []byte
	Amount uint64
	// Note: as long as we require serflags=0x7, we don't need to
	// explicitly store the asset ID here even though it's technically
	// part of the input commitment. We can compute it instead from
	// values in the witness (which, with serflags other than 0x7,
	// might not be present).

	// Witness
	IssuanceWitness
}

func (ii *IssuanceInput) IsIssuance() bool { return true }

func (ii *IssuanceInput) AssetID() AssetID {
	return ComputeAssetID(ii.IssuanceProgram, ii.InitialBlock, ii.VMVersion, ii.AssetDefinitionHash())
}

// readCommitment reads an issuance input commitment AFTER the leading
// type byte has been consumed.
func (ii *IssuanceInput) readCommitment(r io.Reader) (assetID AssetID, err error) {
	ii.Nonce, _, err = blockchain.ReadVarstr31(r)
	if err != nil {
		return assetID, errors.Wrap(err, "reading nonce")
	}

	_, err = io.ReadFull(r, assetID[:])
	if err != nil {
		return assetID, errors.Wrap(err, "reading asset ID")
	}

	ii.Amount, _, err = blockchain.ReadVarint63(r)
	return assetID, errors.Wrap(err, "reading amount")
}

func (ii *IssuanceInput) readWitness(r io.Reader, assetVersion uint64) error {
	return ii.IssuanceWitness.readFrom(r, assetVersion)
}

func NewIssuanceInput(
	nonce []byte,
	amount uint64,
	referenceData []byte,
	initialBlock Hash,
	issuanceProgram []byte,
	arguments [][]byte,
	assetDefinition []byte,
) *TxInput {
	return &TxInput{
		AssetVersion:  1,
		ReferenceData: referenceData,
		TypedInput: &IssuanceInput{
			Nonce:  nonce,
			Amount: amount,
			IssuanceWitness: IssuanceWitness{
				InitialBlock:    initialBlock,
				AssetDefinition: assetDefinition,
				VMVersion:       1,
				IssuanceProgram: issuanceProgram,
				Arguments:       arguments,
			},
		},
	}
}

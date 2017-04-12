package legacy

import (
	"chain/crypto/sha3pool"
	"chain/protocol/bc"
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

func (ii *IssuanceInput) AssetID() bc.AssetID {
	defhash := ii.AssetDefinitionHash()
	return bc.ComputeAssetID(ii.IssuanceProgram, &ii.InitialBlock, ii.VMVersion, &defhash)
}

func (ii *IssuanceInput) AssetDefinitionHash() (defhash bc.Hash) {
	sha := sha3pool.Get256()
	defer sha3pool.Put256(sha)
	sha.Write(ii.AssetDefinition)
	defhash.ReadFrom(sha)
	return defhash
}

func NewIssuanceInput(
	nonce []byte,
	amount uint64,
	referenceData []byte,
	initialBlock bc.Hash,
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

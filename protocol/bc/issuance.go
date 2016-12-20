package bc

import "chain/crypto/sha3pool"

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

func (ii *IssuanceInput) AssetDefinitionHash() (defhash Hash) {
	sha := sha3pool.Get256()
	defer sha3pool.Put256(sha)
	sha.Write(ii.AssetDefinition)
	sha.Read(defhash[:])
	return
}


func (ii *IssuanceInput) IsIssuance() bool { return true }

func (ii *IssuanceInput) AssetID() AssetID {
	return ComputeAssetID(ii.IssuanceProgram, ii.InitialBlock, ii.VMVersion)
}

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

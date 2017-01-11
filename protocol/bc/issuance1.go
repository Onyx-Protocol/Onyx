package bc

import (
	"io"

	"chain-stealth/encoding/blockchain"
	"chain-stealth/errors"
)

type IssuanceInput1 struct {
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

func (ii *IssuanceInput1) IsIssuance() bool { return true }

func (ii *IssuanceInput1) AssetID() AssetID {
	return ComputeAssetID(ii.IssuanceProgram, ii.InitialBlock, 1, ii.VMVersion)
}

func (ii1 *IssuanceInput1) readCommitment(r io.Reader) (assetID AssetID, err error) {
	ii1.Nonce, _, err = blockchain.ReadVarstr31(r)
	if err != nil {
		return assetID, errors.Wrap(err, "reading nonce")
	}

	_, err = io.ReadFull(r, assetID[:])
	if err != nil {
		return assetID, errors.Wrap(err, "reading asset ID")
	}

	ii1.Amount, _, err = blockchain.ReadVarint63(r)
	return assetID, errors.Wrap(err, "reading amount")
}

func (ii1 *IssuanceInput1) readWitness(r io.Reader, assetVersion uint64) error {
	return ii1.IssuanceWitness.readFrom(r, assetVersion)
}

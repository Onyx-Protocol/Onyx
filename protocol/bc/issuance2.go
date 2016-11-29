package bc

import (
	"bytes"

	"io"

	"chain-stealth/crypto/ca"
	"chain-stealth/encoding/blockchain"
	"chain-stealth/errors"
)

type IssuanceInput2 struct {
	// Commitment
	Nonce           []byte
	assetDescriptor ca.AssetDescriptor
	valueDescriptor ca.ValueDescriptor

	// Witness
	AssetChoices            []AssetWitness
	issuanceAssetRangeProof *ca.IssuanceAssetRangeProof
	valueRangeProof         *ca.ValueRangeProof
}

func (ii *IssuanceInput2) IsIssuance() bool { return true }

func (ii2 *IssuanceInput2) readCommitment(r io.Reader) (err error) {
	ii2.Nonce, _, err = blockchain.ReadVarstr31(r)
	if err != nil {
		return errors.Wrap(err, "reading nonce")
	}
	ii2.assetDescriptor, err = ca.ReadAssetDescriptor(r)
	if err != nil {
		return errors.Wrap(err, "reading asset descriptor")
	}
	ii2.valueDescriptor, err = ca.ReadValueDescriptor(r, ii2.assetDescriptor)
	return errors.Wrap(err, "reading value descriptor")
}

func (ii2 *IssuanceInput2) readWitness(r io.Reader, assetVersion uint64) error {
	nchoices, _, err := blockchain.ReadVarint31(r)
	if err != nil {
		return errors.Wrap(err, "reading number of asset choices")
	}
	ii2.AssetChoices = nil
	for i := uint32(0); i < nchoices; i++ {
		var c AssetWitness
		err = c.readFrom(r, assetVersion)
		if err != nil {
			return errors.Wrapf(err, "reading asset witness %d", i)
		}
		ii2.AssetChoices = append(ii2.AssetChoices, c)
	}

	b, _, err := blockchain.ReadVarstr31(r)
	if err != nil {
		return errors.Wrap(err, "reading issuance asset range proof")
	}
	if len(b) > 0 {
		ii2.issuanceAssetRangeProof = new(ca.IssuanceAssetRangeProof)
		err = ii2.issuanceAssetRangeProof.ReadFrom(bytes.NewReader(b), uint32(len(ii2.AssetChoices)))
		if err != nil {
			return errors.Wrap(err, "parsing issuance asset range proof")
		}
	}

	b, _, err = blockchain.ReadVarstr31(r)
	if err != nil {
		return errors.Wrap(err, "reading value range proof")
	}
	if len(b) > 0 {
		ii2.valueRangeProof = new(ca.ValueRangeProof)
		err = ii2.valueRangeProof.ReadFrom(bytes.NewReader(b))
		if err != nil {
			return errors.Wrap(err, "parsing value range proof")
		}
	}

	return nil
}

func (ii2 *IssuanceInput2) VMVersion() uint64 {
	return ii2.issuanceAssetRangeProof.VMVersion()
}

func (ii2 *IssuanceInput2) Program() []byte {
	return ii2.issuanceAssetRangeProof.Program()
}

func (ii2 *IssuanceInput2) Arguments() [][]byte {
	return ii2.issuanceAssetRangeProof.Arguments()
}

func (ii2 *IssuanceInput2) AssetDescriptor() ca.AssetDescriptor {
	return ii2.assetDescriptor
}

func (ii2 *IssuanceInput2) ValueDescriptor() ca.ValueDescriptor {
	return ii2.valueDescriptor
}

func (ii2 *IssuanceInput2) AssetIDs() []ca.AssetID {
	result := make([]ca.AssetID, 0, len(ii2.AssetChoices))
	for _, c := range ii2.AssetChoices {
		result = append(result, ca.AssetID(c.AssetID(2))) // assumes IssuanceInput2 implies asset version 2
	}
	return result
}

func (ii2 *IssuanceInput2) IssuanceAssetRangeProof() *ca.IssuanceAssetRangeProof {
	return ii2.issuanceAssetRangeProof
}

func (ii2 *IssuanceInput2) ValueRangeProof() *ca.ValueRangeProof {
	return ii2.valueRangeProof
}

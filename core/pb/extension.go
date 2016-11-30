package pb

import "chain/crypto/ed25519/chainkd"

func (a *AssetIdentifier) SetAssetId(id []byte) {
	a.Identifier = &AssetIdentifier_AssetId{AssetId: id}
}

func (a *AccountIdentifier) SetAccountId(id string) {
	a.Identifier = &AccountIdentifier_AccountId{AccountId: id}
}

func SignatureWitness(xpubs []chainkd.XPub, path [][]byte, quorum int) *TxTemplate_WitnessComponent {
	sig := &TxTemplate_SignatureComponent{
		Quorum: int32(quorum),
	}
	for _, k := range xpubs {
		sig.KeyIds = append(sig.KeyIds, &TxTemplate_KeyID{
			Xpub:           k[:],
			DerivationPath: path,
		})
	}
	return &TxTemplate_WitnessComponent{
		Component: &TxTemplate_WitnessComponent_Signature{Signature: sig},
	}
}

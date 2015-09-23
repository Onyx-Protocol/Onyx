package validation

import (
	"chain/crypto/hash256"
	"chain/fedchain/bc"
	"chain/fedchain/script"
	"chain/fedchain/state"
)

// GenesisHash must be initialized before using this package.
var GenesisHash [32]byte

// ADPUpdates holds a list of new values to apply
// to the asset definition pointer.
type ADPUpdates struct {
	Updates []*AssetDefinition
}

// AssetDefinition annotates AssetDefinitionPointer
// with relevant metadata
// such as which transaction it is declared in,
// is it mutable or not, etc.
// It also optionally holds the raw asset definition data.
// Used during validation/ingestion process
// so the underlying storage can index that data.
type AssetDefinition struct {
	// Raw pointer (assetID -> hash)
	Pointer bc.AssetDefinitionPointer

	// Raw data matching the hash in the Pointer.
	Data []byte

	// Mutable is false if AssetDefinition is a built-in asset id
	// via pushdata+OP_DROP prefix in the P2SH redeem script.
	Mutable bool

	// InnerAssetID differs from asset id if AssetDefinition is immutable,
	// otherwise it's the same as AssetID in AssetDefinition.
	InnerAssetID bc.AssetID

	// IssuanceScript defines the asset and is used for issuing units
	// and re-defining the asset definition.
	IssuanceScript script.Script

	// RedeemScript is non-nil for immutable AssetDefinition.
	RedeemScript script.Script

	// TxHash and InputIndex identify the tx input
	// in which this pointer is declared.
	TxHash     [32]byte
	InputIndex uint32
}

// NewAssetDefinition validates the given data and returns
// a new AssetDefinition.
func NewAssetDefinition(tx *bc.Tx, unspent *state.Output, txin *bc.TxInput, inIndex uint32) (ad *AssetDefinition, err error) {
	// Check asset definition according to:
	// https://github.com/OpenAssets/open-assets-protocol/blob/08a72c4c05f9ea3780bf421fcd75550644224bde/asset-definition-protocol.mediawiki#determining-the-asset-definition-pointer-associated-to-an-asset
	//
	// 1. If output script is p2sh and redeem script contains OP_DROP prefix data, then use it as a hash.
	// 2. Otherwise use the hash of the metadata if it's not empty.

	adp, err := ExtractADP(unspent.Script, txin.SignatureScript, GenesisHash)
	if err != nil {
		return nil, err
	}
	if adp != nil {
		// Found immutable asset definition in the issuance P2SH script
		ad = new(AssetDefinition)
		ad.Pointer = *adp
		ad.TxHash = tx.Hash()
		ad.InputIndex = inIndex
		if len(txin.Metadata) != 0 && hash256.Sum(txin.Metadata) == adp.DefinitionHash {
			ad.Data = txin.Metadata
		}
	} else if len(txin.Metadata) != 0 {
		// If we have not found a valid asset definition hash
		// in the AssetID (via issuance script),
		// try to use non-empty metadata if it's present.
		ad = new(AssetDefinition)
		ad.Pointer.AssetID = bc.ComputeAssetID(unspent.Script, GenesisHash)
		ad.Pointer.DefinitionHash = hash256.Sum(ad.Data)
		ad.TxHash = tx.Hash()
		ad.InputIndex = inIndex
		ad.Data = txin.Metadata
		ad.Mutable = true
		ad.IssuanceScript = unspent.Script
		ad.InnerAssetID = ad.Pointer.AssetID
	}
	return ad, nil
}

func ExtractADP(issuanceScript, sigScript script.Script, genesisHash [32]byte) (*bc.AssetDefinitionPointer, error) {
	// TODO(erykwalder): add asset definition extractor
	return nil, nil
}

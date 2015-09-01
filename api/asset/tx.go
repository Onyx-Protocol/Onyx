package asset

import "chain/encoding/json"

// Tx represents a partially- or fully-signed transaction
// belonging to a Chain app wallet.
type Tx struct {
	Unsigned   json.HexBytes `json:"unsigned_hex"`
	BlockChain string        `json:"block_chain"`
	Inputs     []*Input      `json:"inputs"`
}

// Input is an input for an app wallet Tx.
type Input struct {
	AssetGroupID  string        `json:"asset_group_id,omitempty"`
	WalletID      string        `json:"wallet_id,omitempty"`
	RedeemScript  json.HexBytes `json:"redeem_script"`
	SignatureData json.HexBytes `json:"signature_data"`
	Sigs          []*Signature  `json:"signatures"`
}

// Signature is an signature for an app wallet Tx.
type Signature struct {
	XPubHash       string        `json:"xpub_hash"`
	DerivationPath []uint32      `json:"derivation_path"`
	DER            json.HexBytes `json:"signature"`
}

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
	WalletID     string        `json:"wallet_id"`
	RedeemScript json.HexBytes `json:"redeem_script"`
	Sigs         []*Signature  `json:"signatures"`
}

// Signature is an signature for an app wallet Tx.
type Signature struct {
	XPubHash       string        `json:"xpub_hash"`
	DerivationPath []uint32      `json:"derivation_path"`
	DER            json.HexBytes `json:"signature"`
}

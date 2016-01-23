package asset

import (
	"chain/encoding/json"
	"chain/fedchain/bc"
)

// TxTemplate represents a partially- or fully-signed transaction
// belonging to a Chain project.
type TxTemplate struct {
	Unsigned   *bc.TxData `json:"unsigned_hex"`
	BlockChain string     `json:"block_chain"`
	Inputs     []*Input   `json:"inputs"`
	OutRecvs   []Receiver `json:"output_receivers"`
}

// Input is an input for a project TxTemplate.
type Input struct {
	RedeemScript  json.HexBytes `json:"redeem_script"`
	SignScript    json.HexBytes `json:"sign_script"`
	SignatureData bc.Hash       `json:"signature_data"`
	Sigs          []*Signature  `json:"signatures"`
}

// Signature is an signature for a project TxTemplate.
type Signature struct {
	XPub           string        `json:"xpub"`
	DerivationPath []uint32      `json:"derivation_path"`
	DER            json.HexBytes `json:"signature"`
}

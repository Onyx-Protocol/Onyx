package asset

import (
	"chain/api/utxodb"
	"chain/encoding/json"
	"chain/fedchain/bc"
)

// Tx represents a partially- or fully-signed transaction
// belonging to a Chain project.
type Tx struct {
	Unsigned   *bc.Tx             `json:"unsigned_hex"`
	BlockChain string             `json:"block_chain"`
	Inputs     []*Input           `json:"inputs"`
	OutRecvs   []*utxodb.Receiver `json:"output_receivers"`
}

// Input is an input for an project Tx.
type Input struct {
	AssetGroupID  string        `json:"issuer_node_id,omitempty"`
	ManagerNodeID string        `json:"manager_node_id,omitempty"`
	RedeemScript  json.HexBytes `json:"redeem_script"`
	SignatureData bc.Hash       `json:"signature_data"`
	Sigs          []*Signature  `json:"signatures"`
}

// Signature is an signature for a project Tx.
type Signature struct {
	XPub           string        `json:"xpub"`
	DerivationPath []uint32      `json:"derivation_path"`
	DER            json.HexBytes `json:"signature"`
}

package txbuilder

import (
	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/encoding/json"
)

// Template represents a partially- or fully-signed transaction.
type Template struct {
	Unsigned *bc.TxData `json:"unsigned_hex"`
	Inputs   []*Input   `json:"inputs"`
}

// Input is an input for a TxTemplate.
type Input struct {
	bc.AssetAmount
	Position      uint32                `json:"position"`
	SigComponents []*SigScriptComponent `json:"signature_components,omitempty"`
}

// SigScriptComponent is an unserialized portion of the sigscript. When
// a tx is finalized, all the sig script components for each input
// are serialized and concatenated to make the final sigscripts. Type
// must be one of 'script', 'data' or 'signature'.
type SigScriptComponent struct {
	Type          string        `json:"type"`           // required
	Data          json.HexBytes `json:"data"`           // required for 'data'
	Quorum        int           `json:"quorum"`         // required for 'signature'
	SignatureData bc.Hash       `json:"signature_data"` // required for 'signature'
	Signatures    []*Signature  `json:"signatures"`     // required for 'signature'
}

// Signature is an signature for a TxTemplate.
type Signature struct {
	XPub           string        `json:"xpub"`
	DerivationPath []uint32      `json:"derivation_path"`
	Bytes          json.HexBytes `json:"signature"`
}

func (inp *Input) AddWitnessData(data []byte) {
	inp.SigComponents = append(inp.SigComponents, &SigScriptComponent{
		Type: "data",
		Data: data,
	})
}

func (inp *Input) AddWitnessSigs(sigs []*Signature, nreq int, sigData *bc.Hash) {
	c := &SigScriptComponent{
		Type:       "signature",
		Quorum:     nreq,
		Signatures: sigs,
	}
	if sigData != nil {
		copy(c.SignatureData[:], (*sigData)[:])
	}
	inp.SigComponents = append(inp.SigComponents, c)
}

type Action interface {
	Build(context.Context) ([]*bc.TxInput, []*bc.TxOutput, []*Input, error)
}

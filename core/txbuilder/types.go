package txbuilder

import (
	"time"

	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/encoding/json"
)

// Template represents a partially- or fully-signed transaction.
type Template struct {
	Unsigned   *bc.TxData `json:"unsigned_hex"`
	BlockChain string     `json:"block_chain"`
	Inputs     []*Input   `json:"inputs"`
}

// Input is an input for a project TxTemplate.
type Input struct {
	bc.AssetAmount
	SigComponents   []*SigScriptComponent `json:"signature_components,omitempty"`
	SigScriptSuffix json.HexBytes         `json:"redeem_script"`  // deprecated
	SignatureData   bc.Hash               `json:"signature_data"` // deprecated
	Sigs            []*Signature          `json:"signatures"`     // deprecated
}

// SigScriptComponent is an unserialized portion of the sigscript. When
// a tx is finalized, all the sig script components for each input
// are serialized and concatenated to make the final sigscripts. Type
// must be one of 'script', 'data' or 'signature'.
type SigScriptComponent struct {
	Type          string        `json:"type"`           // required
	Script        json.HexBytes `json:"script"`         // required for 'script'
	Data          json.HexBytes `json:"data"`           // required for 'data'
	Required      int           `json:"required"`       // required for 'signature'
	SignatureData bc.Hash       `json:"signature_data"` // required for 'signature'
	Signatures    []*Signature  `json:"signatures"`     // required for 'signature'
}

// Signature is an signature for a project TxTemplate.
type Signature struct {
	XPub           string        `json:"xpub"`
	DerivationPath []uint32      `json:"derivation_path"`
	DER            json.HexBytes `json:"signature"`
}

type ReserveResultItem struct {
	TxInput       *bc.TxInput
	TemplateInput *Input
}

type ReserveResult struct {
	Items  []*ReserveResultItem
	Change []*Destination
}

type Reserver interface {
	Reserve(context.Context, *bc.AssetAmount, time.Duration) (*ReserveResult, error)
}

// A Source is a source of funds for a transaction.
type Source struct {
	bc.AssetAmount
	Reserver Reserver
}

type Receiver interface {
	PKScript() []byte
}

// A Destination is a payment destination for a transaction.
type Destination struct {
	bc.AssetAmount
	Metadata []byte
	Receiver Receiver
}

func (source *Source) Reserve(ctx context.Context, ttl time.Duration) (*ReserveResult, error) {
	return source.Reserver.Reserve(ctx, &source.AssetAmount, ttl)
}

func (dest *Destination) PKScript() []byte {
	return dest.Receiver.PKScript()
}

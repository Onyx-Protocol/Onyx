package txbuilder

import (
	"time"

	"golang.org/x/net/context"

	"chain/encoding/json"
	"chain/fedchain/bc"
)

// TxTemplate represents a partially- or fully-signed transaction
// belonging to a Chain project.
type Template struct {
	Unsigned   *bc.TxData `json:"unsigned_hex"`
	BlockChain string     `json:"block_chain"`
	Inputs     []*Input   `json:"inputs"`
}

// Input is an input for a project TxTemplate.
type Input struct {
	bc.AssetAmount

	// The serialized key "redeem_script" is not strictly correct. Changing it
	// will require an update to the Java SDK.
	SigScriptSuffix json.HexBytes `json:"redeem_script"`

	SignatureData bc.Hash      `json:"signature_data"`
	Sigs          []*Signature `json:"signatures"`
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
	Change *Destination
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

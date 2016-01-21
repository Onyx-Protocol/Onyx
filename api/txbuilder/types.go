package txbuilder

import (
	"time"

	"golang.org/x/net/context"

	"chain/api/txdb"
	"chain/encoding/json"
	"chain/fedchain/bc"
)

// TxTemplate represents a partially- or fully-signed transaction
// belonging to a Chain project.
type Template struct {
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
	IsChange() bool
	PKScript() []byte
	// Make sure the UTXOInserter list contains the right kind of
	// UTXOInserter (adding one if necessary), and add data about the
	// txoutput to it.
	AccumulateUTXO(context.Context, *bc.Outpoint, *bc.TxOutput, []UTXOInserter) ([]UTXOInserter, error)
	MarshalJSON() ([]byte, error)
}

// A Destination is a payment destination for a transaction.
type Destination struct {
	bc.AssetAmount
	IsChange bool
	Metadata []byte
	Receiver Receiver
}

type UTXOInserter interface {
	// This function performs UTXO insertion into the db.  It's called
	// as one of the final steps in FinalizeTx().  There may be many
	// UTXOInserters, each inserting utxos of a different type.
	InsertUTXOs(context.Context) ([]*txdb.Output, error)
}

func (source *Source) Reserve(ctx context.Context, ttl time.Duration) (*ReserveResult, error) {
	return source.Reserver.Reserve(ctx, &source.AssetAmount, ttl)
}

func (dest *Destination) PKScript() []byte {
	return dest.Receiver.PKScript()
}

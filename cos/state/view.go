package state

import (
	"golang.org/x/net/context"

	"chain/cos/bc"
)

// View provides access to a consistent snapshot of the blockchain state.
type View interface {
	ViewReader
	ViewWriter
}

// ViewReader provides read access to a consistent snapshot
// of the blockchain state.
//
// It is the ViewReader's responsibility to ensure that
// its methods run fast enough for production throughput.
// If the underlying storage is on a remote server or
// otherwise slow, this requirement typically means the
// view will pre-load or pre-cache many objects in a batch
// so as to avoid multiple round trips.
type ViewReader interface {
	// IsUTXO returns true if the provided output is a valid, unspent
	// output in the view.
	IsUTXO(context.Context, *Output) bool

	// Circulation returns the circulation
	// for the given set of assets.
	Circulation(context.Context, []bc.AssetID) (map[bc.AssetID]int64, error)

	// StateRoot returns the merkle root of the state tree
	StateRoot(context.Context) (bc.Hash, error)
}

type ViewWriter interface {
	// ConsumeUTXO marks the provided utxo as spent.
	ConsumeUTXO(o *Output)

	// AddUTXO adds a new unspent output to the set of available
	// utxos.
	AddUTXO(o *Output)

	// SaveAssetDefinitionPointer updates the asset definition pointer.
	SaveAssetDefinitionPointer(bc.AssetID, bc.Hash)

	// SaveIssuance stores the amount of an asset issued
	SaveIssuance(bc.AssetID, uint64)

	// SaveDestruction stores the amount of an asset destroyed
	SaveDestruction(bc.AssetID, uint64)
}

// Output represents a spent or unspent output
// for the validation process.
type Output struct {
	bc.Outpoint
	bc.TxOutput
}

func NewOutput(o bc.TxOutput, p bc.Outpoint) *Output {
	return &Output{
		TxOutput: o,
		Outpoint: p,
	}
}

// Prevout returns the Output consumed by the provided tx input. It
// only includes the output data that is embedded within inputs (ex,
// excludes metadata).
func Prevout(in *bc.TxInput) *Output {
	return &Output{
		Outpoint: in.Previous,
		TxOutput: bc.TxOutput{
			AssetAmount: in.AssetAmount,
			Script:      in.PrevScript,
		},
	}
}

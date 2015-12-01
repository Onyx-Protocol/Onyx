package txdb

import (
	"golang.org/x/net/context"

	"chain/fedchain/bc"
	"chain/fedchain/state"
)

type poolView struct {
	err *error
}

// NewPoolView returns a new state view on the pool
// of unconfirmed transactions.
// Errors reading and writing outputs
// will be stored in err.
// Any non-nil error value in err will be preserved.
func NewPoolView(err *error) state.ViewReader {
	// TODO(kr): preload several outputs in a batch
	return &poolView{err}
}

func (v *poolView) Output(ctx context.Context, p bc.Outpoint) *state.Output {
	if *v.err != nil {
		return nil
	}
	o, err := loadPoolOutput(ctx, p)
	if err != nil {
		*v.err = err
		return nil
	}
	return o
}

// poolView.AssetDefinitionPointer returns nil because ADPs are encompassed by transactions
// when they're in pools.
func (v *poolView) AssetDefinitionPointer(assetID bc.AssetID) *bc.AssetDefinitionPointer {
	return nil
}

type bcView struct {
	outs map[bc.Outpoint]*state.Output
}

// NewView returns a new state view on the blockchain.
// It loads the prevouts for transactions in txs;
// all other outputs will be omitted from the view.
func NewViewForPrevouts(ctx context.Context, txs []*bc.Tx) (state.ViewReader, error) {
	var p []bc.Outpoint
	for _, tx := range txs {
		for _, in := range tx.Inputs {
			if in.IsIssuance() {
				continue
			}
			p = append(p, in.Previous)
		}
	}
	return NewView(ctx, p)
}

// NewView returns a new state view on the blockchain.
// It loads the outpoints identified in p;
// all other outputs will be omitted from the view.
func NewView(ctx context.Context, p []bc.Outpoint) (state.ViewReader, error) {
	outs, err := loadOutputs(ctx, p)
	if err != nil {
		return nil, err
	}
	return &bcView{outs}, nil
}

func (v *bcView) Output(ctx context.Context, p bc.Outpoint) *state.Output {
	return v.outs[p]
}

func (v *bcView) AssetDefinitionPointer(assetID bc.AssetID) *bc.AssetDefinitionPointer {
	panic("unimplemented")
}

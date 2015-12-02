package txdb

import (
	"golang.org/x/net/context"

	"chain/fedchain/bc"
	"chain/fedchain/state"
)

type view struct {
	outs map[bc.Outpoint]*state.Output
}

// NewPoolViewForPrevouts returns a new state view on the pool
// of unconfirmed transactions.
// It loads the prevouts for transactions in txs;
// all other outputs will be omitted from the view.
func NewPoolViewForPrevouts(ctx context.Context, txs []*bc.Tx) (state.ViewReader, error) {
	var p []bc.Outpoint
	for _, tx := range txs {
		for _, in := range tx.Inputs {
			if in.IsIssuance() {
				continue
			}
			p = append(p, in.Previous)
		}
	}
	return NewPoolView(ctx, p)
}

// NewPoolView returns a new state view on the pool
// of unconfirmed transactions.
// It loads the outpoints identified in p;
// all other outputs will be omitted from the view.
func NewPoolView(ctx context.Context, p []bc.Outpoint) (state.ViewReader, error) {
	outs, err := loadPoolOutputs(ctx, p)
	if err != nil {
		return nil, err
	}
	return &view{outs}, nil
}

// NewViewForPrevouts returns a new state view on the blockchain.
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
	return &view{outs}, nil
}

func (v *view) Output(ctx context.Context, p bc.Outpoint) *state.Output {
	return v.outs[p]
}

func (v *view) AssetDefinitionPointer(assetID bc.AssetID) *bc.AssetDefinitionPointer {
	panic("unimplemented")
}

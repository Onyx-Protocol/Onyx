package txdb

import (
	"golang.org/x/net/context"

	"chain/fedchain/bc"
	"chain/fedchain/state"
)

type view struct {
	outs map[bc.Outpoint]*state.Output
	err  *error
}

func newPoolViewForPrevouts(ctx context.Context, txs []*bc.Tx) (state.ViewReader, error) {
	var p []bc.Outpoint
	for _, tx := range txs {
		for _, in := range tx.Inputs {
			if in.IsIssuance() {
				continue
			}
			p = append(p, in.Previous)
		}
	}
	return newPoolView(ctx, p)
}

// newPoolView returns a new state view on the pool
// of unconfirmed transactions.
// It loads the outpoints identified in p;
// all other outputs will be omitted from the view.
func newPoolView(ctx context.Context, p []bc.Outpoint) (state.ViewReader, error) {
	outs, err := loadPoolOutputs(ctx, p)
	if err != nil {
		return nil, err
	}
	var errbuf error
	result := &view{
		outs: outs,
		err:  &errbuf,
	}
	return result, nil
}

func newViewForPrevouts(ctx context.Context, txs []*bc.Tx) (state.ViewReader, error) {
	var p []bc.Outpoint
	for _, tx := range txs {
		for _, in := range tx.Inputs {
			if in.IsIssuance() {
				continue
			}
			p = append(p, in.Previous)
		}
	}
	return newView(ctx, p)
}

// newView returns a new state view on the blockchain.
// It loads the outpoints identified in p;
// all other outputs will be omitted from the view.
func newView(ctx context.Context, p []bc.Outpoint) (state.ViewReader, error) {
	outs, err := loadOutputs(ctx, p)
	if err != nil {
		return nil, err
	}
	var errbuf error
	result := &view{
		outs: outs,
		err:  &errbuf,
	}
	return result, nil
}

func (v *view) Output(ctx context.Context, p bc.Outpoint) *state.Output {
	if *v.err != nil {
		return nil
	}
	return v.outs[p]
}

func (v *view) AssetDefinitionPointer(assetID bc.AssetID) *bc.AssetDefinitionPointer {
	panic("unimplemented")
}

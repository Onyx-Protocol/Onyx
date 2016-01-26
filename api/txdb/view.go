package txdb

import (
	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
	"chain/fedchain/bc"
	"chain/fedchain/state"
)

type view struct {
	outs                  map[bc.Outpoint]*state.Output
	unspentP2COutputQuery string
	err                   *error
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
	var errbuf error
	result := &view{
		outs: outs,
		unspentP2COutputQuery: poolUnspentP2COutputQuery,
		err: &errbuf,
	}
	return result, nil
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
	var errbuf error
	result := &view{
		outs: outs,
		unspentP2COutputQuery: bcUnspentP2COutputQuery,
		err: &errbuf,
	}
	return result, nil
}

func (v *view) Output(ctx context.Context, p bc.Outpoint) *state.Output {
	if *v.err != nil {
		return nil
	}
	return v.outs[p]
}

func (v *view) UnspentP2COutputs(ctx context.Context, contractHash bc.ContractHash, assetID bc.AssetID) []*state.Output {
	if *v.err != nil {
		return nil
	}
	result, err := loadUnspentP2COutputs(ctx, contractHash, assetID, v.unspentP2COutputQuery)
	if err != nil {
		*v.err = err
		return nil
	}
	return result
}

func loadUnspentP2COutputs(ctx context.Context, contractHash bc.ContractHash, assetID bc.AssetID, query string) (result []*state.Output, err error) {
	rows, err := pg.FromContext(ctx).Query(ctx, query, contractHash, assetID)
	if err != nil {
		return nil, errors.Wrap(err, "query")
	}
	defer rows.Close()
	for rows.Next() {
		var txhash []byte
		var index uint32
		var output state.Output
		err = rows.Scan(&txhash, &index, &output.AssetID, &output.Amount, &output.Script, &output.Metadata)
		if err != nil {
			return nil, errors.Wrap(err, "scan")
		}
		output.Outpoint = *bc.NewOutpoint(txhash, index)
		result = append(result, &output)
	}
	return result, nil
}

func (v *view) AssetDefinitionPointer(assetID bc.AssetID) *bc.AssetDefinitionPointer {
	panic("unimplemented")
}

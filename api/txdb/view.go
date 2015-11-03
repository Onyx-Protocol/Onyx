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
	err *error
}

// NewView returns a new state view on the blockchain.
// Errors reading and writing outputs
// will be stored in err.
// Any non-nil error value in err will be preserved.
func NewView(err *error) state.ViewReader {
	// TODO(kr): preload several outputs in a batch
	return &bcView{err}
}

func (v *bcView) Output(ctx context.Context, p bc.Outpoint) *state.Output {
	if *v.err != nil {
		return nil
	}
	o, err := loadOutput(ctx, p)
	if err != nil {
		*v.err = err
		return nil
	}
	return o
}

func (v *bcView) AssetDefinitionPointer(assetID bc.AssetID) *bc.AssetDefinitionPointer {
	if *v.err != nil {
		return nil
	}

	panic("unimplemented")
}

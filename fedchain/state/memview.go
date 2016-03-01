package state

import (
	"golang.org/x/net/context"

	"chain/fedchain/bc"
)

// MemView satisfies the View interface
type MemView struct {
	Outs map[bc.Outpoint]*Output
	ADPs map[bc.AssetID]bc.Hash
}

// NewMemView returns a new MemView
func NewMemView() *MemView {
	return &MemView{
		Outs: make(map[bc.Outpoint]*Output),
		ADPs: make(map[bc.AssetID]bc.Hash),
	}
}

func (v *MemView) Output(ctx context.Context, p bc.Outpoint) *Output {
	return v.Outs[p]
}

func (v *MemView) SaveOutput(o *Output) {
	v.Outs[o.Outpoint] = o
}

func (v *MemView) SaveAssetDefinitionPointer(asset bc.AssetID, hash bc.Hash) {
	v.ADPs[asset] = hash
}

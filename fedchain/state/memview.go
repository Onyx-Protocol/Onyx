package state

import (
	"golang.org/x/net/context"

	"chain/fedchain/bc"
)

var _ View = (*MemView)(nil)

// MemView satisfies the View interface
type MemView struct {
	Outs     map[bc.Outpoint]*Output
	ADPs     map[bc.AssetID]bc.Hash
	Issuance map[bc.AssetID]uint64
}

// NewMemView returns a new MemView
func NewMemView() *MemView {
	return &MemView{
		Outs:     make(map[bc.Outpoint]*Output),
		ADPs:     make(map[bc.AssetID]bc.Hash),
		Issuance: make(map[bc.AssetID]uint64),
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

func (v *MemView) SaveIssuance(asset bc.AssetID, amount uint64) {
	v.Issuance[asset] += amount
}

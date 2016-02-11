package fedchain

import (
	"golang.org/x/net/context"

	"chain/fedchain/bc"
	"chain/fedchain/state"
)

type MemView struct {
	Outs map[bc.Outpoint]*state.Output
	ADPs map[bc.AssetID]*bc.AssetDefinitionPointer
}

func NewMemView() *MemView {
	return &MemView{
		Outs: make(map[bc.Outpoint]*state.Output),
		ADPs: make(map[bc.AssetID]*bc.AssetDefinitionPointer),
	}
}

func (v *MemView) Output(ctx context.Context, p bc.Outpoint) *state.Output {
	return v.Outs[p]
}

func (v *MemView) AssetDefinitionPointer(assetID bc.AssetID) *bc.AssetDefinitionPointer {
	return v.ADPs[assetID]
}

func (v *MemView) SaveOutput(o *state.Output) {
	v.Outs[o.Outpoint] = o
}

func (v *MemView) SaveAssetDefinitionPointer(adp *bc.AssetDefinitionPointer) {
	v.ADPs[adp.AssetID] = adp
}

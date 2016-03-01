package fedchain

import (
	"golang.org/x/net/context"

	"chain/fedchain/bc"
	"chain/fedchain/state"
)

type memView struct {
	Outs map[bc.Outpoint]*state.Output
	ADPs map[bc.AssetID]*bc.AssetDefinitionPointer
}

func newMemView() *memView {
	return &memView{
		Outs: make(map[bc.Outpoint]*state.Output),
		ADPs: make(map[bc.AssetID]*bc.AssetDefinitionPointer),
	}
}

func (v *memView) Output(ctx context.Context, p bc.Outpoint) *state.Output {
	return v.Outs[p]
}

func (v *memView) SaveOutput(o *state.Output) {
	v.Outs[o.Outpoint] = o
}

func (v *memView) SaveAssetDefinitionPointer(adp *bc.AssetDefinitionPointer) {
	v.ADPs[adp.AssetID] = adp
}

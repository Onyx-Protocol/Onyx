package asset

import (
	"golang.org/x/net/context"

	"chain/api/txdb"
	"chain/fedchain/bc"
	"chain/fedchain/state"
)

type MemView struct {
	Outs map[bc.Outpoint]*txdb.Output
	ADPs map[bc.AssetID]*bc.AssetDefinitionPointer
}

var _ state.View = (*MemView)(nil)

func NewMemView() *MemView {
	return &MemView{
		Outs: make(map[bc.Outpoint]*txdb.Output),
		ADPs: make(map[bc.AssetID]*bc.AssetDefinitionPointer),
	}
}

func (v *MemView) Output(ctx context.Context, p bc.Outpoint) *state.Output {
	o := v.Outs[p]
	if o == nil {
		return nil
	}
	return &o.Output
}

func (v *MemView) AssetDefinitionPointer(assetID bc.AssetID) *bc.AssetDefinitionPointer {
	return v.ADPs[assetID]
}

func (v *MemView) SaveOutput(o *state.Output) {
	v.Outs[o.Outpoint] = &txdb.Output{Output: *o}
}

func (v *MemView) SaveAssetDefinitionPointer(adp *bc.AssetDefinitionPointer) {
	v.ADPs[adp.AssetID] = adp
}

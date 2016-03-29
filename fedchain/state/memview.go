package state

import (
	"golang.org/x/net/context"

	"chain/fedchain/bc"
)

var _ View = (*MemView)(nil)

type AssetState struct {
	ADP                 bc.Hash
	Issuance, Destroyed uint64
}

// MemView satisfies the View interface
type MemView struct {
	Outs   map[bc.Outpoint]*Output
	Assets map[bc.AssetID]*AssetState
}

// NewMemView returns a new MemView
func NewMemView() *MemView {
	return &MemView{
		Outs:   make(map[bc.Outpoint]*Output),
		Assets: make(map[bc.AssetID]*AssetState),
	}
}

func (v *MemView) asset(asset bc.AssetID) *AssetState {
	state := v.Assets[asset]
	if state == nil {
		state = &AssetState{}
		v.Assets[asset] = state
	}
	return state
}

func (v *MemView) Output(ctx context.Context, p bc.Outpoint) *Output {
	return v.Outs[p]
}

func (v *MemView) SaveOutput(o *Output) {
	v.Outs[o.Outpoint] = o
}

func (v *MemView) SaveAssetDefinitionPointer(asset bc.AssetID, hash bc.Hash) {
	v.asset(asset).ADP = hash
}

func (v *MemView) SaveIssuance(asset bc.AssetID, amount uint64) {
	v.asset(asset).Issuance += amount
}

func (v *MemView) SaveDestruction(asset bc.AssetID, amount uint64) {
	v.asset(asset).Destroyed += amount
}

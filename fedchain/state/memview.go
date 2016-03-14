package state

import (
	"golang.org/x/net/context"

	"chain/fedchain/bc"
	"chain/fedchain/patricia"
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

	StateTree *patricia.Tree
}

// NewMemView returns a new MemView. It takes an
// existing state tree. As ViewWriter functions
// are called, the tree will update such that it
// reflects the composite tree of the what was
// entered and the updates that have been made.
func NewMemView(stateTree *patricia.Tree) *MemView {
	return &MemView{
		Outs:      make(map[bc.Outpoint]*Output),
		Assets:    make(map[bc.AssetID]*AssetState),
		StateTree: stateTree,
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

func (v *MemView) Circulation(ctx context.Context, assets []bc.AssetID) (map[bc.AssetID]int64, error) {
	circs := make(map[bc.AssetID]int64)
	for _, a := range assets {
		if state := v.Assets[a]; state != nil {
			circs[a] = int64(state.Issuance - state.Destroyed)
		}
	}
	return circs, nil
}

func (v *MemView) SaveOutput(o *Output) {
	v.Outs[o.Outpoint] = o
	if v.StateTree != nil {
		k, hasher := OutputTreeItem(o)
		if o.Spent {
			v.StateTree.Delete(k)
			return
		}
		v.StateTree.Insert(k, hasher)
	}
}

func (v *MemView) SaveAssetDefinitionPointer(asset bc.AssetID, hash bc.Hash) {
	v.asset(asset).ADP = hash
	if v.StateTree != nil {
		v.StateTree.Insert(ADPTreeItem(asset, hash))
	}
}

func (v *MemView) StateRoot(context.Context) (bc.Hash, error) {
	return v.StateTree.RootHash(), nil
}

func (v *MemView) SaveIssuance(asset bc.AssetID, amount uint64) {
	v.asset(asset).Issuance += amount
}

func (v *MemView) SaveDestruction(asset bc.AssetID, amount uint64) {
	v.asset(asset).Destroyed += amount
}

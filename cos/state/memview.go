package state

import (
	"bytes"

	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/cos/patricia"
	"chain/errors"
)

var _ View = (*MemView)(nil)

type AssetState struct {
	ADP                 bc.Hash
	Issuance, Destroyed uint64
}

// MemView satisfies the View interface. It supports reading from
// in-memory UTXOs, a state tree and optionally another ViewReader.
type MemView struct {
	Added     map[bc.Outpoint]*Output
	Consumed  map[bc.Outpoint]*Output
	Assets    map[bc.AssetID]*AssetState
	Back      ViewReader
	StateTree *patricia.Tree
}

// NewMemView returns a new MemView. It takes an existing state tree,
// and optionally another ViewReader, to read from. As actions are
// are called, the backing tree will update such that it reflects
// the composite tree of the what was entered and the updates that
// have been made.
func NewMemView(stateTree *patricia.Tree, back ViewReader) *MemView {
	return &MemView{
		Added:     make(map[bc.Outpoint]*Output),
		Consumed:  make(map[bc.Outpoint]*Output),
		Assets:    make(map[bc.AssetID]*AssetState),
		Back:      back,
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

func (v *MemView) IsUTXO(ctx context.Context, o *Output) bool {
	if _, ok := v.Consumed[o.Outpoint]; ok {
		return false
	}
	if added, ok := v.Added[o.Outpoint]; ok {
		return bytes.Equal(o.Script, added.Script) && o.AssetAmount == added.AssetAmount
	}
	if v.StateTree != nil {
		k, val := OutputTreeItem(o)
		n := v.StateTree.Lookup(k)
		if n != nil {
			return n.Hash() == val.Value().Hash()
		}
	}

	if v.Back == nil {
		return false
	}
	// If this view didn't spend it, the utxo is valid as long as it's
	// unspent in the backing view.
	return v.Back.IsUTXO(ctx, o)
}

func (v *MemView) Circulation(ctx context.Context, assets []bc.AssetID) (map[bc.AssetID]int64, error) {
	circs := make(map[bc.AssetID]int64)
	for _, a := range assets {
		if state := v.Assets[a]; state != nil {
			circs[a] = int64(state.Issuance - state.Destroyed)
		}
	}
	if v.StateTree != nil {
		for _, a := range assets {
			circs[a] += int64(GetCirculation(v.StateTree, a))
		}
	}
	if v.Back != nil {
		back, err := v.Back.Circulation(ctx, assets)
		if err != nil {
			return nil, errors.Wrap(err, "loading circulation from back")
		}
		for asset, amt := range back {
			circs[asset] += amt
		}
	}
	return circs, nil
}

func (v *MemView) StateRoot(ctx context.Context) (bc.Hash, error) {
	var assets []bc.AssetID
	for asset, amts := range v.Assets {
		if amts.Issuance+amts.Destroyed > 0 {
			assets = append(assets, asset)
		}
	}
	circs, err := v.Circulation(ctx, assets)
	if err != nil {
		return bc.Hash{}, err
	}
	for asset, amt := range circs {
		v.StateTree.Insert(CirculationTreeItem(asset, uint64(amt)))
	}
	return v.StateTree.RootHash(), nil
}

func (v *MemView) SaveAssetDefinitionPointer(asset bc.AssetID, hash bc.Hash) {
	v.asset(asset).ADP = hash
	if v.StateTree != nil {
		v.StateTree.Insert(ADPTreeItem(asset, hash))
	}
}

func (v *MemView) ConsumeUTXO(o *Output) {
	v.Consumed[o.Outpoint] = o
	if v.StateTree != nil {
		k, _ := OutputTreeItem(o)
		v.StateTree.Delete(k)
	}
}

func (v *MemView) AddUTXO(o *Output) {
	v.Added[o.Outpoint] = o
	if v.StateTree != nil {
		k, hasher := OutputTreeItem(o)
		v.StateTree.Insert(k, hasher)
	}
}

func (v *MemView) SaveIssuance(asset bc.AssetID, amount uint64) {
	v.asset(asset).Issuance += amount
}

func (v *MemView) SaveDestruction(asset bc.AssetID, amount uint64) {
	v.asset(asset).Destroyed += amount
}

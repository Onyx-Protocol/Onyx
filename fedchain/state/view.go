package state

import (
	"golang.org/x/net/context"

	"chain/errors"
	"chain/fedchain/bc"
)

// View provides access to a consistent snapshot of the blockchain state.
type View interface {
	ViewReader
	ViewWriter
}

// ViewReader provides read access to a consistent snapshot
// of the blockchain state.
//
// It is the ViewReader's responsibility to ensure that
// its methods run fast enough for production throughput.
// If the underlying storage is on a remote server or
// otherwise slow, this requirement typically means the
// view will pre-load or pre-cache many objects in a batch
// so as to avoid multiple round trips.
type ViewReader interface {
	// Output loads the output from the view.
	// It returns nil if output is not stored or does not exist.
	Output(context.Context, bc.Outpoint) *Output

	// Circulation returns the circulation
	// for the given set of assets.
	Circulation(context.Context, []bc.AssetID) (map[bc.AssetID]int64, error)

	// StateRoot returns the merkle root of the state tree
	StateRoot(context.Context) (bc.Hash, error)
}

type ViewWriter interface {
	// SaveOutput stores output in the view.
	// Saving a spent output may, depending on the type of the view,
	// either erase an existing output or overwrite it with a "spent" flag.
	SaveOutput(*Output)

	// SaveAssetDefinitionPointer updates the asset definition pointer.
	SaveAssetDefinitionPointer(bc.AssetID, bc.Hash)

	// SaveIssuance stores the amount of an asset issued
	SaveIssuance(bc.AssetID, uint64)

	// SaveDestruction stores the amount of an asset destroyed
	SaveDestruction(bc.AssetID, uint64)
}

// Output represents a spent or unspent output
// for the validation process.
// In contrast with bc.TxOutput,
// this stores mandatory extra information
// such as output index and spent flag.
type Output struct {
	bc.TxOutput
	Outpoint bc.Outpoint
	Spent    bool
}

func NewOutput(o bc.TxOutput, p bc.Outpoint, spent bool) *Output {
	return &Output{
		TxOutput: o,
		Outpoint: p,
		Spent:    spent,
	}
}

type compositeView struct {
	ViewWriter
	multiReader
}

// Compose returns a view that combines v
// with the readonly view in r.
// Calls to Output try v
// followed by r.
// Calls to SetOutput go to v.
func Compose(v View, r ViewReader) View {
	return &compositeView{v, multiReader{v, r}}
}

// multiReader

type multiReader struct {
	ViewReader
	back ViewReader
}

// MultiReader returns a view that reads from
// a and then b.
func MultiReader(a, b ViewReader) ViewReader {
	return &multiReader{a, b}
}

func (v *multiReader) Output(ctx context.Context, p bc.Outpoint) *Output {
	o := v.ViewReader.Output(ctx, p)
	if o != nil {
		return o
	}
	return v.back.Output(ctx, p)
}

func (v *multiReader) Circulation(ctx context.Context, assets []bc.AssetID) (map[bc.AssetID]int64, error) {
	front, err := v.ViewReader.Circulation(ctx, assets)
	if err != nil {
		return nil, errors.Wrap(err, "loading circulation from front")
	}
	back, err := v.back.Circulation(ctx, assets)
	if err != nil {
		return nil, errors.Wrap(err, "loading circulation from back")
	}
	for asset, amt := range back {
		front[asset] += amt
	}
	return front, nil
}

func (v *multiReader) StateRoot(ctx context.Context) (bc.Hash, error) {
	if mv, ok := v.ViewReader.(*MemView); ok {
		var assets []bc.AssetID
		for asset, amts := range mv.Assets {
			if amts.Issuance+amts.Destroyed > 0 {
				assets = append(assets, asset)
			}
		}
		circs, err := v.Circulation(ctx, assets)
		if err != nil {
			return bc.Hash{}, err
		}
		for asset, amt := range circs {
			mv.StateTree.Insert(CirculationTreeItem(asset, uint64(amt)))
		}
	}
	return v.ViewReader.StateRoot(ctx)
}

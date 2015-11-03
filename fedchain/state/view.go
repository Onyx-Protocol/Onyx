package state

import (
	"golang.org/x/net/context"

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

	// Streams all the unspent outputs that are paid with the given
	// asset to a given p2c contract.
	UnspentP2COutputs(context.Context, bc.ContractHash, bc.AssetID) []*Output

	// AssetDefinitionPointer looks up the given Asset ID.
	// It returns nil if ADP is not stored or does not exist.
	AssetDefinitionPointer(bc.AssetID) *bc.AssetDefinitionPointer
}

type ViewWriter interface {
	// SaveOutput stores output in the view.
	// Saving a spent output may, depending on the type of the view,
	// either erase an existing output or overwrite it with a "spent" flag.
	SaveOutput(*Output)

	// SaveAssetDefinitionPointer updates the asset definition pointer.
	SaveAssetDefinitionPointer(*bc.AssetDefinitionPointer)
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
	View
	back ViewReader
}

// Compose returns a view that combines v
// with all the readonly views in r.
// Calls to Output try v
// followed by each element of r in order.
// Calls to SetOutput go to v.
func Compose(v View, r ...ViewReader) View {
	if len(r) == 0 {
		return v
	}
	return Compose(&compositeView{v, r[0]}, r[1:]...)
}

func (v *compositeView) Output(ctx context.Context, p bc.Outpoint) *Output {
	o := v.View.Output(ctx, p)
	if o != nil {
		return o
	}
	return v.back.Output(ctx, p)
}

func (v *compositeView) UnspentP2COutputs(ctx context.Context, contractHash bc.ContractHash, assetID bc.AssetID) []*Output {
	// Get the unspent p2c outputs from the "back" view, then check the
	// "front" view to make sure they're still unspent; then add outputs
	// from the "front" view.
	preresult := v.back.UnspentP2COutputs(ctx, contractHash, assetID)
	result := make([]*Output, 0, len(preresult))
	for _, uncheckedOutput := range preresult {
		checkedOutput := v.View.Output(ctx, uncheckedOutput.Outpoint)
		if checkedOutput != nil && !checkedOutput.Spent {
			result = append(result, checkedOutput)
		}
	}
	result = append(result, v.View.UnspentP2COutputs(ctx, contractHash, assetID)...)
	return result
}

// multiReader

type multiReader struct {
	front ViewReader
	back  ViewReader
}

// MultiReader returns a view that reads from
// each element of r in order.
func MultiReader(r ...ViewReader) ViewReader {
	if len(r) == 0 {
		return emptyReader
	}
	return &multiReader{r[0], MultiReader(r[1:]...)}
}

func (v *multiReader) Output(ctx context.Context, p bc.Outpoint) *Output {
	o := v.front.Output(ctx, p)
	if o != nil {
		return o
	}
	return v.back.Output(ctx, p)
}

func (v *multiReader) UnspentP2COutputs(ctx context.Context, contractHash bc.ContractHash, assetID bc.AssetID) []*Output {
	// Get the unspent p2c outputs from the "back" view, then check the
	// "front" view to make sure they're still unspent; then add outputs
	// from the "front" view.
	preresult := v.back.UnspentP2COutputs(ctx, contractHash, assetID)
	result := make([]*Output, 0, len(preresult))
	for _, uncheckedOutput := range preresult {
		checkedOutput := v.front.Output(ctx, uncheckedOutput.Outpoint)
		if checkedOutput != nil && !checkedOutput.Spent {
			result = append(result, checkedOutput)
		}
	}
	result = append(result, v.front.UnspentP2COutputs(ctx, contractHash, assetID)...)
	return result
}

func (v *multiReader) AssetDefinitionPointer(assetID bc.AssetID) *bc.AssetDefinitionPointer {
	adp := v.front.AssetDefinitionPointer(assetID)
	if adp != nil {
		return adp
	}
	return v.back.AssetDefinitionPointer(assetID)
}

var emptyReader ViewReader = empty{}

type empty struct{}

func (empty) Output(ctx context.Context, p bc.Outpoint) *Output {
	return nil
}

func (empty) UnspentP2COutputs(ctx context.Context, contractHash bc.ContractHash, assetID bc.AssetID) []*Output {
	return nil
}

func (empty) AssetDefinitionPointer(assetID bc.AssetID) *bc.AssetDefinitionPointer {
	return nil
}

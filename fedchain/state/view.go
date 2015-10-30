package state

import "chain/fedchain/bc"

// View provides access to a consistent snapshot of the blockchain state.
type View interface {
	// Output loads the output from the view.
	// It returns ErrNotFound if output is not stored or does not exist.
	Output(bc.Outpoint) *Output

	// SaveOutput stores output in the view.
	// Saving a spent output may, depending on the type of the view,
	// either erase an existing output or overwrite it with a "spent" flag.
	SaveOutput(*Output)

	// AssetDefinitionPointer looks up the given Asset ID.
	// It returns ErrNotFound if ADP is not stored or does not exist.
	// AssetDefinitionPointer(bc.AssetID) *bc.AssetDefinitionPointer

	// SaveAssetDefinitionPointer updates the asset definition pointer.
	// SaveAssetDefinitionPointer(*bc.AssetDefinitionPointer)
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

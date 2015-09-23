package state

import (
	"errors"

	"chain/fedchain/bc"
)

// ErrNotFound is returned by View methods
// when an object is not in the view.
var ErrNotFound = errors.New("blockchain state object not found")

// View provides access to a consistent snapshot of the blockchain state.
type View interface {
	// Output loads the output from the view.
	// It returns ErrNotFound if output is not stored or does not exist.
	Output(bc.Outpoint) (*Output, error)

	// SaveOutput stores output in the view.
	// Saving a spent output may, depending on the type of the view,
	// either erase an existing output or overwrite it with a "spent" flag.
	SaveOutput(*Output) error

	// AssetDefinitionPointer looks up the given Asset ID.
	// It returns ErrNotFound if ADP is not stored or does not exist.
	AssetDefinitionPointer(bc.AssetID) (*bc.AssetDefinitionPointer, error)

	// SaveAssetDefinitionPointer updates the asset definition pointer.
	SaveAssetDefinitionPointer(*bc.AssetDefinitionPointer) error
}

// Output represents a spent or unspent output
// for the validation process.
// In contrast with bc.TxOutput,
// this stores mandatory extra information
// such as output index, issuance ID, and spent flag.
type Output struct {
	bc.TxOutput
	Outpoint   bc.Outpoint
	IssuanceID bc.IssuanceID
	Spent      bool
}

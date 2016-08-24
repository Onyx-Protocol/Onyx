package asset

import (
	"context"

	"chain/protocol"
	"chain/protocol/bc"
)

var fc *protocol.FC
var indexer Saver

// A Saver is responsible for saving an annotated asset object
// for indexing and retrieval.
// If the Core is configured not to provide search services,
// SaveAnnotatedAsset can be a no-op.
type Saver interface {
	SaveAnnotatedAsset(context.Context, bc.AssetID, map[string]interface{}) error
}

// Init sets the package level FC. If isManager is true,
// Init registers all necessary callbacks for updating
// application state with the FC.
func Init(chain *protocol.FC, ind Saver, isManager bool) {
	indexer = ind
	if fc == chain {
		// Silently ignore duplicate calls.
		return
	}

	fc = chain
	if isManager {
		fc.AddBlockCallback(func(ctx context.Context, b *bc.Block) {
			recordIssuances(ctx, b)
		})
	}
}

func indexAnnotatedAsset(ctx context.Context, a *Asset) error {
	if indexer == nil {
		return nil
	}
	m := map[string]interface{}{
		"id":               a.AssetID,
		"alias":            a.Alias,
		"definition":       a.Definition,
		"issuance_program": a.IssuanceProgram,
		"tags":             a.Tags,
	}
	return indexer.SaveAnnotatedAsset(ctx, a.AssetID, m)
}

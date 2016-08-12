package asset

import (
	"golang.org/x/net/context"

	"chain/core/query"
	"chain/cos"
	"chain/cos/bc"
)

var fc *cos.FC
var indexer *query.Indexer

// Init sets the package level cos. If isManager is true,
// Init registers all necessary callbacks for updating
// application state with the cos.
func Init(chain *cos.FC, ind *query.Indexer, isManager bool) {
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
		"definition":       a.Definition,
		"issuance_program": a.IssuanceProgram,
		"tags":             a.Tags,
	}
	return indexer.SaveAnnotatedAsset(ctx, a.AssetID, m)
}

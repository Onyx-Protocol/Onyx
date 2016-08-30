package asset

import (
	"context"

	"chain/protocol"
	"chain/protocol/bc"
)

var chain *protocol.Chain
var indexer Saver

// A Saver is responsible for saving an annotated asset object
// for indexing and retrieval.
// If the Core is configured not to provide search services,
// SaveAnnotatedAsset can be a no-op.
type Saver interface {
	SaveAnnotatedAsset(context.Context, bc.AssetID, map[string]interface{}, string) error
}

// Init sets the package level Chain.
func Init(c *protocol.Chain, ind Saver) {
	indexer = ind
	if chain == c {
		// Silently ignore duplicate calls.
		return
	}

	chain = c
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
	return indexer.SaveAnnotatedAsset(ctx, a.AssetID, m, a.sortID)
}

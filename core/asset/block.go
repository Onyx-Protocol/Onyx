package asset

import (
	"context"

	"github.com/lib/pq"

	"chain/database/pg"
	"chain/encoding/json"
	"chain/log"
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
	chain.AddBlockCallback(indexAssets)
}

func indexAnnotatedAsset(ctx context.Context, a *Asset) error {
	if indexer == nil {
		return nil
	}
	m := map[string]interface{}{
		"id":               a.AssetID,
		"alias":            a.Alias,
		"definition":       a.Definition,
		"issuance_program": json.HexBytes(a.IssuanceProgram),
		"tags":             a.Tags,
		"origin":           "external",
	}
	if a.Signer != nil {
		m["xpubs"] = a.Signer.XPubs
		m["quorum"] = a.Signer.Quorum
		m["origin"] = "local"
	}
	return indexer.SaveAnnotatedAsset(ctx, a.AssetID, m, a.sortID)
}

// indexAssets is run on every block and indexes all non-local assets.
func indexAssets(ctx context.Context, b *bc.Block) {
	var (
		assetIDs, definitions pq.StringArray
		issuancePrograms      pq.ByteaArray
		seen                  = make(map[bc.AssetID]bool)
	)
	for _, tx := range b.Transactions {
		for _, in := range tx.Inputs {
			if !in.IsIssuance() {
				continue
			}
			if seen[in.AssetID()] {
				continue
			}

			ic := in.InputCommitment.(*bc.IssuanceInputCommitment)
			definition, err := definitionFromProgram(ic.IssuanceProgram)
			if err != nil {
				continue
			}

			seen[in.AssetID()] = true
			assetIDs = append(assetIDs, in.AssetID().String())
			definitions = append(definitions, string(definition))
			issuancePrograms = append(issuancePrograms, ic.IssuanceProgram)
		}
	}
	if len(assetIDs) == 0 {
		return
	}

	// Grab the intitial block hash.
	initial, err := chain.GetBlock(ctx, 1)
	if err != nil {
		log.Fatal(ctx, log.KeyError, err)
	}

	// Insert these assets into the database. If the asset already exists, don't
	// do anything. Return the asset ID of all inserted assets so we know which
	// ones we have to save to the query indexer.
	//
	// For idempotency concerns, we use `first_block_height` to ensure that this
	// query always returns the full set of new assets at this block. This
	// protects against a crash after inserting into `assets` but before saving
	// the annotated asset to the query indexer.
	const q = `
		WITH new_assets AS (
			INSERT INTO assets (id, issuance_program, definition, created_at, initial_block_hash, first_block_height)
			VALUES(unnest($1::text[]), unnest($2::bytea[]), unnest($3::text[])::jsonb, $4, $5, $6)
			ON CONFLICT (id) DO NOTHING
			RETURNING id
		)
		SELECT id FROM new_assets
			UNION
		SELECT id FROM assets WHERE first_block_height = $6
	`
	var newAssetIDs []bc.AssetID
	err = pg.ForQueryRows(ctx, q, assetIDs, issuancePrograms, definitions, b.Time(), initial.Hash(), b.Height,
		func(assetID bc.AssetID) { newAssetIDs = append(newAssetIDs, assetID) })
	if err != nil {
		log.Fatal(ctx, "at", "error indexing non-local assets", log.KeyError, err)
	}

	// newAssetIDs now contains only the asset IDs of new, non-local
	// assets. We need to index them as annotated assets too.
	for _, assetID := range newAssetIDs {
		// TODO(jackson): Batch the asset lookups.
		a, err := lookupAsset(ctx, assetID, "")
		if err != nil {
			log.Fatal(ctx, "at", "looking up new asset", log.KeyError, err)
		}
		err = indexAnnotatedAsset(ctx, a)
		if err != nil {
			log.Fatal(ctx, "at", "indexing annotated asset", log.KeyError, err)
		}
	}
}

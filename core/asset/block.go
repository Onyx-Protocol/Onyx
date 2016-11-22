package asset

import (
	"context"

	"github.com/lib/pq"

	"chain/core/signers"
	"chain/database/pg"
	"chain/encoding/json"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/vmutil"
)

// PinName is used to identify the pin
// associated with the asset block processor.
const PinName = "asset"

// A Saver is responsible for saving an annotated asset object
// for indexing and retrieval.
// If the Core is configured not to provide search services,
// SaveAnnotatedAsset can be a no-op.
type Saver interface {
	SaveAnnotatedAsset(context.Context, bc.AssetID, map[string]interface{}, string) error
}

func (reg *Registry) indexAnnotatedAsset(ctx context.Context, a *Asset) error {
	if reg.indexer == nil {
		return nil
	}
	m := map[string]interface{}{
		"id":               a.AssetID,
		"alias":            a.Alias,
		"definition":       a.Definition,
		"issuance_program": json.HexBytes(a.IssuanceProgram),
		"tags":             a.Tags,
		"is_local":         "no",
	}
	if a.Signer != nil {
		var keys []map[string]interface{}
		path := signers.Path(a.Signer, signers.AssetKeySpace)
		var jsonPath []json.HexBytes
		for _, p := range path {
			jsonPath = append(jsonPath, p)
		}
		for _, xpub := range a.Signer.XPubs {
			derived := xpub.Derive(path)
			keys = append(keys, map[string]interface{}{
				"root_xpub":             xpub,
				"asset_pubkey":          derived,
				"asset_derivation_path": jsonPath,
			})
		}
		m["keys"] = keys
		m["quorum"] = a.Signer.Quorum
		m["is_local"] = "yes"
	} else {
		pubkeys, quorum, err := vmutil.ParseP2SPMultiSigProgram(a.IssuanceProgram)
		if err == nil {
			var keys []map[string]interface{}
			for _, pubkey := range pubkeys {
				keys = append(keys, map[string]interface{}{
					"asset_pubkey": json.HexBytes(pubkey),
				})
			}
			m["keys"] = keys
			m["quorum"] = quorum
		}
	}
	return reg.indexer.SaveAnnotatedAsset(ctx, a.AssetID, m, a.sortID)
}

func (reg *Registry) ProcessBlocks(ctx context.Context) {
	if reg.pinStore == nil {
		return
	}
	reg.pinStore.ProcessBlocks(ctx, reg.chain, PinName, reg.indexAssets)
}

// indexAssets is run on every block and indexes all non-local assets.
func (reg *Registry) indexAssets(ctx context.Context, b *bc.Block) error {
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
			definition, err := definitionFromProgram(in.IssuanceProgram())
			if err != nil {
				continue
			}
			seen[in.AssetID()] = true
			assetIDs = append(assetIDs, in.AssetID().String())
			definitions = append(definitions, string(definition))
			issuancePrograms = append(issuancePrograms, in.IssuanceProgram())
		}
	}
	if len(assetIDs) == 0 {
		return nil
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
	err := pg.ForQueryRows(ctx, reg.db, q, assetIDs, issuancePrograms, definitions, b.Time(), reg.initialBlockHash, b.Height,
		func(assetID bc.AssetID) { newAssetIDs = append(newAssetIDs, assetID) })
	if err != nil {
		return errors.Wrap(err, "error indexing non-local assets")
	}

	if reg.indexer == nil {
		return nil
	}

	// newAssetIDs now contains only the asset IDs of new, non-local
	// assets. We need to index them as annotated assets too.
	for _, assetID := range newAssetIDs {
		// TODO(jackson): Batch the asset lookups.
		a, err := reg.findByID(ctx, assetID)
		if err != nil {
			return errors.Wrap(err, "looking up new asset")
		}
		err = reg.indexAnnotatedAsset(ctx, a)
		if err != nil {
			return errors.Wrap(err, "indexing annotated asset")
		}
	}
	return nil
}

package asset

import (
	"context"
	"encoding/json"

	"github.com/lib/pq"

	"chain/core/query"
	"chain/core/signers"
	"chain/database/pg"
	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/bc/legacy"
	"chain/protocol/vm/vmutil"
)

// PinName is used to identify the pin
// associated with the asset block processor.
const PinName = "asset"

// A Saver is responsible for saving an annotated asset object
// for indexing and retrieval.
// If the Core is configured not to provide search services,
// SaveAnnotatedAsset can be a no-op.
type Saver interface {
	SaveAnnotatedAsset(context.Context, *query.AnnotatedAsset, string) error
}

func Annotated(a *Asset) (*query.AnnotatedAsset, error) {
	jsonTags := json.RawMessage(`{}`)
	jsonDefinition := json.RawMessage(`{}`)

	// a.RawDefinition is the asset definition as it appears on the
	// blockchain, so it's untrusted and may not be valid json.
	if pg.IsValidJSONB(a.RawDefinition()) {
		jsonDefinition = json.RawMessage(a.RawDefinition())
	}
	if a.Tags != nil {
		b, err := json.Marshal(a.Tags)
		if err != nil {
			return nil, err
		}
		jsonTags = b
	}

	aa := &query.AnnotatedAsset{
		ID:              a.AssetID,
		Definition:      &jsonDefinition,
		Tags:            &jsonTags,
		IssuanceProgram: chainjson.HexBytes(a.IssuanceProgram),
	}
	if a.Alias != nil {
		aa.Alias = *a.Alias
	}
	if a.Signer != nil {
		path := signers.Path(a.Signer, signers.AssetKeySpace)
		var jsonPath []chainjson.HexBytes
		for _, p := range path {
			jsonPath = append(jsonPath, p)
		}
		for _, xpub := range a.Signer.XPubs {
			derived := xpub.Derive(path)
			aa.Keys = append(aa.Keys, &query.AssetKey{
				RootXPub:            xpub,
				AssetPubkey:         derived[:],
				AssetDerivationPath: jsonPath,
			})
		}
		aa.Quorum = a.Signer.Quorum
		aa.IsLocal = true
	} else {
		pubkeys, quorum, err := vmutil.ParseP2SPMultiSigProgram(a.IssuanceProgram)
		if err == nil {
			for _, pubkey := range pubkeys {
				pubkey := pubkey
				aa.Keys = append(aa.Keys, &query.AssetKey{
					AssetPubkey: chainjson.HexBytes(pubkey[:]),
				})
			}
			aa.Quorum = quorum
		}
	}
	return aa, nil
}

func (reg *Registry) indexAnnotatedAsset(ctx context.Context, a *Asset) error {
	if reg.indexer == nil {
		return nil
	}
	aa, err := Annotated(a)
	if err != nil {
		return err
	}
	return reg.indexer.SaveAnnotatedAsset(ctx, aa, a.sortID)
}

func (reg *Registry) ProcessBlocks(ctx context.Context) {
	if reg.pinStore == nil {
		return
	}
	reg.pinStore.ProcessBlocks(ctx, reg.chain, PinName, reg.indexAssets)
}

// indexAssets is run on every block and indexes all non-local assets.
func (reg *Registry) indexAssets(ctx context.Context, b *legacy.Block) error {
	var (
		assetIDs         pq.ByteaArray
		definitions      pq.ByteaArray
		vmVersions       pq.Int64Array
		issuancePrograms pq.ByteaArray
		seen             = make(map[bc.AssetID]bool)
	)
	for _, tx := range b.Transactions {
		for _, in := range tx.Inputs {
			if !in.IsIssuance() {
				continue
			}
			assetID := in.AssetID()
			if seen[assetID] {
				continue
			}
			if ii, ok := in.TypedInput.(*legacy.IssuanceInput); ok {
				definition := ii.AssetDefinition
				seen[assetID] = true
				assetIDs = append(assetIDs, assetID.Bytes())
				definitions = append(definitions, definition)
				vmVersions = append(vmVersions, int64(ii.VMVersion))
				issuancePrograms = append(issuancePrograms, in.IssuanceProgram())
			}
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
			INSERT INTO assets (id, vm_version, issuance_program, definition, created_at, initial_block_hash, first_block_height)
			VALUES(unnest($1::bytea[]), unnest($2::bigint[]), unnest($3::bytea[]), unnest($4::bytea[]), $5, $6, $7)
			ON CONFLICT (id) DO UPDATE SET first_block_height = $7 WHERE assets.first_block_height > $7
			RETURNING id
		)
		SELECT id FROM new_assets
			UNION
		SELECT id FROM assets WHERE first_block_height = $7
	`
	var newAssetIDs []bc.AssetID
	err := pg.ForQueryRows(ctx, reg.db, q, assetIDs, vmVersions, issuancePrograms, definitions, b.Time(), reg.initialBlockHash, b.Height,
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

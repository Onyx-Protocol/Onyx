package asset

import (
	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/cos/validation"
	"chain/database/pg"
	"chain/errors"
	"chain/net/trace/span"
)

// Definitions retrieves a list of asset definitions matching assetIDs.
// The results are returned as a map from an ID to the definition.
func Definitions(ctx context.Context, assetIDs []bc.AssetID) (map[bc.AssetID][]byte, error) {
	assetIDStrings := make([]string, len(assetIDs))
	for i, assetID := range assetIDs {
		assetIDStrings[i] = assetID.String()
	}
	const q = `
		SELECT adp.asset_id, ad.definition
		FROM asset_definition_pointers adp
		JOIN asset_definitions ad ON adp.asset_definition_hash = ad.hash
		WHERE adp.asset_id IN (SELECT unnest($1::text[]))
	`
	res := make(map[bc.AssetID][]byte)
	err := pg.ForQueryRows(ctx, q, pg.Strings(assetIDStrings), func(id bc.AssetID, def []byte) {
		res[id] = def
	})
	return res, err
}

// saveAssetDefinitions saves all asset definitions appearing in the provided
// block and updates all asset definition pointers. It's run as a part of the
// package's block callback.
func saveAssetDefinitions(ctx context.Context, block *bc.Block) error {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	var (
		seen = map[bc.Hash]bool{}
		adps = map[bc.AssetID]bc.Hash{}
		hash []string
		defn [][]byte
	)
	for _, tx := range block.Transactions {
		for _, in := range tx.Inputs {
			if !in.IsIssuance() {
				continue
			}

			assetID, err := validation.AssetIDFromSigScript(in.SignatureScript)
			if err != nil {
				return err
			}
			h := bc.HashAssetDefinition(in.AssetDefinition)
			adps[assetID] = h

			if !seen[h] {
				seen[h] = true
				hash = append(hash, h.String())
				defn = append(defn, in.AssetDefinition)
			}
		}
	}

	const insertDefinitionsQ = `
		WITH defs AS (
			SELECT unnest($1::text[]) h, unnest($2::bytea[]) d
		)
		INSERT INTO asset_definitions (hash, definition)
		SELECT h, d FROM defs
		ON CONFLICT (hash) DO NOTHING
	`
	_, err := pg.Exec(ctx, insertDefinitionsQ, pg.Strings(hash), pg.Byteas(defn))
	if err != nil {
		return errors.Wrap(err, "saving asset definitions")
	}

	aids := make([]string, 0, len(adps))
	ptrs := make([]string, 0, len(adps))
	for assetID, pointer := range adps {
		aids = append(aids, assetID.String())
		ptrs = append(ptrs, pointer.String())
	}

	const insertPointersQ = `
		WITH adps AS (
			SELECT unnest($1::text[]) h, unnest($2::text[]) id
		)
		INSERT INTO asset_definition_pointers (asset_definition_hash, asset_id)
		SELECT h, id FROM adps
		ON CONFLICT (asset_id) DO UPDATE SET asset_definition_hash = excluded.asset_definition_hash
	`
	_, err = pg.Exec(ctx, insertPointersQ, pg.Strings(ptrs), pg.Strings(aids))
	return errors.Wrap(err)
}

package txdb

import (
	"database/sql"

	"golang.org/x/net/context"

	"chain/crypto/hash256"
	"chain/database/pg"
	"chain/errors"
	"chain/fedchain/bc"
	"chain/net/trace/span"
)

func AssetDefinition(ctx context.Context, assetID string) (string, []byte, error) {
	const q = `
		SELECT hash, definition
		FROM asset_definition_pointers adp
		JOIN asset_definitions ON asset_definition_hash=hash
		WHERE asset_id=$1
	`
	var (
		hash     string
		defBytes []byte
	)
	err := pg.FromContext(ctx).QueryRow(ctx, q, assetID).Scan(&hash, &defBytes)
	if err == sql.ErrNoRows {
		err = pg.ErrUserInputNotFound
	}
	if err != nil {
		return "", nil, errors.WithDetailf(err, "asset=%s", assetID)
	}
	return hash, defBytes, nil
}

func DefinitionHashByAssetID(ctx context.Context, assetID string) (string, error) {
	const q = `
		SELECT asset_definition_hash FROM asset_definition_pointers WHERE asset_id=$1
	`

	var hash string
	err := pg.FromContext(ctx).QueryRow(ctx, q, assetID).Scan(&hash)
	if err != nil {
		return "", errors.Wrapf(err, "fetching definition for asset %s", assetID)
	}

	return hash, nil
}

// InsertAssetDefinitionPointers writes the and asset id and the definition hash,
// to the asset_definition_pointers table.
func InsertAssetDefinitionPointers(ctx context.Context, adps map[bc.AssetID]*bc.AssetDefinitionPointer) error {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	const q = `
		WITH adps AS (
			SELECT unnest($1::text[]) h, unnest($2::text[]) id
		), updates AS (
			UPDATE asset_definition_pointers
			SET asset_definition_hash=h
			FROM adps
			WHERE asset_id=id
			RETURNING asset_id
		)
		INSERT INTO asset_definition_pointers (asset_definition_hash, asset_id)
		SELECT * FROM adps
		WHERE id NOT IN (TABLE updates)
	`

	var aids, ptrs []string
	for id, p := range adps {
		aids = append(aids, id.String())
		ptrs = append(ptrs, bc.Hash(p.DefinitionHash).String())
	}

	_, err := pg.FromContext(ctx).Exec(ctx, q, pg.Strings(ptrs), pg.Strings(aids))
	return errors.Wrap(err)
}

// InsertAssetDefinitions inserts a record for each asset definition
// in block. The record maps the hash to the data of the definition.
func InsertAssetDefinitions(ctx context.Context, block *bc.Block) error {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	var (
		seen = map[bc.Hash]bool{}
		hash []string
		defn [][]byte
	)
	for _, tx := range block.Transactions {
		for _, in := range tx.Inputs {
			if in.IsIssuance() && len(in.AssetDefinition) > 0 {
				var h bc.Hash = hash256.Sum(in.AssetDefinition)
				if seen[h] {
					continue
				}
				seen[h] = true
				hash = append(hash, h.String())
				defn = append(defn, in.AssetDefinition)
			}
		}
	}

	const q = `
		WITH defs AS (
			SELECT unnest($1::text[]) h, unnest($2::bytea[]) d
		), filtered_defs AS (
			SELECT h, d FROM defs
			WHERE NOT EXISTS (
				SELECT null FROM asset_definitions
				WHERE h = hash
			)
		)
		INSERT INTO asset_definitions (hash, definition) TABLE filtered_defs
	`
	_, err := pg.FromContext(ctx).Exec(ctx, q, pg.Strings(hash), pg.Byteas(defn))
	return errors.Wrap(err)
}

package txdb

import (
	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/cos/state"
	"chain/database/pg"
	"chain/database/sql"
	"chain/errors"
	"chain/net/trace/span"
)

// GetAssetDefs retrieves a list of asset definitions matching assetIDs. The
// results are returned as a map from an ID to the definition.
func (s *Store) AssetDefinitions(ctx context.Context, assetIDs []string) (map[string][]byte, error) {
	return assetDefinitions(ctx, s.db, assetIDs)
}

func assetDefinitions(ctx context.Context, db pg.DB, assetIDs []string) (map[string][]byte, error) {
	const q = `
		SELECT adp.asset_id, ad.definition
		FROM asset_definition_pointers adp
		JOIN asset_definitions ad ON adp.asset_definition_hash = ad.hash
		WHERE adp.asset_id IN (SELECT unnest($1::text[]))
	`
	res := make(map[string][]byte)
	err := pg.ForQueryRows(pg.NewContext(ctx, db), q, pg.Strings(assetIDs), func(id string, def []byte) {
		res[id] = def
	})
	return res, err
}

func (s *Store) AssetDefinition(ctx context.Context, assetID string) (string, []byte, error) {
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
	err := s.db.QueryRow(ctx, q, assetID).Scan(&hash, &defBytes)
	if err == sql.ErrNoRows {
		err = pg.ErrUserInputNotFound
	}
	if err != nil {
		return "", nil, errors.WithDetailf(err, "asset=%s", assetID)
	}
	return hash, defBytes, nil
}

// insertAssetDefinitionPointers writes the and asset id and the definition hash,
// to the asset_definition_pointers table.
func insertAssetDefinitionPointers(ctx context.Context, dbtx *sql.Tx, assets map[bc.AssetID]*state.AssetState) error {
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
	for id, state := range assets {
		if state.ADP == (bc.Hash{}) {
			continue
		}
		aids = append(aids, id.String())
		ptrs = append(ptrs, state.ADP.String())
	}

	_, err := dbtx.Exec(ctx, q, pg.Strings(ptrs), pg.Strings(aids))
	return errors.Wrap(err)
}

// insertAssetDefinitions inserts a record for each asset definition
// in block. The record maps the hash to the data of the definition.
func insertAssetDefinitions(ctx context.Context, dbtx *sql.Tx, block *bc.Block) error {
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
				h := bc.HashAssetDefinition(in.AssetDefinition)
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
	_, err := dbtx.Exec(ctx, q, pg.Strings(hash), pg.Byteas(defn))
	return errors.Wrap(err)
}

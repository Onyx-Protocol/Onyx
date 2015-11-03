package txdb

import (
	"golang.org/x/net/context"

	"chain/crypto/hash256"
	"chain/database/pg"
	"chain/errors"
	"chain/fedchain/bc"
)

func DefinitionHashByAssetID(ctx context.Context, assetID string) (string, error) {
	const q = `
		SELECT asset_definition_hash FROM asset_definition_pointers WHERE asset_id=$1
	`

	var hash string
	err := pg.FromContext(ctx).QueryRow(q, assetID).Scan(&hash)
	if err != nil {
		return "", errors.Wrapf(err, "fetching definition for asset %s", assetID)
	}

	return hash, nil
}

// InsertAssetDefinitionPointers writes the and asset id and the definition hash,
// to the asset_definition_pointers table.
func InsertAssetDefinitionPointers(ctx context.Context, adps map[bc.AssetID]*bc.AssetDefinitionPointer) error {
	for _, adp := range adps {
		err := insertADP(ctx, adp)
		if err != nil {
			return errors.Wrapf(err, "inserting adp for asset %s", adp.AssetID)
		}
	}

	return nil
}

func insertADP(ctx context.Context, adp *bc.AssetDefinitionPointer) error {
	aid := adp.AssetID.String()
	hash := bc.Hash(adp.DefinitionHash).String()

	const updateQ = `
		UPDATE asset_definition_pointers
		SET asset_definition_hash=$2
		WHERE asset_id=$1
	`

	res, err := pg.FromContext(ctx).Exec(updateQ, aid, hash)
	if err != nil {
		return errors.Wrap(err, "updateQ setting asset definition pointer")
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "checking rows affected, setting asset definition pointer")
	}

	if affected == 0 {
		const insertQ = `
			INSERT INTO asset_definition_pointers (asset_id, asset_definition_hash)
			VALUES ($1, $2)
		`

		_, err = pg.FromContext(ctx).Exec(insertQ, aid, hash)
		if err != nil {
			return errors.Wrap(err, "insertQ setting asset definition pointer")
		}

	}

	return nil
}

// InsertAssetDefinitions writes the maps the hash of an asset definition
// to that definition.
func InsertAssetDefinitions(ctx context.Context, block *bc.Block) error {
	defs := make(map[[32]byte][]byte)
	for _, tx := range block.Transactions {
		for _, in := range tx.Inputs {
			if in.IsIssuance() {
				defs[hash256.Sum(in.Metadata)] = in.Metadata
			}
		}
	}

	for hash, def := range defs {
		err := insertAssetDefinition(ctx, hash, def)
		if err != nil {
			return errors.Wrapf(err, "inserting definition for definition hash %s", hash)
		}
	}

	return nil
}

func insertAssetDefinition(ctx context.Context, hash [32]byte, definition []byte) error {
	hashString := bc.Hash(hash).String()
	const updateQ = `UPDATE asset_definitions SET definition=$2 WHERE hash=$1`
	res, err := pg.FromContext(ctx).Exec(updateQ, hashString, definition)
	if err != nil {
		return errors.Wrap(err, "updateQ setting asset definition")
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "checking rows affected, setting asset definition")
	}

	if affected == 0 {
		const insertQ = `INSERT INTO asset_definitions(hash, definition) VALUES ($1, $2)`

		_, err = pg.FromContext(ctx).Exec(insertQ, hashString, definition)
		if err != nil {
			return errors.Wrap(err, "setting asset definition")
		}
	}

	return nil
}

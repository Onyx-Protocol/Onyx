package appdb

import (
	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/fedchain/wire"
)

// Asset represents an asset type in the blockchain.
// It is made up of extended keys, and paths (indexes) within those keys.
// Assets belong to wallets.
type Asset struct {
	Hash            wire.Hash20 // the raw Asset ID
	GroupID         string
	Label           string
	Keys            []*Key
	AGIndex, AIndex []uint32
	RedeemScript    []byte
}

// AssetByID loads an asset from the database using its ID.
func AssetByID(ctx context.Context, id string) (*Asset, error) {
	const q = `
		SELECT keys, redeem_script, asset_group_id,
			key_index(asset_group.key_index), key_index(assets.key_index),
		FROM assets
		INNER JOIN asset_groups ON asset_groups.id=assets.asset_group_id
		WHERE assets.id=$1
	`
	var (
		keyIDs []string
		a      = new(Asset)
	)
	var err error
	a.Hash, err = wire.NewHash20FromStr(id)
	if err != nil {
		return nil, err
	}
	err = pg.FromContext(ctx).QueryRow(q, id).Scan(
		(*pg.Strings)(&keyIDs),
		&a.RedeemScript,
		&a.GroupID,
		(*pg.Uint32s)(&a.AGIndex),
		(*pg.Uint32s)(&a.AIndex),
	)
	if err != nil {
		return nil, err
	}

	a.Keys, err = getKeys(ctx, keyIDs)
	if err != nil {
		return nil, err
	}

	return a, nil
}

// InsertAsset adds the asset to the database
func InsertAsset(ctx context.Context, asset *Asset) error {
	const q = `
		INSERT INTO assets (id, asset_group_id, key_index, keyset, redeem_script, label)
		VALUES($1, $2, to_key_index($3), $4, $5, $6)
	`
	var xpubs []string
	for _, key := range asset.Keys {
		xpubs = append(xpubs, key.XPub.String())
	}

	_, err := pg.FromContext(ctx).Exec(q,
		asset.Hash.String(),
		asset.GroupID,
		pg.Uint32s(asset.AIndex),
		pg.Strings(xpubs),
		asset.RedeemScript,
		asset.Label,
	)
	return err
}

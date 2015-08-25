package wallets

import "chain/database/pg"

// Asset represents an asset type in the blockchain.
// It is made up of extended keys, and paths (indexes) within those keys.
// Assets belong to wallets.
type Asset struct {
	keyIDs         []string
	wIndex, aIndex int
}

// AssetByID loads an asset from the database using its ID.
func AssetByID(id string) (*Asset, error) {
	const q = `
		SELECT keys, wallets.key_index, assets.key_index
		FROM assets
		INNER JOIN wallets ON wallets.id=assets.wallet_id
		WHERE assets.id=$1
	`
	a := new(Asset)
	err := DB.QueryRow(q, id).Scan((*pg.Strings)(&a.keyIDs), &a.wIndex, &a.aIndex)
	if err != nil {
		return nil, err
	}
	// load keys
	return a, nil
}

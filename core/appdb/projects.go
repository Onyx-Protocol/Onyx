package appdb

import (
	"sort"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
	"chain/strings"
)

// IsAdmin returns true if the user is an admin.
func IsAdmin(ctx context.Context, userID string) (bool, error) {
	const q = `
		SELECT COUNT(*)=1 FROM users
		WHERE id=$1 AND role='admin'
	`
	var isAdmin bool
	row := pg.QueryRow(ctx, q, userID)
	err := row.Scan(&isAdmin)
	return isAdmin, errors.Wrap(err)
}

// CheckActiveAsset returns nil if the provided assets are active.
// If any of the assets are archived, this function returns ErrArchived.
func CheckActiveAsset(ctx context.Context, assetIDs ...string) error {
	// Remove duplicates so that we know how many assets to expect.
	sort.Strings(assetIDs)
	assetIDs = strings.Uniq(assetIDs)

	const q = `
		SELECT COUNT(id),
		       COUNT(CASE WHEN archived THEN 1 ELSE NULL END) AS archived
		FROM assets
		WHERE id=ANY($1)
	`
	var (
		assetsArchived int
		assetsFound    int
	)
	err := pg.QueryRow(ctx, q, pg.Strings(assetIDs)).
		Scan(&assetsFound, &assetsArchived)
	if err != nil {
		return errors.Wrap(err)
	}
	if assetsFound != len(assetIDs) {
		err = pg.ErrUserInputNotFound
	} else if assetsArchived > 0 {
		err = ErrArchived
	}
	return errors.WithDetailf(err, "asset IDs: %+v", assetIDs)
}

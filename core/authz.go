package core

import (
	"golang.org/x/net/context"

	"chain/core/appdb"
	"chain/database/pg"
	"chain/errors"
	"chain/net/http/authn"
)

var (
	errNoAccessToResource = errors.New("Resources are not available to user")
	errNotAdmin           = errors.New("Resource is only available to admins")
)

func adminAuthz(ctx context.Context) error {
	hasAccess, err := appdb.IsAdmin(ctx, authn.GetAuthID(ctx))
	if err != nil {
		return err
	}
	if !hasAccess {
		return errNotAdmin
	}
	return nil
}

// issuerAuthz will verify whether this request has access to the provided asset
// issuer. If the asset issuer is archived, issuerAuthz will return ErrArchived.
func issuerAuthz(ctx context.Context, issuerID string) error {
	return appdb.CheckActiveIssuer(ctx, issuerID)
}

// assetAuthz will verify whether this request has access to the provided
// asset. If the asset is archived, assetAuthz will return ErrArchived.
func assetAuthz(ctx context.Context, assetID string) error {
	return appdb.CheckActiveAsset(ctx, assetID)
}

func buildAuthz(ctx context.Context, reqs ...*BuildRequest) error {
	var (
		assetIDs []string
	)
	for _, req := range reqs {
		for _, source := range req.Sources {
			if source.Type == "issue" && source.AssetID != nil {
				assetIDs = append(assetIDs, source.AssetID.String())
			}
		}
	}
	if len(assetIDs) == 0 {
		return nil
	}

	err := appdb.CheckActiveAsset(ctx, assetIDs...)
	if errors.Root(err) == pg.ErrUserInputNotFound || errors.Root(err) == appdb.ErrArchived {
		return errors.WithDetailf(errNoAccessToResource, "asset IDs: %+v", assetIDs)
	}
	return err
}

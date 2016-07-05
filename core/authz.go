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

func projectAdminAuthz(ctx context.Context, project string) error {
	hasAccess, err := appdb.IsAdmin(ctx, authn.GetAuthID(ctx), project)
	if err != nil {
		return err
	}
	if !hasAccess {
		return errNotAdmin
	}
	return nil
}

// managerAuthz will verify whether this request has access to the provided account
// manager. If the account manager is archived, managerAuthz will return ErrArchived.
func managerAuthz(ctx context.Context, managerID string) error {
	return appdb.CheckActiveManager(ctx, managerID)
}

// accountAuthz will verify whether this request has access to the provided
// account. If the account is archived, accountAuthz will return ErrArchived.
func accountAuthz(ctx context.Context, accountID string) error {
	return appdb.CheckActiveAccount(ctx, accountID)
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
		accountIDs []string
		assetIDs   []string
	)
	for _, req := range reqs {
		for _, source := range req.Sources {
			if source.AccountID != "" {
				accountIDs = append(accountIDs, source.AccountID)
			}
			if source.Type == "issue" && source.AssetID != nil {
				assetIDs = append(assetIDs, source.AssetID.String())
			}
		}
		for _, dest := range req.Dests {
			if dest.AccountID != "" {
				accountIDs = append(accountIDs, dest.AccountID)
			}
		}
	}
	if len(accountIDs) == 0 {
		return nil
	}

	err := appdb.CheckActiveAccount(ctx, accountIDs...)
	if errors.Root(err) == pg.ErrUserInputNotFound || errors.Root(err) == appdb.ErrArchived {
		return errors.WithDetailf(errNoAccessToResource, "account IDs: %+v", accountIDs)
	}
	if err != nil {
		return err
	}

	err = appdb.CheckActiveAsset(ctx, assetIDs...)
	if errors.Root(err) == pg.ErrUserInputNotFound || errors.Root(err) == appdb.ErrArchived {
		return errors.WithDetailf(errNoAccessToResource, "asset IDs: %+v", assetIDs)
	}
	return err
}

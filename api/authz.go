package api

import (
	"golang.org/x/net/context"

	"chain/api/appdb"
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

func projectAuthz(ctx context.Context, projects ...string) error {
	if len(projects) != 1 {
		return errNoAccessToResource
	}

	hasAccess, err := appdb.IsMember(ctx, authn.GetAuthID(ctx), projects[0])
	if err != nil {
		return err
	}
	if !hasAccess {
		return errNoAccessToResource
	}
	return nil
}

func managerAuthz(ctx context.Context, managerID string) error {
	project, err := appdb.ProjectByActiveManager(ctx, managerID)
	if err != nil {
		return err
	}
	return errors.WithDetailf(projectAuthz(ctx, project), "manager node %v", managerID)
}

func accountAuthz(ctx context.Context, accountID string) error {
	projects, err := appdb.ProjectsByAccount(ctx, accountID)
	if err != nil {
		return err
	}
	return errors.WithDetailf(projectAuthz(ctx, projects...), "account %v", accountID)
}

func issuerAuthz(ctx context.Context, issuerID string) error {
	project, err := appdb.ProjectByActiveIssuer(ctx, issuerID)
	if err != nil {
		return err
	}
	return errors.WithDetailf(projectAuthz(ctx, project), "issuer node %v", issuerID)
}

func assetAuthz(ctx context.Context, assetID string) error {
	project, err := appdb.ProjectByActiveAsset(ctx, assetID)
	if err != nil {
		return err
	}
	return errors.WithDetailf(projectAuthz(ctx, project), "asset %v", assetID)
}

func buildAuthz(ctx context.Context, reqs ...*BuildRequest) error {
	var accountIDs []string
	for _, req := range reqs {
		for _, source := range req.Sources {
			if source.AccountID != "" {
				accountIDs = append(accountIDs, source.AccountID)
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
	projects, err := appdb.ProjectsByAccount(ctx, accountIDs...)
	if errors.Root(err) == pg.ErrUserInputNotFound {
		return errors.WithDetailf(errNoAccessToResource, "accounts %+v", accountIDs)
	}
	if err != nil {
		return err
	}
	return errors.WithDetail(projectAuthz(ctx, projects...), "invalid combination of accounts")
}

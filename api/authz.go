package api

import (
	"golang.org/x/net/context"

	"chain/api/appdb"
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
	project, err := appdb.ProjectByManager(ctx, managerID)
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
	project, err := appdb.ProjectByIssuer(ctx, issuerID)
	if err != nil {
		return err
	}
	return errors.WithDetailf(projectAuthz(ctx, project), "issuer node %v", issuerID)
}

func assetAuthz(ctx context.Context, assetID string) error {
	project, err := appdb.ProjectByAsset(ctx, assetID)
	if err != nil {
		return err
	}
	return errors.WithDetailf(projectAuthz(ctx, project), "asset %v", assetID)
}

func buildAuthz(ctx context.Context, reqs ...buildReq) error {
	var accountIDs []string
	for _, req := range reqs {
		for _, input := range req.Inputs {
			accountIDs = append(accountIDs, input.BucketID)
		}
		for _, output := range req.Outputs {
			if output.BucketID != "" {
				accountIDs = append(accountIDs, output.BucketID)
			}
		}
	}
	projects, err := appdb.ProjectsByAccount(ctx, accountIDs...)
	if err != nil {
		return err
	}
	return errors.WithDetail(projectAuthz(ctx, projects...), "invalid combination of accounts")
}

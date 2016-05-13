package core

import (
	"sort"

	"golang.org/x/net/context"

	"chain/core/appdb"
	"chain/database/pg"
	"chain/errors"
	"chain/net/http/authn"
	"chain/strings"
)

var (
	errNoAccessToResource = errors.New("Resources are not available to user")
	errNotAdmin           = errors.New("Resource is only available to admins")

	EnableCrossProjectXferHack = false
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

// managerAuthz will verify whether this request has access to the provided manager
// node. If the manager node is archived, managerAuthz will return ErrArchived.
func managerAuthz(ctx context.Context, managerID string) error {
	project, err := appdb.ProjectByActiveManager(ctx, managerID)
	if err != nil {
		return err
	}
	return errors.WithDetailf(projectAuthz(ctx, project), "manager node %v", managerID)
}

// accountAuthz will verify whether this request has access to the provided account. If
// the account is archived, accountAuthz will return ErrArchived.
func accountAuthz(ctx context.Context, accountID string) error {
	projects, err := appdb.ProjectsByActiveAccount(ctx, accountID)
	if err != nil {
		return err
	}
	return errors.WithDetailf(projectAuthz(ctx, projects...), "account %v", accountID)
}

// issuerAuthz will verify whether this request has access to the provided issuer node.
// If the issuer node is archived, issuerAuthz will return ErrArchived.
func issuerAuthz(ctx context.Context, issuerID string) error {
	project, err := appdb.ProjectByActiveIssuer(ctx, issuerID)
	if err != nil {
		return err
	}
	return errors.WithDetailf(projectAuthz(ctx, project), "issuer node %v", issuerID)
}

// assetAuthz will verify whether this request has access to the provided asset.
// If the asset is archived, assetAuthz will return ErrArchived.
func assetAuthz(ctx context.Context, assetID string) error {
	projects, err := appdb.ProjectsByActiveAsset(ctx, assetID)
	if err != nil {
		return err
	}
	return errors.WithDetailf(projectAuthz(ctx, projects...), "asset %v", assetID)
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

	accountProjects, err := appdb.ProjectsByActiveAccount(ctx, accountIDs...)
	if errors.Root(err) == pg.ErrUserInputNotFound || errors.Root(err) == appdb.ErrArchived {
		return errors.WithDetailf(errNoAccessToResource, "accounts %+v", accountIDs)
	}
	if err != nil {
		return err
	}

	assetProjects, err := appdb.ProjectsByActiveAsset(ctx, assetIDs...)
	if errors.Root(err) == pg.ErrUserInputNotFound || errors.Root(err) == appdb.ErrArchived {
		return errors.WithDetailf(errNoAccessToResource, "accounts %+v", accountIDs)
	}
	if err != nil {
		return err
	}

	// If EnableCrossProjectXferHack is set, we can ignore authz constraints on
	// number of projects and project membership. As a practical concession,
	// this hack does not account for projects that have been archived.
	if EnableCrossProjectXferHack {
		return nil
	}

	projects := append(accountProjects, assetProjects...)
	sort.Strings(projects)
	projects = strings.Uniq(projects)

	return errors.WithDetail(projectAuthz(ctx, projects...), "invalid combination of accounts or assets")
}

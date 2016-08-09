package core

import (
	"golang.org/x/net/context"

	"chain/core/appdb"
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

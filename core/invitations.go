package core

import (
	"chain/core/appdb"
	"chain/database/pg"

	"golang.org/x/net/context"
)

// POST /v3/invitations
func createInvitation(ctx context.Context, in struct{ Email, Role string }) (interface{}, error) {
	if err := adminAuthz(ctx); err != nil {
		return nil, err
	}
	return appdb.CreateInvitation(ctx, in.Email, in.Role)
}

// POST /v3/invitations/:invID/create-user
func createUserFromInvitation(ctx context.Context, invID string, in struct{ Password string }) (interface{}, error) {
	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer dbtx.Rollback(ctx)

	user, err := appdb.CreateUserFromInvitation(ctx, invID, in.Password)
	if err != nil {
		return nil, err
	}

	err = dbtx.Commit(ctx)
	if err != nil {
		return nil, err
	}

	return user, nil
}

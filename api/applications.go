package api

import (
	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/net/http/authn"
)

// GET /v3/applications
func listApplications(ctx context.Context) ([]*appdb.Application, error) {
	uid := authn.GetAuthID(ctx)
	return appdb.ListApplications(ctx, uid)
}

// POST /v3/applications
func createApplication(ctx context.Context, in struct{ Name string }) (*appdb.Application, error) {
	uid := authn.GetAuthID(ctx)
	return appdb.CreateApplication(ctx, in.Name, uid)
}

// PUT /v3/applications/:appID
func updateApplication(ctx context.Context, aid string, in struct{ Name string }) error {
	return appdb.UpdateApplication(ctx, aid, in.Name)
}

// POST /v3/applications/:appID/members
func addMember(ctx context.Context, aid string, in struct{ Email, Role string }) error {
	user, err := appdb.GetUserByEmail(ctx, in.Email)
	if err != nil {
		return err
	}

	return appdb.AddMember(ctx, aid, user.ID, in.Role)
}

// PUT /v3/applications/:appID/members/:userID
func updateMember(ctx context.Context, aid, memberID string, in struct{ Role string }) error {
	return appdb.UpdateMember(ctx, aid, memberID, in.Role)
}

package api

import (
	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/database/pg"
	"chain/net/http/authn"
)

// GET /v3/projects/:projID
func getProject(ctx context.Context, projID string) (*appdb.Project, error) {
	if err := projectAuthz(ctx, projID); err != nil {
		return nil, err
	}
	return appdb.GetProject(ctx, projID)
}

// GET /v3/projects
func listProjects(ctx context.Context) ([]*appdb.Project, error) {
	uid := authn.GetAuthID(ctx)
	return appdb.ListProjects(ctx, uid)
}

// POST /v3/projects
func createProject(ctx context.Context, in struct{ Name string }) (*appdb.Project, error) {
	uid := authn.GetAuthID(ctx)

	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer dbtx.Rollback(ctx)

	p, err := appdb.CreateProject(ctx, in.Name, uid)
	if err != nil {
		return nil, err
	}

	err = dbtx.Commit(ctx)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// PUT /v3/projects/:projID
func updateProject(ctx context.Context, projID string, in struct{ Name string }) error {
	if err := projectAdminAuthz(ctx, projID); err != nil {
		return err
	}
	return appdb.UpdateProject(ctx, projID, in.Name)
}

// DELETE /v3/projects/:projID
func archiveProject(ctx context.Context, projID string) error {
	if err := projectAdminAuthz(ctx, projID); err != nil {
		return err
	}

	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return err
	}
	defer dbtx.Rollback(ctx)

	err = appdb.ArchiveProject(ctx, projID)
	if err != nil {
		return err
	}

	return dbtx.Commit(ctx)
}

// GET /v3/projects/:projID/members
func listMembers(ctx context.Context, projID string) (interface{}, error) {
	if err := projectAuthz(ctx, projID); err != nil {
		return nil, err
	}
	return appdb.ListMembers(ctx, projID)
}

// POST /v3/projects/:projID/members
func addMember(ctx context.Context, aid string, in struct{ Email, Role string }) error {
	if err := projectAdminAuthz(ctx, aid); err != nil {
		return err
	}
	user, err := appdb.GetUserByEmail(ctx, in.Email)
	if err != nil {
		return err
	}

	return appdb.AddMember(ctx, aid, user.ID, in.Role)
}

// PUT /v3/projects/:projID/members/:userID
func updateMember(ctx context.Context, aid, memberID string, in struct{ Role string }) error {
	if err := projectAdminAuthz(ctx, aid); err != nil {
		return err
	}
	return appdb.UpdateMember(ctx, aid, memberID, in.Role)
}

// DELETE /v3/projects/:projID/members/:userID
func removeMember(ctx context.Context, projID, userID string) error {
	if err := projectAdminAuthz(ctx, projID); err != nil {
		return err
	}
	return appdb.RemoveMember(ctx, projID, userID)
}

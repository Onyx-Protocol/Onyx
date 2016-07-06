package core

import (
	"golang.org/x/net/context"

	"chain/core/appdb"
	"chain/database/pg"
)

// GET /v3/projects/:projID
func getProject(ctx context.Context, projID string) (*appdb.Project, error) {
	return appdb.GetProject(ctx, projID)
}

// GET /v3/projects
func listProjects(ctx context.Context) ([]*appdb.Project, error) {
	return appdb.ListProjects(ctx)
}

// POST /v3/projects
func createProject(ctx context.Context, in struct{ Name string }) (*appdb.Project, error) {
	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer dbtx.Rollback(ctx)

	p, err := appdb.CreateProject(ctx, in.Name)
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
	return appdb.UpdateProject(ctx, projID, in.Name)
}

// DELETE /v3/projects/:projID
// Idempotent
func archiveProject(ctx context.Context, projID string) error {
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

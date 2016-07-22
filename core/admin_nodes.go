package core

import "golang.org/x/net/context"

// GET /v3/projects/:projID/admin-node/summary
func (a *api) getAdminNodeSummary(ctx context.Context, projID string) (interface{}, error) {
	return a.generator.GetSummary(ctx, a.store, a.pool, projID)
}

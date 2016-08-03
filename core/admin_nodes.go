package core

import "golang.org/x/net/context"

// GET /v3/admin-node/summary
func (a *api) getAdminNodeSummary(ctx context.Context) (interface{}, error) {
	return a.generator.GetSummary(ctx, a.store, a.pool)
}

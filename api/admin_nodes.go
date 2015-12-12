package api

import (
	"golang.org/x/net/context"

	"chain/api/admin"
)

// GET /v3/projects/:projID/admin-node/summary
func getAdminNodeSummary(ctx context.Context, projID string) (interface{}, error) {
	if err := projectAuthz(ctx, projID); err != nil {
		return nil, err
	}
	return admin.GetSummary(ctx, projID)
}

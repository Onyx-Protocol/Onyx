package api

import (
	"golang.org/x/net/context"

	"chain/api/appdb"
)

// POST /v3/projects/:projID/admin-nodes
func createAdminNode(ctx context.Context, projID string, req struct{ Label string }) (interface{}, error) {
	if err := projectAuthz(ctx, projID); err != nil {
		return nil, err
	}
	return appdb.InsertAdminNode(ctx, projID, req.Label)
}

// PUT /v3/admin-nodes/:anodeID
func updateAdminNode(ctx context.Context, anodeID string, in struct{ Label *string }) error {
	if err := adminNodeAuthz(ctx, anodeID); err != nil {
		return err
	}
	return appdb.UpdateAdminNode(ctx, anodeID, in.Label)
}

// DELETE /v3/admin-nodes/:anodeID
func deleteAdminNode(ctx context.Context, anodeID string) error {
	if err := adminNodeAuthz(ctx, anodeID); err != nil {
		return err
	}
	return appdb.DeleteAdminNode(ctx, anodeID)
}

// GET /v3/projects/:projID/admin-nodes
func listAdminNodes(ctx context.Context, projID string) (interface{}, error) {
	if err := projectAuthz(ctx, projID); err != nil {
		return nil, err
	}
	return appdb.ListAdminNodes(ctx, projID)
}

// GET /v3/admin-nodes/:anodeID
func getAdminNode(ctx context.Context, anodeID string) (interface{}, error) {
	if err := adminNodeAuthz(ctx, anodeID); err != nil {
		return nil, err
	}
	return appdb.GetAdminNode(ctx, anodeID)
}

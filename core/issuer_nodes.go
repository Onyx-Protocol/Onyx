package core

import (
	"encoding/json"
	"time"

	"golang.org/x/net/context"

	"chain/core/appdb"
	"chain/core/asset"
	"chain/core/issuer"
	"chain/database/pg"
	"chain/errors"
	"chain/metrics"
	"chain/net/http/httpjson"
)

// POST /v3/projects/:projID/issuer-nodes
func createIssuerNode(ctx context.Context, projID string, req map[string]interface{}) (interface{}, error) {
	if err := projectAuthz(ctx, projID); err != nil {
		return nil, err
	}

	_, ok := req["keys"]
	isDeprecated := !ok

	bReq, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "trouble marshaling request")
	}

	var (
		issuerNode interface{}
		cnReq      asset.CreateNodeReq
	)

	if isDeprecated {
		var depReq asset.DeprecatedCreateNodeReq
		err = json.Unmarshal(bReq, &depReq)
		if err != nil {
			return nil, errors.Wrap(err, "invalid node creation request")
		}

		for _, xp := range depReq.XPubs {
			key := &asset.CreateNodeKeySpec{Type: "node", XPub: xp}
			cnReq.Keys = append(cnReq.Keys, key)
		}

		if depReq.GenerateKey {
			key := &asset.CreateNodeKeySpec{Type: "node", Generate: true}
			cnReq.Keys = append(cnReq.Keys, key)
		}

		cnReq.SigsRequired = 1
		cnReq.Label = depReq.Label
	} else {
		err = json.Unmarshal(bReq, &cnReq)
		if err != nil {
			return nil, errors.Wrap(err, "invalid node creation request")
		}
	}

	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "begin dbtx")
	}
	defer dbtx.Rollback(ctx)

	issuerNode, err = asset.CreateNode(ctx, asset.IssuerNode, projID, &cnReq)
	if err != nil {
		return nil, err
	}

	err = dbtx.Commit(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "commit dbtx")
	}

	return issuerNode, nil
}

// GET /v3/projects/:projID/issuer-nodes
func listIssuerNodes(ctx context.Context, projID string) (interface{}, error) {
	if err := projectAuthz(ctx, projID); err != nil {
		return nil, err
	}
	return appdb.ListIssuerNodes(ctx, projID)
}

// GET /v3/issuer-nodes/:inodeID
func getIssuerNode(ctx context.Context, inodeID string) (interface{}, error) {
	if err := issuerAuthz(ctx, inodeID); err != nil {
		return nil, err
	}
	return appdb.GetIssuerNode(ctx, inodeID)
}

// PUT /v3/issuer-nodes/:inodeID
func updateIssuerNode(ctx context.Context, inodeID string, in struct{ Label *string }) error {
	if err := issuerAuthz(ctx, inodeID); err != nil {
		return err
	}
	return appdb.UpdateIssuerNode(ctx, inodeID, in.Label)
}

// DELETE /v3/issuer-nodes/:inodeID
// Idempotent
func archiveIssuerNode(ctx context.Context, inodeID string) error {
	if err := issuerAuthz(ctx, inodeID); errors.Root(err) == appdb.ErrArchived {
		// This issuer node was already archived. Return success.
		return nil
	} else if err != nil {
		return err
	}

	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return err
	}
	defer dbtx.Rollback(ctx)

	err = appdb.ArchiveIssuerNode(ctx, inodeID)
	if err != nil {
		return err
	}

	return dbtx.Commit(ctx)
}

// GET /v3/issuer-nodes/:inodeID/assets
func listAssets(ctx context.Context, inodeID string) (interface{}, error) {
	if err := issuerAuthz(ctx, inodeID); err != nil {
		return nil, err
	}
	prev, limit, err := getPageData(ctx, defAssetPageSize)
	if err != nil {
		return nil, err
	}

	assets, last, err := appdb.ListAssets(ctx, inodeID, prev, limit)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"last":   last,
		"assets": httpjson.Array(assets),
	}
	return ret, nil
}

// POST /v3/issuer-nodes/:inodeID/assets
func createAsset(ctx context.Context, inodeID string, in struct {
	Label      string
	Definition map[string]interface{}

	// ClientToken is the application's unique token for the asset. Every asset
	// within an issuer node should have a unique client token. The client token
	// is used to ensure idempotency of create asset requests. Duplicate create
	// asset requests within the same issuer node with the same client_token will
	// only create one asset.
	ClientToken *string `json:"client_token"`
}) (interface{}, error) {
	defer metrics.RecordElapsed(time.Now())
	if err := issuerAuthz(ctx, inodeID); err != nil {
		return nil, err
	}

	ast, err := issuer.CreateAsset(ctx, inodeID, in.Label, in.Definition, in.ClientToken)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"id":             ast.Hash.String(),
		"issuer_node_id": ast.IssuerNodeID,
		"label":          ast.Label,
	}
	return ret, nil
}

// GET /v3/assets/:assetID
func getAsset(ctx context.Context, assetID string) (interface{}, error) {
	if err := assetAuthz(ctx, assetID); err != nil {
		return nil, err
	}
	return appdb.GetAsset(ctx, assetID)
}

// PUT /v3/assets/:assetID
func updateAsset(ctx context.Context, assetID string, in struct{ Label *string }) error {
	if err := assetAuthz(ctx, assetID); err != nil {
		return err
	}
	return appdb.UpdateAsset(ctx, assetID, in.Label)
}

// DELETE /v3/assets/:assetID
// Idempotent
func archiveAsset(ctx context.Context, assetID string) error {
	if err := assetAuthz(ctx, assetID); errors.Root(err) == appdb.ErrArchived {
		// This asset was already archived. Return success.
		return nil
	} else if err != nil {
		return err
	}
	return appdb.ArchiveAsset(ctx, assetID)
}

// GET /v3/issuer-nodes/:inodeID/activity
func getIssuerNodeActivity(ctx context.Context, inodeID string) (interface{}, error) {
	if err := issuerAuthz(ctx, inodeID); err != nil {
		return nil, err
	}
	prev, limit, err := getPageData(ctx, defActivityPageSize)
	if err != nil {
		return nil, err
	}

	nodeTxs, last, err := appdb.IssuerTxs(ctx, inodeID, prev, limit)
	if err != nil {
		return nil, err
	}

	activity, err := nodeTxsToActivity(nodeTxs)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"last":       last,
		"activities": httpjson.Array(activity),
	}
	return ret, nil
}

// GET /v3/assets/:assetID/activity
func getAssetActivity(ctx context.Context, assetID string) (interface{}, error) {
	if err := assetAuthz(ctx, assetID); err != nil {
		return nil, err
	}
	prev, limit, err := getPageData(ctx, defActivityPageSize)
	if err != nil {
		return nil, err
	}

	nodeTxs, last, err := appdb.AssetTxs(ctx, assetID, prev, limit)
	if err != nil {
		return nil, err
	}

	activity, err := nodeTxsToActivity(nodeTxs)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"last":       last,
		"activities": httpjson.Array(activity),
	}
	return ret, nil
}

// GET /v3/issuer-nodes/:inodeID/transactions
func getIssuerNodeTxs(ctx context.Context, inodeID string) (interface{}, error) {
	if err := issuerAuthz(ctx, inodeID); err != nil {
		return nil, err
	}
	prev, limit, err := getPageData(ctx, defActivityPageSize)
	if err != nil {
		return nil, err
	}

	txs, last, err := appdb.IssuerTxs(ctx, inodeID, prev, limit)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"last":         last,
		"transactions": httpjson.Array(txs),
	}
	return ret, nil
}

// GET /v3/assets/:assetID/transactions
func getAssetTxs(ctx context.Context, assetID string) (interface{}, error) {
	if err := assetAuthz(ctx, assetID); err != nil {
		return nil, err
	}
	prev, limit, err := getPageData(ctx, defActivityPageSize)
	if err != nil {
		return nil, err
	}

	txs, last, err := appdb.AssetTxs(ctx, assetID, prev, limit)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"last":         last,
		"transactions": httpjson.Array(txs),
	}
	return ret, nil
}

package core

import (
	"encoding/json"
	"time"

	"golang.org/x/net/context"

	"chain/core/appdb"
	"chain/core/asset"
	"chain/core/issuer"
	"chain/cos/bc"
	"chain/database/pg"
	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/metrics"
	"chain/net/http/httpjson"
)

// POST /v3/projects/:projID/issuer-nodes
func createIssuerNode(ctx context.Context, projID string, req map[string]interface{}) (interface{}, error) {
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
			return nil, errors.Wrap(err, "invalid asset issuer creation request")
		}

		for _, xp := range depReq.XPubs {
			key := &asset.CreateNodeKeySpec{Type: "service", XPub: xp}
			cnReq.Keys = append(cnReq.Keys, key)
		}

		if depReq.GenerateKey {
			key := &asset.CreateNodeKeySpec{Type: "service", Generate: true}
			cnReq.Keys = append(cnReq.Keys, key)
		}

		cnReq.SigsRequired = 1
		cnReq.Label = depReq.Label
	} else {
		err = json.Unmarshal(bReq, &cnReq)
		if err != nil {
			return nil, errors.Wrap(err, "invalid asset issuer creation request")
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

// assetResponse is a JSON-serializable representation of an asset for
// API responses.
type assetResponse struct {
	ID         bc.AssetID         `json:"id"`
	Label      string             `json:"label"`
	Definition chainjson.HexBytes `json:"definition"`
	Issued     assetAmount        `json:"issued"`
	Retired    assetAmount        `json:"retired"`

	// Deprecated in its current form, which is equivalent to Issued.Total
	Circulation uint64 `json:"circulation"`
}

type assetAmount struct {
	Confirmed uint64 `json:"confirmed"`
	Total     uint64 `json:"total"`
}

func (a *api) issuanceAmounts(ctx context.Context, assetIDs []bc.AssetID) (confirmed, unconfirmed asset.Issuances, err error) {
	confirmed, err = asset.Circulation(ctx, assetIDs...)
	if err != nil {
		return confirmed, unconfirmed, errors.Wrap(err, "fetch asset circulation data")
	}
	unconfirmed, err = asset.PoolIssuances(ctx, a.pool)
	if err != nil {
		return confirmed, unconfirmed, errors.Wrap(err, "fetch asset pool circulation data")
	}
	return confirmed, unconfirmed, nil
}

// GET /v3/assets/:assetID
func (a *api) getIssuerAsset(ctx context.Context, assetID string) (interface{}, error) {
	if err := assetAuthz(ctx, assetID); err != nil {
		return nil, err
	}
	asset, err := appdb.GetAsset(ctx, assetID)
	if err != nil {
		return nil, err
	}

	// Pull in the issuance amounts for the asset too.
	confirmed, unconfirmed, err := a.issuanceAmounts(ctx, []bc.AssetID{asset.ID})
	if err != nil {
		return nil, err
	}
	return assetResponse{
		ID:         asset.ID,
		Label:      asset.Label,
		Definition: asset.Definition,
		Issued: assetAmount{
			Confirmed: confirmed.Assets[asset.ID].Issued,
			Total:     confirmed.Assets[asset.ID].Issued + unconfirmed.Assets[asset.ID].Issued,
		},
		Retired: assetAmount{
			Confirmed: confirmed.Assets[asset.ID].Destroyed,
			Total:     confirmed.Assets[asset.ID].Destroyed + unconfirmed.Assets[asset.ID].Destroyed,
		},
		Circulation: confirmed.Assets[asset.ID].Issued + unconfirmed.Assets[asset.ID].Issued,
	}, nil
}

// GET /v3/issuer-nodes/:inodeID/assets
func (a *api) listAssets(ctx context.Context, inodeID string) (interface{}, error) {
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

	// Query the issuance totals indexes for circulation data.
	assetIDs := make([]bc.AssetID, len(assets))
	for i, a := range assets {
		assetIDs[i] = a.ID
	}
	confirmed, unconfirmed, err := a.issuanceAmounts(ctx, assetIDs)
	if err != nil {
		return nil, err
	}

	responses := make([]assetResponse, 0, len(assets))
	for _, a := range assets {
		resp := assetResponse{
			ID:         a.ID,
			Label:      a.Label,
			Definition: a.Definition,
			Issued: assetAmount{
				Confirmed: confirmed.Assets[a.ID].Issued,
				Total:     confirmed.Assets[a.ID].Issued + unconfirmed.Assets[a.ID].Issued,
			},
			Retired: assetAmount{
				Confirmed: confirmed.Assets[a.ID].Destroyed,
				Total:     confirmed.Assets[a.ID].Destroyed + unconfirmed.Assets[a.ID].Destroyed,
			},
			Circulation: confirmed.Assets[a.ID].Issued + unconfirmed.Assets[a.ID].Issued,
		}
		responses = append(responses, resp)
	}
	ret := map[string]interface{}{
		"last":   last,
		"assets": httpjson.Array(responses),
	}
	return ret, nil
}

// POST /v3/issuer-nodes/:inodeID/assets
func (a *api) createAsset(ctx context.Context, inodeID string, in struct {
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

	genesis, err := a.store.GetBlock(ctx, 1)
	if err != nil {
		return nil, err
	}

	ast, err := issuer.CreateAsset(ctx, inodeID, in.Label, genesis.Hash(), in.Definition, in.ClientToken)
	if err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"id":             ast.Hash.String(),
		"issuer_node_id": ast.IssuerNodeID, // deprecated
		"label":          ast.Label,
	}
	return ret, nil
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

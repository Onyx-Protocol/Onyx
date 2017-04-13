package core

import (
	"chain/encoding/json"
	"chain/net/http/httpjson"
	"context"
)

func (a *API) createGrant(ctx context.Context, x struct {
	GuardType string   `json:"guard_type"`
	GuardData json.Map `json:"guard_data"`
	Policy    string

	ClientToken string `json:"client_token"`
}) error {
	return nil
}

func (a *API) listGrants(ctx context.Context, x requestQuery) (*page, error) {
	limit := x.PageSize
	if limit == 0 {
		limit = defGenericPageSize
	}

	// TODO: replace stubbed data with DB call
	grants := [...]map[string]interface{}{
		{
			"guard_type": "access_token",
			"guard_data": map[string]string{
				"id": "test-token",
			},
			"policy": "client-readwrite",
		},
		{
			"guard_type": "access_token",
			"guard_data": map[string]string{
				"id": "test-token",
			},
			"policy": "network",
		},
		{
			"guard_type": "x509",
			"guard_data": map[string]interface{}{
				"subject": map[string]string{
					"CN": "example.com",
				},
			},
			"policy": "network",
		},
	}

	outQuery := x
	// outQuery.After = next

	return &page{
		Items:    httpjson.Array(grants),
		LastPage: len(grants) < limit,
		Next:     outQuery,
	}, nil
}

func (a *API) revokeGrant(ctx context.Context, x struct{ ID string }) error {
	// TODO replace with DB call
	return nil
}

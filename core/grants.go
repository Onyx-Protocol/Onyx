package core

import (
	"bytes"
	"context"

	"github.com/golang/protobuf/proto"

	"chain/encoding/json"
	"chain/errors"
	"chain/net/http/authz"
	"chain/net/http/httpjson"
)

func (a *API) createGrant(ctx context.Context, x struct {
	GuardType string   `json:"guard_type"`
	GuardData json.Map `json:"guard_data"`
	Policy    string
}) error {
	guardData, err := x.GuardData.MarshalJSON()
	if err != nil {
		// json.Map implementation means this should never happen ¯\_(ツ)_/¯
		return errors.Wrap(err)
	}
	g := authz.Grant{
		GuardType: x.GuardType,
		GuardData: guardData,
		Policy:    x.Policy,
	}

	data, err := a.raftDB.Get(ctx, grantPrefix+x.Policy)
	if err != nil {
		return errors.Wrap(err)
	}
	if data == nil {
		// if there aren't any grants associated with this policy, go ahead
		// and chuck this into raftdb
		gList := &authz.GrantList{
			Grants: []*authz.Grant{&g},
		}
		val, err := proto.Marshal(gList)
		if err != nil {
			return errors.Wrap(err)
		}
		err = a.raftDB.Set(ctx, grantPrefix+x.Policy, val)
		if err != nil {
			return errors.Wrap(err)
		}
		return nil
	}

	grantList := new(authz.GrantList)
	err = proto.Unmarshal(data, grantList)
	if err != nil {
		return errors.Wrap(err)
	}

	grants := grantList.GetGrants()
	for _, existing := range grants {
		if existing.GuardType == x.GuardType && bytes.Equal(existing.GuardData, guardData) {
			// this grant already exists, return for idempotency
			return nil
		}
	}

	// create new grant and append to
	grants = append(grants, &g)
	gList := &authz.GrantList{Grants: grants}
	val, err := proto.Marshal(gList)
	if err != nil {
		return errors.Wrap(err)
	}
	err = a.raftDB.Set(ctx, grantPrefix+x.Policy, val)
	if err != nil {
		return errors.Wrap(err)
	}

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

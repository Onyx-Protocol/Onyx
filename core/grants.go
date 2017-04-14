package core

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/golang/protobuf/proto"

	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/net/http/authz"
	"chain/net/http/httpjson"
)

// an api-friendly representation of a grant
type apiGrant struct {
	GuardType string        `json:"guard_type"`
	GuardData chainjson.Map `json:"guard_data"`
	Policy    string        `json:"policy"`
	CreatedAt string        `json:"created_at"`
}

// ErrMissingTokenID is returned when a token does not exist.
var errMissingTokenID = errors.New("id does not exist")

func (a *API) createGrant(ctx context.Context, x apiGrant) error {
	guardData, err := x.GuardData.MarshalJSON()
	if err != nil {
		// chainjson.Map implementation means this should never happen ¯\_(ツ)_/¯
		return errors.Wrap(err)
	}

	// before we go any further, make sure this token is real
	var gd map[string]string
	err = json.Unmarshal(guardData, &gd)
	if err != nil {
		return errors.Wrap(err)
	}
	id, ok := gd["id"]
	if !ok || !a.accessTokens.CheckID(ctx, id) {
		return errMissingTokenID
	}

	g := authz.Grant{
		GuardType: x.GuardType,
		GuardData: guardData,
		Policy:    x.Policy,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
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

func (a *API) listGrants(ctx context.Context) (map[string]interface{}, error) {
	var grants []apiGrant
	for _, p := range policies {
		// perhaps could denormalize the data in storage to speed this up,
		// but for now assume a small number of grants
		data, err := a.raftDB.Get(ctx, grantPrefix+p)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		if data == nil {
			continue
		}

		grantList := new(authz.GrantList)
		err = proto.Unmarshal(data, grantList)
		if err != nil {
			return nil, errors.Wrap(err)
		}
		for _, g := range grantList.GetGrants() {
			var data chainjson.Map
			err = data.UnmarshalJSON(g.GuardData)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			grant := apiGrant{
				GuardType: g.GuardType,
				GuardData: data,
				Policy:    g.Policy,
				CreatedAt: g.CreatedAt,
			}
			grants = append(grants, grant)
		}
	}

	return map[string]interface{}{
		"items": httpjson.Array(grants),
	}, nil
}

func (a *API) revokeGrant(ctx context.Context, x struct {
	GuardType string        `json:"guard_type"`
	GuardData chainjson.Map `json:"guard_data"`
	Policy    string
}) error {
	guardData, err := x.GuardData.MarshalJSON()
	if err != nil {
		// chainjson.Map implementation means this should never happen ¯\_(ツ)_/¯
		return errors.Wrap(err)
	}

	data, err := a.raftDB.Get(ctx, grantPrefix+x.Policy)
	if err != nil {
		return errors.Wrap(err)
	}
	// If there's nothing to revoke, return success
	if data == nil {
		return nil
	}

	grantList := new(authz.GrantList)
	err = proto.Unmarshal(data, grantList)
	if err != nil {
		return errors.Wrap(err)
	}

	grants := grantList.GetGrants()
	toRemove := -1
	for index, existing := range grants {
		if existing.GuardType == x.GuardType && bytes.Equal(existing.GuardData, guardData) {
			toRemove = index
		}
	}

	// If there's no matching grant, return success
	if toRemove == -1 {
		return nil
	}

	grants = append(grants[:toRemove], grants[toRemove+1:]...)
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

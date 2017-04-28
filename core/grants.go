package core

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/golang/protobuf/proto"

	"chain/errors"
	"chain/net/http/authz"
	"chain/net/http/httpjson"
)

// an api-friendly representation of a grant
type apiGrant struct {
	GuardType string                 `json:"guard_type"`
	GuardData map[string]interface{} `json:"guard_data"`
	Policy    string                 `json:"policy"`
	CreatedAt string                 `json:"created_at"`
}

var (
	// errMissingTokenID is returned when a token does not exist.
	errMissingTokenID = errors.New("id does not exist")

	// errProtectedGrant is returned when a grant is protected and therefore cannot
	// be directly modified or deleted by the user.
	errProtectedGrant = errors.New("this grant is protected")
)

func (a *API) createGrant(ctx context.Context, x apiGrant) (*apiGrant, error) {
	var found bool
	for _, p := range policies {
		if p == x.Policy {
			found = true
			break
		}
	}
	if !found {
		return nil, errors.WithDetail(httpjson.ErrBadRequest, "invalid policy: "+x.Policy)
	}

	if x.GuardType == "access_token" {
		if id, _ := x.GuardData["id"].(string); !a.accessTokens.Exists(ctx, id) {
			return nil, errMissingTokenID
		} else if len(x.GuardData) != 1 {
			return nil, errors.WithDetail(httpjson.ErrBadRequest, `guard data should contain exactly one field, "id"`)
		}
	} else if x.GuardType == "x509" {
		if len(x.GuardData) != 1 {
			return nil, errors.WithDetail(httpjson.ErrBadRequest, `guard data should contain exactly one field, "subject"`)
		} else if subj, ok := x.GuardData["subject"].(map[string]interface{}); ok {
			for k := range subj {
				if !authz.ValidX509SubjectField(k) {
					return nil, errors.WithDetail(httpjson.ErrBadRequest, "bad subject field "+k)
				}
			}
		} else {
			return nil, errors.WithDetail(httpjson.ErrBadRequest, "map of subject attributes required")
		}
	} else {
		return nil, errors.WithDetail(httpjson.ErrBadRequest, "invalid guard type: "+x.GuardType)
	}

	// NOTE: package json produces consistent serialization output,
	// effectively an ad hoc canonical form. We rely on this to
	// grant data for equality.
	// TODO(kr): x509 subject field names are case-insensitive,
	// so we should do our equality comparisons accordingly.
	guardData, err := json.Marshal(x.GuardData)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	params := authz.Grant{
		GuardType: x.GuardType,
		GuardData: guardData,
		Policy:    x.Policy,
	}
	g, err := authz.StoreGrant(ctx, a.raftDB, params, grantPrefix)
	if err != nil {
		return nil, err
	}

	// The guard data comes directly from request input, but go ahead and
	// de-serialize for consistency.
	var data map[string]interface{}
	err = json.Unmarshal(g.GuardData, &data)
	if err != nil {
		return nil, err
	}

	return &apiGrant{
		GuardType: g.GuardType,
		GuardData: data,
		Policy:    g.Policy,
		CreatedAt: g.CreatedAt,
	}, nil
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
		for _, g := range grantList.Grants {
			var data map[string]interface{}
			err = json.Unmarshal(g.GuardData, &data)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			if !g.Protected { // is this right?
				grant := apiGrant{
					GuardType: g.GuardType,
					GuardData: data,
					Policy:    g.Policy,
					CreatedAt: g.CreatedAt,
				}
				grants = append(grants, grant)
			}
		}
	}

	return map[string]interface{}{
		"items": httpjson.Array(grants),
	}, nil
}

func (a *API) deleteGrant(ctx context.Context, x apiGrant) error {
	guardData, err := json.Marshal(x.GuardData)
	if err != nil {
		return errors.Wrap(err)
	}

	data, err := a.raftDB.Get(ctx, grantPrefix+x.Policy)
	if err != nil {
		return errors.Wrap(err)
	}
	// If there's nothing to delete, return success
	if data == nil {
		return nil
	}

	grantList := new(authz.GrantList)
	err = proto.Unmarshal(data, grantList)
	if err != nil {
		return errors.Wrap(err)
	}

	var keep []*authz.Grant
	for _, g := range grantList.Grants {
		if g.GuardType != x.GuardType || !bytes.Equal(g.GuardData, guardData) {
			keep = append(keep, g)
		} else if g.Protected {
			return errProtectedGrant
		}
	}

	// We didn't match any grants, don't need to do an update. Return success
	if len(keep) == len(grantList.Grants) {
		return nil
	}

	gList := &authz.GrantList{Grants: keep}
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

func (a *API) deleteGrantsByAccessToken(ctx context.Context, token string) error {
	for _, p := range policies {
		data, err := a.raftDB.Get(ctx, grantPrefix+p)
		if err != nil {
			return errors.Wrap(err)
		}

		grantList := new(authz.GrantList)
		err = proto.Unmarshal(data, grantList)
		if err != nil {
			return errors.Wrap(err)
		}

		var keep []*authz.Grant
		for _, g := range grantList.Grants {
			if g.GuardType != "access_token" {
				keep = append(keep, g)
			}
			var data map[string]interface{}
			err = json.Unmarshal(g.GuardData, &data)
			if err != nil {
				return errors.Wrap(err)
			}

			if id, _ := data["id"].(string); id != token {
				keep = append(keep, g)
			} else if g.Protected {
				return errProtectedGrant
			}
		}

		// We didn't match any grants, don't need to do an update
		if len(keep) == len(grantList.Grants) {
			continue
		}

		gList := &authz.GrantList{Grants: keep}
		val, err := proto.Marshal(gList)
		if err != nil {
			return errors.Wrap(err)
		}
		err = a.raftDB.Set(ctx, grantPrefix+p, val)
		if err != nil {
			return errors.Wrap(err)
		}
	}
	return nil
}

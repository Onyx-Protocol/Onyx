package core

import (
	"context"
	"crypto/x509/pkix"
	"encoding/json"
	"time"

	"chain/database/sinkdb"
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
	Protected bool                   `json:"protected"`
}

var (
	// errMissingTokenID is returned when a token does not exist.
	errMissingTokenID = errors.New("id does not exist")

	// errProtectedGrant is returned when a grant is protected and therefore cannot
	// be directly deleted by the user.
	errProtectedGrant = errors.New("this grant is protected")

	// errCreateProtectedGrant is returned when a createGrant request is called with
	// a protected grant.
	errCreateProtectedGrant = errors.New("cannot manually create a protected grant")
)

// extraGrantLoader is an authz.Loader that wraps loader with extra grants.
type extraGrantLoader struct {
	loader authz.Loader
	extra  map[string][]*authz.Grant // by policy
}

func (s *extraGrantLoader) Load(ctx context.Context, policy []string) ([]*authz.Grant, error) {
	g, err := s.loader.Load(ctx, policy)
	if err != nil {
		return nil, err
	}
	for _, p := range policy {
		g = append(g, s.extra[p]...)
	}
	return g, nil
}

func grantStore(sdb *sinkdb.DB, extra []*authz.Grant, subj *pkix.Name) authz.Loader {
	ext := map[string][]*authz.Grant{
		"public": {{GuardType: "any", Policy: "public"}},
	}
	for _, g := range extra {
		ext[g.Policy] = append(ext[g.Policy], g)
	}
	if subj != nil {
		ext["internal"] = append(ext["internal"], &authz.Grant{
			Policy:    "internal",
			GuardType: "x509",
			GuardData: encodeX509GuardData(*subj),
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			Protected: true,
		})
	}
	return &extraGrantLoader{
		loader: authz.NewStore(sdb, GrantPrefix),
		extra:  ext,
	}
}

func encodeX509GuardData(subj pkix.Name) []byte {
	v := struct {
		Subject authz.PKIXName `json:"subject"`
	}{authz.PKIXName(subj)}
	d, _ := json.Marshal(v)
	return d
}

func (a *API) createGrant(ctx context.Context, x apiGrant) (*apiGrant, error) {
	if x.Protected {
		return nil, errCreateProtectedGrant
	}

	var found bool
	for _, p := range Policies {
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

	g := &authz.Grant{
		GuardType: x.GuardType,
		GuardData: guardData,
		Policy:    x.Policy,
		Protected: false, // grants created through the createGrant RPC cannot be protected
	}
	err = a.sdb.Exec(ctx, a.grants.Save(ctx, g))
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
		Protected: g.Protected,
	}, nil
}

func (a *API) listGrants(ctx context.Context) (map[string]interface{}, error) {
	var grants []apiGrant
	for _, p := range Policies {
		// perhaps could denormalize the data in storage to speed this up,
		// but for now assume a small number of grants
		var grantList authz.GrantList
		_, err := a.sdb.Get(ctx, GrantPrefix+p, &grantList)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		for _, g := range grantList.Grants {
			var data map[string]interface{}
			err = json.Unmarshal(g.GuardData, &data)
			if err != nil {
				return nil, errors.Wrap(err)
			}

			grant := apiGrant{
				GuardType: g.GuardType,
				GuardData: data,
				Policy:    g.Policy,
				CreatedAt: g.CreatedAt,
				Protected: g.Protected,
			}
			grants = append(grants, grant)
		}
	}

	return map[string]interface{}{
		"items": httpjson.Array(grants),
	}, nil
}

func (a *API) deleteGrant(ctx context.Context, x apiGrant) error {
	if x.Protected {
		return errProtectedGrant
	}
	guardData, err := json.Marshal(x.GuardData)
	if err != nil {
		return errors.Wrap(err)
	}

	toDelete := authz.Grant{
		GuardType: x.GuardType,
		GuardData: guardData,
		Protected: x.Protected, // should always be false
	}

	err = a.sdb.Exec(ctx, a.grants.Delete(x.Policy, func(g *authz.Grant) bool {
		return authz.EqualGrants(*g, toDelete)
	}))
	return errors.Wrap(err)
}

// deleteGrantsByAccessToken returns a sinkdb operation to delete all of the
// grants associated with an access token. It will delete a grant even if that
// grant is protected.
func (a *API) deleteGrantsByAccessToken(token string) sinkdb.Op {
	var ops []sinkdb.Op
	for _, p := range Policies {
		ops = append(ops, a.grants.Delete(p, func(g *authz.Grant) bool {
			if g.GuardType != "access_token" {
				return false
			}
			var data map[string]interface{}
			json.Unmarshal(g.GuardData, &data)
			id, _ := data["id"].(string)
			return id == token
		}))
	}
	return sinkdb.All(ops...)
}

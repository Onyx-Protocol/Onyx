package authz

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/golang/protobuf/proto"

	"chain/database/raft"
	"chain/errors"
	"chain/net/http/authn"
)

var ErrNotAuthorized = errors.New("not authorized")

type Authorizer struct {
	raftDB        *raft.Service
	raftPrefix    string
	policyByRoute map[string][]string
}

func NewAuthorizer(rdb *raft.Service, prefix string, policyMap map[string][]string) *Authorizer {
	return &Authorizer{
		raftDB:        rdb,
		raftPrefix:    prefix,
		policyByRoute: policyMap,
	}
}

func (a *Authorizer) Authorize(req *http.Request) error {
	policies := a.policyByRoute[req.RequestURI]
	if policies == nil || len(policies) == 0 {
		return errors.New("missing policy on this route")
	}

	grants, err := a.grantsByPolicies(policies)
	if err != nil {
		return errors.Wrap(err)
	}

	if !authorized(req.Context(), grants) {
		return ErrNotAuthorized
	}

	return nil
}

func authzToken(ctx context.Context, grants []*Grant) bool {
	for _, g := range grants {
		if g.GuardType == "access_token" && accessTokenGuardData(g) == authn.Token(ctx) {
			return true
		}
	}
	return false
}

func authzLocalhost(ctx context.Context, grants []*Grant) bool {
	for _, g := range grants {
		if g.GuardType == "localhost" {
			return true
		}
	}
	return authn.Localhost(ctx)
}

func accessTokenGuardData(grant *Grant) string {
	var v struct{ ID string }
	json.Unmarshal(grant.GuardData, &v) // ignore error, returns "" on failure
	return v.ID
}

func (a *Authorizer) grantsByPolicies(policies []string) ([]*Grant, error) {
	var grants []*Grant
	for _, p := range policies {
		data := a.raftDB.Stale().Get(a.raftPrefix + p)
		if data != nil {
			grantList := new(GrantList)
			err := proto.Unmarshal(data, grantList)
			if err != nil {
				return nil, errors.Wrap(err)
			}
			grants = append(grants, grantList.GetGrants()...)
		}
	}
	return grants, nil
}

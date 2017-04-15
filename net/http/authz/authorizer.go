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

const grantPrefix = "/core/grant/"

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
	if policies == nil {
		return errors.New("missing policy on this route")
	}

	grants, err := grantsByPolicies(a.raftDB, policies)
	if err != nil {
		return errors.Wrap(err)
	}

	if !authorized(req.Context(), grants) {
		return errors.New("not authorized")
	}

	return nil
}

func authzToken(ctx context.Context, grants []*Grant) bool {
	for _, g := range grants {
		if g.GuardType == "access_token" {
			if accessTokenGuardData(g) == authn.Token(ctx) {
				return true
			}
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
	var data map[string]string
	err := json.Unmarshal(grant.GuardData, &data)
	if err != nil {
		return ""
	}
	token, ok := data["id"]
	if ok {
		return token
	}
	return ""
}

func grantsByPolicies(raftDB *raft.Service, policies []string) ([]*Grant, error) {
	var grants []*Grant
	for _, p := range policies {
		data := raftDB.Stale().Get(grantPrefix + p)
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

package authz

import (
	"context"
	"crypto/x509/pkix"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"

	"chain/database/raft"
	"chain/errors"
	"chain/net/http/authn"
)

var ErrNotAuthorized = errors.New("not authorized")

var builtinGrants = []*Grant{{GuardType: "any", Policy: "public"}}

type Authorizer struct {
	raftDB        *raft.Service
	raftPrefix    string
	policyByRoute map[string][]string
	extraGrants   map[string][]*Grant // by policy
}

func NewAuthorizer(rdb *raft.Service, prefix string, policyMap map[string][]string) *Authorizer {
	a := &Authorizer{
		raftDB:        rdb,
		raftPrefix:    prefix,
		policyByRoute: policyMap,
		extraGrants:   make(map[string][]*Grant),
	}
	for _, g := range builtinGrants {
		a.extraGrants[g.Policy] = append(a.extraGrants[g.Policy], g)
	}
	return a
}

// GrantInternal grants access for subj to policy internal.
// This grant is not stored in raft and applies only for
// the current process.
func (a *Authorizer) GrantInternal(subj pkix.Name) {
	a.extraGrants["internal"] = append(a.extraGrants["internal"], &Grant{
		Policy:    "internal",
		GuardType: "x509",
		GuardData: encodeX509GuardData(subj),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
}

func (a *Authorizer) Authorize(req *http.Request) error {
	policies := a.policyByRoute[strings.TrimRight(req.RequestURI, "/")]
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

func authorized(ctx context.Context, grants []*Grant) bool {
	for _, g := range grants {
		switch g.GuardType {
		case "access_token":
			if accessTokenGuardData(g) == authn.Token(ctx) {
				return true
			}
		case "x509":
			pattern := x509GuardData(g.GuardData)
			certs := authn.X509Certs(ctx)
			if len(certs) > 0 && matchesX509(pattern, certs[0].Subject) {
				return true
			}
		case "localhost":
			if authn.Localhost(ctx) {
				return true
			}
		case "any":
			return true
		}
	}
	return false
}

func accessTokenGuardData(grant *Grant) string {
	var v struct{ ID string }
	json.Unmarshal(grant.GuardData, &v) // ignore error, returns "" on failure
	return v.ID
}

func (a *Authorizer) grantsByPolicies(policies []string) ([]*Grant, error) {
	var grants []*Grant
	for _, p := range policies {
		grants = append(grants, a.extraGrants[p]...)
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

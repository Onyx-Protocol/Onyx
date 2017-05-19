package authz

import (
	"context"
	"encoding/json"
	"net/http"
	"path"

	"chain/errors"
	"chain/net/http/authn"
)

var ErrNotAuthorized = errors.New("not authorized")

// Loader loads all grants for any of the given policies.
type Loader interface {
	Load(ctx context.Context, policy []string) ([]*Grant, error)
}

type Authorizer struct {
	loader   Loader
	policies map[string][]string // by route
}

func NewAuthorizer(l Loader, policyMap map[string][]string) *Authorizer {
	return &Authorizer{
		loader:   l,
		policies: policyMap,
	}
}

func (a *Authorizer) Authorize(req *http.Request) error {
	policies, err := a.policiesByRoute(req.RequestURI)
	if err != nil {
		return errors.Wrap(err)
	}

	grants, err := a.loader.Load(req.Context(), policies)
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

func (a *Authorizer) policiesByRoute(route string) ([]string, error) {
	var (
		n        = 0
		policies []string
	)
	for pattern, pol := range a.policies {
		if !pathMatch(pattern, route) {
			continue
		}
		if len(policies) == 0 || len(pattern) > n {
			n = len(pattern)
			policies = pol
		}
	}

	return policies, nil
}

// Return the canonical path for p, eliminating . and .. elements.
// From the stdlib net/http package.
func cleanPath(p string) string {
	if p == "" {
		return "/"
	}
	if p[0] != '/' {
		p = "/" + p
	}
	np := path.Clean(p)
	// path.Clean removes trailing slash except for root;
	// put the trailing slash back if necessary.
	if p[len(p)-1] == '/' && np != "/" {
		np += "/"
	}
	return np
}

// From the stdlib net/http package.
func pathMatch(pattern, path string) bool {
	if len(pattern) == 0 {
		// should not happen
		return false
	}
	n := len(pattern)
	if pattern[n-1] != '/' {
		return pattern == path
	}
	return len(path) >= n && path[0:n] == pattern
}

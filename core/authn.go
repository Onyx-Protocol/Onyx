package core

import (
	"context"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"

	"chain/core/accesstoken"
	"chain/errors"
	"chain/net/http/authn"
)

const tokenExpiry = time.Minute * 5

type apiAuthn struct {
	config   *Config
	tokenMu  sync.Mutex // protects the following
	tokenMap map[string]tokenResult
}

type tokenResult struct {
	valid      bool
	lastLookup time.Time
}

func (a *apiAuthn) auth(req *http.Request) error {
	typ := "client"
	if strings.HasPrefix(req.URL.Path, networkRPCPrefix) {
		typ = "network"
	}

	// Treat "unconfigured" the same as "configured, but
	// auth is disabled".
	// TODO(kr): remove this a.config==nil check when we
	// switch to localhost auth (which ought to remove
	// the dependency on config entirely here).
	if a.config == nil || !a.config.authEnabled(typ) {
		return nil
	}

	user, pw, _ := req.BasicAuth()
	return a.cachedAuthCheck(req.Context(), typ, user, pw)
}

func authCheck(ctx context.Context, typ, user, pw string) (bool, error) {
	pwBytes, err := hex.DecodeString(pw)
	if err != nil {
		return false, nil
	}
	return accesstoken.Check(ctx, user, typ, pwBytes)
}

func (a *apiAuthn) cachedAuthCheck(ctx context.Context, typ, user, pw string) error {
	a.tokenMu.Lock()
	res, ok := a.tokenMap[typ+user+pw]
	a.tokenMu.Unlock()
	if !ok || time.Now().After(res.lastLookup.Add(tokenExpiry)) {
		valid, err := authCheck(ctx, typ, user, pw)
		if err != nil {
			return errors.Wrap(err)
		}
		res = tokenResult{valid: valid, lastLookup: time.Now()}
		a.tokenMu.Lock()
		a.tokenMap[typ+user+pw] = res
		a.tokenMu.Unlock()
	}
	if !res.valid {
		return authn.ErrNotAuthenticated
	}
	return nil
}

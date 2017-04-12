package core

import (
	"context"
	"encoding/hex"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"chain/core/accesstoken"
	"chain/errors"
)

var errNotAuthenticated = errors.New("not authenticated")

const tokenExpiry = time.Minute * 5

type apiAuthn struct {
	tokens *accesstoken.CredentialStore
	// alternative authentication mechanism,
	// used when no basic auth creds are provided.
	// alt is ignored if nil.
	// this will be removed once ACLs are in place.
	alt func(*http.Request) bool

	tokenMu  sync.Mutex // protects the following
	tokenMap map[string]tokenResult
}

type tokenResult struct {
	valid      bool
	lastLookup time.Time
}

func (a *apiAuthn) handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		token, err0 := a.tokenAuth(req)
		ctx := req.Context()
		if err0 != nil {
			// if this request was successfully authenticated, pass the token along
			if token != "" {
				ctx = context.WithValue(ctx, "token", token)
			}
		}

		err1 := a.localhostAuth(req)
		if err1 == nil {
			ctx = context.WithValue(ctx, "localhost", true)
		}

		// TODO(tessr): move this to authz as part of ACL work
		if err0 != nil {
			errorFormatter.Write(ctx, rw, err0)
		}

		next.ServeHTTP(rw, req.WithContext(ctx))
	})
}

func (a *apiAuthn) localhostAuth(req *http.Request) error {
	h, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return errors.Wrap(err)
	}
	if !net.ParseIP(h).IsLoopback() {
		return errNotAuthenticated
	}
	return nil
}

func (a *apiAuthn) tokenAuth(req *http.Request) (string, error) {
	user, pw, ok := req.BasicAuth()
	// TODO(tessr): remove the following clause once ACLs are in place
	if !ok && a.alt != nil && a.alt(req) {
		return "", nil
	}

	// Is this the way we want to encode the token to be passed around? Or just user?
	// Or something just like pulling the Authorization string out of the request header?
	token := user + ":" + pw
	typ := "client"
	if strings.HasPrefix(req.URL.Path, networkRPCPrefix) {
		typ = "network"
	}
	return token, a.cachedAuthCheck(req.Context(), typ, user, pw)
}

func (a *apiAuthn) authCheck(ctx context.Context, typ, user, pw string) (bool, error) {
	pwBytes, err := hex.DecodeString(pw)
	if err != nil {
		return false, nil
	}
	return a.tokens.Check(ctx, user, typ, pwBytes)
}

func (a *apiAuthn) cachedAuthCheck(ctx context.Context, typ, user, pw string) error {
	a.tokenMu.Lock()
	res, ok := a.tokenMap[typ+user+pw]
	a.tokenMu.Unlock()
	if !ok || time.Now().After(res.lastLookup.Add(tokenExpiry)) {
		valid, err := a.authCheck(ctx, typ, user, pw)
		if err != nil {
			return errors.Wrap(err)
		}
		res = tokenResult{valid: valid, lastLookup: time.Now()}
		a.tokenMu.Lock()
		a.tokenMap[typ+user+pw] = res
		a.tokenMu.Unlock()
	}
	if !res.valid {
		return errNotAuthenticated
	}
	return nil
}

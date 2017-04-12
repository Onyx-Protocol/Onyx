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
)

var errNotAuthenticated = errors.New("not authenticated")

const tokenExpiry = time.Minute * 5

type apiAuthn struct {
	tokens *accesstoken.CredentialStore
	// alternative authentication mechanism,
	// used when no basic auth creds are provided.
	// alt is ignored if nil.
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
		token, err := a.auth(req)
		if err != nil {
			errorFormatter.Write(req.Context(), rw, err)
			return
		}

		// if this request was successfully authenticated, pass the token along
		ctx := req.Context()
		if token != "" {
			ctx = context.WithValue(ctx, "token", token)
		}
		next.ServeHTTP(rw, req.WithContext(ctx))
	})
}

func (a *apiAuthn) auth(req *http.Request) (string, error) {
	user, pw, ok := req.BasicAuth()
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

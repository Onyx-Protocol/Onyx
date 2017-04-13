package authn

import (
	"context"
	"encoding/hex"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"chain/core/accesstoken"
	"chain/errors"
	"chain/net/authz"
)

const tokenExpiry = time.Minute * 5

type API struct {
	Tokens           *accesstoken.CredentialStore
	NetworkRPCPrefix string

	tokenMu  sync.Mutex // protects the following
	TokenMap map[string]TokenResult
}

type TokenResult struct {
	valid      bool
	lastLookup time.Time
}

// Authenticate returns the request, with added tokens and/or localhost
// flags in the context, as appropriate.
func (a *API) Authenticate(req *http.Request) *http.Request {
	ctx := req.Context()
	token, err := a.tokenAuthn(req)
	if err == nil && token != "" {
		// if this request was successfully authenticated with a token, pass the token along
		ctx = authz.NewContextWithToken(ctx, token)
	}

	local := a.localhostAuthn(req)
	if local {
		log.Println("authenticated as local connection")
		ctx = authz.NewContextWithLocalhost(ctx)
	} else {
		log.Println("not a local connection")
	}

	return req.WithContext(ctx)
}

// returns true if this request is coming from a loopback address
func (a *API) localhostAuthn(req *http.Request) bool {
	h, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return false
	}
	if !net.ParseIP(h).IsLoopback() {
		return false
	}
	return true
}

func (a *API) tokenAuthn(req *http.Request) (string, error) {
	user, pw, ok := req.BasicAuth()
	if !ok {
		return "", errors.New("no token")
	}
	typ := "client"
	if strings.HasPrefix(req.URL.Path, a.NetworkRPCPrefix) {
		typ = "network"
	}
	return user, a.cachedTokenAuthnCheck(req.Context(), typ, user, pw)
}

func (a *API) tokenAuthnCheck(ctx context.Context, typ, user, pw string) (bool, error) {
	pwBytes, err := hex.DecodeString(pw)
	if err != nil {
		return false, nil
	}
	return a.Tokens.Check(ctx, user, typ, pwBytes)
}

func (a *API) cachedTokenAuthnCheck(ctx context.Context, typ, user, pw string) error {
	a.tokenMu.Lock()
	res, ok := a.TokenMap[typ+user+pw]
	a.tokenMu.Unlock()
	if !ok || time.Now().After(res.lastLookup.Add(tokenExpiry)) {
		valid, err := a.tokenAuthnCheck(ctx, typ, user, pw)
		if err != nil {
			return errors.Wrap(err)
		}
		res = TokenResult{valid: valid, lastLookup: time.Now()}
		a.tokenMu.Lock()
		a.TokenMap[typ+user+pw] = res
		a.tokenMu.Unlock()
	}
	if !res.valid {
		return errors.New("invalid token")
	}
	return nil
}

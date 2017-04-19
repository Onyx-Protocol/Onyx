package authn

import (
	"context"
	"encoding/hex"
	"net"
	"net/http"
	"sync"
	"time"

	"chain/core/accesstoken"
	"chain/errors"
)

const tokenExpiry = time.Minute * 5

type API struct {
	tokens           *accesstoken.CredentialStore
	networkRPCPrefix string

	tokenMu  sync.Mutex // protects the following
	tokenMap map[string]tokenResult
}

type tokenResult struct {
	valid      bool
	lastLookup time.Time
}

func NewAPI(tokens *accesstoken.CredentialStore, networkPrefix string) *API {
	return &API{
		tokens:           tokens,
		networkRPCPrefix: networkPrefix,
		tokenMap:         make(map[string]tokenResult),
	}
}

// Authenticate returns the request, with added tokens and/or localhost
// flags in the context, as appropriate.
func (a *API) Authenticate(req *http.Request) (*http.Request, error) {
	ctx := CertAuthn(req)

	token, err := a.tokenAuthn(req)
	if err == nil && token != "" {
		// if this request was successfully authenticated with a token, pass the token along
		ctx = newContextWithToken(ctx, token)
	}

	local := a.localhostAuthn(req)
	if local {
		ctx = newContextWithLocalhost(ctx)
	}

	// if there is no authentication at all, we return an "unauthenticated" error,
	// which may be helpful when debugging
	if len(X509Certs(ctx)) < 1 && err != nil && !local {
		return req, errors.New("unauthenticated")
	}

	return req.WithContext(ctx), nil
}

// checks the request for a valid client cert list.
// If found, it is added to the request's context.
// This function is exported so it can be used when
// a full API is not available.
func CertAuthn(req *http.Request) context.Context {
	if req.TLS != nil && len(req.TLS.PeerCertificates) > 0 {
		return context.WithValue(req.Context(), x509CertsKey, req.TLS.PeerCertificates)
	}
	return req.Context()
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
	return user, a.cachedTokenAuthnCheck(req.Context(), user, pw)
}

func (a *API) tokenAuthnCheck(ctx context.Context, user, pw string) (bool, error) {
	pwBytes, err := hex.DecodeString(pw)
	if err != nil {
		return false, nil
	}
	return a.tokens.Check(ctx, user, pwBytes)
}

func (a *API) cachedTokenAuthnCheck(ctx context.Context, user, pw string) error {
	a.tokenMu.Lock()
	res, ok := a.tokenMap[user+pw]
	a.tokenMu.Unlock()
	if !ok || time.Now().After(res.lastLookup.Add(tokenExpiry)) {
		valid, err := a.tokenAuthnCheck(ctx, user, pw)
		if err != nil {
			return errors.Wrap(err)
		}
		res = tokenResult{valid: valid, lastLookup: time.Now()}
		a.tokenMu.Lock()
		a.tokenMap[user+pw] = res
		a.tokenMu.Unlock()
	}
	if !res.valid {
		return errors.New("invalid token")
	}
	return nil
}

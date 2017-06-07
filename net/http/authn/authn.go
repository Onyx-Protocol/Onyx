package authn

import (
	"context"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"chain/core/accesstoken"
	"chain/errors"
)

const tokenExpiry = time.Minute * 5

// TODO(kr): This a hack. Please revisit this soon.
// When compiled without localhost_auth, we want to avoid
// running the loopback authenticator at all, except for
// requests to /dashboard/, where we *do* want to run it.
var loopbackOn = false

type API struct {
	tokens             *accesstoken.CredentialStore
	crosscoreRPCPrefix string
	rootCAs            *x509.CertPool

	tokenMu  sync.Mutex // protects the following
	tokenMap map[string]tokenResult
}

type tokenResult struct {
	valid      bool
	lastLookup time.Time
}

func NewAPI(tokens *accesstoken.CredentialStore, crosscorePrefix string, rootCAs *x509.CertPool) *API {
	return &API{
		tokens:             tokens,
		crosscoreRPCPrefix: crosscorePrefix,
		tokenMap:           make(map[string]tokenResult),
		rootCAs:            rootCAs,
	}
}

// Authenticate returns the request, with added tokens and/or localhost
// flags in the context, as appropriate.
func (a *API) Authenticate(req *http.Request) (*http.Request, error) {
	var authnErrors []string

	ctx, err := certAuthn(req, a.rootCAs)
	if err != nil {
		authnErrors = append(authnErrors, err.Error())
	}

	token, err := a.tokenAuthn(req)
	if err != nil {
		authnErrors = append(authnErrors, err.Error())
	} else if token != "" {
		// if this request was successfully authenticated with a token, pass the token along
		ctx = newContextWithToken(ctx, token)
	}

	local := a.localhostAuthn(req)
	if local {
		ctx = newContextWithLocalhost(ctx)
	}

	// Temporary workaround. Dashboard is always ok.
	// See loopbackOn comment above.
	if strings.HasPrefix(req.URL.Path, "/dashboard/") || req.URL.Path == "/dashboard" {
		return req.WithContext(ctx), nil
	}
	if loopbackOn && local {
		return req.WithContext(ctx), nil
	}

	// if there is no authentication at all, we return an "unauthenticated" error,
	// which may be helpful when debugging
	if len(X509Certs(ctx)) < 1 && Token(ctx) == "" {
		err := errors.New("unauthenticated")
		if len(authnErrors) > 0 {
			err = errors.WithDetailf(err, "Invalid credentials: %s", strings.Join(authnErrors, "; "))
		} else {
			err = errors.WithDetail(err, "No authentication credentials provided.")
		}
		return req, err
	}

	return req.WithContext(ctx), nil
}

// checks the request for a valid client cert list.
// If found, it is added to the request's context.
// Note that an *invalid* client cert is treated the
// same as no client cert -- it is omitted from the
// returned context, but the returned error is non-nil.
// The caller should allow the connection to proceed
// even if the error is non-nil.
func certAuthn(req *http.Request, rootCAs *x509.CertPool) (context.Context, error) {
	if req.TLS != nil && len(req.TLS.PeerCertificates) > 0 {
		certs := req.TLS.PeerCertificates

		// Same logic as serverHandshakeState.processCertsFromClient
		// in $GOROOT/src/crypto/tls/handshake_server.go.
		opts := x509.VerifyOptions{
			Roots:         rootCAs,
			CurrentTime:   time.Now(),
			Intermediates: x509.NewCertPool(),
			KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}
		for _, cert := range certs[1:] {
			opts.Intermediates.AddCert(cert)
		}
		_, err := certs[0].Verify(opts)
		if err != nil {
			// crypto/tls treats this as an error:
			// errors.New("tls: failed to verify client's certificate: " + err.Error())
			// For us, it is ok; we want to treat it the same as if there
			// were no client cert presented.
			return req.Context(), err
		}

		return context.WithValue(req.Context(), x509CertsKey, certs), nil
	}
	return req.Context(), nil
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
		return "", nil
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
		return fmt.Errorf("invalid token: %q", user)
	}
	return nil
}

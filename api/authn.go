package api

import (
	"crypto/subtle"
	"database/sql"
	"time"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/errors"
	chainhttp "chain/net/http"
	"chain/net/http/authn"
)

var tokenCache *authn.TokenCache

func init() {
	tokenCache = authn.NewTokenCache()
}

func userCredsAuthn(f chainhttp.HandlerFunc) chainhttp.HandlerFunc {
	return authn.BasicHandler{
		Auth:  appdb.AuthenticateUserCreds,
		Next:  f,
		Realm: "x.chain.com",
	}.ServeHTTPContext
}

func nouserAuthn(secret string, f chainhttp.HandlerFunc) chainhttp.HandlerFunc {
	return authn.BasicHandler{
		Auth: func(_ context.Context, _, p string) (string, error) {
			if subtle.ConstantTimeCompare([]byte(p), []byte(secret)) != 1 {
				return "", authn.ErrNotAuthenticated
			}
			return "", nil
		},
		Next:  f,
		Realm: "x.chain.com",
	}.ServeHTTPContext
}

func tokenAuthn(f chainhttp.HandlerFunc) chainhttp.HandlerFunc {
	return authn.BasicHandler{
		Auth:  authenticateToken,
		Next:  f,
		Realm: "x.chain.com",
	}.ServeHTTPContext
}

func authenticateToken(ctx context.Context, id, secret string) (userID string, err error) {
	if cached := tokenCache.Get(id, secret); cached != "" {
		return cached, nil
	}

	secretHash, userID, exp, err := appdb.GetAuthToken(ctx, id)
	if errors.Root(err) == sql.ErrNoRows {
		return "", authn.ErrNotAuthenticated
	} else if err != nil {
		return "", err
	}

	if !exp.IsZero() && exp.Before(time.Now()) {
		return "", authn.ErrNotAuthenticated
	}

	if bcrypt.CompareHashAndPassword(secretHash, []byte(secret)) != nil {
		return "", authn.ErrNotAuthenticated
	}

	tokenCache.Store(id, secret, userID, exp)

	return userID, nil
}

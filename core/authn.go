package core

import (
	"context"
	"encoding/hex"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/metadata"

	"chain/core/accesstoken"
	"chain/errors"
)

var errNotAuthenticated = errors.New("not authenticated")

const (
	tokenExpiry = time.Minute * 5
	userKey     = iota
	pwKey
)

type apiAuthn struct {
	tokens *accesstoken.CredentialStore
	// alternative authentication mechanism,
	// used when no basic auth creds are provided.
	alt func(context.Context) bool

	tokenMu  sync.Mutex // protects the following
	tokenMap map[string]tokenResult
}

type tokenResult struct {
	valid      bool
	lastLookup time.Time
}

func (a *apiAuthn) authRPC(ctx context.Context, method string) (context.Context, error) {
	md, ok := metadata.FromContext(ctx)
	if !ok {
		if a.alt(ctx) {
			return ctx, nil
		}
		return ctx, errNotAuthenticated
	}

	var user, pw string
	if len(md["username"]) > 0 && len(md["password"]) > 0 {
		user = md["username"][0]
		pw = md["password"][0]
	} else if len(md["username"]) == 0 && len(md["password"]) == 0 && a.alt(ctx) {
		return ctx, nil
	}

	typ := "client"

	if strings.HasPrefix(method, "/pb.Node/") || strings.HasPrefix(method, "/pb.Signer/") {
		typ = "network"
	}

	ctx = context.WithValue(ctx, userKey, user)
	ctx = context.WithValue(ctx, pwKey, pw)

	return ctx, a.cachedAuthCheck(ctx, typ, user, pw)
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

func userPwFromContext(ctx context.Context) (user string, pw string) {
	return ctx.Value(userKey).(string), ctx.Value(pwKey).(string)
}

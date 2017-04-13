package authz

import "context"

type key int

const (
	tokenKey key = iota
	localhostKey
)

func authzToken(ctx context.Context) bool {
	// TODO(tessr): compare against Policies
	_, ok := TokenFromContext(ctx)
	return ok
}

func authzLocalhost(ctx context.Context) bool {
	// TODO(tessr): compare against Policies
	return LocalhostFromContext(ctx)
}

// NewContextWithToken sets the token in a new context and returns the context.
func NewContextWithToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, tokenKey, token)
}

// TokenFromContext returns the token stored in the context, if there is one.
func TokenFromContext(ctx context.Context) (string, bool) {
	t, ok := ctx.Value(tokenKey).(string)
	return t, ok
}

// NewContextWithLocalhost sets the localhost flag to `true` in a new context
// and returns that context.
func NewContextWithLocalhost(ctx context.Context) context.Context {
	return context.WithValue(ctx, localhostKey, true)
}

// LocalhostFromContext returns true if the localhost flag has been set.
func LocalhostFromContext(ctx context.Context) bool {
	l, ok := ctx.Value(localhostKey).(bool)
	if ok && l {
		return true
	}
	return false
}

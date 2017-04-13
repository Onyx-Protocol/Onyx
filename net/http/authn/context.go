package authn

import "context"

type key int

const (
	tokenKey key = iota
	localhostKey
)

// newContextWithToken sets the token in a new context and returns the context.
func newContextWithToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, tokenKey, token)
}

// Token returns the token stored in the context, if there is one.
func Token(ctx context.Context) string {
	t, ok := ctx.Value(tokenKey).(string)
	if !ok {
		return ""
	}
	return t
}

// newContextWithLocalhost sets the localhost flag to `true` in a new context
// and returns that context.
func newContextWithLocalhost(ctx context.Context) context.Context {
	return context.WithValue(ctx, localhostKey, true)
}

// Localhost returns true if the localhost flag has been set.
func Localhost(ctx context.Context) bool {
	l, ok := ctx.Value(localhostKey).(bool)
	if ok && l {
		return true
	}
	return false
}

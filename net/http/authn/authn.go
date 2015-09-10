package authn

import (
	"net/http"

	"golang.org/x/net/context"

	"chain/errors"
	"chain/log"
	chainhttp "chain/net/http"
)

// key is an unexported type for keys defined in this package.
// This prevents collisions with keys defined in other packages.
type key int

// authIDKey is the key for authentication identifiers in Contexts. It is
// unexported; clients use GetAuthID instead of using this key directly.
var authIDKey key

// AuthFunc describes any function that takes a standard username/password pair
// and attempts to perform authentication. If authentication is successful, a
// string uniquely identifying the authenticated resource (usually a user ID)
// should be returned. When used in conjunction with BasicHandler, returning
// ErrNotAuthenticated from an AuthFunc will cause a 401 response to be written.
// Any other error will cause a 500 response.
type AuthFunc func(ctx context.Context, username, password string) (authID string, err error)

// ErrNotAuthenticated should be returned by an AuthFunc if the provided
// credentials are invalid.
var ErrNotAuthenticated = errors.New("not authenticated")

// BasicHandler provides token authentication via HTTP basic auth. If the
// provided token is valid, then the corresponding user ID will be inserted into
// the context. The user ID should be retrieved using authn.GetAuthID.
// BasicHandler satisfies the ContextHandler interface.
type BasicHandler struct {
	Auth AuthFunc
	Next chainhttp.Handler
}

// ServeHTTPContext satisfies the ContextHandler interface.
func (h BasicHandler) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	username, password, _ := req.BasicAuth()
	authID, err := h.Auth(ctx, username, password)
	if err == nil {
		ctx = NewContext(ctx, authID)
		h.Next.ServeHTTPContext(ctx, w, req)
	} else if err == ErrNotAuthenticated {
		log.Write(ctx,
			"status", http.StatusUnauthorized,
			log.KeyError, err,
		)
		http.Error(w, "Request could not be authenticated", http.StatusUnauthorized)
	} else {
		log.Write(ctx,
			"status", http.StatusInternalServerError,
			log.KeyError, err,
			log.KeyStack, errors.Stack(err),
		)
		http.Error(w, "Internal error", http.StatusInternalServerError)
	}
}

// NewContext returns a new Context that carries value authID.
func NewContext(ctx context.Context, authID string) context.Context {
	return context.WithValue(ctx, authIDKey, authID)
}

// GetAuthID retrieves the identifier set by an authentication handler.
func GetAuthID(ctx context.Context) string {
	id, _ := ctx.Value(authIDKey).(string)
	return id
}

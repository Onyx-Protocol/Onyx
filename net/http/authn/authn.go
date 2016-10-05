package authn

import (
	"net/http"

	"chain/errors"
	"chain/log"
)

// AuthFunc describes any function that takes an HTTP request
// and attempts to perform authentication. When used in conjunction with
// BasicHandler, returning ErrNotAuthenticated from an AuthFunc will cause a 401
// response to be written.
// Any other error will cause a 500 response.
type AuthFunc func(*http.Request) error

// ErrNotAuthenticated should be returned by an AuthFunc if the provided
// credentials are invalid.
var ErrNotAuthenticated = errors.New("not authenticated")

// BasicHandler provides token authentication via HTTP basic auth. If the
// provided token is valid, then the corresponding user ID will be inserted into
// the context. The user ID should be retrieved using authn.GetAuthID.
// BasicHandler satisfies the ContextHandler interface.
type BasicHandler struct {
	Auth  AuthFunc
	Realm string
	Next  http.Handler
}

// ServeHTTP satisfies http.Handler.
func (h BasicHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	err := h.Auth(req)
	if err == nil {
		h.Next.ServeHTTP(w, req)
	} else if err == ErrNotAuthenticated {
		log.Write(req.Context(),
			"status", http.StatusUnauthorized,
			log.KeyError, err,
		)
		if u, _, _ := req.BasicAuth(); u == "" {
			w.Header().Add("WWW-Authenticate", `Basic realm="`+h.Realm+`"`)
		}
		http.Error(w, `{"message": "Request could not be authenticated"}`, http.StatusUnauthorized)
	} else {
		log.Write(req.Context(),
			"status", http.StatusInternalServerError,
			log.KeyError, err,
			log.KeyStack, errors.Stack(err),
		)
		http.Error(w, `{"message": "Internal error"}`, http.StatusInternalServerError)
	}
}

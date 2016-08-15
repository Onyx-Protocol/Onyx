package authn

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/net/context"
)

type handlerFunc func(context.Context, http.ResponseWriter, *http.Request)

func (f handlerFunc) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	f(ctx, w, req)
}

func alwaysSuccess(ctx context.Context, u, p string) error     { return nil }
func alwaysNotAuth(ctx context.Context, u, p string) error     { return ErrNotAuthenticated }
func alwaysInternalErr(ctx context.Context, u, p string) error { return errors.New("") }

func TestBasicHandler(t *testing.T) {
	h := BasicHandler{
		Auth: alwaysSuccess,
		Next: handlerFunc(func(context.Context, http.ResponseWriter, *http.Request) {}),
	}
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/whatever", nil)

	h.ServeHTTPContext(context.Background(), rec, req)

	if rec.Code != 200 {
		t.Errorf("status = %v want 200", rec.Code)
	}
}

func TestBasicHandlerNotAuthenticated(t *testing.T) {
	h := BasicHandler{Auth: alwaysNotAuth, Realm: "test-realm"}
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/whatever", nil)

	h.ServeHTTPContext(context.Background(), rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %v want %v", rec.Code, http.StatusUnauthorized)
	}

	if rec.Header().Get("WWW-Authenticate") != `Basic realm="test-realm"` {
		t.Errorf("got WWW-Authenticate header = %#q want %#q",
			rec.Header().Get("WWW-Authenticate"), `Basic realm="test-realm"`)
	}
}

func TestBasicHandlerInternalError(t *testing.T) {
	h := BasicHandler{Auth: alwaysInternalErr}
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/whatever", nil)

	h.ServeHTTPContext(context.Background(), rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %v want %v", rec.Code, http.StatusInternalServerError)
	}
}

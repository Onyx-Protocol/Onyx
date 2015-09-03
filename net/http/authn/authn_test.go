package authn

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/net/context"

	chainhttp "chain/net/http"
)

func alwaysSuccess(ctx context.Context, u, p string) (string, error)     { return "sample-auth-id", nil }
func alwaysNotAuth(ctx context.Context, u, p string) (string, error)     { return "", ErrNotAuthenticated }
func alwaysInternalErr(ctx context.Context, u, p string) (string, error) { return "", errors.New("") }

func TestBasicHandler(t *testing.T) {
	var authID string

	h := BasicHandler{
		Auth: alwaysSuccess,
		Next: chainhttp.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, req *http.Request) {
			authID = GetAuthID(ctx)
		}),
	}
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/whatever", nil)

	h.ServeHTTPContext(context.Background(), rec, req)

	if rec.Code != 200 {
		t.Errorf("status = %v want 200", rec.Code)
	}

	if authID != "sample-auth-id" {
		t.Errorf("authenticated ID = %v want sample-auth-id", authID)
	}
}

func TestBasicHandlerNotAuthenticated(t *testing.T) {
	h := BasicHandler{Auth: alwaysNotAuth}
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/whatever", nil)

	h.ServeHTTPContext(context.Background(), rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %v want %v", rec.Code, http.StatusUnauthorized)
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

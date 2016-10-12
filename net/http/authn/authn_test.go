package authn

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func alwaysSuccess(*http.Request) error     { return nil }
func alwaysNotAuth(*http.Request) error     { return ErrNotAuthenticated }
func alwaysInternalErr(*http.Request) error { return errors.New("") }

func TestBasicHandler(t *testing.T) {
	h := BasicHandler{
		Auth: alwaysSuccess,
		Next: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
	}
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/whatever", nil)

	h.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("status = %v want 200", rec.Code)
	}
}

func TestBasicHandlerNotAuthenticated(t *testing.T) {
	h := BasicHandler{Auth: alwaysNotAuth}
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/whatever", nil)

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %v want %v", rec.Code, http.StatusUnauthorized)
	}
}

func TestBasicHandlerInternalError(t *testing.T) {
	h := BasicHandler{Auth: alwaysInternalErr}
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/whatever", nil)

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %v want %v", rec.Code, http.StatusInternalServerError)
	}
}

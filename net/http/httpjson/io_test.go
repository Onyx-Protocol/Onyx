package httpjson

import (
	"bytes"
	"context"
	"errors"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"chain/log"
)

func TestWriteArray(t *testing.T) {
	examples := []struct {
		in   []int
		want string
	}{
		{nil, "[]"},
		{[]int{}, "[]"},
		{make([]int, 0), "[]"},
	}

	for _, ex := range examples {
		rec := httptest.NewRecorder()
		Write(context.Background(), rec, 200, ex.in)
		got := strings.TrimSpace(rec.Body.String())
		if got != ex.want {
			t.Errorf("Write(%v) = %v want %v", ex.in, got, ex.want)
		}
	}
}

func TestWriteErr(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	want := "test-error"

	ctx := context.Background()
	resp := &errResponse{httptest.NewRecorder(), errors.New(want)}
	Write(ctx, resp, 200, "ok")
	got := buf.String()
	if !strings.Contains(got, want) {
		t.Errorf("log = %v; should contain %q", got, want)
	}
}

type errResponse struct {
	*httptest.ResponseRecorder
	err error
}

func (r *errResponse) Write([]byte) (int, error) {
	return 0, r.err
}

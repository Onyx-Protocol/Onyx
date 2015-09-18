package httpjson

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"

	"golang.org/x/net/context"

	"chain/errors"
	"chain/log"
)

// ErrBadRequest indicates the user supplied malformed JSON input,
// possibly including a datatype that doesn't match what we expected.
var ErrBadRequest = errors.New("httpjson: bad request")

// Read decodes a single JSON text from r into v.
// The only error it returns is ErrBadRequest
// (wrapped with the original error message as context).
func Read(ctx context.Context, r io.Reader, v interface{}) error {
	err := json.NewDecoder(r).Decode(v)
	if err != nil {
		return errors.Wrap(ErrBadRequest, err.Error())
	}
	return nil
}

// Write sets the Content-Type header field to indicate
// JSON data, writes the header using status,
// then writes v to w.
// It logs any error encountered during the write.
func Write(ctx context.Context, w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	// Make sure to render nil slices as "[]", rather than "null"
	if rv := reflect.ValueOf(v); rv.Kind() == reflect.Slice && rv.IsNil() {
		v = []struct{}{}
	}

	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		log.Error(ctx, err)
	}
}

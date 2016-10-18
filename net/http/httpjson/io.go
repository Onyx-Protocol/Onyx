package httpjson

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"

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
	dec := json.NewDecoder(r)
	dec.UseNumber()
	err := dec.Decode(v)
	if err != nil {
		detail := errors.Detail(err)
		if detail == "" {
			detail = "check request parameters for missing and/or incorrect values"
		}
		return errors.WithDetail(ErrBadRequest, err.Error()+": "+detail)
	}
	return err
}

// Write sets the Content-Type header field to indicate
// JSON data, writes the header using status,
// then writes v to w.
// It logs any error encountered during the write.
func Write(ctx context.Context, w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	err := json.NewEncoder(w).Encode(Array(v))
	if err != nil {
		log.Error(ctx, err)
	}
}

// Array returns an empty JSON array if v is a nil slice,
// so that it renders as "[]" rather than "null".
// Otherwise, it returns v.
func Array(v interface{}) interface{} {
	if rv := reflect.ValueOf(v); rv.Kind() == reflect.Slice && rv.IsNil() {
		v = []struct{}{}
	}
	return v
}

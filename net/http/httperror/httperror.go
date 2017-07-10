// Package httperror defines the format for HTTP error responses
// from Chain services.
package httperror

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"chain/errors"
	"chain/log"
	"chain/net/http/httpjson"
)

func init() {
	log.SkipFunc("chain/net/http/httperror.Formatter.Log")
	log.SkipFunc("chain/net/http/httperror.Formatter.Write")
}

// Info contains a set of error codes to send to the user.
type Info struct {
	HTTPStatus int    `json:"-"`
	ChainCode  string `json:"code"`
	Message    string `json:"message"`
}

// Response defines the error response for a Chain error.
type Response struct {
	Info
	Detail    string                 `json:"detail,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Temporary bool                   `json:"temporary"`
}

// Parse reads an error Response from the provided reader.
func Parse(r io.Reader) (*Response, bool) {
	var resp Response
	err := json.NewDecoder(r).Decode(&resp)
	if err != nil || resp.ChainCode == "" {
		return nil, false
	}
	return &resp, true
}

// Formatter defines rules for mapping errors to the Chain error
// response format.
type Formatter struct {
	Default     Info
	IsTemporary func(info Info, err error) bool
	Errors      map[error]Info
}

// Format builds an error Response body describing err by consulting
// the f.Errors lookup table. If no entry is found, it returns f.Default.
func (f Formatter) Format(err error) (body Response) {
	root := errors.Root(err)
	// Some types cannot be used as map keys, for example slices.
	// If an error's underlying type is one of these, don't panic.
	// Just treat it like any other missing entry.
	defer func() {
		if err := recover(); err != nil {
			body = Response{f.Default, "", nil, true}
		}
	}()
	info, ok := f.Errors[root]
	if !ok {
		info = f.Default
	}

	body = Response{
		Info:      info,
		Detail:    errors.Detail(err),
		Data:      errors.Data(err),
		Temporary: f.IsTemporary(info, err),
	}
	return body
}

// Write writes a json encoded Response to the ResponseWriter.
// It uses the status code associated with the error.
//
// Write may be used as an ErrorWriter in the httpjson package.
func (f Formatter) Write(ctx context.Context, w http.ResponseWriter, err error) {
	f.Log(ctx, err)
	resp := f.Format(err)
	httpjson.Write(ctx, w, resp.HTTPStatus, resp)
}

// Log writes a structured log entry to the chain/log logger with
// information about the error and the HTTP response.
func (f Formatter) Log(ctx context.Context, err error) {
	var errorMessage string
	if err != nil {
		// strip the stack trace, if there is one
		errorMessage = err.Error()
	}

	resp := f.Format(err)
	keyvals := []interface{}{
		"status", resp.HTTPStatus,
		"chaincode", resp.ChainCode,
		log.KeyError, errorMessage,
	}
	if resp.HTTPStatus == 500 {
		keyvals = append(keyvals, log.KeyStack, errors.Stack(err))
	}
	log.Printkv(ctx, keyvals...)
}

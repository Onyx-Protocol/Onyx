package core

import (
	"context"
	"fmt"
	"net/http"

	"chain/errors"
	"chain/log"
	"chain/net/http/httpjson"
	"chain/net/http/reqid"
)

// errBadReqHeader indicates the user supplied a malformed request header,
// possibly including a datatype that doesn't match what we expected.
var errBadReqHeader = errors.New("bad request header")

func jsonHandler(f interface{}) http.Handler {
	h, err := httpjson.Handler(f, WriteHTTPError)
	if err != nil {
		panic(err)
	}
	return h
}

// WriteHTTPError writes a json encoded detailedError
// to the ResponseWriter. It uses the status code
// associated with the error.
func WriteHTTPError(ctx context.Context, w http.ResponseWriter, err error) {
	logHTTPError(ctx, err)
	body, info := errInfo(err)
	httpjson.Write(ctx, w, info.HTTPStatus, body)
}

func logHTTPError(ctx context.Context, err error) {
	var errorMessage string
	if err != nil {
		// strip the stack trace, if there is one
		errorMessage = err.Error()
	}

	_, info := errInfo(err)
	keyvals := []interface{}{
		"status", info.HTTPStatus,
		"chaincode", info.ChainCode,
		"path", reqid.PathFromContext(ctx),
		log.KeyError, errorMessage,
	}
	if info.HTTPStatus == 500 {
		keyvals = append(keyvals, log.KeyStack, errors.Stack(err))
	}
	log.Write(ctx, keyvals...)
}

func alwaysError(err error) http.Handler {
	return jsonHandler(func() error { return err })
}

func batchRecover(ctx context.Context, v *interface{}) {
	if r := recover(); r != nil {
		if recoveredErr, ok := r.(error); ok {
			*v = recoveredErr
		} else {
			*v = fmt.Errorf("panic with %T", r)
		}
	}

	if *v == nil {
		return
	}
	// Convert errors into errorInfo responses (including errors
	// from recovered panics above).
	if err, ok := (*v).(error); ok {
		logHTTPError(ctx, err)
		*v, _ = errInfo(err)
	}
}

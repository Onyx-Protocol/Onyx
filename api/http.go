package api

import (
	"net/http"

	"golang.org/x/net/context"

	"chain/errors"
	"chain/log"
	"chain/net/http/httpjson"
)

// ErrBadReqHeader indicates the user supplied a malformed request header,
// possibly including a datatype that doesn't match what we expected.
var ErrBadReqHeader = errors.New("bad request header")

func writeHTTPError(ctx context.Context, w http.ResponseWriter, err error) {
	info := errInfo(err)
	//metrics.Counter("status." + strconv.Itoa(info.HTTPStatus)).Add()
	//metrics.Counter("error." + info.ChainCode).Add()

	keyvals := []interface{}{
		"status", info.HTTPStatus,
		"chaincode", info.ChainCode,
		log.KeyError, err,
	}
	if info.HTTPStatus == 500 {
		keyvals = append(keyvals, log.KeyStack, errors.Stack(err))
	}
	log.Write(ctx, keyvals...)

	var v interface{} = info
	if s := errors.Detail(err); s != "" {
		v = struct {
			errorInfo
			Detail string `json:"detail"`
		}{info, s}
	}
	httpjson.Write(ctx, w, info.HTTPStatus, v)
}

package core

import (
	"net/http"

	"golang.org/x/net/context"

	"chain/errors"
	"chain/log"
	"chain/net/http/httpjson"
)

// errBadReqHeader indicates the user supplied a malformed request header,
// possibly including a datatype that doesn't match what we expected.
var errBadReqHeader = errors.New("bad request header")

func writeHTTPError(ctx context.Context, w http.ResponseWriter, err error) {
	logHTTPError(ctx, err)
	body, info := errInfo(err)
	httpjson.Write(ctx, w, info.HTTPStatus, body)
}

func logHTTPError(ctx context.Context, err error) {
	_, info := errInfo(err)
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
}

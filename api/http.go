package api

import (
	"net/http"
	"strconv"

	"golang.org/x/net/context"

	"chain/errors"
	"chain/log"
	"chain/net/http/httpjson"
)

// errBadReqHeader indicates the user supplied a malformed request header,
// possibly including a datatype that doesn't match what we expected.
var errBadReqHeader = errors.New("bad request header")

func writeHTTPError(ctx context.Context, w http.ResponseWriter, err error) {
	body, info := errInfo(err)
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
	httpjson.Write(ctx, w, info.HTTPStatus, body)
}

func getPageData(ctx context.Context, defaultLimit int) (prev string, limit int, err error) {
	prev = httpjson.Request(ctx).Header.Get("Range-After")

	limit = defaultLimit
	if lstr := httpjson.Request(ctx).Header.Get("Limit"); lstr != "" {
		limit, err = strconv.Atoi(lstr)
		if err != nil {
			err = errors.Wrap(errBadReqHeader, err.Error())
			return "", 0, errors.WithDetail(err, "limit header")
		}
	}
	return
}

package core

import (
	"context"

	"chain/core/account/utxodb"
	"chain/core/asset"
	"chain/core/query"
	"chain/core/signers"
	"chain/core/txbuilder"
	"chain/database/pg"
	"chain/errors"
	"chain/net/http/httpjson"
)

var (
	// ErrBadBuildRequest is returned for malformed build transaction requests.
	ErrBadBuildRequest = errors.New("bad build request")
)

// errorInfo contains a set of error codes to send to the user.
type errorInfo struct {
	HTTPStatus int    `json:"-"`
	ChainCode  string `json:"code"`
	Message    string `json:"message"`
}

type detailedError struct {
	errorInfo
	Detail string `json:"detail,omitempty"`
}

var (
	// infoInternal holds the codes we use for an internal error.
	// It is defined here for easy reference.
	infoInternal = errorInfo{500, "CH000", "Chain API Error"}

	// Map error values to standard chain error codes.
	// Missing entries will map to infoInternal.
	// See chain.com/docs.
	errorInfoTab = map[error]errorInfo{
		// General error namespace (0xx)
		context.DeadlineExceeded: errorInfo{504, "CH001", "Request timed out"},
		pg.ErrUserInputNotFound:  errorInfo{404, "CH002", "Not found"},
		httpjson.ErrBadRequest:   errorInfo{400, "CH003", "Invalid request body"},
		errBadReqHeader:          errorInfo{400, "CH004", "Invalid request header"},
		asset.ErrArchived:        errorInfo{404, "CH005", "Item has been archived"},
		signers.ErrArchived:      errorInfo{404, "CH005", "Item has been archived"},

		// Core error namespace
		ErrProdReset: errorInfo{400, "CH100", "Reset can only be called in a development system"},

		// Signers error namespace (2xx)
		signers.ErrBadQuorum: errorInfo{400, "CH200", "Quorum must be greater than 1 and less than or equal to the length of xpubs"},
		signers.ErrBadXPub:   errorInfo{400, "CH201", "Invalid xpub format"},
		signers.ErrNoXPubs:   errorInfo{400, "CH202", "At least one xpub is required"},
		signers.ErrBadType:   errorInfo{400, "CH203", "Retrieved type does not match expected type"},

		// Query error namespace
		query.ErrBadCursor:              errorInfo{400, "CH600", "Malformed pagination cursor"},
		query.ErrParameterCountMismatch: errorInfo{400, "CH601", "Incorrect number of parameters to filter"},
		ErrBadIndexConfig:               errorInfo{400, "CH602", "Invalid index configuration"},

		// Transaction error namespace
		ErrBadBuildRequest:           errorInfo{400, "CH700", "Invalid build transaction request"},
		txbuilder.ErrBadBuildRequest: errorInfo{400, "CH700", "Invalid build transaction request"},
		txbuilder.ErrBadTxTemplate:   errorInfo{400, "CH701", "Invalid transaction template"},
		txbuilder.ErrRejected:        errorInfo{400, "CH702", "Transaction rejected"},
		utxodb.ErrInsufficient:       errorInfo{400, "CH710", "Insufficient funds for tx"},
		utxodb.ErrReserved:           errorInfo{400, "CH711", "Some outputs are reserved; try again"},
	}
)

// errInfo returns the HTTP status code to use
// and a suitable response body describing err
// by consulting the global lookup table.
// If no entry is found, it returns infoInternal.
func errInfo(err error) (body detailedError, info errorInfo) {
	root := errors.Root(err)
	// Some types cannot be used as map keys, for example slices.
	// If an error's underlying type is one of these, don't panic.
	// Just treat it like any other missing entry.
	defer func() {
		if err := recover(); err != nil {
			info = infoInternal
			body = detailedError{infoInternal, ""}
		}
	}()
	info, ok := errorInfoTab[root]
	if !ok {
		info = infoInternal
	}

	return detailedError{info, errors.Detail(err)}, info
}

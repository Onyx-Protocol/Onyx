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
		pg.ErrUserInputNotFound:  errorInfo{400, "CH002", "Not found"},
		httpjson.ErrBadRequest:   errorInfo{400, "CH003", "Invalid request body"},
		errBadReqHeader:          errorInfo{400, "CH004", "Invalid request header"},
		asset.ErrArchived:        errorInfo{400, "CH005", "Item has been archived"},
		signers.ErrArchived:      errorInfo{400, "CH005", "Item has been archived"},

		// Core error namespace
		errProdReset: errorInfo{400, "CH100", "Reset can only be called in a development system"},

		// Signers error namespace (2xx)
		signers.ErrBadQuorum: errorInfo{400, "CH200", "Quorum must be greater than 1 and less than or equal to the length of xpubs"},
		signers.ErrBadXPub:   errorInfo{400, "CH201", "Invalid xpub format"},
		signers.ErrNoXPubs:   errorInfo{400, "CH202", "At least one xpub is required"},
		signers.ErrBadType:   errorInfo{400, "CH203", "Retrieved type does not match expected type"},

		// Query error namespace (6xx)
		query.ErrBadCursor:              errorInfo{400, "CH600", "Malformed pagination cursor"},
		query.ErrParameterCountMismatch: errorInfo{400, "CH601", "Incorrect number of parameters to filter"},
		errBadIndexConfig:               errorInfo{400, "CH602", "Invalid index configuration"},

		// Transaction error namespace (7xx)
		// Build error namespace (70x)
		txbuilder.ErrBadRefData: errorInfo{400, "CH700", "Reference data does not match previous transaction's reference data"},
		errBadActionType:        errorInfo{400, "CH701", "Invalid action type"},
		errBadAlias:             errorInfo{400, "CH702", "Invalid alias on action"},
		// Submit error namespace (73x)
		txbuilder.ErrMissingRawTx:     errorInfo{400, "CH730", "Missing raw transaction"},
		txbuilder.ErrBadInputCount:    errorInfo{400, "CH731", "Too many inputs in template for transaction"},
		txbuilder.ErrBadTxInputIdx:    errorInfo{400, "CH732", "Invalid transaction input index"},
		txbuilder.ErrBadSigScriptComp: errorInfo{400, "CH733", "Invalid signature script component"},
		txbuilder.ErrMissingSig:       errorInfo{400, "CH734", "Missing signature in template"},
		txbuilder.ErrRejected:         errorInfo{400, "CH735", "Transaction rejected"},
		// account action error namespace (76x)
		utxodb.ErrInsufficient: errorInfo{400, "CH760", "Insufficient funds for tx"},
		utxodb.ErrReserved:     errorInfo{400, "CH761", "Some outputs are reserved; try again"},
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

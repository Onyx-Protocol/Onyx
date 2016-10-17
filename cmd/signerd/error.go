package main

import "chain/errors"

// errorInfo contains a set of error codes to send to the user.
type errorInfo struct {
	HTTPStatus int    `json:"-"`
	ChainCode  string `json:"code"`
	Message    string `json:"message"`
}

var errNotAuthenticated = errors.New("Request could not be authenticated")

var (
	// infoInternal holds the codes we use for an internal error.
	// It is defined here for easy reference.
	infoInternal = errorInfo{500, "CH000", "Chain API Error"}

	// Map error values to standard chain error codes.
	// Missing entries will map to infoInternal.
	// See chain.com/docs.
	errorInfoTab = map[error]errorInfo{
		errNotAuthenticated: errorInfo{401, "CH009", "Request could not be authenticated"},
	}
)

// errInfo returns the HTTP status code to use
// and a suitable response body describing err
// by consulting the global lookup table.
// If no entry is found, it returns infoInternal.
func errInfo(err error) (body interface{}, info errorInfo) {
	root := errors.Root(err)
	// Some types cannot be used as map keys, for example slices.
	// If an error's underlying type is one of these, don't panic.
	// Just treat it like any other missing entry.
	defer func() {
		if err := recover(); err != nil {
			info = infoInternal
			body = infoInternal
		}
	}()
	info, ok := errorInfoTab[root]
	if !ok {
		info = infoInternal
	}

	if s := errors.Detail(err); s != "" {
		return struct {
			errorInfo
			Detail string `json:"detail"`
		}{info, s}, info
	}

	return info, info
}

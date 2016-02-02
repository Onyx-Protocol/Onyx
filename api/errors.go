package api

import (
	"chain/api/appdb"
	"chain/api/asset"
	"chain/api/utxodb"
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

var (
	// infoInternal holds the codes we use for an internal error.
	// It is defined here for easy reference.
	infoInternal = errorInfo{500, "CH000", "Chain API Error"}

	// Map error values to standard chain error codes.
	// Missing entries will map to infoInternal.
	// See chain.com/docs.
	errorInfoTab = map[error]errorInfo{
		pg.ErrUserInputNotFound:     errorInfo{404, "CH005", "Not found."},
		errNoAccessToResource:       errorInfo{404, "CH005", "Not found."},
		httpjson.ErrBadRequest:      errorInfo{400, "CH007", "Invalid request body"},
		errBadReqHeader:             errorInfo{400, "CH008", "Invalid request header"},
		appdb.ErrBadEmail:           errorInfo{400, "CH101", "Invalid email."},
		appdb.ErrBadPassword:        errorInfo{400, "CH102", "Invalid password."},
		appdb.ErrPasswordCheck:      errorInfo{400, "CH103", "Unable to verify password."},
		appdb.ErrNoUserForEmail:     errorInfo{400, "CH104", "No user corresponds to the provided email address."},
		appdb.ErrBadLabel:           errorInfo{400, "CH704", "Invalid label"},
		asset.ErrBadSigsRequired:    errorInfo{400, "CH712", "signatures_required must be at least 1."},
		asset.ErrBadKeySpec:         errorInfo{400, "CH713", "Invalid xpub."},
		asset.ErrTooFewKeys:         errorInfo{400, "CH715", "Cannot have more signatures required than keys."},
		appdb.ErrBadAccountKeyCount: errorInfo{400, "CH716", "Accounts must provide the correct number of keys for a manager node."},
		appdb.ErrPastExpires:        errorInfo{400, "CH720", "Expires, if set, must be in the future"},
		utxodb.ErrInsufficient:      errorInfo{400, "CH733", "Insufficient funds for tx"},
		utxodb.ErrReserved:          errorInfo{400, "CH743", "Some outputs are reserved; try again"},
		asset.ErrBadTx:              errorInfo{400, "CH755", "Invalid transaction template"},
		ErrBadBuildRequest:          errorInfo{400, "CH756", "Invalid build transaction request"},
		appdb.ErrBadProjectName:     errorInfo{400, "CH770", "Invalid project name."},
		appdb.ErrCannotDelete:       errorInfo{400, "CH901", "Cannot delete non-empty object"},
		appdb.ErrArchived:           errorInfo{404, "CH902", "Item has been archived"},
		appdb.ErrBadRole:            errorInfo{400, "CH800", "Member role must be \"developer\" or \"admin\"."},
		appdb.ErrAlreadyMember:      errorInfo{400, "CH801", "User is already a member of the project."},
		errNotAdmin:                 errorInfo{403, "CH781", "Admin privileges are required perform this action"},
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

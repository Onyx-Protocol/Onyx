package core

import (
	"context"

	"chain/core/accesstoken"
	"chain/core/account"
	"chain/core/asset"
	"chain/core/blocksigner"
	"chain/core/config"
	"chain/core/mockhsm"
	"chain/core/query"
	"chain/core/query/filter"
	"chain/core/rpc"
	"chain/core/signers"
	"chain/core/txbuilder"
	"chain/core/txfeed"
	"chain/database/pg"
	"chain/errors"
	"chain/net/http/httpjson"
	"chain/protocol"
)

// errorInfo contains a set of error codes to send to the user.
type errorInfo struct {
	HTTPStatus int    `json:"-"`
	ChainCode  string `json:"code"`
	Message    string `json:"message"`
}

type detailedError struct {
	errorInfo
	Detail    string                 `json:"detail,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Temporary bool                   `json:"temporary"`
}

func isTemporary(info errorInfo, err error) bool {
	switch info.ChainCode {
	case "CH000": // internal server error
		return true
	case "CH001": // request timed out
		return true
	case "CH761": // outputs currently reserved
		return true
	case "CH706": // 1 or more action errors
		errs := errors.Data(err)["actions"].([]detailedError)
		temp := true
		for _, actionErr := range errs {
			temp = temp && isTemporary(actionErr.errorInfo, nil)
		}
		return temp
	default:
		return false
	}
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
		context.DeadlineExceeded:     errorInfo{408, "CH001", "Request timed out"},
		pg.ErrUserInputNotFound:      errorInfo{400, "CH002", "Not found"},
		httpjson.ErrBadRequest:       errorInfo{400, "CH003", "Invalid request body"},
		errBadReqHeader:              errorInfo{400, "CH004", "Invalid request header"},
		errNotFound:                  errorInfo{404, "CH006", "Not found"},
		errRateLimited:               errorInfo{429, "CH007", "Request limit exceeded"},
		errLeaderElection:            errorInfo{503, "CH008", "Electing a new leader for the core; try again soon"},
		errNotAuthenticated:          errorInfo{401, "CH009", "Request could not be authenticated"},
		txbuilder.ErrMissingFields:   errorInfo{400, "CH010", "One or more fields are missing"},
		asset.ErrDuplicateAlias:      errorInfo{400, "CH050", "Alias already exists"},
		account.ErrDuplicateAlias:    errorInfo{400, "CH050", "Alias already exists"},
		txfeed.ErrDuplicateAlias:     errorInfo{400, "CH050", "Alias already exists"},
		mockhsm.ErrDuplicateKeyAlias: errorInfo{400, "CH050", "Alias already exists"},

		// Core error namespace
		errUnconfigured:                errorInfo{400, "CH100", "This core still needs to be configured"},
		errAlreadyConfigured:           errorInfo{400, "CH101", "This core has already been configured"},
		config.ErrBadGenerator:         errorInfo{400, "CH102", "Generator URL returned an invalid response"},
		errBadBlockPub:                 errorInfo{400, "CH103", "Provided Block XPub is invalid"},
		rpc.ErrWrongNetwork:            errorInfo{502, "CH104", "A peer core is operating on a different blockchain network"},
		protocol.ErrTheDistantFuture:   errorInfo{400, "CH105", "Requested height is too far ahead"},
		config.ErrBadSignerURL:         errorInfo{400, "CH106", "Block signer URL is invalid"},
		config.ErrBadSignerPubkey:      errorInfo{400, "CH107", "Block signer pubkey is invalid"},
		config.ErrBadQuorum:            errorInfo{400, "CH108", "Quorum must be greater than 0 if there are signers"},
		errProdReset:                   errorInfo{400, "CH110", "Reset can only be called in a development system"},
		errNoClientTokens:              errorInfo{400, "CH120", "Cannot enable client authentication with no client tokens"},
		blocksigner.ErrConsensusChange: errorInfo{400, "CH150", "Refuse to sign block with consensus change"},

		// Signers error namespace (2xx)
		signers.ErrBadQuorum: errorInfo{400, "CH200", "Quorum must be greater than 1 and less than or equal to the length of xpubs"},
		signers.ErrBadXPub:   errorInfo{400, "CH201", "Invalid xpub format"},
		signers.ErrNoXPubs:   errorInfo{400, "CH202", "At least one xpub is required"},
		signers.ErrBadType:   errorInfo{400, "CH203", "Retrieved type does not match expected type"},
		signers.ErrDupeXPub:  errorInfo{400, "CH204", "Root XPubs cannot contain the same key more than once"},

		// Access token error namespace (3xx)
		accesstoken.ErrBadID:       errorInfo{400, "CH300", "Malformed or empty access token id"},
		accesstoken.ErrBadType:     errorInfo{400, "CH301", "Access tokens must be type client or network"},
		accesstoken.ErrDuplicateID: errorInfo{400, "CH302", "Access token id is already in use"},
		errCurrentToken:            errorInfo{400, "CH310", "The access token used to authenticate this request cannot be deleted"},

		// Query error namespace (6xx)
		query.ErrBadAfter:               errorInfo{400, "CH600", "Malformed pagination parameter `after`"},
		query.ErrParameterCountMismatch: errorInfo{400, "CH601", "Incorrect number of parameters to filter"},
		filter.ErrBadFilter:             errorInfo{400, "CH602", "Malformed query filter"},

		// Transaction error namespace (7xx)
		// Build error namespace (70x)
		txbuilder.ErrBadRefData: errorInfo{400, "CH700", "Reference data does not match previous transaction's reference data"},
		errBadActionType:        errorInfo{400, "CH701", "Invalid action type"},
		errBadAlias:             errorInfo{400, "CH702", "Invalid alias on action"},
		errBadAction:            errorInfo{400, "CH703", "Invalid action object"},
		txbuilder.ErrBadAmount:  errorInfo{400, "CH704", "Invalid asset amount"},
		txbuilder.ErrBlankCheck: errorInfo{400, "CH705", "Unsafe transaction: leaves assets to be taken without requiring payment"},
		txbuilder.ErrAction:     errorInfo{400, "CH706", "One or more actions had an error: see attached data"},

		// Submit error namespace (73x)
		txbuilder.ErrMissingRawTx:          errorInfo{400, "CH730", "Missing raw transaction"},
		txbuilder.ErrBadInstructionCount:   errorInfo{400, "CH731", "Too many signing instructions in template for transaction"},
		txbuilder.ErrBadTxInputIdx:         errorInfo{400, "CH732", "Invalid transaction input index"},
		txbuilder.ErrBadWitnessComponent:   errorInfo{400, "CH733", "Invalid witness component"},
		txbuilder.ErrRejected:              errorInfo{400, "CH735", "Transaction rejected"},
		txbuilder.ErrNoTxSighashCommitment: errorInfo{400, "CH736", "Transaction is not final, additional actions still allowed"},

		// account action error namespace (76x)
		account.ErrInsufficient: errorInfo{400, "CH760", "Insufficient funds for tx"},
		account.ErrReserved:     errorInfo{400, "CH761", "Some outputs are reserved; try again"},

		// Mock HSM error namespace (80x)
		mockhsm.ErrInvalidAfter:         errorInfo{400, "CH801", "Invalid `after` in query"},
		mockhsm.ErrTooManyAliasesToList: errorInfo{400, "CH802", "Too many aliases to list"},
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
			body = detailedError{infoInternal, "", nil, true}
		}
	}()
	info, ok := errorInfoTab[root]
	if !ok {
		info = infoInternal
	}

	body = detailedError{
		errorInfo: info,
		Detail:    errors.Detail(err),
		Data:      errors.Data(err),
		Temporary: isTemporary(info, err),
	}
	return body, info
}

// errInfoBodyList calls errInfo for each element in errs
// and returns the "body".
func errInfoBodyList(errs []error) (a []detailedError) {
	for _, err := range errs {
		body, _ := errInfo(err)
		a = append(a, body)
	}
	return a
}

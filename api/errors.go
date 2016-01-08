package api

import (
	"chain/api/appdb"
	"chain/api/asset"
	"chain/api/utxodb"
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
		asset.ErrBadAddr:            errorInfo{400, "CH300", "Invalid address"},
		appdb.ErrBadLabel:           errorInfo{400, "CH704", "Invalid label"},
		asset.ErrBadSigsRequired:    errorInfo{400, "CH712", "signatures_required must be at least 1."},
		asset.ErrBadKeySpec:         errorInfo{400, "CH713", "Invalid xpub."},
		asset.ErrTooFewKeys:         errorInfo{400, "CH715", "Cannot have more signatures required than keys."},
		appdb.ErrBadAccountKeyCount: errorInfo{400, "CH716", "Accounts must provide the correct number of keys for a manager node."},
		appdb.ErrPastExpires:        errorInfo{400, "CH720", "Expires, if set, must be in the future"},
		utxodb.ErrInsufficient:      errorInfo{400, "CH733", "Insufficient funds for tx"},
		utxodb.ErrReserved:          errorInfo{400, "CH743", "Some outputs are reserved; try again"},
		asset.ErrBadOutDest:         errorInfo{400, "CH744", "Invalid input sources or output destinations"},
		asset.ErrBadTx:              errorInfo{400, "CH755", "Invalid transaction template"},
		appdb.ErrBadAsset:           errorInfo{400, "CH761", "Invalid asset"},
		appdb.ErrCannotDelete:       errorInfo{400, "CH901", "Cannot delete non-empty object"},
		appdb.ErrBadRole:            errorInfo{400, "CH800", "Member role must be \"developer\" or \"admin\"."},
		appdb.ErrAlreadyMember:      errorInfo{400, "CH801", "User is already a member of the project."},
		errNotAdmin:                 errorInfo{403, "CH781", "Admin privileges are required perform this action"},

		// Error codes imported from papi for convenient reference.
		// Please delete lines from this block when you add them
		// to the live code above or when you know they won't be used.
		//
		// ErrAPITryAgain     = errorInfo{500, "CH009", "Chain API Error, Try Again"}
		// ErrRateLimit       = errorInfo{429, "CH011", "Exceeded rate limit. Email enterprise@chain.com to learn about our production services."}
		//
		// ErrTxStatusUnknown = errorInfo{500, "CH203", "Transaction relayed, status unknown"}
		//
		// ErrBadNotifURL   = errorInfo{400, "CH400", "Invalid notification URL"}
		// ErrDupNotif      = errorInfo{400, "CH401", "New notification conflicts with an existing notification"}
		// ErrBadBlockChain = errorInfo{400, "CH403", "Invalid block chain"}
		// ErrBadNotifType  = errorInfo{400, "CH408", "Invalid notification type"}
		//
		// ErrMissingWallet       = errorInfo{404, "CH703", "Requested wallet could not be found"}
		// ErrMissingWallet400    = errorInfo{400, "CH703", "Requested wallet could not be found"}
		// ErrBadAppID            = errorInfo{400, "CH705", "Invalid application ID"}
		// ErrBadKeyRotate        = errorInfo{400, "CH706", "Old XPub is not in current wallet key rotation"}
		// ErrWalletOwnership     = errorInfo{400, "CH707", "Chain must provide fewer keys than required signatures"}
		// ErrZeroSigReq          = errorInfo{400, "CH708", "Signatures required must be at least one"}
		// ErrRequestedKeys       = errorInfo{400, "CH709", "Chain can only provide one key"}
		// ErrMissingBucket400    = errorInfo{400, "CH710", "Requested bucket could not be found"}
		// ErrMissingApp          = errorInfo{404, "CH711", "Requested application could not be found"}
		// ErrMissingReceiver     = errorInfo{404, "CH721", "Requested receiver could not be found"}
		// ErrMaxInputs           = errorInfo{400, "CH730", "Maximum number of inputs passed"}
		// ErrNoInputs            = errorInfo{400, "CH731", "No inputs provided"}
		// ErrNoOutputs           = errorInfo{400, "CH732", "No outputs provided"}
		//
		// ErrZeroInput           = errorInfo{400, "CH734", "Input amount must be greater than 0"}
		// ErrZeroOutput          = errorInfo{400, "CH735", "Output amount must be greater than 0"}
		// ErrSoloWallet          = errorInfo{400, "CH736", "Wallet input must be the only input"}
		// ErrBadOut              = errorInfo{400, "CH737", "Invalid output"}
		//
		// ErrFeePayer            = errorInfo{400, "CH739", "There must be exactly one fee payer"}
		// ErrMetadataHex         = errorInfo{400, "CH740", "Metadata must be hex encoded"}
		// ErrMetadataLen         = errorInfo{400, "CH741", "Metadata cannot be longer than 40 bytes"}
		// ErrTxTooBig            = errorInfo{400, "CH742", "Transaction byte size is too big"}
		//
		// ErrMetaAndOA           = errorInfo{400, "CH745", "Metadata cannot be added to open asset transactions"}
		//
		// ErrInvalidSig          = errorInfo{400, "CH751", "Signature was not valid for transaction"}
		// ErrNTplInput           = errorInfo{400, "CH752", "Transaction template has wrong number of inputs"}
		// ErrExtraSig            = errorInfo{400, "CH754", "Too many signatures"}
		//
		// ErrBadRedeem           = errorInfo{400, "CH756", "Redeem script is not a valid multisig script"}
		// ErrMissingAsset        = errorInfo{404, "CH760", "Requested asset could not be found"}
		//
		// ErrAssetOpRetTooBig    = errorInfo{400, "CH762", "Too many outputs and/or asset defintion too large"}
		// ErrAssetDefIDForbidden = errorInfo{400, "CH763", "Asset definition should not have asset IDs"}
		// ErrInvalidPolicy       = errorInfo{400, "CH772", "Invalid policy"}
		// ErrMissingApproval     = errorInfo{404, "CH773", "Requested approval could not be found"}
		// ErrBadApprovals        = errorInfo{400, "CH774", "Approval request cannot be satisfied"}
		// ErrMissingPolicy       = errorInfo{404, "CH775", "Requested policy could not be found"}
		//
		// ErrNoTxActivity        = errorInfo{404, "CH790", "Transaction details could not be found."}
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

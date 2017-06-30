package core

import (
	"context"

	"chain/core/accesstoken"
	"chain/core/account"
	"chain/core/asset"
	"chain/core/blocksigner"
	"chain/core/config"
	"chain/core/leader"
	"chain/core/query"
	"chain/core/query/filter"
	"chain/core/rpc"
	"chain/core/signers"
	"chain/core/txbuilder"
	"chain/core/txfeed"
	"chain/database/pg"
	"chain/database/sinkdb"
	"chain/errors"
	"chain/net/http/authz"
	"chain/net/http/httperror"
	"chain/net/http/httpjson"
	"chain/net/raft"
	"chain/protocol"
)

func isTemporary(info httperror.Info, err error) bool {
	switch info.ChainCode {
	case "CH000": // internal server error
		return true
	case "CH001": // request timed out
		return true
	case "CH761": // outputs currently reserved
		return true
	case "CH706": // 1 or more action errors
		errs := errors.Data(err)["actions"].([]httperror.Response)
		temp := true
		for _, actionErr := range errs {
			temp = temp && isTemporary(actionErr.Info, nil)
		}
		return temp
	default:
		return false
	}
}

// Map error values to standard chain error codes. Missing entries
// will map to internalErrInfo.
// See chain.com/docs.
//
// TODO(jackson): Share one error table across Chain
// products/services so that errors are consistent.
var errorFormatter = httperror.Formatter{
	Default:     httperror.Info{500, "CH000", "Chain API Error"},
	IsTemporary: isTemporary,
	Errors: map[error]httperror.Info{
		// General error namespace (0xx)
		context.DeadlineExceeded:   {408, "CH001", "Request timed out"},
		pg.ErrUserInputNotFound:    {400, "CH002", "Not found"},
		httpjson.ErrBadRequest:     {400, "CH003", "Invalid request body"},
		errNotFound:                {404, "CH006", "Not found"},
		errRateLimited:             {429, "CH007", "Request limit exceeded"},
		leader.ErrNoLeader:         {503, "CH008", "Electing a new leader for the core; try again soon"},
		errNotAuthenticated:        {401, "CH009", "Request could not be authenticated"},
		txbuilder.ErrMissingFields: {400, "CH010", "One or more fields are missing"},
		authz.ErrNotAuthorized:     {403, "CH011", "Request is unauthorized"},
		sinkdb.ErrConflict:         {409, "CH012", "Conflict processing request"},
		asset.ErrDuplicateAlias:    {400, "CH050", "Alias already exists"},
		account.ErrDuplicateAlias:  {400, "CH050", "Alias already exists"},
		txfeed.ErrDuplicateAlias:   {400, "CH050", "Alias already exists"},
		account.ErrBadIdentifier:   {400, "CH051", "Either an ID or alias must be provided, but not both"},
		asset.ErrBadIdentifier:     {400, "CH051", "Either an ID or alias must be provided, but not both"},

		// Core error namespace
		errUnconfigured:                {400, "CH100", "This core still needs to be configured"},
		errAlreadyConfigured:           {400, "CH101", "This core has already been configured"},
		config.ErrBadGenerator:         {400, "CH102", "Generator URL returned an invalid response"},
		errBadBlockPub:                 {400, "CH103", "Provided Block XPub is invalid"},
		rpc.ErrWrongNetwork:            {502, "CH104", "A peer core is operating on a different blockchain network"},
		protocol.ErrTheDistantFuture:   {400, "CH105", "Requested height is too far ahead"},
		config.ErrBadSignerURL:         {400, "CH106", "Block signer URL is invalid"},
		config.ErrBadSignerPubkey:      {400, "CH107", "Block signer pubkey is invalid"},
		config.ErrBadQuorum:            {400, "CH108", "Quorum must be greater than 0 if there are signers"},
		config.ErrNoBlockPub:           {400, "CH109", "Block Pub cannot be empty when configuring a mockhsm disabled signer"},
		errNoMockHSM:                   {400, "CH110", "This endpoint is disabled for this server's configuration"},
		errNoReset:                     {400, "CH110", "This endpoint is disabled for this server's configuration"},
		config.ErrNoBlockHSMURL:        {400, "CH111", "Block HSM URL cannot be empty when configuring a non mockhsm signer"},
		errNoClientTokens:              {400, "CH120", "Cannot enable client authentication with no client tokens"},
		blocksigner.ErrConsensusChange: {400, "CH150", "Refuse to sign block with consensus change"},
		errMissingAddr:                 {400, "CH160", "Address is missing"},
		errInvalidAddr:                 {400, "CH161", "Address is invalid"},
		raft.ErrAddressNotAllowed:      {400, "CH162", "Address is not allowed"},
		raft.ErrUninitialized:          {400, "CH163", "Cluster not initialized"},
		raft.ErrExistingCluster:        {400, "CH164", "Already connected to a cluster"},
		raft.ErrPeerUninitialized:      {400, "CH165", "Peer node is uninitialized"},
		raft.ErrUnknownPeer:            {400, "CH166", "Unknown peer"},
		config.ErrConfigOp:             {400, "CH170", "Invalid configuration operation"},

		// Signers error namespace (2xx)
		signers.ErrBadQuorum: {400, "CH200", "Quorum must be greater than 1 and less than or equal to the length of xpubs"},
		signers.ErrBadXPub:   {400, "CH201", "Invalid xpub format"},
		signers.ErrNoXPubs:   {400, "CH202", "At least one xpub is required"},
		signers.ErrBadType:   {400, "CH203", "Retrieved type does not match expected type"},
		signers.ErrDupeXPub:  {400, "CH204", "Root XPubs cannot contain the same key more than once"},

		// Access token and grant error namespace (3xx)
		accesstoken.ErrBadID:       {400, "CH300", "Malformed or empty access token id"},
		accesstoken.ErrBadType:     {400, "CH301", "Access tokens must be type client or network"},
		accesstoken.ErrDuplicateID: {400, "CH302", "Access token id is already in use"},
		errMissingTokenID:          {400, "CH303", "Access token id does not exist"},
		errCurrentToken:            {400, "CH310", "The access token used to authenticate this request cannot be deleted"},
		errProtectedGrant:          {400, "CH320", "Protected grants cannot be manually deleted"},
		errCreateProtectedGrant:    {400, "CH321", "Protected grants cannot be manually created"},

		// Query error namespace (6xx)
		query.ErrBadAfter:               {400, "CH600", "Malformed pagination parameter `after`"},
		query.ErrParameterCountMismatch: {400, "CH601", "Incorrect number of parameters to filter"},
		filter.ErrBadFilter:             {400, "CH602", "Malformed query filter"},

		// Transaction error namespace (7xx)
		// Build error namespace (70x)
		txbuilder.ErrBadRefData: {400, "CH700", "Reference data does not match previous transaction's reference data"},
		errBadActionType:        {400, "CH701", "Invalid action type"},
		errBadAlias:             {400, "CH702", "Invalid alias on action"},
		errBadAction:            {400, "CH703", "Invalid action object"},
		txbuilder.ErrBadAmount:  {400, "CH704", "Invalid asset amount"},
		txbuilder.ErrBlankCheck: {400, "CH705", "Unsafe transaction: leaves assets to be taken without requiring payment"},
		txbuilder.ErrAction:     {400, "CH706", "One or more actions had an error: see attached data"},

		// Submit error namespace (73x)
		txbuilder.ErrMissingRawTx:          {400, "CH730", "Missing raw transaction"},
		txbuilder.ErrBadInstructionCount:   {400, "CH731", "Too many signing instructions in template for transaction"},
		txbuilder.ErrBadTxInputIdx:         {400, "CH732", "Invalid transaction input index"},
		txbuilder.ErrBadWitnessComponent:   {400, "CH733", "Invalid witness component"},
		txbuilder.ErrRejected:              {400, "CH735", "Transaction rejected"},
		txbuilder.ErrNoTxSighashCommitment: {400, "CH736", "Transaction is not final, additional actions still allowed"},
		txbuilder.ErrTxSignatureFailure:    {400, "CH737", "Transaction signature missing, client may be missing signature key"},
		txbuilder.ErrNoTxSighashAttempt:    {400, "CH738", "Transaction signature was not attempted"},

		// account action error namespace (76x)
		account.ErrInsufficient: {400, "CH760", "Insufficient funds for tx"},
		account.ErrReserved:     {400, "CH761", "Some outputs are reserved; try again"},

		// Mock HSM error namespace (80x)
	},
}

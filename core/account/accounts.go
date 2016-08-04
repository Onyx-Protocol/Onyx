package account

import (
	"sync"
	"time"

	"golang.org/x/net/context"

	"chain/core/signers"
	"chain/cos/txscript"
	"chain/crypto/ed25519/hd25519"
	"chain/database/pg"
	"chain/errors"
	"chain/metrics"
)

var (
	acpIndexNext int64 // next acp index in our block
	acpIndexCap  int64 // points to end of block
	acpMu        sync.Mutex
)

// Create creates a new Account.
func Create(ctx context.Context, xpubs []string, quorum int, tags map[string]interface{}, clientToken *string) (*signers.Signer, error) {
	return signers.Create(ctx, "account", xpubs, quorum, tags, clientToken)
}

// Find retrieves an Account record from signer
func Find(ctx context.Context, id string) (*signers.Signer, error) {
	return signers.Find(ctx, "account", id)
}

// Archive marks an Account record as archived,
// effectively "deleting" it.
func Archive(ctx context.Context, id string) error {
	return signers.Archive(ctx, "account", id)
}

// List returns a paginated set of Accounts
func List(ctx context.Context, prev string, limit int) ([]*signers.Signer, string, error) {
	return signers.List(ctx, "account", prev, limit)
}

// CreateControlProgram creates a control program
// that is tied to the Account and stores it in the database.
func CreateControlProgram(ctx context.Context, accountID string) ([]byte, error) {
	account, err := Find(ctx, accountID)
	if err != nil {
		return nil, err
	}

	idx, err := nextIndex(ctx)
	if err != nil {
		return nil, err
	}

	path := signers.Path(account, signers.AccountKeySpace, idx)
	derivedXPubs := hd25519.DeriveXPubs(account.XPubs, path)
	derivedPKs := hd25519.XPubKeys(derivedXPubs)
	control, redeem, err := txscript.TxScripts(derivedPKs, account.Quorum)
	if err != nil {
		return nil, err
	}

	err = insertAccountControlProgram(ctx, account.ID, idx, control, redeem)
	if err != nil {
		return nil, err
	}

	return control, nil
}

func insertAccountControlProgram(ctx context.Context, accountID string, idx []uint32, control, redeem []byte) error {
	const q = `
		INSERT INTO account_control_programs (signer_id, key_index, control_program, redeem_program)
		VALUES($1, to_key_index($2), $3, $4)
	`

	_, err := pg.Exec(ctx, q, accountID, pg.Uint32s(idx), control, redeem)
	return errors.Wrap(err)
}

func nextIndex(ctx context.Context) ([]uint32, error) {
	defer metrics.RecordElapsed(time.Now())
	acpMu.Lock()
	defer acpMu.Unlock()

	if acpIndexNext >= acpIndexCap {
		var cap int64
		const incrby = 10000 // account_control_program_seq increments by 10,000
		const q = `SELECT nextval('account_control_program_seq')`
		err := pg.QueryRow(ctx, q).Scan(&cap)
		if err != nil {
			return nil, errors.Wrap(err, "scan")
		}
		acpIndexCap = cap
		acpIndexNext = cap - incrby
	}

	n := acpIndexNext
	acpIndexNext++
	return keyIndex(n), nil
}

func keyIndex(n int64) []uint32 {
	index := make([]uint32, 2)
	index[0] = uint32(n >> 31)
	index[1] = uint32(n & 0x7fffffff)
	return index
}

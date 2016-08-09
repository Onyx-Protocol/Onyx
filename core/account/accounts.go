package account

import (
	"database/sql"
	"encoding/json"
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

type Account struct {
	*signers.Signer
	Tags map[string]interface{} `json:"tags"`
}

var (
	acpIndexNext int64 // next acp index in our block
	acpIndexCap  int64 // points to end of block
	acpMu        sync.Mutex
)

// Create creates a new Account.
func Create(ctx context.Context, xpubs []string, quorum int, tags map[string]interface{}, clientToken *string) (*Account, error) {
	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "create signer")
	}
	defer dbtx.Rollback(ctx)

	signer, err := signers.Create(ctx, "account", xpubs, quorum, clientToken)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	account := &Account{Signer: signer}
	err = insertAccountTags(ctx, account.ID, tags)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	account.Tags = tags

	err = dbtx.Commit(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "committing create account dbtx")
	}
	return account, nil
}

// SetTags updates the tags on the provided Account.
func SetTags(ctx context.Context, id string, tags map[string]interface{}) (*Account, error) {
	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "setting tags")
	}
	defer dbtx.Rollback(ctx)

	signer, err := Find(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	err = insertAccountTags(ctx, id, tags)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	err = dbtx.Commit(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "committing create account dbtx")
	}

	return &Account{
		Signer: signer,
		Tags:   tags,
	}, nil
}

// insertAccountTags inserts a set of tags for the given accountID.
// It must take place inside a database transaction.
func insertAccountTags(ctx context.Context, accountID string, tags map[string]interface{}) error {
	_ = pg.FromContext(ctx).(pg.Tx) // panics if not in a db transaction
	tagsParam, err := tagsToNullString(tags)
	if err != nil {
		return err
	}

	const q = `
		INSERT INTO account_tags (account_id, tags) VALUES ($1, $2)
		ON CONFLICT (account_id) DO UPDATE SET tags = $2
	`

	_, err = pg.Exec(ctx, q, accountID, tagsParam)
	if err != nil {
		return errors.Wrap(err)
	}

	return nil
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

func tagsToNullString(tags map[string]interface{}) (*sql.NullString, error) {
	var tagsJSON []byte
	if len(tags) != 0 {
		var err error
		tagsJSON, err = json.Marshal(tags)
		if err != nil {
			return nil, errors.Wrap(err)
		}
	}
	return &sql.NullString{String: string(tagsJSON), Valid: len(tagsJSON) > 0}, nil
}

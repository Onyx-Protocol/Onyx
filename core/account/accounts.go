package account

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"

	"github.com/golang/groupcache/lru"

	"chain/core/signers"
	"chain/crypto/ed25519/chainkd"
	"chain/database/pg"
	"chain/errors"
	"chain/net/http/httpjson"
	"chain/protocol/vmutil"
)

const maxAccountCache = 100

var (
	cacheMu sync.Mutex
	cache   = lru.New(maxAccountCache)
)

type Account struct {
	*signers.Signer
	Alias string
	Tags  map[string]interface{}
}

var (
	acpIndexNext uint64 // next acp index in our block
	acpIndexCap  uint64 // points to end of block
	acpMu        sync.Mutex
)

// Create creates a new Account.
func Create(ctx context.Context, xpubs []string, quorum int, alias string, tags map[string]interface{}, clientToken *string) (*Account, error) {
	signer, err := signers.Create(ctx, "account", xpubs, quorum, clientToken)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	tagsParam, err := tagsToNullString(tags)
	if err != nil {
		return nil, err
	}

	aliasSQL := sql.NullString{
		String: alias,
		Valid:  alias != "",
	}

	const q = `
		INSERT INTO accounts (account_id, alias, tags) VALUES ($1, $2, $3)
		ON CONFLICT (account_id) DO UPDATE SET alias = $2, tags = $3
	`
	_, err = pg.Exec(ctx, q, signer.ID, aliasSQL, tagsParam)
	if pg.IsUniqueViolation(err) {
		return nil, errors.WithDetail(httpjson.ErrBadRequest, "non-unique alias")
	} else if err != nil {
		return nil, errors.Wrap(err)
	}

	account := &Account{
		Signer: signer,
		Alias:  alias,
		Tags:   tags,
	}

	err = indexAnnotatedAccount(ctx, account)
	if err != nil {
		return nil, errors.Wrap(err, "indexing annotated account")
	}

	return account, nil
}

// FindByAlias retrieves an account's Signer record by its alias
func FindByAlias(ctx context.Context, alias string) (*signers.Signer, error) {
	const q = `SELECT account_id FROM accounts WHERE alias=$1`
	var accountID string
	err := pg.QueryRow(ctx, q, alias).Scan(&accountID)
	if err == sql.ErrNoRows {
		return nil, errors.WithDetailf(pg.ErrUserInputNotFound, "alias: %s", alias)
	}
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return signers.Find(ctx, "account", accountID)
}

// findByID returns an account's Signer record by its ID.
func findByID(ctx context.Context, id string) (*signers.Signer, error) {
	cacheMu.Lock()
	cached, ok := cache.Get(id)
	cacheMu.Unlock()
	if ok {
		return cached.(*signers.Signer), nil
	}
	account, err := signers.Find(ctx, "account", id)
	if err != nil {
		return nil, err
	}
	cacheMu.Lock()
	cache.Add(id, account)
	cacheMu.Unlock()
	return account, nil
}

// CreateControlProgram creates a control program
// that is tied to the Account and stores it in the database.
func CreateControlProgram(ctx context.Context, accountID string, change bool) ([]byte, error) {
	account, err := findByID(ctx, accountID)
	if err != nil {
		return nil, err
	}

	idx, err := nextIndex(ctx)
	if err != nil {
		return nil, err
	}

	path := signers.Path(account, signers.AccountKeySpace, idx)
	derivedXPubs := chainkd.DeriveXPubs(account.XPubs, path)
	derivedPKs := chainkd.XPubKeys(derivedXPubs)
	control, err := vmutil.P2SPMultiSigProgram(derivedPKs, account.Quorum)
	if err != nil {
		return nil, err
	}
	err = insertAccountControlProgram(ctx, account.ID, idx, control, change)
	if err != nil {
		return nil, err
	}

	return control, nil
}

func insertAccountControlProgram(ctx context.Context, accountID string, idx uint64, control []byte, change bool) error {
	const q = `
		INSERT INTO account_control_programs (signer_id, key_index, control_program, change)
		VALUES($1, $2, $3, $4)
	`

	_, err := pg.Exec(ctx, q, accountID, idx, control, change)
	return errors.Wrap(err)
}

func nextIndex(ctx context.Context) (uint64, error) {
	acpMu.Lock()
	defer acpMu.Unlock()

	if acpIndexNext >= acpIndexCap {
		var cap uint64
		const incrby = 10000 // account_control_program_seq increments by 10,000
		const q = `SELECT nextval('account_control_program_seq')`
		err := pg.QueryRow(ctx, q).Scan(&cap)
		if err != nil {
			return 0, errors.Wrap(err, "scan")
		}
		acpIndexCap = cap
		acpIndexNext = cap - incrby
	}

	n := acpIndexNext
	acpIndexNext++
	return n, nil
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

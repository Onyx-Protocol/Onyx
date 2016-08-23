package account

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"chain/core/signers"
	"chain/cos/vmutil"
	"chain/crypto/ed25519/hd25519"
	"chain/database/pg"
	"chain/errors"
	"chain/metrics"
	"chain/net/http/httpjson"
)

type Account struct {
	*signers.Signer
	Alias string                 `json:"alias"`
	Tags  map[string]interface{} `json:"tags"`
}

var (
	acpIndexNext int64 // next acp index in our block
	acpIndexCap  int64 // points to end of block
	acpMu        sync.Mutex
)

// Create creates a new Account.
func Create(ctx context.Context, xpubs []string, quorum int, alias string, tags map[string]interface{}, clientToken *string) (*Account, error) {
	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "create signer")
	}
	defer dbtx.Rollback(ctx)

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

	const q = `INSERT INTO accounts (account_id, alias, tags) VALUES ($1, $2, $3)`
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

	err = dbtx.Commit(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "committing create account dbtx")
	}

	err = indexAnnotatedAccount(ctx, account)
	if err != nil {
		return nil, errors.Wrap(err, "indexing annotated account")
	}

	return account, nil
}

// SetTags updates the tags on the provided Account.
func SetTags(ctx context.Context, id, alias string, tags map[string]interface{}) (*Account, error) {
	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "setting tags")
	}
	defer dbtx.Rollback(ctx)

	var acc *Account
	if id != "" {
		acc, err = FindByID(ctx, id)
		if err != nil {
			return nil, errors.Wrap(err)
		}
	} else {
		acc, err = FindByAlias(ctx, alias)
		if err != nil {
			return nil, errors.Wrap(err)
		}
	}

	err = insertAccountTags(ctx, acc.ID, tags)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	err = dbtx.Commit(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "committing create account dbtx")
	}
	acc.Tags = tags

	err = indexAnnotatedAccount(ctx, acc)
	if err != nil {
		return nil, errors.Wrap(err, "indexing annotated account")
	}

	return acc, nil
}

// insertAccountTags inserts a set of tags for the given identifier
// It must take place inside a database transaction.
func insertAccountTags(ctx context.Context, id string, tags map[string]interface{}) error {
	_ = pg.FromContext(ctx).(pg.Tx) // panics if not in a db transaction
	tagsParam, err := tagsToNullString(tags)
	if err != nil {
		return err
	}

	const q = `
		INSERT INTO accounts (account_id, tags) VALUES ($1, $2)
		ON CONFLICT (account_id) DO UPDATE SET tags = $2
`

	_, err = pg.Exec(ctx, q, id, tagsParam)
	if err != nil {
		return errors.Wrap(err)
	}

	return nil
}

// FindByID retrieves an Account record from signer by its accountID
func FindByID(ctx context.Context, id string) (*Account, error) {
	s, err := signers.Find(ctx, "account", id)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	alias, tags, err := fetchAccountData(ctx, s.ID)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return &Account{
		Signer: s,
		Alias:  alias,
		Tags:   tags,
	}, nil
}

func fetchAccountData(ctx context.Context, id string) (string, map[string]interface{}, error) {
	const q = `SELECT alias, tags FROM accounts WHERE account_id=$1`
	var (
		tagBytes []byte
		alias    sql.NullString
	)
	err := pg.QueryRow(ctx, q, id).Scan(&alias, &tagBytes)
	if err != nil {
		return "", nil, errors.Wrap(err)
	}

	var tags map[string]interface{}
	if len(tagBytes) > 0 {
		err := json.Unmarshal(tagBytes, &tags)
		if err != nil {
			return "", nil, errors.Wrap(err)
		}
	}

	if alias.Valid {
		return alias.String, tags, nil
	}

	return "", tags, nil
}

// FindByAlias retrieves an Account record by its alias
func FindByAlias(ctx context.Context, alias string) (*Account, error) {
	const q = `SELECT account_id, tags, archived FROM accounts WHERE alias=$1`
	var (
		tagBytes  []byte
		accountID string
		archived  bool
	)
	err := pg.QueryRow(ctx, q, alias).Scan(&accountID, &tagBytes, &archived)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	if archived {
		return nil, signers.ErrArchived
	}

	var tags map[string]interface{}
	if len(tagBytes) > 0 {
		err := json.Unmarshal(tagBytes, &tags)
		if err != nil {
			return nil, errors.Wrap(err)
		}
	}

	s, err := signers.Find(ctx, "account", accountID)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return &Account{
		Signer: s,
		Alias:  alias,
		Tags:   tags,
	}, nil
}

// Archive marks an Account record as archived,
// effectively "deleting" it.
func Archive(ctx context.Context, id, alias string) error {
	identifier := id
	column := "account_id"
	if id == "" {
		identifier = alias
		column = "alias"
	}

	const q = `
		UPDATE accounts SET archived='t' WHERE %s=$1
		RETURNING account_id
	`

	err := pg.QueryRow(ctx, fmt.Sprintf(q, column), identifier).Scan(&id)
	if err == sql.ErrNoRows {
		return errors.Wrap(pg.ErrUserInputNotFound)
	}

	return signers.Archive(ctx, "account", id)
}

// FindBatch returns a map of Accounts for the provided IDs. The
// account tags on the returned Accounts will not be populated.
func FindBatch(ctx context.Context, ids ...string) (map[string]*Account, error) {
	const q = `
		SELECT id, xpubs, quorum, key_index(key_index)
		FROM signers
		WHERE type='account' AND id = ANY ($1)
	`

	accounts := make(map[string]*Account, len(ids))
	err := pg.ForQueryRows(ctx, q, pg.Strings(ids),
		func(id string, xpubs pg.Strings, quorum int, keyIndex pg.Uint32s) error {
			keys, err := signers.ConvertKeys(xpubs)
			if err != nil {
				return errors.WithDetail(errors.New("bad xpub in databse"), errors.Detail(err))
			}

			a := &Account{
				Signer: &signers.Signer{
					ID:       id,
					Type:     "account",
					XPubs:    keys,
					Quorum:   quorum,
					KeyIndex: keyIndex,
				}}
			accounts[id] = a
			return nil
		},
	)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return accounts, nil
}

// CreateControlProgram creates a control program
// that is tied to the Account and stores it in the database.
func CreateControlProgram(ctx context.Context, accountID string) ([]byte, error) {
	account, err := FindByID(ctx, accountID)
	if err != nil {
		return nil, err
	}

	idx, err := nextIndex(ctx)
	if err != nil {
		return nil, err
	}

	path := signers.Path(account.Signer, signers.AccountKeySpace, idx)
	derivedXPubs := hd25519.DeriveXPubs(account.XPubs, path)
	derivedPKs := hd25519.XPubKeys(derivedXPubs)
	control, redeem, err := vmutil.TxScripts(derivedPKs, account.Quorum)
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

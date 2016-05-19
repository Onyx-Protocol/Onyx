package appdb

import (
	"database/sql"
	"time"

	"golang.org/x/net/context"

	"chain/cos/hdkey"
	"chain/database/pg"
	"chain/errors"
	"chain/log"
	"chain/metrics"
)

// ErrBadAccountKeyCount is returned by CreateAccount when the
// number of keys provided doesn't match the number required by
// the manager node.
var ErrBadAccountKeyCount = errors.New("account has provided wrong number of keys")

// ErrInvalidAccountKey is returned by CreateAccount when the
// key provided isn't valid
var ErrInvalidAccountKey = errors.New("account has provided invalid key")

// Account represents an indexed namespace inside of a manager node
type Account struct {
	ID    string   `json:"id"`
	Label string   `json:"label"`
	Index []uint32 `json:"account_index"`
	Keys  []string `json:"keys"`
}

// CreateAccount inserts an account database record
// for the given manager node, and returns the new Account.
// Parameter keys will be concatenated with the manager node's
// keys when creating redeem scripts for this account.
// The len(keys) must equal variable_keys in the manager node.
func CreateAccount(ctx context.Context, managerNodeID, label string, keys []string, clientToken *string) (*Account, error) {
	defer metrics.RecordElapsed(time.Now())
	if label == "" {
		return nil, errors.WithDetail(ErrBadLabel, "missing/null value")
	}

	account := &Account{Label: label}

	keyCount, err := managerNodeVariableKeys(ctx, managerNodeID)
	if err != nil {
		return nil, errors.Wrap(err, "fetching variable key count for manager node")
	}

	if keyCount != len(keys) {
		return nil, ErrBadAccountKeyCount
	}

	for i, key := range keys {
		xpub, err := hdkey.NewXKey(key)
		if err != nil {
			return nil, errors.WithDetailf(ErrInvalidAccountKey, "key %d: xpub is not valid", i)
		} else if xpub.IsPrivate() {
			return nil, errors.WithDetailf(ErrInvalidAccountKey, "key %d: is xpriv, not xpub", i)
		}
	}

	if len(keys) > 0 {
		account.Keys = keys
	}

	const attempts = 3
	for i := 0; i < attempts; i++ {
		const q = `
			WITH incr AS (
				UPDATE manager_nodes
				SET
					accounts_count=accounts_count+1,
					next_account_index=next_account_index+1
				WHERE id=$1
				RETURNING (next_account_index - 1)
			)
			INSERT INTO accounts (manager_node_id, key_index, label, keys, client_token)
			VALUES ($1, (TABLE incr), $2, $3, $4)
			ON CONFLICT (manager_node_id, client_token) DO NOTHING
			RETURNING id, key_index(key_index)
		`
		err := pg.QueryRow(ctx, q, managerNodeID, label, pg.Strings(keys), clientToken).Scan(
			&account.ID,
			(*pg.Uint32s)(&account.Index),
		)

		// A unique violation error is expected and caused by contention on the account
		// index. We should retry the query.
		if pg.IsUniqueViolation(err) {
			log.Write(ctx, "attempt", i, "error", err)
			if i == attempts-1 {
				return nil, err
			}
			continue
		}

		// No rows indicates that the insertion failed because of a unique index violation
		// on the (maanger_node_id, client_token) index. This indicates that the account
		// was successfully created in a previous request.
		if err == sql.ErrNoRows && clientToken != nil {
			return getAccountByClientToken(ctx, managerNodeID, *clientToken)
		}

		// Any other error is unexpected.
		if err != nil {
			return nil, err
		}

		// The new account was succesfully inserted.
		break
	}

	return account, nil
}

func getAccountByClientToken(ctx context.Context, managerNodeID, clientToken string) (*Account, error) {
	const q = `
		SELECT id, label, key_index(key_index), keys
		FROM accounts
		WHERE manager_node_id = $1 AND client_token = $2
	`
	a := &Account{}
	err := pg.QueryRow(ctx, q, managerNodeID, clientToken).Scan(
		&a.ID, &a.Label, (*pg.Uint32s)(&a.Index), (*pg.Strings)(&a.Keys))
	if err == sql.ErrNoRows {
		return nil, pg.ErrUserInputNotFound
	}
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}
	return a, nil
}

// ListAccounts returns a list of accounts contained in the given manager node.
func ListAccounts(ctx context.Context, managerNodeID string, prev string, limit int) ([]*Account, string, error) {
	q := `
		SELECT id, label, key_index(key_index)
		FROM accounts
		WHERE manager_node_id = $1 AND ($2='' OR id<$2) AND NOT archived
		ORDER BY id DESC LIMIT $3
	`
	var (
		accounts []*Account
		last     string
	)
	err := pg.ForQueryRows(ctx, q, managerNodeID, prev, limit, func(id, label string, index pg.Uint32s) {
		account := &Account{
			ID:    id,
			Label: label,
			Index: index,
		}
		accounts = append(accounts, account)
		last = id
	})
	return accounts, last, err
}

// GetAccount returns a single account.
func GetAccount(ctx context.Context, accountID string) (*Account, error) {
	const q = `
		SELECT label, key_index(key_index)
		FROM accounts
		WHERE id = $1
	`
	a := &Account{ID: accountID}
	err := pg.QueryRow(ctx, q, accountID).Scan(&a.Label, (*pg.Uint32s)(&a.Index))
	if err == sql.ErrNoRows {
		return nil, pg.ErrUserInputNotFound
	}
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}

	return a, nil
}

// UpdateAccount updates the label of an account.
func UpdateAccount(ctx context.Context, accountID string, label *string) error {
	if label == nil {
		return nil
	}
	const q = `UPDATE accounts SET label = $2 WHERE id = $1`
	_, err := pg.Exec(ctx, q, accountID, *label)
	return errors.Wrap(err, "update query")
}

// ArchiveAccount marks an account as archived. Once an account has
// been archived, it does not appear for its manager node, and it cannot
// be used in transactions.
func ArchiveAccount(ctx context.Context, accountID string) error {
	const q = `UPDATE accounts SET archived = true WHERE id = $1`
	_, err := pg.Exec(ctx, q, accountID)
	return errors.Wrap(err, "archive query")
}

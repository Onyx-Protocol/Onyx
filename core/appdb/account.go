package appdb

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/crypto/ed25519/hd25519"
	"chain/database/pg"
	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/log"
	"chain/metrics"
)

var (
	// ErrBadAccountKeyCount is returned by CreateAccount when the
	// number of keys provided doesn't match the number required by
	// the manager node.
	ErrBadAccountKeyCount = errors.New("account has provided wrong number of keys")

	// ErrInvalidAccountKey is returned by CreateAccount when the
	// key provided isn't valid
	ErrInvalidAccountKey = errors.New("account has provided invalid key")

	// ErrBadCursor is returned by functions that receive an invalid
	// pagination cursor.
	ErrBadCursor = errors.New("invalid pagination cursor")
)

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
func CreateAccount(ctx context.Context, managerNodeID, label string, pubkeyStrs []string, clientToken *string) (*Account, error) {
	defer metrics.RecordElapsed(time.Now())
	if label == "" {
		return nil, errors.WithDetail(ErrBadLabel, "missing/null value")
	}

	account := &Account{Label: label}

	keyCount, err := managerNodeVariableKeys(ctx, managerNodeID)
	if err != nil {
		return nil, errors.Wrap(err, "fetching variable key count for account manager")
	}

	if keyCount != len(pubkeyStrs) {
		return nil, ErrBadAccountKeyCount
	}

	for i, pubkeyStr := range pubkeyStrs {
		_, err := hd25519.XPubFromString(pubkeyStr)
		if err != nil {
			return nil, errors.WithDetailf(ErrInvalidAccountKey, "key %d: xpub is not valid", i)
		}
	}

	if len(pubkeyStrs) > 0 {
		account.Keys = pubkeyStrs
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
		err := pg.QueryRow(ctx, q, managerNodeID, label, pg.Strings(pubkeyStrs), clientToken).Scan(
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

// TxOutput describes a single transaction output, and is intended for use in
// API responses.
type TxOutput struct {
	TxHash   bc.Hash            `json:"transaction_id"`
	TxIndex  uint32             `json:"transaction_output"`
	AssetID  bc.AssetID         `json:"asset_id"`
	Amount   uint64             `json:"amount"`
	Address  chainjson.HexBytes `json:"address"` // deprecated
	Script   chainjson.HexBytes `json:"script"`
	Metadata chainjson.HexBytes `json:"metadata"`
}

// ListAccountUTXOs returns UTXOs held by the specified account, optionally
// filtered by a list of assets. In order to simplify ordered pagination, only
// UTXOs in blocks are considered.
func ListAccountUTXOs(ctx context.Context, accountID string, assetIDs []bc.AssetID, cursor string, limit int) ([]*TxOutput, string, error) {
	q := `
		SELECT tx_hash, index, asset_id, amount, script, metadata,
		       confirmed_in, block_pos
		FROM account_utxos
	`

	qparams := []interface{}{accountID}
	criteria := []string{
		"confirmed_in IS NOT NULL", // Only examine UTXOs in blocks
		"account_id = $1",          // filter by the specified account
	}

	// Add asset ID filter, if necessary.
	if len(assetIDs) > 0 {
		var s []string
		for _, id := range assetIDs {
			s = append(s, id.String())
		}
		qparams = append(qparams, pg.Strings(s))
		criteria = append(criteria, fmt.Sprintf("asset_id = ANY($%d)", len(qparams)))
	}

	// Handle pagination, if necessary. We paginate over UTXOs, which are
	// ordered by a compound cursor (confirmed_in, block_pos, index).
	if len(cursor) > 0 {
		var (
			block    int64
			blockPos int
			index    int
		)
		_, err := fmt.Sscanf(cursor, "%d-%d-%d", &block, &blockPos, &index)
		if err != nil {
			return nil, "", errors.WithDetailf(ErrBadCursor, "cursor: %q", cursor)
		}

		qparams = append(qparams, block, blockPos, index)
		criteria = append(
			criteria,
			fmt.Sprintf(
				`(confirmed_in, block_pos, index) > ($%d, $%d, $%d)`,
				len(qparams)-2, len(qparams)-1, len(qparams), // block, block_pos, index
			),
		)
	}

	// Add all criteria to query as a WHERE clause.
	q += " WHERE " + strings.Join(criteria, " AND ")

	// Ensure consistent ordering.
	q += " ORDER BY confirmed_in, block_pos, index"

	// Limit responses based on user input.
	qparams = append(qparams, limit)
	q += fmt.Sprintf(" LIMIT $%d", len(qparams))

	rows, err := pg.Query(ctx, q, qparams...)
	if err != nil {
		return nil, "", errors.Wrap(err, "select query")
	}

	var (
		res        []*TxOutput
		nextCursor string
	)
	for rows.Next() {
		var (
			o        TxOutput
			block    int64
			blockPos int
		)
		err = rows.Scan(
			&o.TxHash,
			&o.TxIndex,
			&o.AssetID,
			&o.Amount,
			(*[]byte)(&o.Script),
			(*[]byte)(&o.Metadata),
			&block,
			&blockPos,
		)
		if err != nil {
			return nil, "", errors.Wrap(err, "row scan")
		}

		// For backward compatibility
		o.Address = o.Script

		// Ensure non-nil empty object for metadata
		if o.Metadata == nil {
			o.Metadata = chainjson.HexBytes{}
		}

		res = append(res, &o)
		nextCursor = fmt.Sprintf("%d-%d-%d", block, blockPos, o.TxIndex)
	}

	if err = rows.Close(); err != nil {
		return nil, "", errors.Wrap(err, "end row scan loop")
	}

	return res, nextCursor, nil
}

// Package account stores and tracks accounts within a Chain Core.
package account

import (
	"context"
	stdsql "database/sql"
	"encoding/json"
	"sync"
	"time"

	"github.com/golang/groupcache/lru"
	"github.com/lib/pq"

	"chain/core/pin"
	"chain/core/signers"
	"chain/core/txbuilder"
	"chain/crypto/ed25519/chainkd"
	"chain/database/pg"
	"chain/errors"
	"chain/log"
	"chain/protocol"
	"chain/protocol/vm/vmutil"
)

const maxAccountCache = 1000

var (
	ErrDuplicateAlias = errors.New("duplicate account alias")
	ErrBadIdentifier  = errors.New("either ID or alias must be specified, and not both")
)

func NewManager(db pg.DB, chain *protocol.Chain, pinStore *pin.Store) *Manager {
	return &Manager{
		db:          db,
		chain:       chain,
		utxoDB:      newReserver(db, chain, pinStore),
		pinStore:    pinStore,
		cache:       lru.New(maxAccountCache),
		aliasCache:  lru.New(maxAccountCache),
		delayedACPs: make(map[*txbuilder.TemplateBuilder][]*controlProgram),
	}
}

// Manager stores accounts and their associated control programs.
type Manager struct {
	db       pg.DB
	chain    *protocol.Chain
	utxoDB   *reserver
	indexer  Saver
	pinStore *pin.Store

	cacheMu    sync.Mutex
	cache      *lru.Cache
	aliasCache *lru.Cache

	delayedACPsMu sync.Mutex
	delayedACPs   map[*txbuilder.TemplateBuilder][]*controlProgram

	acpMu        sync.Mutex
	acpIndexNext uint64 // next acp index in our block
	acpIndexCap  uint64 // points to end of block
}

func (m *Manager) IndexAccounts(indexer Saver) {
	m.indexer = indexer
}

// ExpireReservations removes reservations that have expired periodically.
// It blocks until the context is canceled.
func (m *Manager) ExpireReservations(ctx context.Context, period time.Duration) {
	ticks := time.Tick(period)
	for {
		select {
		case <-ctx.Done():
			log.Printf(ctx, "Deposed, ExpireReservations exiting")
			return
		case <-ticks:
			err := m.utxoDB.ExpireReservations(ctx)
			if err != nil {
				log.Error(ctx, err)
			}
		}
	}
}

type Account struct {
	*signers.Signer
	Alias string
	Tags  map[string]interface{}
}

// Create creates a new Account.
func (m *Manager) Create(ctx context.Context, xpubs []chainkd.XPub, quorum int, alias string, tags map[string]interface{}, clientToken string) (*Account, error) {
	signer, err := signers.Create(ctx, m.db, "account", xpubs, quorum, clientToken)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	tagsParam, err := tagsToNullString(tags)
	if err != nil {
		return nil, err
	}

	aliasSQL := stdsql.NullString{
		String: alias,
		Valid:  alias != "",
	}

	const q = `
		INSERT INTO accounts (account_id, alias, tags) VALUES ($1, $2, $3)
		ON CONFLICT (account_id) DO UPDATE SET alias = $2, tags = $3
	`
	_, err = m.db.ExecContext(ctx, q, signer.ID, aliasSQL, tagsParam)
	if pg.IsUniqueViolation(err) {
		return nil, errors.WithDetail(ErrDuplicateAlias, "an account with the provided alias already exists")
	} else if err != nil {
		return nil, errors.Wrap(err)
	}

	account := &Account{
		Signer: signer,
		Alias:  alias,
		Tags:   tags,
	}

	err = m.indexAnnotatedAccount(ctx, account)
	if err != nil {
		return nil, errors.Wrap(err, "indexing annotated account")
	}

	return account, nil
}

// UpdateTags modifies the tags of the specified account. The account may be
// identified either by ID or Alias, but not both.
func (m *Manager) UpdateTags(ctx context.Context, id, alias *string, tags map[string]interface{}) error {
	if (id == nil) == (alias == nil) {
		return errors.Wrap(ErrBadIdentifier)
	}

	tagsParam, err := tagsToNullString(tags)
	if err != nil {
		return errors.Wrap(err, "convert tags")
	}

	var (
		signer   *signers.Signer
		aliasStr string
	)

	if id != nil {
		signer, err = m.findByID(ctx, *id)
		if err != nil {
			return errors.Wrap(err, "get account by ID")
		}

		// An alias is required by indexAnnotatedAccount. The latter is a somewhat
		// complex function, so in the interest of not making a near-duplicate,
		// we'll satisfy its contract and provide an alias.
		const q = `SELECT alias FROM accounts WHERE account_id = $1`
		var a stdsql.NullString
		err := m.db.QueryRowContext(ctx, q, *id).Scan(&a)
		if err != nil {
			return errors.Wrap(err, "alias lookup")
		}
		if a.Valid {
			aliasStr = a.String
		}
	} else { // alias is guaranteed to be not nil due to bad identifier check
		aliasStr = *alias
		signer, err = m.FindByAlias(ctx, aliasStr)
		if err != nil {
			return errors.Wrap(err, "get account by alias")
		}
	}

	const q = `
		UPDATE accounts
		SET tags = $1
		WHERE account_id = $2
	`
	_, err = m.db.ExecContext(ctx, q, tagsParam, signer.ID)
	if err != nil {
		return errors.Wrap(err, "update entry in accounts table")
	}

	return errors.Wrap(m.indexAnnotatedAccount(ctx, &Account{
		Signer: signer,
		Alias:  aliasStr,
		Tags:   tags,
	}), "update account index")
}

// FindByAlias retrieves an account's Signer record by its alias
func (m *Manager) FindByAlias(ctx context.Context, alias string) (*signers.Signer, error) {
	var accountID string

	m.cacheMu.Lock()
	cachedID, ok := m.aliasCache.Get(alias)
	m.cacheMu.Unlock()
	if ok {
		accountID = cachedID.(string)
	} else {
		const q = `SELECT account_id FROM accounts WHERE alias=$1`
		err := m.db.QueryRowContext(ctx, q, alias).Scan(&accountID)
		if err == stdsql.ErrNoRows {
			return nil, errors.WithDetailf(pg.ErrUserInputNotFound, "alias: %s", alias)
		}
		if err != nil {
			return nil, errors.Wrap(err)
		}
		m.cacheMu.Lock()
		m.aliasCache.Add(alias, accountID)
		m.cacheMu.Unlock()
	}
	return m.findByID(ctx, accountID)
}

// findByID returns an account's Signer record by its ID.
func (m *Manager) findByID(ctx context.Context, id string) (*signers.Signer, error) {
	m.cacheMu.Lock()
	cached, ok := m.cache.Get(id)
	m.cacheMu.Unlock()
	if ok {
		return cached.(*signers.Signer), nil
	}
	account, err := signers.Find(ctx, m.db, "account", id)
	if err != nil {
		return nil, err
	}
	m.cacheMu.Lock()
	m.cache.Add(id, account)
	m.cacheMu.Unlock()
	return account, nil
}

type controlProgram struct {
	accountID      string
	keyIndex       uint64
	controlProgram []byte
	change         bool
	expiresAt      time.Time
}

func (m *Manager) createControlProgram(ctx context.Context, accountID string, change bool, expiresAt time.Time) (*controlProgram, error) {
	account, err := m.findByID(ctx, accountID)
	if err != nil {
		return nil, err
	}

	idx, err := m.nextIndex(ctx)
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
	return &controlProgram{
		accountID:      account.ID,
		keyIndex:       idx,
		controlProgram: control,
		change:         change,
		expiresAt:      expiresAt,
	}, nil
}

// CreateControlProgram creates a control program
// that is tied to the Account and stores it in the database.
func (m *Manager) CreateControlProgram(ctx context.Context, accountID string, change bool, expiresAt time.Time) ([]byte, error) {
	cp, err := m.createControlProgram(ctx, accountID, change, expiresAt)
	if err != nil {
		return nil, err
	}
	err = m.insertAccountControlProgram(ctx, cp)
	if err != nil {
		return nil, err
	}
	return cp.controlProgram, nil
}

func (m *Manager) insertAccountControlProgram(ctx context.Context, progs ...*controlProgram) error {
	const q = `
		INSERT INTO account_control_programs (signer_id, key_index, control_program, change, expires_at)
		SELECT unnest($1::text[]), unnest($2::bigint[]), unnest($3::bytea[]), unnest($4::boolean[]),
			unnest($5::timestamp with time zone[])
	`
	var (
		accountIDs   pq.StringArray
		keyIndexes   pq.Int64Array
		controlProgs pq.ByteaArray
		change       pq.BoolArray
		expirations  []stdsql.NullString
	)
	for _, p := range progs {
		accountIDs = append(accountIDs, p.accountID)
		keyIndexes = append(keyIndexes, int64(p.keyIndex))
		controlProgs = append(controlProgs, p.controlProgram)
		change = append(change, p.change)
		expirations = append(expirations, stdsql.NullString{
			String: p.expiresAt.Format(time.RFC3339),
			Valid:  !p.expiresAt.IsZero(),
		})
	}

	_, err := m.db.ExecContext(ctx, q, accountIDs, keyIndexes, controlProgs, change, pq.Array(expirations))
	return errors.Wrap(err)
}

func (m *Manager) nextIndex(ctx context.Context) (uint64, error) {
	m.acpMu.Lock()
	defer m.acpMu.Unlock()

	if m.acpIndexNext >= m.acpIndexCap {
		var cap uint64
		const incrby = 10000 // account_control_program_seq increments by 10,000
		const q = `SELECT nextval('account_control_program_seq')`
		err := m.db.QueryRowContext(ctx, q).Scan(&cap)
		if err != nil {
			return 0, errors.Wrap(err, "scan")
		}
		m.acpIndexCap = cap
		m.acpIndexNext = cap - incrby
	}

	n := m.acpIndexNext
	m.acpIndexNext++
	return n, nil
}

func tagsToNullString(tags map[string]interface{}) (*stdsql.NullString, error) {
	var tagsJSON []byte
	if len(tags) != 0 {
		var err error
		tagsJSON, err = json.Marshal(tags)
		if err != nil {
			return nil, errors.Wrap(err)
		}
	}
	return &stdsql.NullString{String: string(tagsJSON), Valid: len(tagsJSON) > 0}, nil
}

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
	"chain/database/sql"
	"chain/errors"
	"chain/log"
	"chain/protocol"
	"chain/protocol/vmutil"
)

const maxAccountCache = 1000

var ErrDuplicateAlias = errors.New("duplicate account alias")

func NewManager(db *sql.DB, chain *protocol.Chain, pinStore *pin.Store) *Manager {
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
	db       *sql.DB
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
			log.Messagef(ctx, "Deposed, ExpireReservations exiting")
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
func (m *Manager) Create(ctx context.Context, xpubs []string, quorum int, alias string, tags map[string]interface{}, clientToken *string) (*Account, error) {
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
	_, err = m.db.Exec(ctx, q, signer.ID, aliasSQL, tagsParam)
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
		err := m.db.QueryRow(ctx, q, alias).Scan(&accountID)
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
}

func (m *Manager) createControlProgram(ctx context.Context, accountID string, change bool) (*controlProgram, error) {
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
	}, nil
}

// CreateControlProgram creates a control program
// that is tied to the Account and stores it in the database.
func (m *Manager) CreateControlProgram(ctx context.Context, accountID string, change bool) ([]byte, error) {
	cp, err := m.createControlProgram(ctx, accountID, change)
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
		INSERT INTO account_control_programs (signer_id, key_index, control_program, change)
		SELECT unnest($1::text[]), unnest($2::bigint[]), unnest($3::bytea[]), unnest($4::boolean[])
	`
	var (
		accountIDs   pq.StringArray
		keyIndexes   pq.Int64Array
		controlProgs pq.ByteaArray
		change       pq.BoolArray
	)
	for _, p := range progs {
		accountIDs = append(accountIDs, p.accountID)
		keyIndexes = append(keyIndexes, int64(p.keyIndex))
		controlProgs = append(controlProgs, p.controlProgram)
		change = append(change, p.change)
	}

	_, err := m.db.Exec(ctx, q, accountIDs, keyIndexes, controlProgs, change)
	return errors.Wrap(err)
}

func (m *Manager) nextIndex(ctx context.Context) (uint64, error) {
	m.acpMu.Lock()
	defer m.acpMu.Unlock()

	if m.acpIndexNext >= m.acpIndexCap {
		var cap uint64
		const incrby = 10000 // account_control_program_seq increments by 10,000
		const q = `SELECT nextval('account_control_program_seq')`
		err := m.db.QueryRow(ctx, q).Scan(&cap)
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

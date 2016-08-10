package query

import (
	"database/sql"
	"sync"
	"time"

	"golang.org/x/net/context"

	"chain/core/query/chql"
	"chain/cos"
	"chain/database/pg"
	"chain/errors"
	chainlog "chain/log"
)

const (
	indexRefreshPeriod = time.Minute
)

var (
	ErrParsingQuery      = errors.New("error parsing ChQL query")
	ErrTooManyParameters = errors.New("transaction ChQL queries support up to 1 parameter")
)

// NewIndexer constructs a new indexer for indexing transactions.
func NewIndexer(db pg.DB, fc *cos.FC) *Indexer {
	indexer := &Indexer{
		db:      db,
		indexes: make(map[string]*Index),
	}
	fc.AddBlockCallback(indexer.indexBlockCallback)
	return indexer
}

// Indexer creates, updates and queries against ChQL indexes.
type Indexer struct {
	db      pg.DB
	mu      sync.Mutex // protects indexes
	indexes map[string]*Index
}

// BeginIndexing must be called before blocks are processed to refresh
// the indexes.
func (i *Indexer) BeginIndexing(ctx context.Context) error {
	err := i.refreshIndexes(ctx)
	if err != nil {
		return err
	}

	go func() {
		ticker := time.NewTicker(indexRefreshPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				err := i.refreshIndexes(ctx)
				if err != nil {
					chainlog.Error(ctx, err)
				}
			}
		}
	}()
	return nil
}

// Index represents a transaction index on a particular ChQL query.
type Index struct {
	ID         string     `json:"id"`   // unique, external string identifier
	Type       string     `json:"type"` // 'transaction', 'output', etc.
	Query      chql.Query `json:"query"`
	internalID int        // unique, internal pg serial id
	rawQuery   string
	createdAt  time.Time
}

// GetIndex looks up an individual index by its ID and its type.
func (i *Indexer) GetIndex(ctx context.Context, id, typ string) (*Index, error) {
	const selectQ = `
		SELECT internal_id, id, type, query, created_at FROM query_indexes
		WHERE id = $1 AND type = $2
	`
	var idx Index
	err := i.db.QueryRow(ctx, selectQ, id, typ).
		Scan(&idx.internalID, &idx.ID, &idx.Type, &idx.rawQuery, &idx.createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "looking up index")
	}
	idx.Query, err = chql.Parse(idx.rawQuery)
	return &idx, err
}

// CreateIndex commits a new index in the database. Blockchain data
// will not be indexed until the leader process picks up the new index.
func (i *Indexer) CreateIndex(ctx context.Context, id string, typ string, rawQuery string) (*Index, error) {
	q, err := chql.Parse(rawQuery)
	if err != nil {
		return nil, errors.WithDetail(ErrParsingQuery, err.Error())
	}
	if typ == "transaction" && q.Parameters > 1 {
		return nil, ErrTooManyParameters
	}

	const insertQ = `
		INSERT INTO query_indexes (id, type, query) VALUES($1, $2, $3)
		ON CONFLICT (id) DO UPDATE SET type = $2, query = $3
		RETURNING internal_id, created_at
	`
	idx := &Index{
		ID:       id,
		Type:     typ,
		Query:    q,
		rawQuery: rawQuery,
	}
	err = i.db.QueryRow(ctx, insertQ, id, typ, rawQuery).Scan(&idx.internalID, &idx.createdAt)
	if err != nil {
		return nil, errors.Wrap(err, "saving tx index in db")
	}
	return idx, nil
}

// ListIndexes lists all registered indexes.
func (i *Indexer) ListIndexes(ctx context.Context, cursor string, limit int) ([]*Index, string, error) {
	indexes, newCursor, err := i.listIndexes(ctx, cursor, limit)
	if err != nil {
		return nil, "", errors.Wrap(err, "retrieving indexes")
	}

	// Parse all the queries so that we can print a cleaned
	// represenation of the query.
	for _, idx := range indexes {
		idx.Query, err = chql.Parse(idx.rawQuery)
		if err != nil {
			return nil, "", errors.Wrap(err, "parsing raw query")
		}
	}
	return indexes, newCursor, nil
}

func (i *Indexer) isIndexActive(id string) bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	_, ok := i.indexes[id]
	return ok
}

func (i *Indexer) setupIndex(idx *Index) (err error) {
	idx.Query, err = chql.Parse(idx.rawQuery)
	if err != nil {
		return errors.Wrap(err, "parsing raw query for index", idx.ID)
	}

	i.mu.Lock()
	defer i.mu.Unlock()
	i.indexes[idx.ID] = idx
	return nil
}

func (i *Indexer) refreshIndexes(ctx context.Context) error {
	indexes, err := i.getIndexes(ctx)
	if err != nil {
		return err
	}

	for _, index := range indexes {
		if i.isIndexActive(index.ID) {
			continue
		}

		err := i.setupIndex(index)
		if err != nil {
			chainlog.Fatal(ctx, chainlog.KeyError, err)
		}
	}
	return nil
}

// getIndexes queries the database for all active indexes.
// getIndexes does not parse idx.RawQuery and leaves
// idx.Query as nil.
func (i *Indexer) getIndexes(ctx context.Context) ([]*Index, error) {
	const q = `SELECT internal_id, id, type, query, created_at FROM query_indexes`
	rows, err := i.db.Query(ctx, q)
	if err != nil {
		return nil, errors.Wrap(err, "reload indexes sql query")
	}
	defer rows.Close()

	var indexes []*Index
	for rows.Next() {
		idx := new(Index)
		err = rows.Scan(&idx.internalID, &idx.ID, &idx.Type, &idx.rawQuery, &idx.createdAt)
		if err != nil {
			return nil, errors.Wrap(err, "scanning query_indexes row")
		}
		indexes = append(indexes, idx)
	}
	return indexes, errors.Wrap(rows.Err())
}

// listIndexes behaves almost identically to getIndexes.
// The caveat is listIndexes returns a paged result.
func (i *Indexer) listIndexes(ctx context.Context, cursor string, limit int) ([]*Index, string, error) {
	const q = `
		SELECT internal_id, id, type, query, created_at
		FROM query_indexes WHERE ($1='' OR $1<id)
		ORDER BY id ASC LIMIT $2
	`

	rows, err := i.db.Query(ctx, q, cursor, limit)
	if err != nil {
		return nil, "", errors.Wrap(err, "reload indexes sql query")
	}
	defer rows.Close()

	var indexes []*Index
	for rows.Next() {
		idx := new(Index)
		err = rows.Scan(&idx.internalID, &idx.ID, &idx.Type, &idx.rawQuery, &idx.createdAt)
		if err != nil {
			return nil, "", errors.Wrap(err, "scanning query_indexes row")
		}
		indexes = append(indexes, idx)
	}

	var last string
	if len(indexes) > 0 {
		last = indexes[len(indexes)-1].ID
	}
	return indexes, last, errors.Wrap(rows.Err())
}

package query

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
	"time"

	"chain/core/query/chql"
	"chain/cos"
	"chain/database/pg"
	"chain/errors"
	chainlog "chain/log"
	"chain/net/http/httpjson"
)

const (
	indexRefreshPeriod = time.Minute
)

// Valid index types
const (
	IndexTypeAsset       = "asset"
	IndexTypeBalance     = "balance"
	IndexTypeTransaction = "transaction"
)

var IndexTypes = map[string]bool{
	IndexTypeAsset:       true,
	IndexTypeBalance:     true,
	IndexTypeTransaction: true,
}

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
	db         pg.DB
	mu         sync.Mutex // protects indexes
	indexes    map[string]*Index
	annotators []Annotator
}

// BeginIndexing must be called before blocks are processed to refresh
// the indexes.
func (ind *Indexer) BeginIndexing(ctx context.Context) error {
	err := ind.refreshIndexes(ctx)
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
				err := ind.refreshIndexes(ctx)
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
	ID        string // unique, chain ID
	Alias     string // unique, external string identifier
	Type      string // 'transaction', 'balance', etc.
	Query     chql.Query
	Unspents  bool // only for balance indexes
	rawQuery  string
	createdAt time.Time
}

// MarshalJSON implements json.Marshaler and correctly marshals the 'unspents'
// field only if the index is a balance index.
func (i *Index) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"id":    i.ID,
		"alias": i.Alias,
		"type":  i.Type,
		"query": i.Query.String(),
	}
	if i.Type == "balance" {
		m["unspents"] = i.Unspents
	}
	return json.Marshal(m)
}

// GetIndex looks up an individual index by its ID or alias and its type.
func (ind *Indexer) GetIndex(ctx context.Context, id, alias, typ string) (*Index, error) {
	const selectQ = `
		SELECT id, alias, type, query, created_at, unspent_outputs FROM query_indexes
		WHERE (($1 != '' AND id = $1) OR ($2 != '' AND alias = $2)) AND type = $3
	`
	var idx Index
	err := ind.db.QueryRow(ctx, selectQ, id, alias, typ).
		Scan(&idx.ID, &idx.Alias, &idx.Type, &idx.rawQuery, &idx.createdAt, &idx.Unspents)
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
func (ind *Indexer) CreateIndex(ctx context.Context, alias, typ, rawQuery string, unspents bool) (*Index, error) {
	q, err := chql.Parse(rawQuery)
	if err != nil {
		return nil, errors.WithDetail(ErrParsingQuery, err.Error())
	}
	if typ == IndexTypeTransaction && q.Parameters > 1 {
		return nil, ErrTooManyParameters
	}

	const insertQ = `
		INSERT INTO query_indexes (alias, type, query, unspent_outputs) VALUES($1, $2, $3, $4)
		RETURNING id, created_at
	`
	idx := &Index{
		Alias:    alias,
		Type:     typ,
		Query:    q,
		rawQuery: rawQuery,
	}
	err = ind.db.QueryRow(ctx, insertQ, alias, typ, rawQuery, unspents).Scan(&idx.ID, &idx.createdAt)
	if err != nil {
		if pg.IsUniqueViolation(err) {
			return nil, errors.WithDetail(httpjson.ErrBadRequest, "non-unique alias")
		}
		return nil, errors.Wrap(err, "saving tx index in db")
	}
	return idx, nil
}

// ListIndexes lists all registered indexes.
func (ind *Indexer) ListIndexes(ctx context.Context, cursor string, limit int) ([]*Index, string, error) {
	indexes, newCursor, err := ind.listIndexes(ctx, cursor, limit)
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

func (ind *Indexer) isIndexActive(id string) bool {
	ind.mu.Lock()
	defer ind.mu.Unlock()
	_, ok := ind.indexes[id]
	return ok
}

func (ind *Indexer) setupIndex(idx *Index) (err error) {
	idx.Query, err = chql.Parse(idx.rawQuery)
	if err != nil {
		return errors.Wrap(err, "parsing raw query for index", idx.Alias)
	}

	ind.mu.Lock()
	defer ind.mu.Unlock()
	ind.indexes[idx.Alias] = idx
	return nil
}

func (ind *Indexer) refreshIndexes(ctx context.Context) error {
	indexes, err := ind.getIndexes(ctx)
	if err != nil {
		return err
	}

	for _, index := range indexes {
		if ind.isIndexActive(index.Alias) {
			continue
		}

		err := ind.setupIndex(index)
		if err != nil {
			chainlog.Fatal(ctx, chainlog.KeyError, err)
		}
	}
	return nil
}

// getIndexes queries the database for all active indexes.
// getIndexes does not parse idx.RawQuery and leaves
// idx.Query as nil.
func (ind *Indexer) getIndexes(ctx context.Context) ([]*Index, error) {
	const q = `SELECT id, alias, type, query, created_at, unspent_outputs FROM query_indexes`
	rows, err := ind.db.Query(ctx, q)
	if err != nil {
		return nil, errors.Wrap(err, "reload indexes sql query")
	}
	defer rows.Close()

	var indexes []*Index
	for rows.Next() {
		idx := new(Index)
		err = rows.Scan(&idx.ID, &idx.Alias, &idx.Type, &idx.rawQuery, &idx.createdAt, &idx.Unspents)
		if err != nil {
			return nil, errors.Wrap(err, "scanning query_indexes row")
		}
		indexes = append(indexes, idx)
	}
	return indexes, errors.Wrap(rows.Err())
}

// listIndexes behaves almost identically to getIndexes.
// The caveat is listIndexes returns a paged result.
func (ind *Indexer) listIndexes(ctx context.Context, cursor string, limit int) ([]*Index, string, error) {
	const q = `
		SELECT id, alias, type, query, created_at, unspent_outputs
		FROM query_indexes WHERE ($1='' OR $1<id)
		ORDER BY id ASC LIMIT $2
	`

	rows, err := ind.db.Query(ctx, q, cursor, limit)
	if err != nil {
		return nil, "", errors.Wrap(err, "reload indexes sql query")
	}
	defer rows.Close()

	var indexes []*Index
	for rows.Next() {
		idx := new(Index)
		err = rows.Scan(&idx.ID, &idx.Alias, &idx.Type, &idx.rawQuery, &idx.createdAt, &idx.Unspents)
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

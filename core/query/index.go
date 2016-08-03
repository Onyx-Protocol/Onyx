package query

import (
	"time"

	"golang.org/x/net/context"

	"chain/core/query/cql"
	"chain/database/pg"
	"chain/errors"
)

var (
	ErrParsingQuery      = errors.New("error parsing CQL query")
	ErrTooManyParameters = errors.New("transaction CQL queries support up to 1 parameter")
)

// NewIndexer constructs a new indexer for indexing transactions.
func NewIndexer(db pg.DB) *Indexer {
	return &Indexer{db: db}
}

// Indexer creates, updates and queries against CQL indexes.
type Indexer struct {
	db pg.DB
}

// Index represents a transaction index on a particular CQL query.
type Index struct {
	ID         string    `json:"id"`   // unique, external string identifier
	Type       string    `json:"type"` // 'transaction', 'output', etc.
	Query      cql.Query `json:"query"`
	internalID int       // unique, internal pg serial id
	rawQuery   string
	createdAt  time.Time
}

// CreateIndex commits a new index in the database. Blockchain data
// will not be indexed until the leader process picks up the new index.
func (i *Indexer) CreateIndex(ctx context.Context, id string, typ string, rawQuery string) (*Index, error) {
	q, err := cql.Parse(rawQuery)
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
func (i *Indexer) ListIndexes(ctx context.Context) ([]*Index, error) {
	indexes, err := i.getIndexes(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving indexes")
	}

	// Parse all the queries so that we can print a cleaned
	// represenation of the query.
	for _, idx := range indexes {
		idx.Query, err = cql.Parse(idx.rawQuery)
		if err != nil {
			return nil, errors.Wrap(err, "parsing raw query")
		}
	}
	return indexes, nil
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

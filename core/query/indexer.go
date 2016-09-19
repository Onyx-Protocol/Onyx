package query

import (
	"encoding/json"
	"sync"
	"time"

	"chain/core/query/filter"
	"chain/database/pg"
	"chain/errors"
	"chain/protocol"
)

const (
	indexRefreshPeriod = time.Minute
)

// Valid index types
const (
	IndexTypeAsset       = "asset"
	IndexTypeBalance     = "balance"
	IndexTypeOutput      = "output"
	IndexTypeTransaction = "transaction"
)

var IndexTypes = map[string]bool{
	IndexTypeAsset:       true,
	IndexTypeBalance:     true,
	IndexTypeOutput:      true,
	IndexTypeTransaction: true,
}

var (
	ErrParsingFilter     = errors.New("error parsing filter")
	ErrTooManyParameters = errors.New("transaction filters support up to 1 parameter")
)

// NewIndexer constructs a new indexer for indexing transactions.
func NewIndexer(db pg.DB, c *protocol.Chain) *Indexer {
	indexer := &Indexer{
		db:      db,
		indexes: make(map[string]*Index),
		c:       c,
	}
	c.AddBlockCallback(indexer.indexBlockCallback)
	return indexer
}

// Indexer creates, updates and queries against indexes.
type Indexer struct {
	db         pg.DB
	c          *protocol.Chain
	mu         sync.Mutex // protects indexes
	indexes    map[string]*Index
	annotators []Annotator
}

// Index represents a transaction index configured with a particular filter.
type Index struct {
	ID           string // unique, chain ID
	Alias        string // unique, external string identifier
	Type         string // 'transaction', 'balance', etc.
	Predicate    filter.Predicate
	SumBy        []filter.Field // only for 'balance' indexes
	rawPredicate string
	rawSumBy     []string
	createdAt    time.Time
}

// Parse parses the Index's rawQuery and rawSumBy, populating the Query
// and SumBy fields with the AST representations.
func (i *Index) Parse() (err error) {
	i.Predicate, err = filter.Parse(i.rawPredicate)
	if err != nil {
		return errors.Wrap(err, "parsing index query")
	}
	for _, rawField := range i.rawSumBy {
		field, err := filter.ParseField(rawField)
		if err != nil {
			return errors.Wrap(err, "parsing index field")
		}
		i.SumBy = append(i.SumBy, field)
	}
	return nil
}

// MarshalJSON implements json.Marshaler and correctly marshals the 'sum_by'
// field only if the index is a balance index.
func (i *Index) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"id":     i.ID,
		"alias":  i.Alias,
		"type":   i.Type,
		"filter": i.Predicate.String(),
	}

	if i.Type == IndexTypeBalance {
		cleaned := []string{}
		for _, f := range i.SumBy {
			cleaned = append(cleaned, f.String())
		}
		m["sum_by"] = cleaned
	}
	return json.Marshal(m)
}

package query

import (
	"chain/database/pg"
	"chain/protocol"
)

// Valid index types
const (
	IndexTypeAsset       = "asset"
	IndexTypeBalance     = "balance"
	IndexTypeOutput      = "output"
	IndexTypeTransaction = "transaction"
)

// NewIndexer constructs a new indexer for indexing transactions.
func NewIndexer(db pg.DB, c *protocol.Chain) *Indexer {
	indexer := &Indexer{
		db: db,
		c:  c,
	}
	c.AddBlockCallback(indexer.indexBlockCallback)
	return indexer
}

// Indexer creates, updates and queries against indexes.
type Indexer struct {
	db         pg.DB
	c          *protocol.Chain
	annotators []Annotator
}

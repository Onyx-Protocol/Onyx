package query

import (
	"chain/database/pg"
	"chain/protocol"
)

// NewIndexer constructs a new indexer for indexing transactions.
func NewIndexer(db pg.DB, c *protocol.Chain) *Indexer {
	indexer := &Indexer{
		db: db,
		c:  c,
	}
	return indexer
}

// Indexer creates, updates and queries against indexes.
type Indexer struct {
	db         pg.DB
	c          *protocol.Chain
	annotators []Annotator
}

func (ind *Indexer) IndexTransactions() {
	ind.c.AddBlockCallback(ind.indexBlockCallback)
}

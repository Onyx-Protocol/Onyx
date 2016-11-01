package query

import (
	"chain/core/pin"
	"chain/database/pg"
	"chain/protocol"
)

// NewIndexer constructs a new indexer for indexing transactions.
func NewIndexer(db pg.DB, c *protocol.Chain, pinStore *pin.Store) *Indexer {
	indexer := &Indexer{
		db:       db,
		c:        c,
		pinStore: pinStore,
	}
	return indexer
}

// Indexer creates, updates and queries against indexes.
type Indexer struct {
	db         pg.DB
	c          *protocol.Chain
	pinStore   *pin.Store
	annotators []Annotator
}

package query

import (
	"chain/core/processor"
	"chain/database/pg"
	"chain/protocol"
)

// NewIndexer constructs a new indexer for indexing transactions.
func NewIndexer(db pg.DB, c *protocol.Chain, cursorStore *processor.CursorStore) *Indexer {
	indexer := &Indexer{
		db:          db,
		c:           c,
		cursorStore: cursorStore,
	}
	return indexer
}

// Indexer creates, updates and queries against indexes.
type Indexer struct {
	db          pg.DB
	c           *protocol.Chain
	cursorStore *processor.CursorStore
	annotators  []Annotator
}

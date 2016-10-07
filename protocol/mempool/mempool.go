// Package mempool provides a Pool implementation that keeps
// all pending transactions in memory.
//
// It is used in tests to avoid needing a database and is not
// safe for concurrent access.
package mempool

import (
	"context"

	"chain/protocol/bc"
)

// MemPool satisfies the protocol.Pool interface.
type MemPool struct {
	pool []*bc.Tx // in topological order
}

// New returns a new MemPool.
func New() *MemPool {
	return &MemPool{}
}

// Insert adds a new pending tx to the pending tx pool.
func (m *MemPool) Insert(ctx context.Context, tx *bc.Tx) error {
	m.pool = append(m.pool, tx)
	return nil
}

// Dump returns all pending transactions in the pool and
// empties the pool.
func (m *MemPool) Dump(context.Context) ([]*bc.Tx, error) {
	txs := m.pool[:len(m.pool):len(m.pool)]
	m.pool = nil
	return txs, nil
}

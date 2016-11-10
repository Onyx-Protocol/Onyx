// Package mempool provides a Pool implementation that keeps
// all pending transactions in memory.
//
// It is used in tests to avoid needing a database and is not
// safe for concurrent access.
package mempool

import (
	"context"
	"sync"

	"chain/protocol/bc"
)

// MemPool satisfies the protocol.Pool interface.
type MemPool struct {
	mu   sync.Mutex
	pool []*bc.Tx // in topological order
}

// New returns a new MemPool.
func New() *MemPool {
	return &MemPool{}
}

// Insert adds a new pending tx to the pending tx pool.
func (m *MemPool) Insert(ctx context.Context, tx *bc.Tx) error {
	m.mu.Lock()
	m.pool = append(m.pool, tx)
	m.mu.Unlock()
	return nil
}

// Dump returns all pending transactions in the pool and
// empties the pool.
func (m *MemPool) Dump(context.Context) ([]*bc.Tx, error) {
	m.mu.Lock()
	txs := m.pool
	m.pool = make([]*bc.Tx, 0, len(txs))
	m.mu.Unlock()
	return txs, nil
}

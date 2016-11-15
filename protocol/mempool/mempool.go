// Package mempool provides a Pool implementation that keeps
// all pending transactions in memory.
//
// It is used in tests to avoid needing a database and is not
// safe for concurrent access.
package mempool

import (
	"context"
	"sync"

	"chain/log"
	"chain/protocol/bc"
)

// MemPool satisfies the protocol.Pool interface.
type MemPool struct {
	mu     sync.Mutex
	pool   []*bc.Tx // in topological order
	hashes map[bc.Hash]bool
}

// New returns a new MemPool.
func New() *MemPool {
	return &MemPool{hashes: make(map[bc.Hash]bool)}
}

// Insert adds a new pending tx to the pending tx pool.
func (m *MemPool) Insert(ctx context.Context, tx *bc.Tx) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.hashes[tx.Hash] {
		return nil
	}

	m.hashes[tx.Hash] = true
	m.pool = append(m.pool, tx)
	return nil
}

// Dump returns all pending transactions in the pool and
// empties the pool.
func (m *MemPool) Dump(ctx context.Context) ([]*bc.Tx, error) {
	m.mu.Lock()
	txs := m.pool
	m.pool = nil
	m.hashes = make(map[bc.Hash]bool)
	m.mu.Unlock()

	if !isTopSorted(txs) {
		log.Messagef(ctx, "set of %d txs not in topo order; sorting", len(txs))
		txs = topSort(txs)
	}

	return txs, nil
}

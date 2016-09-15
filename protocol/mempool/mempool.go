package mempool

import (
	"context"

	"chain/protocol/bc"
)

// MemPool satisfies the protocol.Pool interface.
// It is used by tests to avoid needing a database.
type MemPool struct {
	pool    []*bc.Tx // used for keeping topological order
	poolMap map[bc.Hash]*bc.Tx
}

// New returns a new MemPool.
func New() *MemPool {
	return &MemPool{
		poolMap: make(map[bc.Hash]*bc.Tx),
	}
}

// Insert adds a new pending tx to the pending tx pool.
func (m *MemPool) Insert(ctx context.Context, tx *bc.Tx) error {
	m.poolMap[tx.Hash] = tx
	m.pool = append(m.pool, tx)
	return nil
}

// Dump returns all pending transactions in the pool.
func (m *MemPool) Dump(context.Context) ([]*bc.Tx, error) {
	return m.pool[:len(m.pool):len(m.pool)], nil
}

// Clean removes txs from the pool.
func (m *MemPool) Clean(
	ctx context.Context,
	txs []*bc.Tx,
) error {
	for _, tx := range txs {
		delete(m.poolMap, tx.Hash)
		for i := range m.pool {
			if m.pool[i].Hash == tx.Hash {
				m.pool = append(m.pool[:i], m.pool[i+1:]...)
				break
			}
		}
	}
	return nil
}

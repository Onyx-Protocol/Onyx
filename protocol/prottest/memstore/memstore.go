// MemStore is a Store implementation that
// keeps all blockchain state in memory.
//
// It is used in tests to avoid needing a database.
package memstore

import (
	"context"
	"fmt"
	"sync"

	"chain/protocol/bc/bcvm"
	"chain/protocol/state"
)

// MemStore satisfies the Store interface.
type MemStore struct {
	mu          sync.Mutex
	Blocks      map[uint64]*bcvm.Block
	State       *state.Snapshot
	StateHeight uint64
}

// New returns a new MemStore
func New() *MemStore {
	return &MemStore{Blocks: make(map[uint64]*bcvm.Block)}
}

func (m *MemStore) Height(context.Context) (uint64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return uint64(len(m.Blocks)), nil
}

func (m *MemStore) SaveBlock(ctx context.Context, b *bcvm.Block) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.Blocks[b.Height]
	if ok && existing.Hash() != b.Hash() {
		return fmt.Errorf("already have a block at height %d", b.Height)
	}
	m.Blocks[b.Height] = b
	return nil
}

func (m *MemStore) SaveSnapshot(ctx context.Context, height uint64, snapshot *state.Snapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.State = state.Copy(snapshot)
	m.StateHeight = height
	return nil
}

func (m *MemStore) GetBlock(ctx context.Context, height uint64) (*bcvm.Block, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.Blocks[height]
	if !ok {
		return nil, fmt.Errorf("memstore: no block at height %d", height)
	}
	return b, nil
}

func (m *MemStore) LatestSnapshot(context.Context) (*state.Snapshot, uint64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.State == nil {
		m.State = state.Empty()
	}
	return state.Copy(m.State), m.StateHeight, nil
}

func (m *MemStore) FinalizeBlock(context.Context, uint64) error { return nil }

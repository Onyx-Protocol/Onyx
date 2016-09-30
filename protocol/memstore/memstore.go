package memstore

import (
	"context"
	"sync"

	"chain/protocol/bc"
	"chain/protocol/state"
)

// MemStore satisfies the protocol.Store interface.
// It is used by tests to avoid needing a database.
// All its fields are exported
// so tests can directly inspect their values.
type MemStore struct {
	mu          sync.Mutex
	Blocks      []*bc.Block
	State       *state.Snapshot
	StateHeight uint64
}

// New returns a new MemStore
func New() *MemStore {
	return new(MemStore)
}

func (m *MemStore) Height(context.Context) (uint64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return uint64(len(m.Blocks)), nil
}

func (m *MemStore) SaveBlock(ctx context.Context, b *bc.Block) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Blocks = append(m.Blocks, b)
	return nil
}

func (m *MemStore) SaveSnapshot(ctx context.Context, height uint64, snapshot *state.Snapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.State = state.Copy(snapshot)
	m.StateHeight = height
	return nil
}

func (m *MemStore) GetBlock(ctx context.Context, height uint64) (*bc.Block, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	index := height - 1
	if index < 0 || index >= uint64(len(m.Blocks)) {
		return nil, nil
	}
	return m.Blocks[index], nil
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

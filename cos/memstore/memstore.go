package memstore

import (
	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/cos/state"
)

// MemStore satisfies the cos.Store interface.
// It is used by tests to avoid needing a database.
// All its fields are exported
// so tests can directly inspect their values.
type MemStore struct {
	Blocks      []*bc.Block
	BlockTxs    map[bc.Hash]*bc.Tx
	State       *state.Snapshot
	StateHeight uint64
}

// New returns a new MemStore
func New() *MemStore {
	return &MemStore{BlockTxs: make(map[bc.Hash]*bc.Tx)}
}

func (m *MemStore) Height(context.Context) (uint64, error) {
	return uint64(len(m.Blocks)), nil
}

func (m *MemStore) GetTxs(ctx context.Context, hashes ...bc.Hash) (bcTxs map[bc.Hash]*bc.Tx, err error) {
	bcTxs = make(map[bc.Hash]*bc.Tx)
	for _, hash := range hashes {
		if tx := m.BlockTxs[hash]; tx != nil {
			bcTxs[hash] = m.BlockTxs[hash]
		}
	}
	return bcTxs, nil
}

func (m *MemStore) SaveBlock(ctx context.Context, b *bc.Block) error {
	m.Blocks = append(m.Blocks, b)

	// Record all the new transactions.
	for _, tx := range b.Transactions {
		m.BlockTxs[tx.Hash] = tx
	}
	return nil
}

func (m *MemStore) SaveSnapshot(ctx context.Context, height uint64, snapshot *state.Snapshot) error {
	m.State = state.Copy(snapshot)
	m.StateHeight = height
	return nil
}

func (m *MemStore) GetBlock(ctx context.Context, height uint64) (*bc.Block, error) {
	index := height - 1
	if index < 0 || index >= uint64(len(m.Blocks)) {
		return nil, nil
	}
	return m.Blocks[index], nil
}

func (m *MemStore) LatestSnapshot(context.Context) (*state.Snapshot, uint64, error) {
	if m.State == nil {
		m.State = state.Empty()
	}
	return state.Copy(m.State), m.StateHeight, nil
}

func (m *MemStore) FinalizeBlock(context.Context, uint64) error { return nil }

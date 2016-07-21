package memstore

import (
	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/cos/patricia"
)

// MemStore satisfies the cos.Store interface.
// It is used by tests to avoid needing a database.
// All its fields are exported
// so tests can directly inspect their values.
type MemStore struct {
	Blocks      []*bc.Block
	BlockTxs    map[bc.Hash]*bc.Tx
	State       *patricia.Tree
	StateHeight uint64
}

// New returns a new MemStore
func New() *MemStore {
	return &MemStore{BlockTxs: make(map[bc.Hash]*bc.Tx)}
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

func (m *MemStore) SaveStateTree(ctx context.Context, height uint64, tree *patricia.Tree) error {
	m.State = patricia.Copy(tree)
	m.StateHeight = height
	return nil
}

func (m *MemStore) LatestBlock(context.Context) (*bc.Block, error) {
	if len(m.Blocks) == 0 {
		return nil, nil
	}
	return m.Blocks[len(m.Blocks)-1], nil
}

func (m *MemStore) LatestStateTree(context.Context) (*patricia.Tree, uint64, error) {
	if m.State == nil {
		m.State = patricia.NewTree(nil)
	}
	return patricia.Copy(m.State), m.StateHeight, nil
}

func (m *MemStore) FinalizeBlock(context.Context, uint64) error { return nil }

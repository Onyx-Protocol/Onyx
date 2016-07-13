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
	Blocks   []*bc.Block
	BlockTxs map[bc.Hash]*bc.Tx
	State    *patricia.Tree
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

func (m *MemStore) ApplyBlock(
	ctx context.Context,
	b *bc.Block,
	stateTree *patricia.Tree,
) ([]*bc.Tx, error) {
	m.Blocks = append(m.Blocks, b)

	// Record all the new transactions.
	var newTxs []*bc.Tx
	for _, tx := range b.Transactions {
		newTxs = append(newTxs, tx)
		m.BlockTxs[tx.Hash] = tx
	}

	m.State = patricia.Copy(stateTree)
	return newTxs, nil
}

func (m *MemStore) LatestBlock(context.Context) (*bc.Block, error) {
	if len(m.Blocks) == 0 {
		return nil, nil
	}
	return m.Blocks[len(m.Blocks)-1], nil
}

func (m *MemStore) StateTree(context.Context, uint64) (*patricia.Tree, error) {
	if m.State == nil {
		m.State = patricia.NewTree(nil)
	}
	return patricia.Copy(m.State), nil
}

func (m *MemStore) FinalizeBlock(context.Context, uint64) error { return nil }

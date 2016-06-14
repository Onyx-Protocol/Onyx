package memstore

import (
	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/cos/patricia"
	"chain/cos/state"
)

// MemStore satisfies the cos.Store interface.
// It is used by tests to avoid needing a database.
type MemStore struct {
	blocks   []*bc.Block
	blockTxs map[bc.Hash]*bc.Tx

	stateTree *patricia.Tree
}

// New returns a new MemStore
func New() *MemStore {
	return &MemStore{blockTxs: make(map[bc.Hash]*bc.Tx)}
}

func (m *MemStore) GetTxs(ctx context.Context, hashes ...bc.Hash) (bcTxs map[bc.Hash]*bc.Tx, err error) {
	bcTxs = make(map[bc.Hash]*bc.Tx)
	for _, hash := range hashes {
		if tx := m.blockTxs[hash]; tx != nil {
			bcTxs[hash] = m.blockTxs[hash]
		}
	}
	return bcTxs, nil
}

func (m *MemStore) ApplyBlock(
	ctx context.Context,
	b *bc.Block,
	assets map[bc.AssetID]*state.AssetState,
	stateTree *patricia.Tree,
) ([]*bc.Tx, error) {
	m.blocks = append(m.blocks, b)

	// Record all the new transactions.
	var newTxs []*bc.Tx
	for _, tx := range b.Transactions {
		newTxs = append(newTxs, tx)
		m.blockTxs[tx.Hash] = tx
	}

	m.stateTree = patricia.Copy(stateTree)
	return newTxs, nil
}

func (m *MemStore) LatestBlock(context.Context) (*bc.Block, error) {
	if len(m.blocks) == 0 {
		return nil, nil
	}
	return m.blocks[len(m.blocks)-1], nil
}

func (m *MemStore) StateTree(context.Context, uint64) (*patricia.Tree, error) {
	if m.stateTree == nil {
		m.stateTree = patricia.NewTree(nil)
	}
	return patricia.Copy(m.stateTree), nil
}

func (m *MemStore) FinalizeBlock(context.Context, uint64) error { return nil }

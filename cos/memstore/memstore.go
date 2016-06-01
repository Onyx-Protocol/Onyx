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
	blocks     []*bc.Block
	blockTxs   map[bc.Hash]*bc.Tx
	blockUTXOs map[bc.Outpoint]*state.Output

	pool      []*bc.Tx // used for keeping topological order
	poolMap   map[bc.Hash]*bc.Tx
	poolUTXOs map[bc.Outpoint]*state.Output

	stateTree *patricia.Tree
}

// New returns a new MemStore
func New() *MemStore {
	return &MemStore{
		blockTxs:   make(map[bc.Hash]*bc.Tx),
		blockUTXOs: make(map[bc.Outpoint]*state.Output),

		poolMap:   make(map[bc.Hash]*bc.Tx),
		poolUTXOs: make(map[bc.Outpoint]*state.Output),
	}
}

func (m *MemStore) GetTxs(ctx context.Context, hashes ...bc.Hash) (poolTxs, bcTxs map[bc.Hash]*bc.Tx, err error) {
	poolTxs = make(map[bc.Hash]*bc.Tx)
	bcTxs = make(map[bc.Hash]*bc.Tx)
	for _, hash := range hashes {
		if tx := m.blockTxs[hash]; tx != nil {
			bcTxs[hash] = m.blockTxs[hash]
		}
		if tx := m.poolMap[hash]; tx != nil {
			poolTxs[hash] = tx
		}
	}
	return poolTxs, bcTxs, nil
}

func (m *MemStore) ApplyTx(ctx context.Context, tx *bc.Tx, assets map[bc.AssetID]*state.AssetState) error {
	m.poolMap[tx.Hash] = tx
	m.pool = append(m.pool, tx)

	for i, out := range tx.Outputs {
		op := bc.Outpoint{Hash: tx.Hash, Index: uint32(i)}
		m.poolUTXOs[op] = state.NewOutput(*out, op)
	}
	return nil
}

func (m *MemStore) CleanPool(
	ctx context.Context,
	confirmed,
	conflicting []*bc.Tx,
	assets map[bc.AssetID]*state.AssetState,
) error {
	for _, tx := range append(confirmed, conflicting...) {
		delete(m.poolMap, tx.Hash)
		for i := range m.pool {
			if m.pool[i].Hash == tx.Hash {
				m.pool = append(m.pool[:i], m.pool[i+1:]...)
				break
			}
		}
		for i := range tx.Outputs {
			delete(m.poolUTXOs, bc.Outpoint{Hash: tx.Hash, Index: uint32(i)})
		}
	}

	return nil
}

func (m *MemStore) PoolTxs(context.Context) ([]*bc.Tx, error) {
	return m.pool[:len(m.pool):len(m.pool)], nil
}

func (m *MemStore) GetPoolPrevouts(ctx context.Context, txs []*bc.Tx) (map[bc.Outpoint]*state.Output, error) {
	prevouts := map[bc.Outpoint]*state.Output{}
	for _, tx := range txs {
		for _, in := range tx.Inputs {
			if in.IsIssuance() {
				continue
			}
			if o, ok := m.poolUTXOs[in.Previous]; ok {
				prevouts[in.Previous] = o
			}
		}
	}
	return prevouts, nil
}

func (m *MemStore) ApplyBlock(
	ctx context.Context,
	b *bc.Block,
	addedUTXOs []*state.Output,
	consumedUTXOs []*state.Output,
	assets map[bc.AssetID]*state.AssetState,
	stateTree *patricia.Tree,
) ([]*bc.Tx, error) {
	m.blocks = append(m.blocks, b)

	// Record all the new transactions.
	var newTxs []*bc.Tx
	for _, tx := range b.Transactions {
		if m.poolMap[tx.Hash] == nil {
			newTxs = append(newTxs, tx)
		}
		m.blockTxs[tx.Hash] = tx
	}

	// Add in all of the new UTXOs.
	for _, out := range addedUTXOs {
		m.blockUTXOs[out.Outpoint] = out
	}

	// Now mark all prevouts in consumed, both in the bc and the pool. The
	// order of adding UTXOs and consuming them is important for txs that
	// consume the outputs of other pool txs.
	for _, out := range consumedUTXOs {
		delete(m.poolUTXOs, out.Outpoint)
		delete(m.blockUTXOs, out.Outpoint)
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

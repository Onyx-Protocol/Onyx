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
		m.poolUTXOs[op] = &state.Output{
			Outpoint: op,
			TxOutput: *out,
		}
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

func (m *MemStore) NewPoolViewForPrevouts(context.Context, []*bc.Tx) (state.ViewReader, error) {
	return &state.MemView{
		Outs: cloneUTXOs(m.poolUTXOs),
	}, nil
}

func (m *MemStore) ApplyBlock(
	ctx context.Context,
	b *bc.Block,
	utxos []*state.Output,
	assets map[bc.AssetID]*state.AssetState,
	stateTree *patricia.Tree,
) ([]*bc.Tx, error) {
	m.blocks = append(m.blocks, b)

	var newTxs []*bc.Tx
	for _, tx := range b.Transactions {
		if m.poolMap[tx.Hash] == nil {
			newTxs = append(newTxs, tx)
		}
		m.blockTxs[tx.Hash] = tx

		for _, in := range tx.Inputs {
			if in.IsIssuance() {
				continue
			}
			out := m.poolUTXOs[in.Previous]
			if out == nil {
				out = &state.Output{Outpoint: in.Previous}
				m.blockUTXOs[in.Previous] = out
			}
			out.Spent = true
		}

		for i, out := range tx.Outputs {
			op := bc.Outpoint{Hash: tx.Hash, Index: uint32(i)}
			m.blockUTXOs[op] = &state.Output{
				Outpoint: op,
				TxOutput: *out,
			}
		}
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

func (m *MemStore) NewViewForPrevouts(context.Context, []*bc.Tx) (state.ViewReader, error) {
	return &state.MemView{
		Outs: cloneUTXOs(m.blockUTXOs),
	}, nil
}

func (m *MemStore) StateTree(context.Context, uint64) (*patricia.Tree, error) {
	if m.stateTree == nil {
		m.stateTree = patricia.NewTree(nil)
	}
	return patricia.Copy(m.stateTree), nil
}

func (m *MemStore) FinalizeBlock(context.Context, uint64) error { return nil }

func cloneUTXOs(utxos map[bc.Outpoint]*state.Output) map[bc.Outpoint]*state.Output {
	outs := make(map[bc.Outpoint]*state.Output, len(utxos))
	for outpoint, output := range utxos {
		clone := *output
		outs[outpoint] = &clone
	}
	return outs
}

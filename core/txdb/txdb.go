package txdb

import (
	"context"
	"sort"
	"strconv"

	"chain/database/pg"
	"chain/database/sql"
	"chain/errors"
	"chain/log"
	"chain/net/trace/span"
	"chain/protocol/bc"
	"chain/strings"
)

// New creates a Store and Pool backed by the txdb with the provided
// db handle.
func New(db *sql.DB) (*Store, *Pool) {
	return NewStore(db), NewPool(db)
}

// getPoolTxs looks up transactions by their hashes in the pending tx pool.
func getPoolTxs(ctx context.Context, db pg.DB, hashes ...bc.Hash) (poolTxs map[bc.Hash]*bc.Tx, err error) {
	hashStrings := make([]string, 0, len(hashes))
	for _, h := range hashes {
		hashStrings = append(hashStrings, h.String())
	}
	sort.Strings(hashStrings)
	hashStrings = strings.Uniq(hashStrings)
	const q = `
		SELECT t.tx_hash, t.data
		FROM pool_txs t
		WHERE t.tx_hash = ANY($1)
	`
	poolTxs = make(map[bc.Hash]*bc.Tx, len(hashes))
	err = pg.ForQueryRows(pg.NewContext(ctx, db), q, pg.Strings(hashStrings), func(hash bc.Hash, data bc.TxData) {
		tx := &bc.Tx{TxData: data, Hash: hash}
		poolTxs[hash] = tx
	})
	return poolTxs, errors.Wrap(err, "get txs query")
}

// dumpPoolTxs returns all of the pending transactions in the pool.
func dumpPoolTxs(ctx context.Context, db pg.DB) ([]*bc.Tx, error) {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	const q = `SELECT tx_hash, data FROM pool_txs ORDER BY sort_id`
	var txs []*bc.Tx
	err := pg.ForQueryRows(pg.NewContext(ctx, db), q, func(hash bc.Hash, data bc.TxData) {
		txs = append(txs, &bc.Tx{TxData: data, Hash: hash})
	})
	if err != nil {
		return nil, err
	}
	txs = topSort(ctx, txs)
	return txs, nil
}

func insertBlock(ctx context.Context, dbtx *sql.Tx, block *bc.Block) error {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	const q = `
		INSERT INTO blocks (block_hash, height, data, header)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (block_hash) DO NOTHING
	`
	_, err := dbtx.Exec(ctx, q, block.Hash(), block.Height, block, &block.BlockHeader)
	return errors.Wrap(err, "insert query")
}

// ListBlocks returns a list of the most recent blocks,
// potentially offset by a previous query's results.
func (s *Store) ListBlocks(ctx context.Context, prev string, limit int) ([]*bc.Block, error) {
	return listBlocks(ctx, s.db, prev, limit)
}

func listBlocks(ctx context.Context, db pg.DB, prev string, limit int) ([]*bc.Block, error) {
	const q = `
		SELECT data FROM blocks WHERE ($1='' OR height<$1::bigint)
		ORDER BY height DESC LIMIT $2
	`
	var blocks []*bc.Block
	err := pg.ForQueryRows(pg.NewContext(ctx, db), q, prev, limit, func(b bc.Block) {
		blocks = append(blocks, &b)
	})
	return blocks, err
}

func ListenBlocks(ctx context.Context, dbURL string) (<-chan uint64, error) {
	listener, err := pg.NewListener(ctx, dbURL, "newblock")
	if err != nil {
		return nil, err
	}

	c := make(chan uint64)
	go func() {
		defer func() {
			listener.Close()
			close(c)
		}()

		for {
			select {
			case <-ctx.Done():
				return

			case n := <-listener.Notify:
				height, err := strconv.ParseUint(n.Extra, 10, 64)
				if err != nil {
					log.Error(ctx, errors.Wrap(err, "parsing db notification payload"))
					return
				}
				c <- height
			}
		}
	}()

	return c, nil
}

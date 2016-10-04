package txdb

import (
	"context"
	"strconv"

	"chain/database/pg"
	"chain/errors"
	"chain/log"
	"chain/protocol/bc"
)

// New creates a Store and Pool backed by the txdb with the provided
// db handle.
func New(db pg.DB) (*Store, *Pool) {
	return NewStore(db), NewPool(db)
}

// dumpPoolTxs returns all of the pending transactions in the pool.
func dumpPoolTxs(ctx context.Context, db pg.DB) ([]*bc.Tx, error) {
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

// GetRawBlocks queries the database for blocks after the provided height,
// returning up to count blocks. The blocks are returned as raw bytes.
func GetRawBlocks(ctx context.Context, afterHeight uint64, count int) ([][]byte, error) {
	const q = `SELECT data FROM blocks WHERE height > $1 ORDER BY height LIMIT `
	var blocks [][]byte
	err := pg.ForQueryRows(ctx, q+strconv.Itoa(count), afterHeight, func(rawBlock []byte) {
		blocks = append(blocks, rawBlock)
	})
	return blocks, errors.Wrap(err, "querying blocks from the db")
}

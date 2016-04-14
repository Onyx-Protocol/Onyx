package txdb

import (
	"bytes"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/cos/state"
	"chain/database/pg"
	"chain/errors"
	"chain/metrics"
	"chain/net/trace/span"
	"chain/strings"
)

func poolTxs(ctx context.Context) ([]*bc.Tx, error) {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	const q = `SELECT tx_hash, data FROM pool_txs ORDER BY sort_id`
	var txs []*bc.Tx
	err := pg.ForQueryRows(ctx, q, func(hash bc.Hash, data bc.TxData) {
		txs = append(txs, &bc.Tx{TxData: data, Hash: hash, Stored: true})
	})
	if err != nil {
		return nil, err
	}
	txs = topSort(ctx, txs)
	return txs, nil
}

// GetTxs looks up transactions by their hashes
// in the block chain and in the pool.
func GetTxs(ctx context.Context, hashes ...bc.Hash) (map[bc.Hash]*bc.Tx, error) {
	hashStrings := make([]string, 0, len(hashes))
	for _, h := range hashes {
		hashStrings = append(hashStrings, h.String())
	}
	sort.Strings(hashStrings)
	hashStrings = strings.Uniq(hashStrings)
	const q = `SELECT tx_hash, data FROM txs WHERE tx_hash=ANY($1)`
	txs := make(map[bc.Hash]*bc.Tx, len(hashes))
	err := pg.ForQueryRows(ctx, q, pg.Strings(hashStrings), func(hash bc.Hash, data bc.TxData) {
		txs[hash] = &bc.Tx{TxData: data, Hash: hash, Stored: true}
	})
	return txs, errors.Wrap(err, "get txs query")
}

func GetTxBlockHeader(ctx context.Context, hash bc.Hash) (*bc.BlockHeader, error) {
	const q = `
		SELECT header
		FROM blocks b
		JOIN blocks_txs bt ON b.block_hash = bt.block_hash
		WHERE bt.tx_hash=$1
	`
	b := new(bc.BlockHeader)
	err := pg.QueryRow(ctx, q, hash).Scan(b)
	if err == sql.ErrNoRows {
		return nil, nil // tx "not being in a block" is not an error
	}
	return b, errors.Wrap(err, "select query")
}

// insertTx inserts tx into txs. It returns true if the insert query inserted the
// transaction. It returns false if the transaction already existed and the query
// had no effect.
func insertTx(ctx context.Context, tx *bc.Tx) (bool, error) {
	const q = `INSERT INTO txs (tx_hash, data) VALUES($1, $2) ON CONFLICT DO NOTHING`
	res, err := pg.Exec(ctx, q, tx.Hash, tx)
	if err != nil {
		return false, errors.Wrap(err, "insert query")
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, errors.Wrap(err, "insert query rows affected")
	}
	return affected > 0, nil
}

func insertBlock(ctx context.Context, block *bc.Block) ([]bc.Hash, error) {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	const q = `
		INSERT INTO blocks (block_hash, height, data, header)
		VALUES ($1, $2, $3, $4)
	`
	_, err := pg.Exec(ctx, q, block.Hash(), block.Height, block, &block.BlockHeader)
	if err != nil {
		return nil, errors.Wrap(err, "insert query")
	}

	newHashes, err := insertBlockTxs(ctx, block)
	return newHashes, errors.Wrap(err, "inserting txs")
}

func insertBlockTxs(ctx context.Context, block *bc.Block) ([]bc.Hash, error) {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	var (
		hashInBlock []string // all txs in block
		blockPos    []int32  // position of txs in block
		hashHist    []string // historical txs not already stored
		data        [][]byte // parallel with hashHist
	)
	for i, tx := range block.Transactions {
		blockPos = append(blockPos, int32(i))
		hashInBlock = append(hashInBlock, tx.Hash.String())
		if !tx.Stored {
			var buf bytes.Buffer
			_, err := tx.WriteTo(&buf)
			if err != nil {
				return nil, errors.Wrap(err, "serializing tx")
			}
			data = append(data, buf.Bytes())
			hashHist = append(hashHist, tx.Hash.String())
		}
	}

	const txQ = `
		WITH t AS (SELECT unnest($1::text[]) tx_hash, unnest($2::bytea[]) dat)
		INSERT INTO txs (tx_hash, data)
		SELECT tx_hash, dat FROM t
		WHERE t.tx_hash NOT IN (SELECT tx_hash FROM txs)
		RETURNING tx_hash;
	`
	var (
		newHashes []bc.Hash
	)
	err := pg.ForQueryRows(ctx, txQ, pg.Strings(hashHist), pg.Byteas(data), func(hash bc.Hash) {
		newHashes = append(newHashes, hash)
	})
	if err != nil {
		return nil, errors.Wrap(err, "insert txs")
	}

	const blockTxQ = `
		INSERT INTO blocks_txs (tx_hash, block_pos, block_hash, block_height)
		SELECT unnest($1::text[]), unnest($2::int[]), $3, $4;
	`
	_, err = pg.Exec(
		ctx,
		blockTxQ,
		pg.Strings(hashInBlock),
		pg.Int32s(blockPos),
		block.Hash(),
		block.Height,
	)
	if err != nil {
		return nil, errors.Wrap(err, "insert block txs")
	}
	return newHashes, nil
}

// ListBlocks returns a list of the most recent blocks,
// potentially offset by a previous query's results.
func ListBlocks(ctx context.Context, prev string, limit int) ([]*bc.Block, error) {
	const q = `
		SELECT data FROM blocks WHERE ($1='' OR height<$1::bigint)
		ORDER BY height DESC LIMIT $2
	`
	var blocks []*bc.Block
	err := pg.ForQueryRows(ctx, q, prev, limit, func(b bc.Block) {
		blocks = append(blocks, &b)
	})
	return blocks, err
}

// GetBlock fetches a block by its hash
func GetBlock(ctx context.Context, hash string) (*bc.Block, error) {
	const q = `SELECT data FROM blocks WHERE block_hash=$1`
	block := new(bc.Block)
	err := pg.QueryRow(ctx, q, hash).Scan(block)
	if err == sql.ErrNoRows {
		err = pg.ErrUserInputNotFound
	}
	return block, errors.WithDetailf(err, "block hash=%v", hash)
}

func removeBlockSpentOutputs(ctx context.Context, delta []*state.Output) error {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	var (
		txHashes []string
		ids      []uint32
	)
	for _, out := range delta {
		if !out.Spent {
			continue
		}
		txHashes = append(txHashes, out.Outpoint.Hash.String())
		ids = append(ids, out.Outpoint.Index)
	}

	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return errors.Wrap(err, "begin db transaction for deleting utxos")
	}
	defer dbtx.Rollback(ctx)

	// account_utxos are deleted by a foreign key constraint
	_, err = pg.Exec(ctx, `LOCK TABLE account_utxos IN EXCLUSIVE MODE`)
	if err != nil {
		return errors.Wrap(err, "acquire lock for deleting utxos")
	}

	const q = `
		DELETE FROM utxos
		WHERE (tx_hash, index) IN (SELECT unnest($1::text[]), unnest($2::integer[]))
	`
	_, err = pg.Exec(ctx, q, pg.Strings(txHashes), pg.Uint32s(ids))
	if err != nil {
		return errors.Wrap(err, "delete query")
	}

	return errors.Wrap(dbtx.Commit(ctx), "commit transaction for deleting utxos")
}

// insertBlockOutputs updates utxos to mark
// unconfirmed records as confirmed and to insert new
// records as necessary, one for each unspent item
// in delta.
//
// It returns a new list containing all spent items
// from delta, plus all newly-inserted unspent outputs
// from delta, omitting the updated items.
func insertBlockOutputs(ctx context.Context, delta []*state.Output) error {
	defer metrics.RecordElapsed(time.Now())
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	var outs utxoSet
	for _, out := range delta {
		if out.Spent {
			continue
		}
		addToUTXOSet(&outs, &Output{Output: *out})
	}

	// Insert the ones not upgraded above.
	const insertQ1 = `
		WITH new_utxos AS (
			SELECT
				unnest($1::text[]) AS tx_hash,
				unnest($2::bigint[]) AS index,
				unnest($3::text[]),
				unnest($4::bigint[]),
				unnest($5::bytea[]),
				unnest($6::bytea[]),
				unnest($7::bytea[])
		)
		INSERT INTO utxos (
			tx_hash, index, asset_id, amount,
			script, contract_hash, metadata
		)
		SELECT * FROM new_utxos n WHERE NOT EXISTS
			(SELECT 1 FROM utxos u WHERE (n.tx_hash, n.index) = (u.tx_hash, u.index))
	`

	_, err := pg.Exec(ctx, insertQ1,
		outs.txHash,
		outs.index,
		outs.assetID,
		outs.amount,
		outs.script,
		outs.contractHash,
		outs.metadata,
	)
	if err != nil {
		return errors.Wrap(err, "insert into utxos")
	}

	const insertQ2 = `
		INSERT INTO blocks_utxos (tx_hash, index)
		    SELECT unnest($1::text[]), unnest($2::bigint[])
	`
	_, err = pg.Exec(ctx, insertQ2, outs.txHash, outs.index)
	return errors.Wrap(err, "insert into blocks_utxos")
}

// CountBlockTxs returns the total number of confirmed transactions.
// TODO: Instead running a count query, we should increment a value each time a
// new block lands.
func CountBlockTxs(ctx context.Context) (uint64, error) {
	const q = `SELECT count(tx_hash) FROM blocks_txs`
	var res uint64
	err := pg.QueryRow(ctx, q).Scan(&res)
	return res, errors.Wrap(err)
}

func ListUTXOsByAsset(ctx context.Context, assetID bc.AssetID, prev string, limit int) ([]*state.Output, string, error) {
	const q = `
		SELECT blocks_txs.block_height,
			blocks_txs.block_pos,
			utxos.tx_hash,
			utxos.index,
			utxos.asset_id,
			utxos.amount,
			utxos.metadata,
			utxos.script
		FROM utxos
		JOIN blocks_txs ON utxos.tx_hash = blocks_txs.tx_hash
		WHERE utxos.asset_id = $1
		AND blocks_txs.block_height >= $2
		-- Block pos only matters if we're in the same block
		AND (blocks_txs.block_height != $2 OR blocks_txs.block_pos >= $3)
		-- Output index only matters if we're in the same block and the same transaction
		AND (blocks_txs.block_height != $2 OR blocks_txs.block_pos != $3 OR utxos.index > $4)
		ORDER BY blocks_txs.block_height, blocks_txs.block_pos, utxos.index
		LIMIT $5
	`

	// Since the sort criteria is composite, the cursor is composite.
	var (
		prevBlock    int64
		prevBlockPos int
		prevOutIndex int
	)
	_, err := fmt.Sscanf(prev, "%d-%d-%d", &prevBlock, &prevBlockPos, &prevOutIndex)
	if err != nil {
		// tolerate malformed cursors
		prevBlock = 0
		prevBlockPos = -1
		prevOutIndex = -1
	}

	var (
		res  []*state.Output
		last string
	)
	err = pg.ForQueryRows(ctx, q, assetID, prevBlock, prevBlockPos, prevOutIndex, limit, func(bh int64, bpos int, hash bc.Hash, index uint32, assetID bc.AssetID, amount uint64, metadata, script []byte) {
		o := &state.Output{
			Outpoint: bc.Outpoint{Hash: hash, Index: index},
			TxOutput: bc.TxOutput{
				AssetAmount: bc.AssetAmount{AssetID: assetID, Amount: amount},
				Script:      script,
				Metadata:    metadata,
			},
		}
		res = append(res, o)
		last = fmt.Sprintf("%d-%d-%d", bh, bpos, index)
	})
	if err != nil {
		return nil, "", err
	}

	return res, last, nil
}

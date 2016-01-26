package txdb

import (
	"bytes"
	"database/sql"
	"sort"
	"time"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
	"chain/fedchain/bc"
	"chain/fedchain/txscript"
	"chain/metrics"
	"chain/net/trace/span"
	"chain/strings"
)

// PoolTxs returns the pooled transactions
// in topological order.
// If max is negative, there is no limit.
// TODO(jeffomatic) - at some point in the future, will we want to keep this
// cached in an in-memory pool, a la btcd's TxMemPool?
func PoolTxs(ctx context.Context) ([]*bc.Tx, error) {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	const q = `SELECT tx_hash, data FROM pool_txs ORDER BY sort_id`
	rows, err := pg.FromContext(ctx).Query(ctx, q)
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}
	defer rows.Close()

	var txs []*bc.Tx
	for rows.Next() {
		var hash bc.Hash
		var data bc.TxData
		err := rows.Scan(&hash, &data)
		if err != nil {
			return nil, errors.Wrap(err, "row scan")
		}
		txs = append(txs, &bc.Tx{TxData: data, Hash: hash, Stored: true})
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "end row scan loop")
	}

	txs = topSort(ctx, txs)
	return txs, nil
}

// GetTxs looks up transactions by their hashes
// in the block chain and in the pool.
func GetTxs(ctx context.Context, hashes ...string) (map[string]*bc.Tx, error) {
	sort.Strings(hashes)
	hashes = strings.Uniq(hashes)
	const q = `SELECT tx_hash, data FROM txs WHERE tx_hash=ANY($1)`
	rows, err := pg.FromContext(ctx).Query(ctx, q, pg.Strings(hashes))
	if err != nil {
		return nil, errors.Wrap(err, "get txs query")
	}
	defer rows.Close()

	txs := make(map[string]*bc.Tx, len(hashes))
	for rows.Next() {
		var hash bc.Hash
		var data bc.TxData
		err = rows.Scan(&hash, &data)
		if err != nil {
			return nil, errors.Wrap(err, "rows scan")
		}
		txs[hash.String()] = &bc.Tx{TxData: data, Hash: hash, Stored: true}
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "rows end")
	}
	if len(txs) < len(hashes) {
		return nil, errors.Wrap(pg.ErrUserInputNotFound, "missing tx")
	}
	return txs, nil
}

func GetTxBlockHeader(ctx context.Context, hash string) (*bc.BlockHeader, error) {
	const q = `
		SELECT header
		FROM blocks b
		JOIN blocks_txs bt ON b.block_hash = bt.block_hash
		WHERE bt.tx_hash=$1
	`
	b := new(bc.BlockHeader)
	err := pg.FromContext(ctx).QueryRow(ctx, q, hash).Scan(b)
	if err == sql.ErrNoRows {
		return nil, nil // tx "not being in a block" is not an error
	}
	return b, errors.Wrap(err, "select query")
}

// InsertTx inserts tx into txs.
func InsertTx(ctx context.Context, tx *bc.Tx) error {
	const q = `INSERT INTO txs (tx_hash, data) VALUES($1, $2)`
	_, err := pg.FromContext(ctx).Exec(ctx, q, tx.Hash, tx)
	return errors.Wrap(err, "insert query")
}

// LatestBlock returns the most recent block.
func LatestBlock(ctx context.Context) (*bc.Block, error) {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)
	const q = `SELECT data FROM blocks ORDER BY height DESC LIMIT 1`
	b := new(bc.Block)
	err := pg.FromContext(ctx).QueryRow(ctx, q).Scan(b)
	if err == sql.ErrNoRows {
		return nil, errors.Wrap(err, "blocks table is empty; please seed with genesis block")
	}
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}
	return b, nil
}

func InsertBlock(ctx context.Context, block *bc.Block) error {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	const q = `
		INSERT INTO blocks (block_hash, height, data, header)
		VALUES ($1, $2, $3, $4)
	`
	_, err := pg.FromContext(ctx).Exec(ctx, q, block.Hash(), block.Height, block, &block.BlockHeader)
	if err != nil {
		return errors.Wrap(err, "insert query")
	}

	err = insertBlockTxs(ctx, block)
	return errors.Wrap(err, "inserting txs")
}

func insertBlockTxs(ctx context.Context, block *bc.Block) error {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	var (
		hashInBlock []string // all txs in block
		hashHist    []string // historical txs not already stored
		data        [][]byte // parallel with hashHist
	)
	for _, tx := range block.Transactions {
		hashInBlock = append(hashInBlock, tx.Hash.String())
		if !tx.Stored {
			var buf bytes.Buffer
			_, err := tx.WriteTo(&buf)
			if err != nil {
				return errors.Wrap(err, "serializing tx")
			}
			data = append(data, buf.Bytes())
			hashHist = append(hashHist, tx.Hash.String())
		}
	}

	const txQ = `
		WITH t AS (SELECT unnest($1::text[]) tx_hash, unnest($2::bytea[]) dat)
		INSERT INTO txs (tx_hash, data)
		SELECT tx_hash, dat FROM t
		WHERE t.tx_hash NOT IN (SELECT tx_hash FROM txs);
	`
	_, err := pg.FromContext(ctx).Exec(ctx, txQ, pg.Strings(hashHist), pg.Byteas(data))
	if err != nil {
		return errors.Wrap(err, "insert txs")
	}

	const blockTxQ = `
		INSERT INTO blocks_txs (tx_hash, block_hash)
		SELECT unnest($1::text[]), $2;
	`
	_, err = pg.FromContext(ctx).Exec(ctx, blockTxQ, pg.Strings(hashInBlock), block.Hash())
	return errors.Wrap(err, "insert block txs")
}

// ListBlocks returns a list of the most recent blocks,
// potentially offset by a previous query's results.
func ListBlocks(ctx context.Context, prev string, limit int) ([]*bc.Block, error) {
	const q = `
		SELECT data FROM blocks WHERE ($1='' OR height<$1::bigint)
		ORDER BY height DESC LIMIT $2
	`
	rows, err := pg.FromContext(ctx).Query(ctx, q, prev, limit)
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}
	defer rows.Close()
	var blocks []*bc.Block
	for rows.Next() {
		var block bc.Block
		err := rows.Scan(&block)
		if err != nil {
			return nil, errors.Wrap(err, "row scan")
		}
		blocks = append(blocks, &block)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "rows loop")
	}
	return blocks, nil
}

// GetBlock fetches a block by its hash
func GetBlock(ctx context.Context, hash string) (*bc.Block, error) {
	const q = `SELECT data FROM blocks WHERE block_hash=$1`
	block := new(bc.Block)
	err := pg.FromContext(ctx).QueryRow(ctx, q, hash).Scan(block)
	if err == sql.ErrNoRows {
		err = pg.ErrUserInputNotFound
	}
	return block, errors.WithDetailf(err, "block hash=%v", hash)
}

func RemoveBlockSpentOutputs(ctx context.Context, delta []*Output) error {
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

	const q = `
		DELETE FROM utxos
		WHERE (tx_hash, index) IN (SELECT unnest($1::text[]), unnest($2::integer[]))
	`
	_, err := pg.FromContext(ctx).Exec(ctx, q, pg.Strings(txHashes), pg.Uint32s(ids))
	if err != nil {
		return errors.Wrap(err, "delete query")
	}

	return nil
}

// InsertBlockOutputs updates utxos to mark
// unconfirmed records as confirmed and to insert new
// records as necessary, one for each unspent item
// in delta.
//
// It returns a new list containing all spent items
// from delta, plus all newly-inserted unspent outputs
// from delta, omitting the updated items.
func InsertBlockOutputs(ctx context.Context, block *bc.Block, delta []*Output) ([]*Output, error) {
	defer metrics.RecordElapsed(time.Now())
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	txHashes := make([]string, 0, len(delta))
	indexes := make([]uint32, 0, len(delta))
	for _, out := range delta {
		txHashes = append(txHashes, out.Outpoint.Hash.String())
		indexes = append(indexes, out.Outpoint.Index)
	}

	blockHashStr := block.Hash().String()

	// Some of the utxos may already be in the utxos table as
	// unconfirmed pool utxos.  Upgrade them to confirmed.
	const updateQ = `
		UPDATE utxos SET block_hash = $1, block_height = $2, confirmed = TRUE, pool_tx_hash = NULL
			WHERE NOT confirmed
			      AND (tx_hash, index) IN (SELECT unnest($3::text[]), unnest($4::integer[]))
			RETURNING tx_hash, index
	`
	rows, err := pg.FromContext(ctx).Query(ctx, updateQ, blockHashStr, block.Height, pg.Strings(txHashes), pg.Uint32s(indexes))
	if err != nil {
		return nil, errors.Wrap(err, "update utxos")
	}
	defer rows.Close()

	updated := make(map[bc.Outpoint]bool)
	for rows.Next() {
		var outpoint bc.Outpoint
		err = rows.Scan(&outpoint.Hash, &outpoint.Index)
		if err != nil {
			return nil, errors.Wrap(err, "scanning update utxos result")
		}
		updated[outpoint] = true
	}

	var (
		outs     utxoSet
		newDelta = make([]*Output, 0, len(delta)-len(updated))
	)
	for _, out := range delta {
		if updated[out.Outpoint] {
			// already updated above
			continue
		}
		newDelta = append(newDelta, out)
		if out.Spent {
			continue
		}
		outs.txHash = append(outs.txHash, out.Outpoint.Hash.String())
		outs.index = append(outs.index, out.Outpoint.Index)
		outs.assetID = append(outs.assetID, out.AssetID.String())
		outs.amount = append(outs.amount, int64(out.Amount))
		outs.accountID = append(outs.accountID, out.AccountID)
		outs.managerNodeID = append(outs.managerNodeID, out.ManagerNodeID)
		outs.aIndex = append(outs.aIndex, toKeyIndex(out.AddrIndex[:]))
		outs.script = append(outs.script, out.Script)
		outs.metadata = append(outs.metadata, out.Metadata)

		isPayToContract, contractHash := txscript.TestPayToContract(out.Script)
		if isPayToContract {
			outs.contractHash = append(outs.contractHash, contractHash[:])
		} else {
			outs.contractHash = append(outs.contractHash, nil)
		}
	}

	// Insert the ones not upgraded above.
	const insertQ = `
		INSERT INTO utxos (
			tx_hash, index, asset_id, amount,
			account_id, manager_node_id, addr_index,
			script, contract_hash, metadata,
			block_hash, block_height, confirmed
		)
		SELECT
			unnest($1::text[]),
			unnest($2::bigint[]),
			unnest($3::text[]),
			unnest($4::bigint[]),
			unnest($5::text[]),
			unnest($6::text[]),
			unnest($7::bigint[]),
			unnest($8::bytea[]),
			unnest($9::bytea[]),
			unnest($10::bytea[]),
			$11,
			$12,
			TRUE
	`
	_, err = pg.FromContext(ctx).Exec(ctx, insertQ,
		outs.txHash,
		outs.index,
		outs.assetID,
		outs.amount,
		outs.accountID,
		outs.managerNodeID,
		outs.aIndex,
		outs.script,
		outs.contractHash,
		outs.metadata,
		blockHashStr,
		block.Height,
	)
	return newDelta, errors.Wrap(err, "insert utxos")
}

// CountBlockTxs returns the total number of confirmed transactions.
// TODO: Instead running a count query, we should increment a value each time a
// new block lands.
func CountBlockTxs(ctx context.Context) (uint64, error) {
	const q = `SELECT count(tx_hash) FROM blocks_txs`
	var res uint64
	err := pg.FromContext(ctx).QueryRow(ctx, q).Scan(&res)
	return res, errors.Wrap(err)
}

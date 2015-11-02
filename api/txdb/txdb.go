package txdb

import (
	"bytes"
	"database/sql"
	"sort"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
	"chain/fedchain/bc"
	"chain/fedchain/txscript"
	"chain/net/trace/span"
	"chain/strings"
)

// PoolTxs loads all pooled transactions from the database in order received.
// TODO(jeffomatic) - at some point in the future, will we want to keep this
// cached in an in-memory pool, a la btcd's TxMemPool?
func PoolTxs(ctx context.Context) ([]*bc.Tx, error) {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	const q = `SELECT data FROM pool_txs ORDER BY sort_id`
	rows, err := pg.FromContext(ctx).Query(q)
	if err != nil {
		return nil, errors.Wrap(err, "select query")
	}
	defer rows.Close()

	var txs []*bc.Tx
	for rows.Next() {
		tx := new(bc.Tx)
		err := rows.Scan(tx)
		if err != nil {
			return nil, errors.Wrap(err, "row scan")
		}
		txs = append(txs, tx)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "end row scan loop")
	}

	return txs, nil
}

// GetTxs looks up transactions by their hashes
// in the block chain and in the pool.
func GetTxs(ctx context.Context, hashes ...string) (map[string]*bc.Tx, error) {
	sort.Strings(hashes)
	hashes = strings.Uniq(hashes)
	const q = `SELECT data FROM txs WHERE tx_hash=ANY($1)`
	rows, err := pg.FromContext(ctx).Query(q, pg.Strings(hashes))
	if err != nil {
		return nil, errors.Wrap(err, "get txs query")
	}
	defer rows.Close()

	txs := make(map[string]*bc.Tx, len(hashes))
	for rows.Next() {
		tx := new(bc.Tx)
		err = rows.Scan(&tx)
		if err != nil {
			return nil, errors.Wrap(err, "rows scan")
		}
		txs[tx.Hash().String()] = tx
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "rows end")
	}
	if len(txs) < len(hashes) {
		return nil, errors.Wrap(pg.ErrUserInputNotFound, "missing tx")
	}
	return txs, nil
}

func GetTxBlock(ctx context.Context, hash string) (*bc.Block, error) {
	const q = `
		SELECT data
		FROM blocks b
		JOIN blocks_txs bt ON b.block_hash = bt.block_hash
		WHERE bt.tx_hash=$1
	`
	b := new(bc.Block)
	err := pg.FromContext(ctx).QueryRow(q, hash).Scan(b)
	if err == sql.ErrNoRows {
		return nil, nil // tx "not being in a block" is not an error
	}
	return b, errors.Wrap(err, "select query")
}

func InsertTx(ctx context.Context, tx *bc.Tx) error {
	const q = `INSERT INTO txs (tx_hash, data) VALUES($1, $2)`
	_, err := pg.FromContext(ctx).Exec(q, tx.Hash(), tx)
	return errors.Wrap(err, "insert query")
}

// LatestBlock returns the most recent block.
func LatestBlock(ctx context.Context) (*bc.Block, error) {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)
	const q = `SELECT data FROM blocks ORDER BY height DESC LIMIT 1`
	b := new(bc.Block)
	err := pg.FromContext(ctx).QueryRow(q).Scan(b)
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
		INSERT INTO blocks (block_hash, height, data)
		VALUES ($1, $2, $3)
	`
	_, err := pg.FromContext(ctx).Exec(q, block.Hash().String(), block.Height, block)
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
		hashes []string
		data   [][]byte
	)
	for _, tx := range block.Transactions {
		hashes = append(hashes, tx.Hash().String())
		var buf bytes.Buffer
		_, err := tx.WriteTo(&buf)
		if err != nil {
			return errors.Wrap(err, "serializing tx")
		}
		data = append(data, buf.Bytes())
	}

	const txQ = `
		WITH t AS (SELECT unnest($1::text[]) txid, unnest($2::bytea[]) dat)
		INSERT INTO txs (tx_hash, data)
		SELECT txid, dat FROM t
		WHERE t.txid NOT IN (SELECT tx_hash FROM txs);
	`
	_, err := pg.FromContext(ctx).Exec(txQ, pg.Strings(hashes), pg.Byteas(data))
	if err != nil {
		return errors.Wrap(err, "insert txs")
	}

	const blockTxQ = `
		INSERT INTO blocks_txs (tx_hash, block_hash)
		SELECT unnest($1::text[]), $2;
	`
	_, err = pg.FromContext(ctx).Exec(blockTxQ, pg.Strings(hashes), block.Hash())
	return errors.Wrap(err, "insert block txs")
}

// ListBlocks returns a list of the most recent blocks,
// potentially offset by a previous query's results.
func ListBlocks(ctx context.Context, prev string, limit int) ([]*bc.Block, error) {
	const q = `
		SELECT data FROM blocks WHERE ($1='' OR height<$1::bigint)
		ORDER BY height DESC LIMIT $2
	`
	rows, err := pg.FromContext(ctx).Query(q, prev, limit)
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
	err := pg.FromContext(ctx).QueryRow(q, hash).Scan(block)
	if err == sql.ErrNoRows {
		err = pg.ErrUserInputNotFound
	}
	return block, errors.WithDetailf(err, "block hash=%v", hash)
}

func RemoveBlockSpentOutputs(ctx context.Context, delta []*Output) error {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	var (
		txids []string
		ids   []uint32
	)
	for _, out := range delta {
		if !out.Spent {
			continue
		}
		txids = append(txids, out.Outpoint.Hash.String())
		ids = append(ids, out.Outpoint.Index)
	}

	const q = `
		DELETE FROM utxos
		WHERE (txid, index) IN (SELECT unnest($1::text[]), unnest($2::integer[]))
	`
	_, err := pg.FromContext(ctx).Exec(q, pg.Strings(txids), pg.Uint32s(ids))
	if err != nil {
		return errors.Wrap(err, "delete query")
	}

	return nil
}

func InsertBlockOutputs(ctx context.Context, block *bc.Block, delta []*Output) error {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	var outs utxoSet
	for _, out := range delta {
		if out.Spent {
			continue
		}
		outs.txid = append(outs.txid, out.Outpoint.Hash.String())
		outs.index = append(outs.index, out.Outpoint.Index)
		outs.assetID = append(outs.assetID, out.AssetID.String())
		outs.amount = append(outs.amount, int64(out.Value))
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

	const q = `
		INSERT INTO utxos (
			txid, index, asset_id, amount,
			account_id, manager_node_id, addr_index,
			script, contract_hash, metadata,
			block_hash, block_height
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
			$12
	`
	_, err := pg.FromContext(ctx).Exec(q,
		outs.txid,
		outs.index,
		outs.assetID,
		outs.amount,
		outs.accountID,
		outs.managerNodeID,
		outs.aIndex,
		outs.script,
		outs.contractHash,
		outs.metadata,
		block.Hash().String(),
		block.Height,
	)
	return errors.Wrap(err)
}

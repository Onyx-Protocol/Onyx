package asset

import (
	"runtime"
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/txdb"
	"chain/api/utxodb"
	"chain/database/pg"
	"chain/errors"
	"chain/fedchain-sandbox/txscript"
	"chain/fedchain/bc"
	"chain/fedchain/state"
	"chain/fedchain/validation"
	"chain/log"
)

// ErrBadBlock is returned when a block is invalid.
var ErrBadBlock = errors.New("invalid block")

// MakeBlocks runs forever,
// attempting to make one block per period.
// The caller should call it exactly once.
func MakeBlocks(ctx context.Context, period time.Duration) {
	for range time.Tick(period) {
		makeBlock(ctx)
	}
}

func makeBlock(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			log.Write(ctx,
				log.KeyMessage, "panic",
				log.KeyError, err,
				log.KeyStack, buf,
			)
		}
	}()
	log.Messagef(ctx, "making block")
	b, err := GenerateBlock(ctx, time.Now())
	if err != nil {
		log.Error(ctx, errors.Wrap(err, "generate"))
		return
	}
	err = ApplyBlock(ctx, b)
	if err != nil {
		log.Error(ctx, errors.Wrap(err, "apply"))
	}
}

// GenerateBlock creates a new bc.Block using the current tx pool and blockchain
// state.
// TODO - receive parameters for script config.
func GenerateBlock(ctx context.Context, now time.Time) (*bc.Block, error) {
	ts := uint64(now.Unix())

	prevBlock, err := txdb.LatestBlock(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "fetch latest block")
	}

	if ts < prevBlock.Timestamp {
		return nil, errors.New("timestamp is earlier than prevblock timestamp")
	}

	txs, err := txdb.PoolTxs(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get pool TXs")
	}

	block := &bc.Block{
		BlockHeader: bc.BlockHeader{
			Version:           bc.NewBlockVersion,
			Height:            prevBlock.Height + 1,
			PreviousBlockHash: prevBlock.Hash(),

			// TODO: Calculate merkle hashes of txs and blockchain state.
			//TxRoot:
			//StateRoot:

			// It's possible to generate a block whose timestamp is prior to the
			// previous block, but we won't validate that here.
			Timestamp: ts,

			// TODO: Generate sigscript/outscript.
			//SignatureScript:
			//OutputScript:
		},
	}

	poolView := NewMemView()
	bcView, err := txdb.NewViewForPrevouts(ctx, txs)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	for _, tx := range txs {
		vview := NewMemView()
		view := state.Compose(vview, poolView, bcView)
		if validation.ValidateTx(ctx, view, tx, ts, &block.PreviousBlockHash) == nil {
			block.Transactions = append(block.Transactions, tx)
			for op, out := range vview.Outs {
				poolView.Outs[op] = out
			}
		}
	}
	log.Messagef(ctx, "generated block with %d txs", len(block.Transactions))
	return block, nil
}

func ApplyBlock(ctx context.Context, block *bc.Block) error {
	delta, err := applyBlock(ctx, block)
	if err != nil {
		return errors.Wrap(err)
	}

	// update reserver
	applyToReserver(delta)

	conflictTxs, err := rebuildPool(ctx, block)
	if err != nil {
		return errors.Wrap(err)
	}

	conflictOuts, err := getRestoreableOutputs(ctx, conflictTxs)
	if err != nil {
		return errors.Wrap(err)
	}

	// update reserver
	applyToReserver(conflictOuts)
	return nil
}

func applyBlock(ctx context.Context, block *bc.Block) ([]*txdb.Output, error) {
	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "blockchain update dbtx begin")
	}
	defer dbtx.Rollback()

	delta, adps, err := validateBlock(ctx, block)
	if err != nil {
		return nil, errors.Wrap(err, "block validation")
	}

	err = txdb.InsertBlock(ctx, block)
	if err != nil {
		return nil, errors.Wrap(err, "insert block")
	}

	err = txdb.InsertAssetDefinitionPointers(ctx, adps)
	if err != nil {
		return nil, errors.Wrap(err, "insert ADPs")
	}

	err = txdb.InsertAssetDefinitions(ctx, block)
	if err != nil {
		return nil, errors.Wrap(err, "writing asset definitions")
	}

	err = loadAccountInfo(ctx, delta)
	if err != nil {
		return nil, errors.Wrap(err, "block outputs")
	}

	err = txdb.RemoveBlockSpentOutputs(ctx, delta)
	if err != nil {
		return nil, errors.Wrap(err, "remove block spent outputs")
	}

	err = txdb.InsertBlockOutputs(ctx, block, delta)
	if err != nil {
		return nil, errors.Wrap(err, "insert block outputs")
	}

	err = appdb.UpdateIssuances(ctx, issuedAssets(block.Transactions), true)
	if err != nil {
		return nil, errors.Wrap(err, "update issuances")
	}

	err = dbtx.Commit()
	if err != nil {
		return nil, errors.Wrap(err, "blockchain update dbtx commit")
	}
	return delta, nil
}

func rebuildPool(ctx context.Context, block *bc.Block) ([]*bc.Tx, error) {
	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "pool update dbtx begin")
	}
	defer dbtx.Rollback()

	txInBlock := make(map[bc.Hash]bool)
	for _, tx := range block.Transactions {
		txInBlock[tx.Hash()] = true
	}

	var (
		conflictTxs          []*bc.Tx
		deleteTxs            []*bc.Tx
		deleteTxHashes       []string
		deleteInputTxHashes  []string
		deleteInputTxIndexes []uint32
	)

	poolView := NewMemView()

	txs, err := txdb.PoolTxs(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	blockHash := block.Hash()
	bcView, err := txdb.NewViewForPrevouts(ctx, txs)
	if err != nil {
		return nil, errors.Wrap(err, "blockchain view")
	}
	for _, tx := range txs {
		vview := NewMemView()
		view := state.Compose(vview, poolView, bcView)
		if validation.ValidateTx(ctx, view, tx, uint64(time.Now().Unix()), &blockHash) == nil {
			for op, out := range vview.Outs {
				poolView.Outs[op] = out
			}
		} else {
			deleteTxs = append(deleteTxs, tx)
			deleteTxHashes = append(deleteTxHashes, tx.Hash().String())
			for _, in := range tx.Inputs {
				if in.IsIssuance() {
					continue
				}
				deleteInputTxHashes = append(deleteInputTxHashes, in.Previous.Hash.String())
				deleteInputTxIndexes = append(deleteInputTxIndexes, in.Previous.Index)
			}

			if !txInBlock[tx.Hash()] {
				conflictTxs = append(conflictTxs, tx)
			}
		}
	}

	// Delete pool_txs
	const txq = `DELETE FROM pool_txs WHERE tx_hash IN (SELECT unnest($1::text[]))`
	_, err = pg.FromContext(ctx).Exec(txq, pg.Strings(deleteTxHashes))
	if err != nil {
		return nil, errors.Wrap(err, "delete from pool_txs")
	}

	// Delete pool_outputs
	const outq = `DELETE FROM pool_outputs WHERE tx_hash IN (SELECT unnest($1::text[]))`
	_, err = pg.FromContext(ctx).Exec(outq, pg.Strings(deleteTxHashes))
	if err != nil {
		return nil, errors.Wrap(err, "delete from pool_outputs")
	}

	// Delete pool_inputs
	const inq = `
		DELETE FROM pool_inputs
		WHERE (tx_hash, index) IN (
			SELECT unnest($1::text[]), unnest($2::integer[])
		)
	`
	_, err = pg.FromContext(ctx).Exec(inq, pg.Strings(deleteInputTxHashes), pg.Uint32s(deleteInputTxIndexes))
	if err != nil {
		return nil, errors.Wrap(err, "delete from pool_inputs")
	}

	// Update issuance totals
	deltas := issuedAssets(deleteTxs)
	for aid, v := range deltas {
		deltas[aid] = -v // reverse polarity, we want decrements
	}
	err = appdb.UpdateIssuances(ctx, deltas, false)
	if err != nil {
		return nil, errors.Wrap(err, "undo pool issuances")
	}

	err = dbtx.Commit()
	if err != nil {
		return nil, errors.Wrap(err, "pool update dbtx commit")
	}
	return conflictTxs, nil
}

func getRestoreableOutputs(ctx context.Context, txs []*bc.Tx) (outs []*txdb.Output, err error) {
	poolView, err := txdb.NewPoolViewForPrevouts(ctx, txs)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	bcView, err := txdb.NewViewForPrevouts(ctx, txs)
	if err != nil {
		return nil, errors.Wrap(err, "load prev outs from conflicting txs")
	}

	// undo conflicting txs in reserver
	view := state.MultiReader(poolView, bcView)
	for _, tx := range txs {
		for _, in := range tx.Inputs {
			if in.IsIssuance() {
				continue
			}
			o := view.Output(ctx, in.Previous)
			if o == nil || o.Spent {
				continue
			}
			outs = append(outs, &txdb.Output{Output: *o})
		}

		for i, out := range tx.Outputs {
			op := bc.Outpoint{Hash: tx.Hash(), Index: uint32(i)}
			outs = append(outs, &txdb.Output{
				Output: state.Output{
					TxOutput: *out,
					Outpoint: op,
					Spent:    true,
				},
			})
		}
	}

	err = loadAccountInfo(ctx, outs)
	if err != nil {
		return nil, errors.Wrap(err, "load conflict out account info")
	}

	return outs, nil
}

func applyToReserver(outs []*txdb.Output) {
	var del, ins []*utxodb.UTXO
	for _, out := range outs {
		u := &utxodb.UTXO{
			AccountID: out.AccountID,
			AssetID:   out.AssetID.String(),
			Amount:    out.Value,
			Outpoint:  out.Outpoint,
			AddrIndex: out.AddrIndex,
		}
		if out.Spent {
			del = append(del, u)
		} else {
			ins = append(ins, u)
		}
	}
	utxoDB.Apply(del, ins)
}

// loadAccountInfo returns annotated UTXO data (outputs + account mappings) for
// addresses known to this manager node. It is only concerned with outputs that
// actually have account mappings, which come from either the pool_outputs or
// addresses tables.
func loadAccountInfo(ctx context.Context, outs []*txdb.Output) error {
	var (
		hashes          []string
		indexes         []uint32
		addrs           []string
		outpointsByAddr = make(map[string]bc.Outpoint)
		outputs         = make(map[bc.Outpoint]*txdb.Output)
	)
	for _, out := range outs {
		outputs[out.Outpoint] = out
		hashes = append(hashes, out.Outpoint.Hash.String())
		indexes = append(indexes, out.Outpoint.Index)

		addr, err := txscript.PkScriptAddr(out.Script)
		if err != nil {
			return errors.Wrapf(err, "output %s: bad script: %x", out.Outpoint, out.Script)
		}
		s := addr.String()
		addrs = append(addrs, s)
		outpointsByAddr[s] = out.Outpoint
	}

	// addresses table

	const addrq = `
		SELECT address, manager_node_id, account_id, key_index(key_index)
		FROM addresses
		WHERE address IN (SELECT unnest($1::text[]))
	`
	rows, err := pg.FromContext(ctx).Query(addrq, pg.Strings(addrs))
	if err != nil {
		return errors.Wrap(err, "addresses select query")
	}
	defer rows.Close()

	for rows.Next() {
		var (
			addr, mnodeID, accID string
			addrIndex            []uint32
		)
		err := rows.Scan(&addr, &mnodeID, &accID, (*pg.Uint32s)(&addrIndex))
		if err != nil {
			return errors.Wrap(err, "addresses row scan")
		}
		out := outputs[outpointsByAddr[addr]]
		out.ManagerNodeID = mnodeID
		out.AccountID = accID
		copy(out.AddrIndex[:], addrIndex)
	}
	if err := rows.Err(); err != nil {
		return errors.Wrap(err, "addresses end row scan loop")
	}

	// pool_outputs table

	const poolq = `
		SELECT tx_hash, index, manager_node_id, account_id, key_index(addr_index)
		FROM pool_outputs
		WHERE (tx_hash, index) IN (SELECT unnest($1::text[]), unnest($2::integer[]))
	`
	rows, err = pg.FromContext(ctx).Query(poolq, pg.Strings(hashes), pg.Uint32s(indexes))
	if err != nil {
		return errors.Wrap(err, "pool_outputs select query")
	}
	defer rows.Close()

	for rows.Next() {
		var (
			op             bc.Outpoint
			mnodeID, accID string
			addrIndex      []uint32
		)
		err := rows.Scan(&op.Hash, &op.Index, &mnodeID, &accID, (*pg.Uint32s)(&addrIndex))
		if err != nil {
			return errors.Wrap(err, "pool_outputs row scan")
		}
		out := outputs[op]
		out.ManagerNodeID = mnodeID
		out.AccountID = accID
		copy(out.AddrIndex[:], addrIndex)
	}
	if err := rows.Err(); err != nil {
		return errors.Wrap(err, "pool_outputs end row scan loop")
	}

	// utxos table

	const utxoq = `
		SELECT txid, index, manager_node_id, account_id, key_index(addr_index)
		FROM utxos
		WHERE (txid, index) IN (SELECT unnest($1::text[]), unnest($2::integer[]))
	`
	rows, err = pg.FromContext(ctx).Query(utxoq, pg.Strings(hashes), pg.Uint32s(indexes))
	if err != nil {
		return errors.Wrap(err, "utxos select query")
	}
	defer rows.Close()

	for rows.Next() {
		var (
			op             bc.Outpoint
			mnodeID, accID string
			addrIndex      []uint32
		)
		err := rows.Scan(&op.Hash, &op.Index, &mnodeID, &accID, (*pg.Uint32s)(&addrIndex))
		if err != nil {
			return errors.Wrap(err, "utxos row scan")
		}
		out := outputs[op]
		out.ManagerNodeID = mnodeID
		out.AccountID = accID
		copy(out.AddrIndex[:], addrIndex)
	}
	if err := rows.Err(); err != nil {
		return errors.Wrap(err, "utxos end row scan loop")
	}

	return nil
}

// validateBlock performs validation on an incoming block, in advance of
// applying the block to the txdb.
func validateBlock(ctx context.Context, block *bc.Block) (outs []*txdb.Output, adps map[bc.AssetID]*bc.AssetDefinitionPointer, err error) {
	bcView, err := txdb.NewViewForPrevouts(ctx, block.Transactions)
	if err != nil {
		return nil, nil, errors.Wrap(err, "txdb")
	}
	mv := NewMemView()
	err = validation.ValidateBlock(ctx, state.Compose(mv, bcView), block)
	if err != nil {
		return nil, nil, errors.Wrapf(ErrBadBlock, "validate block: %v", err)
	}

	for _, out := range mv.Outs {
		outs = append(outs, out)
	}

	return outs, mv.ADPs, nil
}

func issuedAssets(txs []*bc.Tx) map[string]int64 {
	issued := make(map[string]int64)
	for _, tx := range txs {
		if !tx.IsIssuance() {
			continue
		}
		for _, out := range tx.Outputs {
			issued[out.AssetID.String()] += int64(out.Value)
		}
	}
	return issued
}

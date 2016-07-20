package asset

import (
	"time"

	"golang.org/x/net/context"

	"chain/core/rpcclient"
	"chain/core/txbuilder"
	"chain/core/txdb"
	"chain/cos"
	"chain/cos/bc"
	"chain/cos/state"
	"chain/cos/validation"
	"chain/database/pg"
	"chain/errors"
	chainlog "chain/log"
	"chain/metrics"
)

var (
	// ErrBadTxTemplate is returned by FinalizeTx
	ErrBadTxTemplate = errors.New("bad transaction template")

	// ErrRejected means the network rejected a tx (as a double-spend)
	ErrRejected = errors.New("transaction rejected")
)

var Generator *string

// FinalizeTx validates a transaction signature template,
// assembles a fully signed tx, and stores the effects of
// its changes on the UTXO set.
func FinalizeTx(ctx context.Context, txTemplate *txbuilder.Template) (*bc.Tx, error) {
	defer metrics.RecordElapsed(time.Now())

	if txTemplate.Unsigned == nil {
		return nil, errors.WithDetail(ErrBadTxTemplate, "missing unsigned tx")
	}

	if len(txTemplate.Inputs) > len(txTemplate.Unsigned.Inputs) {
		return nil, errors.WithDetail(ErrBadTxTemplate, "too many inputs in template")
	}

	msg, err := txbuilder.AssembleSignatures(txTemplate)
	if err != nil {
		return nil, errors.WithDetail(ErrBadTxTemplate, err.Error())
	}

	err = publishTx(ctx, msg)
	if err != nil {
		rawtx, err2 := msg.MarshalText()
		if err2 != nil {
			// ignore marshalling errors (they should never happen anyway)
			return nil, err
		}
		return nil, errors.Wrapf(err, "tx=%s", rawtx)
	}

	return msg, nil
}

// TODO(boymanjor): Refactor into fc.LatestBlockHeight
func latestBlockHeight(ctx context.Context) (uint64, error) {
	b, err := fc.LatestBlock(ctx)
	if errors.Root(err) == cos.ErrNoBlocks {
		return 0, nil
	} else if err != nil {
		return 0, errors.Wrap(err, "getting latest block")
	}
	return b.Height, nil
}

// FinalizeTxWait calls FinalizeTx and then waits for confirmation of
// the transaction.  A nil error return means the transaction is
// confirmed on the blockchain.  ErrRejected means a conflicting tx is
// on the blockchain.  context.DeadlineExceeded means ctx is an
// expiring context that timed out.
func FinalizeTxWait(ctx context.Context, txTemplate *txbuilder.Template) (*bc.Tx, error) {
	// Avoid a race condition.  Calling latestBlockHeight here ensures that
	// when we start waiting for blocks below, we don't begin waiting at
	// block N+1 when the tx we want is in block N.
	height, err := latestBlockHeight(ctx)
	if err != nil {
		return nil, err
	}

	tx, err := FinalizeTx(ctx, txTemplate)
	if err != nil {
		return nil, err
	}

	for {
		height++
		select {
		case <-ctx.Done():
			return nil, ctx.Err()

		case err := <-waitBlock(ctx, height):
			if err != nil {
				// This should be impossible, since the only error produced by
				// WaitForBlock is ErrTheDistantFuture, and height is known
				// not to be in "the distant future."
				return nil, errors.Wrapf(err, "waiting for block %d", height)
			}
			// TODO(bobg): This technique is not future-proof.  The database
			// won't necessarily contain all the txs we might care about.
			// An alternative approach will be to scan through each block as
			// it lands, looking for the tx or a tx that conflicts with it.
			// For now, though, this is probably faster and simpler.
			bcTxs, err := fc.ConfirmedTxs(ctx, tx.Hash)
			if err != nil {
				return nil, errors.Wrap(err, "getting bc txs")
			}
			if _, ok := bcTxs[tx.Hash]; ok {
				// confirmed
				return tx, nil
			}

			poolTxs, err := fc.PendingTxs(ctx, tx.Hash)
			if err != nil {
				return nil, errors.Wrap(err, "getting pool txs")
			}
			if _, ok := poolTxs[tx.Hash]; !ok {
				// rejected
				return nil, ErrRejected
			}

			// still in the pool; iterate
		}
	}
}

func waitBlock(ctx context.Context, height uint64) <-chan error {
	c := make(chan error, 1)
	go func() { c <- fc.WaitForBlock(ctx, height) }()
	return c
}

func publishTx(ctx context.Context, msg *bc.Tx) error {
	// Make sure there is atleast one block in case client is
	// trying to finalize a tx before the genesis block has landed
	fc.WaitForBlock(ctx, 1)
	err := fc.AddTx(ctx, msg)
	if errors.Root(err) == validation.ErrBadTx {
		detail := errors.Detail(err)
		err = errors.Wrap(ErrBadTxTemplate, err)
		return errors.WithDetail(err, detail)
	} else if err != nil {
		return errors.Wrap(err, "add tx to blockchain")
	}

	if Generator != nil && *Generator != "" {
		err = rpcclient.Submit(ctx, msg)
		if err != nil {
			err = errors.Wrap(err, "generator transaction notice")
			chainlog.Error(ctx, err)

			// Return an error so that the client knows that it needs to
			// retry the request.
			return err
		}
	}
	return nil
}

// Note, FC guarantees it will call the tx callback
// for every tx in b before we get here.
func indexAccountUTXOs(ctx context.Context, b *bc.Block) {
	var (
		pos    []int32
		txhash []string
	)
	for i, tx := range b.Transactions {
		pos = append(pos, int32(i))
		txhash = append(txhash, tx.Hash.String())
	}

	const q = `
		UPDATE account_utxos SET confirmed_in=$3, block_timestamp=$4, block_pos=pos
		FROM (SELECT unnest($1::text[]) AS txhash, unnest($2::integer[]) AS pos) t
		WHERE tx_hash=txhash
	`
	_, err := pg.Exec(
		ctx,
		q,
		pg.Strings(txhash),
		pg.Int32s(pos),
		b.Height,
		b.Timestamp,
	)
	if err != nil {
		// TODO(kr): make these errors stop log replay (e.g. crash the process)
		chainlog.Write(ctx, "at", "account utxos indexing block", "block", b.Height, "error", errors.Wrap(err))
	}

	deltxhash, delindex := prevoutDBKeys(b.Transactions...)

	const delQ = `
		DELETE FROM account_utxos
		WHERE (tx_hash, index) IN (SELECT unnest($1::text[]), unnest($2::integer[]))
	`
	_, err = pg.Exec(ctx, delQ, deltxhash, delindex)
	if err != nil {
		chainlog.Write(ctx, "block", b.Height, "error", errors.Wrap(err))
		panic(err)
	}
}

func addAccountData(ctx context.Context, tx *bc.Tx) error {
	var outs []*state.Output
	for i, out := range tx.Outputs {
		stateOutput := &state.Output{
			TxOutput: *out,
			Outpoint: bc.Outpoint{Hash: tx.Hash, Index: uint32(i)},
		}
		outs = append(outs, stateOutput)
	}

	addrOuts, err := LoadAccountInfo(ctx, outs)
	if err != nil {
		return errors.Wrap(err, "loading account info from addresses")
	}

	err = insertAccountOutputs(ctx, addrOuts)
	return errors.Wrap(err, "updating pool outputs")
}

func prevoutDBKeys(txs ...*bc.Tx) (txhash pg.Strings, index pg.Uint32s) {
	for _, tx := range txs {
		for _, in := range tx.Inputs {
			if in.IsIssuance() {
				continue
			}
			txhash = append(txhash, in.Previous.Hash.String())
			index = append(index, in.Previous.Index)
		}
	}
	return
}

// insertAccountOutputs records the account data for utxos
func insertAccountOutputs(ctx context.Context, outs []*txdb.Output) error {
	var (
		txHash        pg.Strings
		index         pg.Uint32s
		assetID       pg.Strings
		amount        pg.Int64s
		accountID     pg.Strings
		managerNodeID pg.Strings
		aIndex        pg.Int64s
		script        pg.Byteas
		metadata      pg.Byteas
	)
	for _, out := range outs {
		txHash = append(txHash, out.Outpoint.Hash.String())
		index = append(index, out.Outpoint.Index)
		assetID = append(assetID, out.AssetID.String())
		amount = append(amount, int64(out.Amount))
		accountID = append(accountID, out.AccountID)
		managerNodeID = append(managerNodeID, out.ManagerNodeID)
		aIndex = append(aIndex, toKeyIndex(out.AddrIndex[:]))
		script = append(script, out.ControlProgram)
		metadata = append(metadata, out.ReferenceData)
	}

	const q = `
		WITH outputs AS (
			SELECT t.* FROM unnest($1::text[], $2::bigint[], $3::text[], $4::bigint[], $5::text[], $6::text[], $7::bigint[], $8::bytea[], $9::bytea[])
			AS t(tx_hash, index, asset_id, amount, mnode, acc, addr_index, script, metadata)
		)
		INSERT INTO account_utxos (tx_hash, index, asset_id, amount, manager_node_id, account_id, addr_index, script, metadata)
		SELECT * FROM outputs o
		ON CONFLICT (tx_hash, index) DO NOTHING
	`
	_, err := pg.Exec(ctx, q,
		txHash,
		index,
		assetID,
		amount,
		managerNodeID,
		accountID,
		aIndex,
		script,
		metadata,
	)

	return errors.Wrap(err)
}

func toKeyIndex(i []uint32) int64 {
	return int64(i[0])<<31 | int64(i[1]&0x7fffffff)
}

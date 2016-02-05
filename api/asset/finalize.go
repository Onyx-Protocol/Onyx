package asset

import (
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/asset/nodetxlog"
	"chain/api/txbuilder"
	"chain/api/txdb"
	"chain/api/utxodb"
	"chain/database/pg"
	"chain/errors"
	"chain/fedchain/bc"
	"chain/fedchain/state"
	"chain/fedchain/validation"
	"chain/metrics"
)

// ErrBadTx is returned by FinalizeTx
var ErrBadTx = errors.New("bad transaction template")

// FinalizeTx validates a transaction signature template,
// assembles a fully signed tx, and stores the effects of
// its changes on the UTXO set.
func FinalizeTx(ctx context.Context, txTemplate *txbuilder.Template) (*bc.Tx, error) {
	defer metrics.RecordElapsed(time.Now())

	if len(txTemplate.Inputs) > len(txTemplate.Unsigned.Inputs) {
		return nil, errors.WithDetail(ErrBadTx, "too many inputs in template")
	} else if len(txTemplate.Unsigned.Outputs) != len(txTemplate.OutRecvs) {
		return nil, errors.Wrapf(ErrBadTx, "txTemplate has %d outputs but output receivers list has %d", len(txTemplate.Unsigned.Outputs), len(txTemplate.OutRecvs))
	}

	msg, err := txbuilder.AssembleSignatures(txTemplate)
	if err != nil {
		return nil, errors.WithDetail(ErrBadTx, err.Error())
	}

	err = publishTx(ctx, msg, txTemplate.OutRecvs)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func publishTx(ctx context.Context, msg *bc.Tx, receivers []txbuilder.Receiver) (err error) {
	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return errors.Wrap(err)
	}
	defer dbtx.Rollback(ctx)

	poolView, err := txdb.NewPoolViewForPrevouts(ctx, []*bc.Tx{msg})
	if err != nil {
		return errors.Wrap(err)
	}

	bcView, err := txdb.NewViewForPrevouts(ctx, []*bc.Tx{msg})
	if err != nil {
		return errors.Wrap(err)
	}

	view := state.MultiReader(poolView, bcView)
	// TODO(kr): get current block hash for last argument to ValidateTx
	err = validation.ValidateTx(ctx, view, msg, uint64(time.Now().Unix()), nil)
	if err != nil {
		return errors.Wrapf(ErrBadTx, "validate tx: %v", err)
	}

	// Update persistent tx pool state
	deleted, inserted, err := applyTx(ctx, msg, receivers)
	if err != nil {
		return errors.Wrap(err, "apply TX")
	}

	err = nodetxlog.Write(ctx, msg, time.Now())
	if err != nil {
		return errors.Wrap(err, "writing activitiy")
	}

	if msg.IsIssuance() {
		asset, amt := issued(msg.Outputs)
		err = appdb.UpdateIssuances(
			ctx,
			map[bc.AssetID]int64{asset: int64(amt)},
			false,
		)
		if err != nil {
			return errors.Wrap(err, "update issuances")
		}
	}

	// Fetch account data for deleted UTXOs so we can apply the deletions to
	// the reservation system.
	delUTXOs, err := getUTXOsForDeletion(ctx, deleted)
	if err != nil {
		return errors.Wrap(err, "get UTXOs for deletion")
	}

	// Repack the inserted UTXO data into a format the reservation system can
	// understand.
	var insUTXOs []*utxodb.UTXO
	for _, o := range inserted {
		// The reserver is only interested in outputs that have a defined
		// account ID. Outputs with blank account IDs are external to this
		// manager node.
		if o.AccountID == "" {
			continue
		}

		insUTXOs = append(insUTXOs, &utxodb.UTXO{
			Outpoint:  o.Outpoint,
			AssetID:   o.AssetID,
			Amount:    o.Amount,
			AccountID: o.AccountID,
			AddrIndex: o.AddrIndex,
		})
	}

	err = dbtx.Commit(ctx)
	if err != nil {
		return errors.Wrap(err)
	}

	// Update reservation state
	utxoDB.Apply(delUTXOs, insUTXOs)
	return nil
}

// issued returns the asset issued, as well as the amount.
// It should only be called with outputs from transactions
// where isIssuance is true.
func issued(outs []*bc.TxOutput) (asset bc.AssetID, amt uint64) {
	for _, out := range outs {
		amt += out.Amount
	}
	return outs[0].AssetID, amt
}

// getUTXOsForDeletion takes a set of outpoints and retrieves a list of
// partial utxodb.UTXOs, with enough information to be used in
// utxodb.Reserver.delete.
// TODO(jeffomatic) - consider revising the signature for utxodb.Reserver.delete
// so that it takes a smaller data structure. This way, we don't have to
// generate and propagate partially-filled data structures.
func getUTXOsForDeletion(ctx context.Context, ops []bc.Outpoint) ([]*utxodb.UTXO, error) {
	defer metrics.RecordElapsed(time.Now())

	var (
		hashes  []string
		indexes []uint32
	)
	for _, op := range ops {
		hashes = append(hashes, op.Hash.String())
		indexes = append(indexes, op.Index)
	}

	const q = `
		SELECT tx_hash, index, account_id, asset_id
		FROM utxos
		WHERE (tx_hash, index) IN (SELECT unnest($1::text[]), unnest($2::bigint[]))
	`
	rows, err := pg.FromContext(ctx).Query(ctx, q, pg.Strings(hashes), pg.Uint32s(indexes))
	if err != nil {
		return nil, errors.Wrap(err)
	}
	defer rows.Close()

	var utxos []*utxodb.UTXO
	for rows.Next() {
		u := new(utxodb.UTXO)
		err := rows.Scan(&u.Outpoint.Hash, &u.Outpoint.Index, &u.AccountID, &u.AssetID)
		if err != nil {
			return nil, errors.Wrap(err, "scan")
		}
		utxos = append(utxos, u)
	}
	return utxos, errors.Wrap(rows.Err())
}

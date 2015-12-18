package asset

import (
	"time"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/api/txdb"
	"chain/api/utxodb"
	"chain/database/pg"
	"chain/errors"
	"chain/fedchain/bc"
	"chain/fedchain/state"
	"chain/fedchain/txscript"
	"chain/fedchain/validation"
	"chain/metrics"
)

// ErrBadTx is returned by FinalizeTx
var ErrBadTx = errors.New("bad transaction template")

// FinalizeTx validates a transaction signature template,
// assembles a fully signed tx, and stores the effects of
// its changes on the UTXO set.
func FinalizeTx(ctx context.Context, tx *TxTemplate) (*bc.Tx, error) {
	defer metrics.RecordElapsed(time.Now())

	if len(tx.Inputs) > len(tx.Unsigned.Inputs) {
		return nil, errors.WithDetail(ErrBadTx, "too many inputs in template")
	} else if len(tx.Unsigned.Outputs) != len(tx.OutRecvs) {
		return nil, errors.Wrapf(ErrBadTx, "tx has %d outputs but output receivers list has %d", len(tx.Unsigned.Outputs), len(tx.OutRecvs))
	}

	msg, err := assembleSignatures(tx)
	if err != nil {
		return nil, err
	}

	err = publishTx(ctx, msg, tx.OutRecvs)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func assembleSignatures(tx *TxTemplate) (*bc.Tx, error) {
	msg := tx.Unsigned
	for i, input := range tx.Inputs {
		sigsAdded := 0
		sigsReqd, err := getSigsRequired(input.RedeemScript)
		if err != nil {
			return nil, err
		}
		if len(input.Sigs) == 0 {
			return nil, errors.WithDetailf(ErrBadTx, "input %d must contain signatures", i)
		}
		builder := txscript.NewScriptBuilder()
		builder.AddOp(txscript.OP_FALSE)
		for _, sig := range input.Sigs {
			if len(sig.DER) > 0 {
				builder.AddData(sig.DER)
				sigsAdded++
				if sigsAdded == sigsReqd {
					break
				}
			}
		}
		builder.AddData(input.RedeemScript)
		script, err := builder.Script()
		if err != nil {
			return nil, errors.Wrap(err)
		}
		msg.Inputs[i].SignatureScript = script
	}
	return bc.NewTx(*msg), nil
}

func getSigsRequired(script []byte) (sigsReqd int, err error) {
	sigsReqd = 1
	if txscript.GetScriptClass(script) == txscript.MultiSigTy {
		_, sigsReqd, err = txscript.CalcMultiSigStats(script)
		if err != nil {
			return 0, err
		}
	}
	return sigsReqd, nil
}

func publishTx(ctx context.Context, msg *bc.Tx, receivers []*utxodb.Receiver) (err error) {
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

	mv := NewMemView()
	view := state.Compose(mv, poolView, bcView)
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

	outIsChange := make(map[int]bool)
	for i, r := range receivers {
		if r != nil && r.IsChange {
			outIsChange[i] = true
		}
	}
	err = appdb.WriteActivity(ctx, msg, outIsChange, time.Now())
	if err != nil {
		return errors.Wrap(err, "writing activitiy")
	}

	err = WriteNodeTxs(ctx, msg, time.Now())
	if err != nil {
		return errors.Wrap(err, "writing activitiy")
	}

	if msg.IsIssuance() {
		asset, amt := issued(msg.Outputs)
		err = appdb.UpdateIssuances(
			ctx,
			map[string]int64{asset.String(): int64(amt)},
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
			AssetID:   o.AssetID.String(),
			Amount:    o.Value,
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
		amt += out.Value
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

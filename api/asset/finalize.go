package asset

import (
	"time"

	"golang.org/x/net/context"

	"chain/api/rpcclient"
	"chain/api/txbuilder"
	"chain/api/txdb"
	"chain/database/pg"
	"chain/errors"
	"chain/fedchain/bc"
	"chain/fedchain/state"
	chainlog "chain/log"
	"chain/metrics"
	"chain/net/trace/span"
)

// ErrBadTx is returned by FinalizeTx
var ErrBadTx = errors.New("bad transaction template")

var Generator *string

// FinalizeTx validates a transaction signature template,
// assembles a fully signed tx, and stores the effects of
// its changes on the UTXO set.
func FinalizeTx(ctx context.Context, txTemplate *txbuilder.Template) (*bc.Tx, error) {
	defer metrics.RecordElapsed(time.Now())

	if len(txTemplate.Inputs) > len(txTemplate.Unsigned.Inputs) {
		return nil, errors.WithDetail(ErrBadTx, "too many inputs in template")
	}

	msg, err := txbuilder.AssembleSignatures(txTemplate)
	if err != nil {
		return nil, errors.WithDetail(ErrBadTx, err.Error())
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

func publishTx(ctx context.Context, msg *bc.Tx) error {
	err := fc.AddTx(ctx, msg)
	if err != nil {
		return errors.Wrap(err, "add tx to fedchain")
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

func addAccountData(ctx context.Context, tx *bc.Tx) error {
	var outs []*txdb.Output
	for i, out := range tx.Outputs {
		txdbOutput := &txdb.Output{
			Output: state.Output{
				TxOutput: *out,
				Outpoint: bc.Outpoint{Hash: tx.Hash, Index: uint32(i)},
			},
		}
		outs = append(outs, txdbOutput)
	}

	addrOuts, err := loadAccountInfo(ctx, outs)
	if err != nil {
		return errors.Wrap(err, "loading account info from addresses")
	}

	err = txdb.InsertAccountOutputs(ctx, addrOuts)
	return errors.Wrap(err, "updating pool outputs")
}

// loadAccountInfo queries the addresses table
// to load account information using output scripts
func loadAccountInfo(ctx context.Context, outs []*txdb.Output) ([]*txdb.Output, error) {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	var (
		scripts      [][]byte
		outsByScript = make(map[string][]*txdb.Output)
	)
	for _, out := range outs {
		scripts = append(scripts, out.Script)
		outsByScript[string(out.Script)] = append(outsByScript[string(out.Script)], out)
	}

	const addrq = `
		SELECT pk_script, manager_node_id, account_id, key_index(key_index)
		FROM addresses
		WHERE pk_script IN (SELECT unnest($1::bytea[]))
	`
	rows, err := pg.FromContext(ctx).Query(ctx, addrq, pg.Byteas(scripts))
	if err != nil {
		return nil, errors.Wrap(err, "addresses select query")
	}
	defer rows.Close()

	var addrOuts []*txdb.Output
	for rows.Next() {
		var (
			script         []byte
			mnodeID, accID string
			addrIndex      []uint32
		)
		err := rows.Scan(&script, &mnodeID, &accID, (*pg.Uint32s)(&addrIndex))
		if err != nil {
			return nil, errors.Wrap(err, "addresses row scan")
		}
		for _, out := range outsByScript[string(script)] {
			out.ManagerNodeID = mnodeID
			out.AccountID = accID
			copy(out.AddrIndex[:], addrIndex)
			addrOuts = append(addrOuts, out)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(rows.Err(), "addresses end row scan loop")
	}
	return addrOuts, nil
}

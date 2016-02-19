package asset

import (
	"time"

	"golang.org/x/net/context"

	"chain/api/asset/nodetxlog"
	"chain/api/txdb"
	"chain/database/pg"
	"chain/errors"
	"chain/fedchain"
	"chain/fedchain/bc"
	"chain/log"
	"chain/net/trace/span"
)

var fc *fedchain.FC

// Init sets the package level fedchain. If isManager is true,
// Init registers all necessary callbacks for updating
// application state with the fedchain.
func Init(chain *fedchain.FC, isManager bool) {
	if fc == chain {
		// Silently ignore duplicate calls.
		return
	}

	fc = chain
	if isManager {
		fc.AddTxCallback(func(ctx context.Context, tx *bc.Tx) {
			err := addAccountData(ctx, tx)
			if err != nil {
				log.Error(ctx, errors.Wrap(err, "adding account data"))
			}
		})
		fc.AddTxCallback(func(ctx context.Context, tx *bc.Tx) {
			err := nodetxlog.Write(ctx, tx, time.Now())
			if err != nil {
				log.Error(ctx, errors.Wrap(err, "writing activitiy"))
			}
		})
	}
}

// loadAccountInfoFromAddrs queries the addresses table
// to load account information using output scripts
func loadAccountInfoFromAddrs(ctx context.Context, outs map[bc.Outpoint]*txdb.Output) error {
	ctx = span.NewContext(ctx)
	defer span.Finish(ctx)

	var (
		scripts           [][]byte
		outpointsByScript = make(map[string]bc.Outpoint)
	)
	for _, out := range outs {

		scripts = append(scripts, out.Script)
		outpointsByScript[string(out.Script)] = out.Outpoint
	}

	const addrq = `
		SELECT pk_script, manager_node_id, account_id, key_index(key_index)
		FROM addresses
		WHERE pk_script IN (SELECT unnest($1::bytea[]))
	`
	rows, err := pg.FromContext(ctx).Query(ctx, addrq, pg.Byteas(scripts))
	if err != nil {
		return errors.Wrap(err, "addresses select query")
	}
	defer rows.Close()

	for rows.Next() {
		var (
			script         []byte
			mnodeID, accID string
			addrIndex      []uint32
		)
		err := rows.Scan(&script, &mnodeID, &accID, (*pg.Uint32s)(&addrIndex))
		if err != nil {
			return errors.Wrap(err, "addresses row scan")
		}
		out := outs[outpointsByScript[string(script)]]
		out.ManagerNodeID = mnodeID
		out.AccountID = accID
		copy(out.AddrIndex[:], addrIndex)
	}
	return errors.Wrap(rows.Err(), "addresses end row scan loop")
}

func issuedAssets(txs []*bc.Tx) map[bc.AssetID]int64 {
	issued := make(map[bc.AssetID]int64)
	for _, tx := range txs {
		if !tx.IsIssuance() {
			continue
		}
		for _, out := range tx.Outputs {
			issued[out.AssetID] += int64(out.Amount)
		}
	}
	return issued
}

// Optional features

package explorer

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/context"

	"chain/core/asset"
	"chain/cos"
	"chain/cos/bc"
	"chain/cos/state"
	"chain/database/pg"
	"chain/errors"
	chainlog "chain/log"
	"chain/net/http/httpjson"
)

var historicalOutputs bool

// Connect adds a block callback for indexing utxos in the
// explorer_outputs table and occasionally pruning ones spent
// spent more than maxAgeDays ago.  (If maxAgeDays is <= 0, no
// pruning is done.)
func Connect(ctx context.Context, fc *cos.FC, historical bool, maxAgeDays int, isManager bool) {
	historicalOutputs = historical

	maxAge := time.Duration(maxAgeDays*24) * time.Hour
	var lastPrune time.Time

	fc.AddBlockCallback(func(ctx context.Context, block *bc.Block, conflicts []*bc.Tx) {
		var (
			newTxHashes   pg.Strings
			newIndexes    pg.Uint32s
			newAssetIDs   pg.Strings
			newAmounts    pg.Uint64s
			newScripts    pg.Byteas
			newMetadatas  pg.Byteas
			spentTxHashes pg.Strings
			spentIndexes  pg.Uint32s
		)

		const insertQ = `
			INSERT INTO explorer_outputs (tx_hash, index, asset_id, amount, script, metadata, timespan)
				SELECT UNNEST($1::TEXT[]), UNNEST($2::INTEGER[]), UNNEST($3::TEXT[]), UNNEST($4::BIGINT[]), UNNEST($5::BYTEA[]), UNNEST($6::BYTEA[]), INT8RANGE($7, NULL)
		`
		const updateQ = `
			UPDATE explorer_outputs SET timespan = INT8RANGE(LOWER(timespan), $3)
				WHERE (tx_hash, index) IN (SELECT unnest($1::text[]), unnest($2::integer[]))
		`
		const deleteQ = `
			DELETE FROM explorer_outputs
				WHERE (tx_hash, index) IN (SELECT unnest($1::text[]), unnest($2::integer[]))
		`

		var stateOuts []*state.Output

		for _, tx := range block.Transactions {
			txHash := tx.Hash
			txHashStr := txHash.String()
			for _, txin := range tx.Inputs {
				if txin.IsIssuance() {
					continue
				}
				spentTxHashes = append(spentTxHashes, txin.Previous.Hash.String())
				spentIndexes = append(spentIndexes, txin.Previous.Index)
			}
			for index, txout := range tx.Outputs {
				newTxHashes = append(newTxHashes, txHashStr)
				newIndexes = append(newIndexes, uint32(index))
				newAssetIDs = append(newAssetIDs, txout.AssetID.String())
				newAmounts = append(newAmounts, txout.Amount)
				newScripts = append(newScripts, txout.Script)
				newMetadatas = append(newMetadatas, txout.Metadata)

				if isManager {
					stateOut := &state.Output{
						TxOutput: *txout,
						Outpoint: bc.Outpoint{
							Hash:  txHash,
							Index: uint32(index),
						},
					}
					stateOuts = append(stateOuts, stateOut)
				}
			}
		}
		_, err := pg.Exec(ctx, insertQ, newTxHashes, newIndexes, newAssetIDs, newAmounts, newScripts, newMetadatas, block.Timestamp)
		if err != nil {
			chainlog.Error(ctx, errors.Wrap(err, "inserting to explorer_outputs"))
			return // or panic?
		}
		if historical {
			_, err = pg.Exec(ctx, updateQ, spentTxHashes, spentIndexes, block.Timestamp)
			if err != nil {
				chainlog.Error(ctx, errors.Wrap(err, "updating explorer_outputs"))
				return // or panic?
			}
		} else {
			_, err = pg.Exec(ctx, deleteQ, spentTxHashes, spentIndexes)
			if err != nil {
				chainlog.Error(ctx, errors.Wrap(err, "deleting explorer_outputs"))
				return // or panic?
			}
		}

		if isManager {
			txdbOutputs, err := asset.LoadAccountInfo(ctx, stateOuts)
			if err != nil {
				chainlog.Error(ctx, errors.Wrap(err, "loading account info for explorer_outputs"))
				return // or panic?
			}
			var (
				accountIDs pg.Strings
				txHashes   pg.Strings
				indexes    pg.Uint32s
			)
			for _, out := range txdbOutputs {
				accountIDs = append(accountIDs, out.AccountID)
				txHashes = append(txHashes, out.Outpoint.Hash.String())
				indexes = append(indexes, out.Outpoint.Index)
			}

			const annotateQ = `
				UPDATE explorer_outputs h SET account_id = t.account_id
					FROM (SELECT unnest($1::text[]) AS account_id, unnest($2::text[]) AS tx_hash, unnest($3::integer[]) AS index) t
					WHERE h.tx_hash = t.tx_hash AND h.index = t.index
			`
			_, err = pg.Exec(ctx, annotateQ, accountIDs, txHashes, indexes)
			if err != nil {
				chainlog.Error(ctx, errors.Wrap(err, "annotating explorer_outputs with account info"))
				return // or panic?
			}
		}

		if historical && maxAge > 0 && time.Since(lastPrune) >= 24*time.Hour {
			now := time.Now()
			_, err := pg.Exec(ctx, "DELETE FROM explorer_outputs WHERE UPPER(timespan) < $1", now.Add(-maxAge))
			if err == nil {
				lastPrune = now
			} else {
				chainlog.Error(ctx, errors.Wrap(err, "pruning explorer_outputs"))
			}
		}
	})
}

// HistoricalBalancesByAccount queries the explorer_outputs table
// for outputs in the given account at the given time and sums them by
// assetID.  If the assetID parameter is non-nil, the output is
// constrained to the balance of that asset only.
func HistoricalBalancesByAccount(ctx context.Context, accountID string, timestamp time.Time, assetID *bc.AssetID, prev string, limit int) ([]bc.AssetAmount, string, error) {
	if limit > 0 && assetID != nil {
		return nil, "", errors.New("cannot set both pagination and asset id filter")
	}

	q := "SELECT asset_id, SUM(amount) FROM explorer_outputs WHERE account_id = $1 AND timespan @> $2::int8"
	args := []interface{}{
		accountID,
		timestamp.Unix(),
	}

	if assetID != nil {
		q += " AND asset_id = $3"
		args = append(args, *assetID)
	} else if limit > 0 {
		q += " AND asset_id>$3"
		args = append(args, *assetID)
		q += fmt.Sprintf("LIMIT %d", limit)
	}

	q += " GROUP BY asset_id"
	if limit > 0 {
		q += " ORDER BY asset_id"
	}

	var (
		output []bc.AssetAmount
		last   string
	)
	args = append(args, func(assetID bc.AssetID, amount uint64) {
		output = append(output, bc.AssetAmount{assetID, amount})

	})
	err := pg.ForQueryRows(ctx, q, args...)
	if err != nil {
		return nil, "", err
	}

	if limit > 0 || len(output) > 0 {
		last = output[len(output)-1].AssetID.String()
	}
	return output, last, nil
}

// ListHistoricalOutputsByAsset returns an array of every UTXO that contains assetID at timestamp.
// When paginating, it takes a limit as well as `prev`, the last UTXO returned on the previous call.
// ListHistoricalOutputsByAsset expects prev to be of the format "hash:index".
func ListHistoricalOutputsByAsset(ctx context.Context, assetID bc.AssetID, timestamp time.Time, prev string, limit int) ([]*TxOutput, string, error) {
	if !historicalOutputs {
		return nil, "", errors.WithDetail(httpjson.ErrBadRequest, "historical outputs aren't enabled on this core")
	}
	return listHistoricalOutputsByAssetAndAccount(ctx, assetID, "", timestamp, prev, limit)
}

func listHistoricalOutputsByAssetAndAccount(ctx context.Context, assetID bc.AssetID, accountID string, timestamp time.Time, prev string, limit int) ([]*TxOutput, string, error) {
	ts := timestamp.Unix()
	prevs := strings.Split(prev, ":")
	var (
		prevHash  string
		prevIndex int64
		err       error
	)

	if len(prevs) != 2 {
		// tolerate malformed/empty cursors
		prevHash = ""
		prevIndex = -1
	} else {
		prevHash = prevs[0]
		prevIndex, err = strconv.ParseInt(prevs[1], 10, 64)
		if err != nil {
			prevIndex = -1
		}
	}

	conditions := []string{
		"asset_id = $1",
		"timespan @> $2::int8",
		"tx_hash >= $3",
		"(tx_hash != $3 OR index > $4)", // prev index only matters if we're in the same tx
	}
	args := []interface{}{
		assetID, ts, prevHash, prevIndex,
	}
	if accountID != "" {
		conditions = append(conditions, "account_id = $5")
		args = append(args, accountID)
	}

	var limitClause string
	if limit > 0 {
		limitClause = fmt.Sprintf("LIMIT %d", limit)
	}

	var (
		res  []*state.Output
		last string
	)
	args = append(args, func(hash bc.Hash, index uint32, amount uint64, script, metadata []byte) {
		outpt := bc.Outpoint{Hash: hash, Index: index}
		o := &state.Output{
			Outpoint: outpt,
			TxOutput: bc.TxOutput{
				AssetAmount: bc.AssetAmount{AssetID: assetID, Amount: amount},
				Script:      script,
				Metadata:    metadata,
			},
		}
		res = append(res, o)
	})
	q := fmt.Sprintf("SELECT tx_hash, index, amount, script, metadata FROM explorer_outputs WHERE %s %s", strings.Join(conditions, " AND "), limitClause)

	err = pg.ForQueryRows(ctx, q, args...)
	if err != nil {
		return nil, "", err
	}

	if len(res) > 0 {
		last = res[len(res)-1].Outpoint.String()
	}
	return stateOutsToTxOuts(res), last, nil
}

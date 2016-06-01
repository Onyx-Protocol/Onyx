// Optional features

package explorer

import (
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

func InitHistoricalOutputs(fc *cos.FC, isManager bool) {
	historicalOutputs = true
	// TODO(bobg): Launch a goroutine that prunes old data from
	// historical_outputs
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
			INSERT INTO historical_outputs (tx_hash, index, asset_id, amount, script, metadata, timespan)
				SELECT UNNEST($1::TEXT[]), UNNEST($2::INTEGER[]), UNNEST($3::TEXT[]), UNNEST($4::BIGINT[]), UNNEST($5::BYTEA[]), UNNEST($6::BYTEA[]), INT8RANGE($7, NULL)
		`
		const updateQ = `
			UPDATE historical_outputs SET timespan = INT8RANGE(LOWER(timespan), $3)
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
			chainlog.Error(ctx, errors.Wrap(err, "inserting to historical_outputs"))
			return // or panic?
		}
		_, err = pg.Exec(ctx, updateQ, spentTxHashes, spentIndexes, block.Timestamp)
		if err != nil {
			chainlog.Error(ctx, errors.Wrap(err, "updating historical_outputs"))
			return // or panic?
		}

		if isManager {
			txdbOutputs, err := asset.LoadAccountInfo(ctx, stateOuts)
			if err != nil {
				chainlog.Error(ctx, errors.Wrap(err, "loading account info for historical_outputs"))
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
				UPDATE historical_outputs h SET account_id = t.account_id
					FROM (SELECT unnest($1::text[]) AS account_id, unnest($2::text[]) AS tx_hash, unnest($3::integer[]) AS index) t
					WHERE h.tx_hash = t.tx_hash AND h.index = t.index
			`
			_, err = pg.Exec(ctx, annotateQ, accountIDs, txHashes, indexes)
			if err != nil {
				chainlog.Error(ctx, errors.Wrap(err, "annotating historical_outputs with account info"))
				return // or panic?
			}
		}
	})
}

// ListHistoricalOutputsByAsset returns an array of every UTXO that contains assetID at timestamp.
// When paginating, it takes a limit as well as `prev`, the last UTXO returned on the previous call.
// ListHistoricalOutputsByAsset expects prev to be of the format "hash:index".
func ListHistoricalOutputsByAsset(ctx context.Context, assetID bc.AssetID, timestamp time.Time, prev string, limit int) ([]*TxOutput, string, error) {
	if !historicalOutputs {
		return nil, "", errors.WithDetail(httpjson.ErrBadRequest, "historical outputs aren't enabled on this core")
	}
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

	const q = `
		SELECT tx_hash, index, amount, script, metadata
		FROM historical_outputs
		WHERE asset_id = $1
			AND timespan @> $2::int8
			AND tx_hash >= $3
			-- prev index only matters if we're in the same tx
			AND (tx_hash != $3 OR index > $4)
		LIMIT $5
	`

	var (
		res  []*state.Output
		last string
	)
	err = pg.ForQueryRows(ctx, q, assetID, ts, prevHash, prevIndex, limit, func(hash bc.Hash, index uint32, amount uint64, script, metadata []byte) {
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
	if err != nil {
		return nil, "", err
	}

	if len(res) > 0 {
		last = res[len(res)-1].Outpoint.String()
	}
	return stateOutsToTxOuts(res), last, nil
}

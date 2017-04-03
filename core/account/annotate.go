package account

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/lib/pq"

	"chain/core/query"
	"chain/database/pg"
	"chain/errors"
	"chain/protocol/bc"
)

var empty = json.RawMessage(`{}`)

// AnnotateTxs adds account data to transactions
func (m *Manager) AnnotateTxs(ctx context.Context, txs []*query.AnnotatedTx) error {
	var (
		outputIDs [][]byte
		inputs    = make(map[bc.Hash]*query.AnnotatedInput)
		outputs   = make(map[bc.Hash]*query.AnnotatedOutput)
	)

	for _, tx := range txs {
		for _, in := range tx.Inputs {
			if in.SpentOutputID == nil {
				continue
			}

			inputs[*in.SpentOutputID] = in
			outputIDs = append(outputIDs, in.SpentOutputID.Bytes())
		}
		for _, out := range tx.Outputs {
			if out.Type == "retire" {
				continue
			}

			outputs[out.OutputID] = out
			outputIDs = append(outputIDs, out.OutputID.Bytes())
		}
	}

	// Look up all of the spent and created outputs. If any of them are
	// account UTXOs add the account annotations to the inputs and outputs.
	const q = `
		SELECT o.output_id, o.account_id, a.alias, a.tags, o.change
		FROM account_utxos o
		LEFT JOIN accounts a ON o.account_id = a.account_id
		WHERE o.output_id = ANY($1::bytea[])
	`
	err := pg.ForQueryRows(ctx, m.db, q, pq.ByteaArray(outputIDs),
		func(outputID bc.Hash, accID string, alias sql.NullString, accountTags []byte, change bool) {
			spendingInput, ok := inputs[outputID]
			if ok {
				spendingInput.AccountID = accID
				if alias.Valid {
					spendingInput.AccountAlias = alias.String
				}
				if len(accountTags) > 0 {
					spendingInput.AccountTags = (*json.RawMessage)(&accountTags)
				} else {
					spendingInput.AccountTags = &empty
				}
			}

			out, ok := outputs[outputID]
			if ok {
				out.AccountID = accID
				if alias.Valid {
					out.AccountAlias = alias.String
				}
				if len(accountTags) > 0 {
					out.AccountTags = (*json.RawMessage)(&accountTags)
				} else {
					out.AccountTags = &empty
				}
				if change {
					out.Purpose = "change"
				} else {
					out.Purpose = "receive"
				}
			}
		})
	return errors.Wrap(err, "annotating with account data")
}

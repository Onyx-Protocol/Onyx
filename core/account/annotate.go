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
		inputs            = make(map[bc.Hash]*query.AnnotatedInput)
		outputs           = make(map[string][]*query.AnnotatedOutput)
		controlProgramSet = make(map[string]bool)
		controlPrograms   [][]byte
		spentOutputIDs    [][]byte
	)

	for _, tx := range txs {
		for _, in := range tx.Inputs {
			if in.SpentOutputID == nil {
				continue
			}

			inputs[*in.SpentOutputID] = in
			spentOutputIDs = append(spentOutputIDs, in.SpentOutputID[:])
		}
		for _, out := range tx.Outputs {
			if out.Type == "retire" {
				continue
			}
			outputs[string(out.ControlProgram)] = append(outputs[string(out.ControlProgram)], out)
			if controlProgramSet[string(out.ControlProgram)] {
				continue
			}
			controlPrograms = append(controlPrograms, out.ControlProgram)
			controlProgramSet[string(out.ControlProgram)] = true
		}
	}

	// Look up all of the spent outputs. If any of them are account UTXOs
	// add the account annotations to the input.
	const inputsQ = `
		SELECT o.output_id, o.account_id, a.alias, a.tags
		FROM account_utxos o
		LEFT JOIN accounts a ON o.account_id = a.account_id
		WHERE o.output_id = ANY($1::bytea[])
	`
	err := pg.ForQueryRows(ctx, m.db, inputsQ, pq.ByteaArray(spentOutputIDs),
		func(outputID bc.Hash, accID string, alias sql.NullString, accountTags []byte) {
			spendingInput := inputs[outputID]
			spendingInput.AccountID = accID
			if alias.Valid {
				spendingInput.AccountAlias = alias.String
			}
			if len(accountTags) > 0 {
				spendingInput.AccountTags = (*json.RawMessage)(&accountTags)
			} else {
				spendingInput.AccountTags = &empty
			}
		})
	if err != nil {
		return errors.Wrap(err, "annotating input account data")
	}

	// Compare all new outputs' control programs to our own account
	// control programs. If we recognize any, add the relevant account
	// annotations.
	//
	// TODO(jackson): Instead of using `account_control_programs` here,
	// we should use `account_utxos`. We will need to add and backfill
	// the `change` field into `account_utxos` first.
	const outputsQ = `
		SELECT acp.control_program, a.account_id, acp.change, a.alias, a.tags
		FROM account_control_programs acp
		LEFT JOIN accounts a ON a.account_id = acp.signer_id
		WHERE acp.control_program = ANY($1::bytea[])
	`
	err = pg.ForQueryRows(ctx, m.db, outputsQ, pq.ByteaArray(controlPrograms),
		func(program []byte, accountID string, change bool, alias sql.NullString, accountTags []byte) {
			for _, out := range outputs[string(program)] {
				out.AccountID = accountID
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
	return errors.Wrap(err, "annotating output account data")
}

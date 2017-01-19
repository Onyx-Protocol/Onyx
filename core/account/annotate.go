package account

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/lib/pq"

	"chain/core/query"
	"chain/database/pg"
)

// AnnotateTxs adds account data to transactions
func (m *Manager) AnnotateTxs(ctx context.Context, txs []*query.AnnotatedTx) error {
	inputs := make(map[string][]*query.AnnotatedInput)
	outputs := make(map[string][]*query.AnnotatedOutput)
	controlProgramSet := make(map[string]bool)
	var controlPrograms [][]byte

	for _, tx := range txs {
		for _, in := range tx.Inputs {
			if len(in.ControlProgram) == 0 {
				continue
			}
			inputs[string(in.ControlProgram)] = append(inputs[string(in.ControlProgram)], in)
			if controlProgramSet[string(in.ControlProgram)] {
				continue
			}
			controlPrograms = append(controlPrograms, in.ControlProgram)
			controlProgramSet[string(in.ControlProgram)] = true
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

	const q = `
		SELECT signer_id, control_program, change, alias, tags
		FROM account_control_programs
		LEFT JOIN signers ON signers.id=account_control_programs.signer_id
		LEFT JOIN accounts ON accounts.account_id=signers.id
		WHERE control_program=ANY($1::bytea[])
	`
	var (
		ids         []string
		programs    [][]byte
		changeFlags []bool
		aliases     []sql.NullString
		tags        []*json.RawMessage
	)
	err := pg.ForQueryRows(ctx, m.db, q, pq.ByteaArray(controlPrograms), func(accountID string, program []byte, change bool, alias sql.NullString, accountTags []byte) {
		ids = append(ids, accountID)
		programs = append(programs, program)
		changeFlags = append(changeFlags, change)
		aliases = append(aliases, alias)
		if len(accountTags) > 0 {
			tags = append(tags, (*json.RawMessage)(&accountTags))
		} else {
			tags = append(tags, nil)
		}
	})
	if err != nil {
		return err
	}

	empty := json.RawMessage(`{}`)
	for i := range ids {
		inps := inputs[string(programs[i])]
		for _, inp := range inps {
			inp.AccountID = ids[i]
			if aliases[i].Valid {
				inp.AccountAlias = aliases[i].String
			}
			if tags[i] != nil {
				inp.AccountTags = tags[i]
			} else {
				inp.AccountTags = &empty
			}
		}

		outs := outputs[string(programs[i])]
		for _, out := range outs {
			out.AccountID = ids[i]
			if aliases[i].Valid {
				out.AccountAlias = aliases[i].String
			}
			if tags[i] != nil {
				out.AccountTags = tags[i]
			} else {
				out.AccountTags = &empty
			}

			if changeFlags[i] {
				out.Purpose = "change"
			} else {
				out.Purpose = "receive"
			}
		}
	}

	return nil
}

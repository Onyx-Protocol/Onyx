package account

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/lib/pq"

	"chain-stealth/database/pg"
	"chain-stealth/errors"
	"chain-stealth/log"
	"chain-stealth/protocol/bc"
)

// AnnotateTxs adds account data to transactions
func (m *Manager) AnnotateTxs(ctx context.Context, txs []map[string]interface{}, _ []*bc.Tx) error {
	controlMaps := make(map[string][]map[string]interface{})
	outputMaps := make(map[string][]map[string]interface{})
	var controlPrograms [][]byte

	update := func(s interface{}, maps ...map[string][]map[string]interface{}) {
		asSlice, ok := s.([]interface{})
		if !ok {
			log.Error(ctx, errors.Wrap(fmt.Errorf("expected slice, got %T", s)))
			return
		}
		for _, m := range asSlice {
			asMap, ok := m.(map[string]interface{})
			if !ok {
				log.Error(ctx, errors.Wrap(fmt.Errorf("expected map, got %T", m)))
				continue
			}
			if asMap["control_program"] == nil {
				// Issuance inputs don't have control_programs
				continue
			}
			controlString, ok := asMap["control_program"].(string)
			if !ok {
				log.Error(ctx, errors.Wrap(fmt.Errorf("expected string, got %T", asMap["control_program"])))
				continue
			}
			controlProgram, err := hex.DecodeString(controlString)
			if err != nil {
				log.Error(ctx, errors.Wrap(err, "could not decode control program"))
				continue
			}
			controlPrograms = append(controlPrograms, controlProgram)
			for _, m := range maps {
				m[string(controlProgram)] = append(m[string(controlProgram)], asMap)
			}
		}
	}

	for _, tx := range txs {
		update(tx["outputs"], controlMaps, outputMaps)
		update(tx["inputs"], controlMaps)
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
	for i := range ids {
		maps := controlMaps[string(programs[i])]
		for _, m := range maps {
			m["account_id"] = ids[i]
			if tags[i] != nil {
				m["account_tags"] = tags[i]
			}
			if aliases[i].Valid {
				m["account_alias"] = aliases[i].String
			}
		}

		// Add output-only annotations.
		outs := outputMaps[string(programs[i])]
		for _, out := range outs {
			if changeFlags[i] {
				out["purpose"] = "change"
			} else {
				out["purpose"] = "receive"
			}
		}
	}

	return nil
}

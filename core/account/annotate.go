package account

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"golang.org/x/net/context"

	"chain/cos/txscript"
	"chain/database/pg"
	"chain/errors"
)

// AnnotateTxs adds account data to transactions
func AnnotateTxs(ctx context.Context, txs []map[string]interface{}) error {
	controlMaps := make(map[string][]map[string]interface{})
	var controlPrograms [][]byte
	for _, tx := range txs {
		outs, ok := tx["outputs"].([]interface{})
		if !ok {
			return errors.Wrap(fmt.Errorf("bad outputs type %T", tx["outputs"]))
		}

		for _, out := range outs {
			txOut, ok := out.(map[string]interface{})
			if !ok {
				return errors.Wrap(fmt.Errorf("bad output type %T", out))
			}
			controlString, ok := txOut["control_program"].(string)
			if !ok {
				return errors.Wrap(fmt.Errorf("bad control program type %T", txOut["control_program"]))
			}
			controlProgram, err := hex.DecodeString(controlString)
			if err != nil {
				return err
			}
			controlPrograms = append(controlPrograms, controlProgram)
			controlMaps[string(controlProgram)] = append(controlMaps[string(controlProgram)], txOut)
		}

		ins, ok := tx["inputs"].([]interface{})
		if !ok {
			return errors.Wrap(fmt.Errorf("bad inputs type %T", tx["inputs"]))
		}

		for _, in := range ins {
			txIn, ok := in.(map[string]interface{})
			if !ok {
				return errors.Wrap(fmt.Errorf("bad input type %T", in))
			}
			inputWitness, ok := txIn["input_witness"].([]interface{})
			if !ok {
				return errors.Wrap(fmt.Errorf("bad input witness type %T", txIn["input_witness"]))
			}
			maybeRedeemStr, ok := inputWitness[len(inputWitness)-1].(string)
			if !ok {
				return errors.Wrap(fmt.Errorf("bad input witness item type %T", inputWitness[len(inputWitness)-1]))
			}
			maybeRedeem, err := hex.DecodeString(maybeRedeemStr)
			if err != nil {
				return err
			}
			controlProgram := txscript.RedeemToPkScript(maybeRedeem)
			controlPrograms = append(controlPrograms, controlProgram)
			controlMaps[string(controlProgram)] = append(controlMaps[string(controlProgram)], txIn)
		}
	}

	const q = `
		SELECT signer_id, control_program, alias, tags
		FROM account_control_programs
		LEFT JOIN signers ON signers.id=account_control_programs.signer_id
		LEFT JOIN accounts ON accounts.account_id=signers.id
		WHERE control_program=ANY($1::bytea[])
	`
	var (
		ids      []string
		programs [][]byte
		aliases  []sql.NullString
		tags     []*json.RawMessage
	)
	err := pg.ForQueryRows(ctx, q, pg.Byteas(controlPrograms), func(accountID string, program []byte, alias sql.NullString, accountTags []byte) {
		ids = append(ids, accountID)
		programs = append(programs, program)
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
	}

	return nil
}

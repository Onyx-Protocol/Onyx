// Package confidentiality stores data required for blinding and unblinding
// confidential data on the blockchain.
package confidentiality

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"

	"github.com/lib/pq"

	"chain-stealth/crypto/ca"
	"chain-stealth/database/pg"
	"chain-stealth/encoding/json"
	"chain-stealth/errors"
	"chain-stealth/log"
	"chain-stealth/protocol/bc"
	"chain-stealth/protocol/state"
)

type Storage struct {
	DB pg.DB
}

type Key struct {
	Key            ca.RecordKey
	ControlProgram []byte
}

func NewKey() (ca.RecordKey, error) {
	var rek [32]byte
	_, err := rand.Read(rek[:])
	return rek, err
}

// GetKeys looks up confidentiality keys for the provided control programs.
func (s *Storage) GetKeys(ctx context.Context, controlPrograms [][]byte) ([]*Key, error) {
	const q = `
		SELECT control_program, key FROM confidentiality_keys
		WHERE control_program IN (SELECT unnest($1::bytea[]))
	`

	var keys []*Key
	err := pg.ForQueryRows(ctx, s.DB, q, pq.ByteaArray(controlPrograms), func(cp, k []byte) {
		key := &Key{ControlProgram: cp}
		copy(key.Key[:], k)
		keys = append(keys, key)
	})
	if err != nil {
		return nil, errors.Wrap(err, "looking up confidentiality keys")
	}
	return keys, nil
}

// StoreKeys saves the provided confidentiality keys to persisent storage.
func (s *Storage) StoreKeys(ctx context.Context, keys []*Key) error {
	var (
		cps     = make([][]byte, 0, len(keys))
		rawKeys = make([][]byte, 0, len(keys))
	)
	for _, k := range keys {
		cps = append(cps, k.ControlProgram)
		rawKeys = append(rawKeys, k.Key[:])
	}

	const q = `
		INSERT INTO confidentiality_keys (control_program, key)
		SELECT unnest($1::bytea[]), unnest($2::bytea[])
		ON CONFLICT (control_program) DO UPDATE SET key = excluded.key;
	`
	_, err := s.DB.Exec(ctx, q, pq.ByteaArray(cps), pq.ByteaArray(rawKeys))
	return errors.Wrap(err, "inserting confidentiality keys")
}

// RecordIssuance records the record of a confidential issuance to
// persistent storage. It is used by the indexer to annotate amounts
// of confidential issuances without requiring decrypting the blinded
// issuance (which is not yet supported.
func (s *Storage) RecordIssuance(ctx context.Context, assetID bc.AssetID, nonce []byte, amount uint64) error {
	const q = `
		INSERT INTO confidential_issuances (asset_id, nonce, amount)
		VALUES($1, $2, $3)
	`
	_, err := s.DB.Exec(ctx, q, assetID, nonce, amount)
	return errors.Wrap(err, "inserting confidential issuance")
}

func (s *Storage) lookupIssuance(ctx context.Context, assetID bc.AssetID, nonce []byte) (amt uint64, ok bool, err error) {
	const q = `
		SELECT amount FROM confidential_issuances
		WHERE asset_id = $1 AND nonce = $2
	`
	err = s.DB.QueryRow(ctx, q, assetID, nonce).Scan(&amt)
	if err == sql.ErrNoRows {
		return 0, false, nil
	} else if err != nil {
		return 0, false, err
	}
	return amt, true, nil
}

// DecryptOutputs takes a slice outputs and returns a mapping from
// outpoint to decrypted asset amount. Unblinded outputs are ignored.
func (s *Storage) DecryptOutputs(ctx context.Context, outs []*state.Output) (map[bc.Outpoint]bc.AssetAmount, map[bc.Outpoint]*bc.CAValues, error) {
	// Collect the control programs of encrypted outputs.
	var controlPrograms [][]byte
	for _, out := range outs {
		_, ok := out.TypedOutput.GetAssetAmount()
		if !ok {
			controlPrograms = append(controlPrograms, out.Program())
		}
	}

	// Load the confidentiality keys corresponding to the control programs.
	keys, err := s.GetKeys(ctx, controlPrograms)
	if err != nil {
		return nil, nil, err
	}

	keysByControlProgram := make(map[string][32]byte, len(controlPrograms))
	for _, k := range keys {
		keysByControlProgram[hex.EncodeToString(k.ControlProgram)] = k.Key
	}

	// Build a map from outpoint to asset amount for encrypted outputs for
	// which we have a valid confidentiality key.
	amts := make(map[bc.Outpoint]bc.AssetAmount, len(keysByControlProgram))
	params := make(map[bc.Outpoint]*bc.CAValues, len(keysByControlProgram))
	for _, out := range outs {
		_, ok := out.GetAssetAmount()
		if ok {
			continue // unencrypted output
		}

		_, _, p, decryptedAA := decryptOutput(out.TypedOutput, keysByControlProgram)
		if decryptedAA == nil {
			continue // unreadable output
		}

		amts[out.Outpoint] = *decryptedAA
		params[out.Outpoint] = p
	}
	return amts, params, nil
}

// AnnotateTxs unblinds any transaction data that it can. The tx is annotated
// with the unblinded data. Additionally, it adds the following fields to the
// annotated txs inputs and outputs:
//
// * `confidential` — 'yes' if the entry is confidential and encrypted on the
//                    blockchain. 'no' otherwise.
// * `readable`     — 'yes' if the entry was not confidential or the Core was
//                    able to unblind the data using its stored confidentiality
//                    keys.
func (s *Storage) AnnotateTxs(ctx context.Context, annotatedTxs []map[string]interface{}, txs []*bc.Tx) error {
	var controlPrograms [][]byte
	for _, tx := range txs {
		for _, in := range tx.Inputs {
			if _, ok := in.AssetAmount(); ok {
				// If the data is already readable, skip it.
				continue
			}
			if in.IsIssuance() {
				continue
			}

			// We use the prevout control program to lookup the
			// confidentiality key.
			controlPrograms = append(controlPrograms, in.ControlProgram())
		}

		for _, out := range tx.Outputs {
			if _, ok := out.GetAssetAmount(); ok {
				// If the data is already readable, skip it.
				continue
			}

			controlPrograms = append(controlPrograms, out.Program())
		}
	}

	// Load the confidentiality keys corresponding to the control programs.
	keys, err := s.GetKeys(ctx, controlPrograms)
	if err != nil {
		return err
	}
	keysByControlProgram := map[string][32]byte{}
	for _, k := range keys {
		if len(k.ControlProgram) > 0 {
			keysByControlProgram[hex.EncodeToString(k.ControlProgram)] = k.Key
		}
	}

	for i, tx := range txs {
		annotatedTx := annotatedTxs[i]
		inputs := annotatedTx["inputs"].([]interface{})
		outputs := annotatedTx["outputs"].([]interface{})

		for j, in := range tx.Inputs {
			var aa *bc.AssetAmount
			var confidential, readable string
			var assetCommitment, valueCommitment []byte
			var possibleAssetIDs []bc.AssetID

			switch typedIn := in.TypedInput.(type) {
			case *bc.IssuanceInput2:
				assetCommitment = typedIn.AssetDescriptor().Commitment().Bytes()
				valueCommitment = typedIn.ValueDescriptor().Commitment().Bytes()

				for _, assetChoice := range typedIn.AssetChoices {
					possibleAssetIDs = append(possibleAssetIDs, assetChoice.AssetID(in.AssetVersion))
				}

				if len(typedIn.AssetChoices) != 1 {
					confidential, readable = "yes", "no"
					break
				}

				// TODO(jackson): Lookup issuances in a batch instead of one-at-a-time.
				assetID := typedIn.AssetChoices[0].AssetID(in.AssetVersion)
				amt, ok, err := s.lookupIssuance(ctx, assetID, typedIn.Nonce)
				if err != nil {
					confidential, readable = "yes", "no"
					log.Error(ctx, err)
					break
				}
				if !ok {
					confidential, readable = "yes", "no"
					break
				}

				confidential, readable = "yes", "yes"
				aa = &bc.AssetAmount{AssetID: assetID, Amount: amt}
			case *bc.SpendInput:
				// If the input is a spend input, the asset and amount might be
				// confidential because the prevout is confidential.
				confidential, readable, _, aa = decryptOutput(typedIn.TypedOutput, keysByControlProgram)

				if o2, ok := typedIn.TypedOutput.(*bc.Outputv2); ok {
					assetCommitment = o2.AssetDescriptor().Commitment().Bytes()
					valueCommitment = o2.ValueDescriptor().Commitment().Bytes()
				}
			default:
				// Other input types (like v1 issuance) are not confidential.
				confidential, readable = "no", "yes"
			}
			m := inputs[j].(map[string]interface{})
			m["confidential"] = confidential
			m["readable"] = readable
			m["asset_id_commitment"] = json.HexBytes(assetCommitment)
			m["asset_id_candidates"] = possibleAssetIDs
			m["amount_commitment"] = json.HexBytes(valueCommitment)
			if aa != nil {
				m["amount"] = aa.Amount
				m["asset_id"] = aa.AssetID.String()
			}
		}

		for j, out := range tx.Outputs {
			m := outputs[j].(map[string]interface{})
			confidential, rreadable, _, aa := decryptOutput(out.TypedOutput, keysByControlProgram)
			m["confidential"] = confidential
			m["readable"] = rreadable
			if o2, ok := out.TypedOutput.(*bc.Outputv2); ok {
				m["asset_id_commitment"] = json.HexBytes(o2.AssetDescriptor().Commitment().Bytes())
				m["amount_commitment"] = json.HexBytes(o2.ValueDescriptor().Commitment().Bytes())
			}
			if aa != nil {
				m["amount"] = aa.Amount
				m["asset_id"] = aa.AssetID.String()
			}
		}
	}

	return nil
}

func decryptOutput(out bc.TypedOutput, keys map[string][32]byte) (confidential, readable string, params *bc.CAValues, encryptedAA *bc.AssetAmount) {
	o2, ok := out.(*bc.Outputv2)
	if !ok {
		// If it's not an output v2, then it's already readable.
		return "no", "yes", nil, nil
	}

	vd, ad := o2.ValueDescriptor(), o2.AssetDescriptor()
	if !vd.IsBlinded() && !ad.IsBlinded() {
		// The output is v2 but not blinded.
		return "no", "yes", nil, nil
	}

	key, ok := keys[hex.EncodeToString(o2.ControlProgram)]
	if !ok {
		// We don't have the confidentiality key to decrypt this output.
		return "yes", "no", nil, nil
	}

	caassetID, amount, c, f, _, err := ca.DecryptOutput(key, ad, vd, o2.ValueRangeProof())
	if err != nil {
		// We're unable to decrypt this output. Maybe the confidentiality
		// key is incorrect?
		fmt.Printf("failed to decrypt using key: %#v", key)
		return "yes", "no", nil, nil
	}

	ac, vc := ad.Commitment(), vd.Commitment()
	output := &bc.CAValues{
		Value:                    amount,
		AssetCommitment:          ac,
		CumulativeBlindingFactor: c,
		ValueCommitment:          vc,
		ValueBlindingFactor:      f,
	}
	aa := bc.AssetAmount{
		AssetID: bc.AssetID(caassetID),
		Amount:  amount,
	}
	return "yes", "yes", output, &aa
}

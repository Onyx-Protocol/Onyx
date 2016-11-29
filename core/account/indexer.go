package account

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/lib/pq"

	"chain-stealth/core/account/internal/storage"
	"chain-stealth/core/signers"
	"chain-stealth/database/pg"
	"chain-stealth/encoding/json"
	"chain-stealth/errors"
	"chain-stealth/log"
	"chain-stealth/protocol/bc"
	"chain-stealth/protocol/state"
)

// PinName is used to identify the pin associated with
// the account block processor.
const PinName = "account"

// A Saver is responsible for saving an annotated account object.
// for indexing and retrieval.
// If the Core is configured not to provide search services,
// SaveAnnotatedAccount can be a no-op.
type Saver interface {
	SaveAnnotatedAccount(context.Context, string, map[string]interface{}) error
}

func (m *Manager) indexAnnotatedAccount(ctx context.Context, a *Account) error {
	if m.indexer == nil {
		return nil
	}
	var keys []map[string]interface{}
	path := signers.Path(a.Signer, signers.AccountKeySpace)
	var jsonPath []json.HexBytes
	for _, p := range path {
		jsonPath = append(jsonPath, p)
	}
	for _, xpub := range a.XPubs {
		keys = append(keys, map[string]interface{}{
			"root_xpub":               xpub,
			"account_xpub":            xpub.Derive(path),
			"account_derivation_path": jsonPath,
		})
	}
	return m.indexer.SaveAnnotatedAccount(ctx, a.ID, map[string]interface{}{
		"id":     a.ID,
		"alias":  a.Alias,
		"keys":   keys,
		"tags":   a.Tags,
		"quorum": a.Quorum,
	})
}

type output struct {
	state.Output
	AccountID string
	keyIndex  uint64
}

func (m *Manager) ProcessBlocks(ctx context.Context) {
	if m.pinStore == nil {
		return
	}
	m.pinStore.ProcessBlocks(ctx, m.chain, PinName, m.indexAccountUTXOs)
}

func (m *Manager) indexAccountUTXOs(ctx context.Context, b *bc.Block) error {
	// Upsert any UTXOs belonging to accounts managed by this Core.
	outs := make([]*state.Output, 0, len(b.Transactions))
	blockPositions := make(map[bc.Hash]uint32, len(b.Transactions))
	for i, tx := range b.Transactions {
		blockPositions[tx.Hash] = uint32(i)
		for j, out := range tx.Outputs {
			stateOutput := &state.Output{
				TypedOutput: out.TypedOutput,
				Outpoint:    bc.Outpoint{Hash: tx.Hash, Index: uint32(j)},
			}
			outs = append(outs, stateOutput)
		}
	}
	accOuts, accStateOuts, err := m.loadAccountInfo(ctx, outs)
	if err != nil {
		return errors.Wrap(err, "loading account info from control programs")
	}

	decryptedAmts, caParams, err := m.confidentiality.DecryptOutputs(ctx, accStateOuts)
	if err != nil {
		return errors.Wrap(err, "decrypting confidential output amounts")
	}

	err = m.upsertConfirmedAccountOutputs(ctx, accOuts, decryptedAmts, caParams, blockPositions, b)
	if err != nil {
		return errors.Wrap(err, "upserting confirmed account utxos")
	}

	// Delete consumed account UTXOs.
	deltxhash, delindex := prevoutDBKeys(b.Transactions...)
	const delQ = `
		DELETE FROM account_utxos
		WHERE (tx_hash, index) IN (SELECT unnest($1::text[]), unnest($2::integer[]))
	`
	_, err = m.db.Exec(ctx, delQ, deltxhash, delindex)
	return errors.Wrap(err, "deleting spent account utxos")
}

func prevoutDBKeys(txs ...*bc.Tx) (txhash pq.StringArray, index pg.Uint32s) {
	for _, tx := range txs {
		for _, in := range tx.Inputs {
			o, ok := in.Outpoint()
			if !ok {
				continue
			}
			txhash = append(txhash, o.Hash.String())
			index = append(index, o.Index)
		}
	}
	return
}

// loadAccountInfo turns a set of state.Outputs into a set of
// outputs by adding account annotations.  Outputs that can't be
// annotated are excluded from the result.
func (m *Manager) loadAccountInfo(ctx context.Context, outs []*state.Output) ([]*output, []*state.Output, error) {
	outsByScript := make(map[string][]*state.Output, len(outs))
	for _, out := range outs {
		scriptStr := string(out.Program())
		outsByScript[scriptStr] = append(outsByScript[scriptStr], out)
	}

	var scripts pq.ByteaArray
	for s := range outsByScript {
		scripts = append(scripts, []byte(s))
	}

	result := make([]*output, 0, len(outs))
	stateOuts := make([]*state.Output, 0, len(outs))

	const q = `
		SELECT signer_id, key_index, control_program
		FROM account_control_programs
		WHERE control_program IN (SELECT unnest($1::bytea[]))
	`
	err := pg.ForQueryRows(ctx, m.db, q, scripts, func(accountID string, keyIndex uint64, program []byte) {
		for _, out := range outsByScript[string(program)] {
			newOut := &output{
				Output:    *out,
				AccountID: accountID,
				keyIndex:  keyIndex,
			}
			result = append(result, newOut)
			stateOuts = append(stateOuts, out)
		}
	})
	if err != nil {
		return nil, nil, err
	}

	return result, stateOuts, nil
}

// upsertConfirmedAccountOutputs records the account data for confirmed utxos.
// If the account utxo already exists (because it's from a local tx), the
// block confirmation data will in the row will be updated.
func (m *Manager) upsertConfirmedAccountOutputs(
	ctx context.Context,
	outs []*output,
	decryptedAAs map[bc.Outpoint]bc.AssetAmount,
	caValues map[bc.Outpoint]*bc.CAValues,
	pos map[bc.Hash]uint32,
	block *bc.Block,
) error {
	var (
		txHash      pq.StringArray
		index       pg.Uint32s
		assetID     pq.StringArray
		amount      pq.Int64Array
		accountID   pq.StringArray
		cpIndex     pq.Int64Array
		program     pq.ByteaArray
		rawOuts     pq.ByteaArray
		rawCAParams pq.ByteaArray
	)
	for _, out := range outs {
		aa, ok := out.GetAssetAmount()

		if !ok {
			// Maybe the output is just encrypted?
			aa, ok = decryptedAAs[out.Outpoint]
		}

		if !ok {
			// This output was sent to a control program managed by this Core,
			// but we're unable to decrypt the details!
			log.Messagef(ctx, "output sent to %s control program but unable to decrypt it", out.AccountID)
			continue
		}

		var err error
		var encodedParams []byte
		if vals, ok := caValues[out.Outpoint]; ok {
			encodedParams, err = proto.Marshal(&storage.ConfidentialAssetParams{
				Value:                    vals.Value,
				AssetCommitment:          vals.AssetCommitment.Bytes(),
				CumulativeBlindingFactor: vals.CumulativeBlindingFactor[:],
				ValueCommitment:          vals.ValueCommitment.Bytes(),
				ValueBlindingFactor:      vals.ValueBlindingFactor[:],
			})
			if err != nil {
				return err
			}
		}

		raw := state.OutputBytes(&out.Output)

		txHash = append(txHash, out.Outpoint.Hash.String())
		index = append(index, out.Outpoint.Index)
		assetID = append(assetID, aa.AssetID.String())
		amount = append(amount, int64(aa.Amount)) // TODO(bobg): range-check aa.Amount before casting to int64
		accountID = append(accountID, out.AccountID)
		cpIndex = append(cpIndex, int64(out.keyIndex))
		program = append(program, out.Program())
		rawOuts = append(rawOuts, raw)
		rawCAParams = append(rawCAParams, encodedParams)
	}

	const q = `
		INSERT INTO account_utxos (tx_hash, index, asset_id, amount, account_id, control_program_index,
			control_program, confirmed_in, raw_output, ca_params)
		SELECT unnest($1::text[]), unnest($2::bigint[]), unnest($3::text[]),  unnest($4::bigint[]),
		unnest($5::text[]), unnest($6::bigint[]), unnest($7::bytea[]), $8, unnest($9::bytea[]), unnest($10::bytea[])
		ON CONFLICT (tx_hash, index) DO NOTHING
	`
	_, err := m.db.Exec(ctx, q,
		txHash,
		index,
		assetID,
		amount,
		accountID,
		cpIndex,
		program,
		block.Height,
		rawOuts,
		rawCAParams,
	)
	return errors.Wrap(err)
}

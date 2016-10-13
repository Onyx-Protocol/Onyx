package account

import (
	"context"

	"github.com/lib/pq"

	"chain/core/signers"
	"chain/database/pg"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/state"
)

const (
	// unconfirmedExpiration configures when an unconfirmed UTXO must
	// be confirmed in a block before it expires and is deleted. If a
	// UTXO is deleted but later confirmed, it'll be re-inserted.
	unconfirmedExpiration = 5
)

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
	for _, xpub := range a.XPubs {
		keys = append(keys, map[string]interface{}{
			"root_xpub":               xpub,
			"account_xpub":            xpub.Derive(path),
			"account_derivation_path": path,
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

// IndexUnconfirmedUTXOs looks up a transaction's control programs for matching
// account control programs. If any control programs match, the unconfirmed
// UTXOs are inserted into account_utxos with an expiry_height. If not confirmed
// by the expiry_height, the UTXOs will be deleted (and assumed rejected).
func (m *Manager) IndexUnconfirmedUTXOs(ctx context.Context, tx *bc.Tx) error {
	stateOuts := make([]*state.Output, 0, len(tx.Outputs))
	for i, out := range tx.Outputs {
		stateOutput := &state.Output{
			TxOutput: *out,
			Outpoint: bc.Outpoint{Hash: tx.Hash, Index: uint32(i)},
		}
		stateOuts = append(stateOuts, stateOutput)
	}
	accOuts, err := m.loadAccountInfo(ctx, stateOuts)
	if err != nil {
		return errors.Wrap(err, "loading account info")
	}
	err = m.upsertUnconfirmedAccountOutputs(ctx, accOuts, m.chain.Height()+unconfirmedExpiration)
	return errors.Wrap(err, "upserting confirmed account utxos")
}

func (m *Manager) indexAccountUTXOs(ctx context.Context, b *bc.Block) error {
	// Upsert any UTXOs belonging to accounts managed by this Core.
	outs := make([]*state.Output, 0, len(b.Transactions))
	blockPositions := make(map[bc.Hash]uint32, len(b.Transactions))
	for i, tx := range b.Transactions {
		blockPositions[tx.Hash] = uint32(i)
		for j, out := range tx.Outputs {
			stateOutput := &state.Output{
				TxOutput: *out,
				Outpoint: bc.Outpoint{Hash: tx.Hash, Index: uint32(j)},
			}
			outs = append(outs, stateOutput)
		}
	}
	accOuts, err := m.loadAccountInfo(ctx, outs)
	if err != nil {
		return errors.Wrap(err, "loading account info from control programs")
	}
	err = m.upsertConfirmedAccountOutputs(ctx, accOuts, blockPositions, b)
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
	if err != nil {
		return errors.Wrap(err, "deleting spent account utxos")
	}

	// Delete any unconfirmed account UTXOs that are now expired because they
	// have not been confirmed after several blocks.
	const expiryQ = `
		DELETE FROM account_utxos WHERE expiry_height <= $1 AND confirmed_in IS NULL
	`
	_, err = m.db.Exec(ctx, expiryQ, b.Height)
	return errors.Wrap(err, "deleting expired account utxos")
}

func prevoutDBKeys(txs ...*bc.Tx) (txhash pq.StringArray, index pg.Uint32s) {
	for _, tx := range txs {
		for _, in := range tx.Inputs {
			if in.IsIssuance() {
				continue
			}
			o := in.Outpoint()
			txhash = append(txhash, o.Hash.String())
			index = append(index, o.Index)
		}
	}
	return
}

// loadAccountInfo turns a set of state.Outputs into a set of
// outputs by adding account annotations.  Outputs that can't be
// annotated are excluded from the result.
func (m *Manager) loadAccountInfo(ctx context.Context, outs []*state.Output) ([]*output, error) {
	outsByScript := make(map[string][]*state.Output, len(outs))
	for _, out := range outs {
		scriptStr := string(out.ControlProgram)
		outsByScript[scriptStr] = append(outsByScript[scriptStr], out)
	}

	var scripts pq.ByteaArray
	for s := range outsByScript {
		scripts = append(scripts, []byte(s))
	}

	result := make([]*output, 0, len(outs))

	const q = `
		SELECT signer_id, key_index, control_program
		FROM account_control_programs
		WHERE control_program IN (SELECT unnest($1::bytea[]))
	`
	dbctx := pg.NewContext(ctx, m.db)
	err := pg.ForQueryRows(dbctx, q, scripts, func(accountID string, keyIndex uint64, program []byte) {
		for _, out := range outsByScript[string(program)] {
			newOut := &output{
				Output:    *out,
				AccountID: accountID,
				keyIndex:  keyIndex,
			}
			result = append(result, newOut)
		}
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// upsertUnconfirmedAccountOutputs records the account data for unconfirmed
// account utxos.
func (m *Manager) upsertUnconfirmedAccountOutputs(ctx context.Context, outs []*output, expiryHeight uint64) error {
	var (
		txHash    pq.StringArray
		index     pg.Uint32s
		assetID   pq.StringArray
		amount    pq.Int64Array
		accountID pq.StringArray
		cpIndex   pq.Int64Array
		program   pq.ByteaArray
		metadata  pq.ByteaArray
	)
	for _, out := range outs {
		txHash = append(txHash, out.Outpoint.Hash.String())
		index = append(index, out.Outpoint.Index)
		assetID = append(assetID, out.AssetID.String())
		amount = append(amount, int64(out.Amount))
		accountID = append(accountID, out.AccountID)
		cpIndex = append(cpIndex, int64(out.keyIndex))
		program = append(program, out.ControlProgram)
		metadata = append(metadata, out.ReferenceData)
	}

	const q = `
		INSERT INTO account_utxos (tx_hash, index, asset_id, amount, account_id, control_program_index,
			control_program, metadata, expiry_height)
		SELECT unnest($1::text[]), unnest($2::bigint[]), unnest($3::text[]),  unnest($4::bigint[]),
			   unnest($5::text[]), unnest($6::bigint[]), unnest($7::bytea[]), unnest($8::bytea[]), $9
		ON CONFLICT (tx_hash, index) DO NOTHING;
	`
	_, err := m.db.Exec(ctx, q,
		txHash,
		index,
		assetID,
		amount,
		accountID,
		cpIndex,
		program,
		metadata,
		expiryHeight,
	)
	return errors.Wrap(err)
}

// upsertConfirmedAccountOutputs records the account data for confirmed utxos.
// If the account utxo already exists (because it's from a local tx), the
// block confirmation data will in the row will be updated.
func (m *Manager) upsertConfirmedAccountOutputs(ctx context.Context, outs []*output, pos map[bc.Hash]uint32, block *bc.Block) error {
	var (
		txHash    pq.StringArray
		index     pg.Uint32s
		assetID   pq.StringArray
		amount    pq.Int64Array
		accountID pq.StringArray
		cpIndex   pq.Int64Array
		program   pq.ByteaArray
		metadata  pq.ByteaArray
		blockPos  pg.Uint32s
	)
	for _, out := range outs {
		txHash = append(txHash, out.Outpoint.Hash.String())
		index = append(index, out.Outpoint.Index)
		assetID = append(assetID, out.AssetID.String())
		amount = append(amount, int64(out.Amount))
		accountID = append(accountID, out.AccountID)
		cpIndex = append(cpIndex, int64(out.keyIndex))
		program = append(program, out.ControlProgram)
		metadata = append(metadata, out.ReferenceData)
		blockPos = append(blockPos, pos[out.Outpoint.Hash])
	}

	const q = `
		INSERT INTO account_utxos (tx_hash, index, asset_id, amount, account_id, control_program_index,
			control_program, metadata, confirmed_in, block_pos, block_timestamp, expiry_height)
		SELECT unnest($1::text[]), unnest($2::bigint[]), unnest($3::text[]),  unnest($4::bigint[]),
			   unnest($5::text[]), unnest($6::bigint[]), unnest($7::bytea[]), unnest($8::bytea[]),
			   $9, unnest($10::bigint[]), $11, NULL
		ON CONFLICT (tx_hash, index) DO UPDATE SET
			confirmed_in    = excluded.confirmed_in,
			block_pos       = excluded.block_pos,
			block_timestamp = excluded.block_timestamp,
			expiry_height   = excluded.expiry_height;
	`
	_, err := m.db.Exec(ctx, q,
		txHash,
		index,
		assetID,
		amount,
		accountID,
		cpIndex,
		program,
		metadata,
		block.Height,
		blockPos,
		block.TimestampMS,
	)
	return errors.Wrap(err)
}

func toKeyIndex(i []uint32) int64 {
	return int64(i[0])<<31 | int64(i[1]&0x7fffffff)
}

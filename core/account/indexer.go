package account

import (
	"context"
	"encoding/json"

	"github.com/lib/pq"

	"chain/core/query"
	"chain/core/signers"
	"chain/database/pg"
	chainjson "chain/encoding/json"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/bc/legacy"
)

const (
	// PinName is used to identify the pin associated with
	// the account indexer block processor.
	PinName = "account"
	// ExpirePinName is used to identify the pin associated
	// with the account control program expiration processor.
	ExpirePinName = "expire-control-programs"
	// DeleteSpentsPinName is used to identify the pin associated
	// with the processor that deletes spent account UTXOs.
	DeleteSpentsPinName = "delete-account-spents"
)

var emptyJSONObject = json.RawMessage(`{}`)

// A Saver is responsible for saving an annotated account object.
// for indexing and retrieval.
// If the Core is configured not to provide search services,
// SaveAnnotatedAccount can be a no-op.
type Saver interface {
	SaveAnnotatedAccount(context.Context, *query.AnnotatedAccount) error
}

func Annotated(a *Account) (*query.AnnotatedAccount, error) {
	aa := &query.AnnotatedAccount{
		ID:     a.ID,
		Alias:  a.Alias,
		Quorum: a.Quorum,
		Tags:   &emptyJSONObject,
	}

	tags, err := json.Marshal(a.Tags)
	if err != nil {
		return nil, err
	}
	if len(tags) > 0 {
		rawTags := json.RawMessage(tags)
		aa.Tags = &rawTags
	}

	path := signers.Path(a.Signer, signers.AccountKeySpace)
	var jsonPath []chainjson.HexBytes
	for _, p := range path {
		jsonPath = append(jsonPath, p)
	}
	for _, xpub := range a.XPubs {
		aa.Keys = append(aa.Keys, &query.AccountKey{
			RootXPub:              xpub,
			AccountXPub:           xpub.Derive(path),
			AccountDerivationPath: jsonPath,
		})
	}
	return aa, nil
}

func (m *Manager) indexAnnotatedAccount(ctx context.Context, a *Account) error {
	if m.indexer == nil {
		return nil
	}
	aa, err := Annotated(a)
	if err != nil {
		return err
	}
	return m.indexer.SaveAnnotatedAccount(ctx, aa)
}

type rawOutput struct {
	OutputID bc.Hash
	bc.AssetAmount
	ControlProgram []byte
	txHash         bc.Hash
	outputIndex    uint32
	sourceID       bc.Hash
	sourcePos      uint64
	refData        bc.Hash
}

type accountOutput struct {
	rawOutput
	AccountID string
	keyIndex  uint64
	change    bool
}

func (m *Manager) ProcessBlocks(ctx context.Context) {
	if m.pinStore == nil {
		return
	}
	go m.pinStore.ProcessBlocks(ctx, m.chain, ExpirePinName, func(ctx context.Context, b *legacy.Block) error {
		<-m.pinStore.PinWaiter(PinName, b.Height)
		return m.expireControlPrograms(ctx, b)
	})
	go m.pinStore.ProcessBlocks(ctx, m.chain, DeleteSpentsPinName, func(ctx context.Context, b *legacy.Block) error {
		<-m.pinStore.PinWaiter(PinName, b.Height)
		<-m.pinStore.PinWaiter(query.TxPinName, b.Height)
		return m.deleteSpentOutputs(ctx, b)
	})
	m.pinStore.ProcessBlocks(ctx, m.chain, PinName, m.indexAccountUTXOs)
}

func (m *Manager) expireControlPrograms(ctx context.Context, b *legacy.Block) error {
	// Delete expired account control programs.
	const deleteQ = `DELETE FROM account_control_programs WHERE expires_at IS NOT NULL AND expires_at < $1`
	_, err := m.db.ExecContext(ctx, deleteQ, b.Time())
	return err
}

func (m *Manager) deleteSpentOutputs(ctx context.Context, b *legacy.Block) error {
	// Delete consumed account UTXOs.
	delOutputIDs := prevoutDBKeys(b.Transactions...)
	const delQ = `
		DELETE FROM account_utxos
		WHERE output_id IN (SELECT unnest($1::bytea[]))
	`
	_, err := m.db.ExecContext(ctx, delQ, delOutputIDs)
	return errors.Wrap(err, "deleting spent account utxos")
}

func (m *Manager) indexAccountUTXOs(ctx context.Context, b *legacy.Block) error {
	// Upsert any UTXOs belonging to accounts managed by this Core.
	outs := make([]*rawOutput, 0, len(b.Transactions))
	blockPositions := make(map[bc.Hash]uint32, len(b.Transactions))
	for i, tx := range b.Transactions {
		blockPositions[tx.ID] = uint32(i)
		for j, out := range tx.Outputs {
			resOutID := tx.ResultIds[j]
			resOut, ok := tx.Entries[*resOutID].(*bc.Output)
			if !ok {
				continue
			}
			out := &rawOutput{
				OutputID:       *tx.OutputID(j),
				AssetAmount:    out.AssetAmount,
				ControlProgram: out.ControlProgram,
				txHash:         tx.ID,
				outputIndex:    uint32(j),
				sourceID:       *resOut.Source.Ref,
				sourcePos:      resOut.Source.Position,
				refData:        *resOut.Data,
			}
			outs = append(outs, out)
		}
	}
	accOuts, err := m.loadAccountInfo(ctx, outs)
	if err != nil {
		return errors.Wrap(err, "loading account info from control programs")
	}

	err = m.upsertConfirmedAccountOutputs(ctx, accOuts, blockPositions, b)
	return errors.Wrap(err, "upserting confirmed account utxos")
}

func prevoutDBKeys(txs ...*legacy.Tx) (outputIDs pq.ByteaArray) {
	for _, tx := range txs {
		for _, inpID := range tx.Tx.InputIDs {
			if sp, err := tx.Spend(inpID); err == nil {
				outputIDs = append(outputIDs, sp.SpentOutputId.Bytes())
			}
		}
	}
	return
}

// loadAccountInfo turns a set of output IDs into a set of
// outputs by adding account annotations.  Outputs that can't be
// annotated are excluded from the result.
func (m *Manager) loadAccountInfo(ctx context.Context, outs []*rawOutput) ([]*accountOutput, error) {
	outsByScript := make(map[string][]*rawOutput, len(outs))
	for _, out := range outs {
		scriptStr := string(out.ControlProgram)
		outsByScript[scriptStr] = append(outsByScript[scriptStr], out)
	}

	var scripts pq.ByteaArray
	for s := range outsByScript {
		scripts = append(scripts, []byte(s))
	}

	result := make([]*accountOutput, 0, len(outs))

	const q = `
		SELECT signer_id, key_index, control_program, change
		FROM account_control_programs
		WHERE control_program IN (SELECT unnest($1::bytea[]))
	`
	err := pg.ForQueryRows(ctx, m.db, q, scripts, func(accountID string, keyIndex uint64, program []byte, change bool) {
		for _, out := range outsByScript[string(program)] {
			newOut := &accountOutput{
				rawOutput: *out,
				AccountID: accountID,
				keyIndex:  keyIndex,
				change:    change,
			}
			result = append(result, newOut)
		}
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// upsertConfirmedAccountOutputs records the account data for confirmed utxos.
// If the account utxo already exists (because it's from a local tx), the
// block confirmation data will in the row will be updated.
func (m *Manager) upsertConfirmedAccountOutputs(ctx context.Context, outs []*accountOutput, pos map[bc.Hash]uint32, block *legacy.Block) error {
	var (
		outputID  pq.ByteaArray
		assetID   pq.ByteaArray
		amount    pq.Int64Array
		accountID pq.StringArray
		cpIndex   pq.Int64Array
		program   pq.ByteaArray
		sourceID  pq.ByteaArray
		sourcePos pq.Int64Array
		refData   pq.ByteaArray
		change    pq.BoolArray
	)
	for _, out := range outs {
		outputID = append(outputID, out.OutputID.Bytes())
		assetID = append(assetID, out.AssetId.Bytes())
		amount = append(amount, int64(out.Amount))
		accountID = append(accountID, out.AccountID)
		cpIndex = append(cpIndex, int64(out.keyIndex))
		program = append(program, out.ControlProgram)
		sourceID = append(sourceID, out.sourceID.Bytes())
		sourcePos = append(sourcePos, int64(out.sourcePos))
		refData = append(refData, out.refData.Bytes())
		change = append(change, out.change)
	}

	const q = `
		INSERT INTO account_utxos (output_id, asset_id, amount, account_id, control_program_index,
			control_program, confirmed_in, source_id, source_pos, ref_data_hash, change)
		SELECT unnest($1::bytea[]), unnest($2::bytea[]),  unnest($3::bigint[]),
			   unnest($4::text[]), unnest($5::bigint[]), unnest($6::bytea[]), $7,
			   unnest($8::bytea[]), unnest($9::bigint[]), unnest($10::bytea[]), unnest($11::boolean[])
		ON CONFLICT (output_id) DO NOTHING
	`
	_, err := m.db.ExecContext(ctx, q,
		outputID,
		assetID,
		amount,
		accountID,
		cpIndex,
		program,
		block.Height,
		sourceID,
		sourcePos,
		refData,
		change,
	)
	return errors.Wrap(err)
}

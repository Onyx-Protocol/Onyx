package query

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/lib/pq"

	"chain/core/pin"
	"chain/database/pg"
	"chain/errors"
	"chain/protocol"
	"chain/protocol/bc/legacy"
)

const (
	// TxPinName is used to identify the pin associated
	// with the transaction block processor.
	TxPinName = "tx"
)

// NewIndexer constructs a new indexer for indexing transactions.
func NewIndexer(db pg.DB, c *protocol.Chain, pinStore *pin.Store) *Indexer {
	indexer := &Indexer{
		db:       db,
		c:        c,
		pinStore: pinStore,
	}
	return indexer
}

// Indexer creates, updates and queries against indexes.
type Indexer struct {
	db         pg.DB
	c          *protocol.Chain
	pinStore   *pin.Store
	annotators []Annotator
}

// Annotator describes a function capable of adding annotations
// to transactions, inputs and outputs.
type Annotator func(ctx context.Context, txs []*AnnotatedTx) error

// RegisterAnnotator adds an additional annotator capable of mutating
// the annotated transaction object.
func (ind *Indexer) RegisterAnnotator(annotator Annotator) {
	ind.annotators = append(ind.annotators, annotator)
}

func (ind *Indexer) ProcessBlocks(ctx context.Context) {
	if ind.pinStore == nil {
		return
	}
	ind.pinStore.ProcessBlocks(ctx, ind.c, TxPinName, ind.IndexTransactions)
}

// IndexTransactions is registered as a block callback on the Chain. It
// saves all annotated transactions to the database.
func (ind *Indexer) IndexTransactions(ctx context.Context, b *legacy.Block) error {
	<-ind.pinStore.PinWaiter("asset", b.Height)
	<-ind.pinStore.PinWaiter("account", b.Height)
	<-ind.pinStore.PinWaiter(TxPinName, b.Height-1)

	err := ind.insertBlock(ctx, b)
	if err != nil {
		return err
	}
	txs, err := ind.insertAnnotatedTxs(ctx, b)
	if err != nil {
		return err
	}
	err = ind.insertAnnotatedOutputs(ctx, b, txs)
	if err != nil {
		return err
	}
	err = ind.insertAnnotatedInputs(ctx, b, txs)
	return err
}

func (ind *Indexer) insertBlock(ctx context.Context, b *legacy.Block) error {
	const q = `
		INSERT INTO query_blocks (height, timestamp) VALUES($1, $2)
		ON CONFLICT (height) DO NOTHING
	`
	_, err := ind.db.ExecContext(ctx, q, b.Height, b.TimestampMS)
	return errors.Wrap(err, "inserting block timestamp")
}

func (ind *Indexer) insertAnnotatedTxs(ctx context.Context, b *legacy.Block) ([]*AnnotatedTx, error) {
	var (
		hashes           = pq.ByteaArray(make([][]byte, 0, len(b.Transactions)))
		positions        = make([]uint32, 0, len(b.Transactions))
		annotatedTxBlobs = pq.StringArray(make([]string, 0, len(b.Transactions)))
		annotatedTxs     = make([]*AnnotatedTx, 0, len(b.Transactions))
		locals           = pq.BoolArray(make([]bool, 0, len(b.Transactions)))
		referenceDatas   = pq.StringArray(make([]string, 0, len(b.Transactions)))
	)

	// Build the fully annotated transactions.
	for pos, tx := range b.Transactions {
		annotatedTxs = append(annotatedTxs, buildAnnotatedTransaction(tx, b, uint32(pos)))
	}
	for _, annotator := range ind.annotators {
		err := annotator(ctx, annotatedTxs)
		if err != nil {
			return nil, errors.Wrap(err, "adding external annotations")
		}
	}
	localAnnotator(ctx, annotatedTxs)

	// Collect the fields we need to commit to the DB.
	for pos, tx := range annotatedTxs {
		b, err := json.Marshal(tx)
		if err != nil {
			return nil, err
		}
		annotatedTxBlobs = append(annotatedTxBlobs, string(b))
		hashes = append(hashes, tx.ID.Bytes())
		positions = append(positions, uint32(pos))
		locals = append(locals, bool(tx.IsLocal))
		referenceDatas = append(referenceDatas, string(*tx.ReferenceData))
	}

	// Save the annotated txs to the database.
	const insertQ = `
		INSERT INTO annotated_txs(block_height, block_id, timestamp,
			tx_pos, tx_hash, data, local, reference_data, block_tx_count)
		SELECT $1, $2, $3, unnest($4::integer[]), unnest($5::bytea[]),
			unnest($6::jsonb[]), unnest($7::boolean[]), unnest($8::jsonb[]), $9
		ON CONFLICT (block_height, tx_pos) DO NOTHING;
	`
	_, err := ind.db.ExecContext(ctx, insertQ, b.Height, b.Hash(), b.Time(),
		pq.Array(positions), hashes, annotatedTxBlobs, locals,
		referenceDatas, len(b.Transactions))
	if err != nil {
		return nil, errors.Wrap(err, "inserting annotated_txs to db")
	}
	return annotatedTxs, nil
}

func (ind *Indexer) insertAnnotatedInputs(ctx context.Context, b *legacy.Block, annotatedTxs []*AnnotatedTx) error {
	var (
		inputTxHashes         pq.ByteaArray
		inputIndexes          pq.Int64Array
		inputTypes            pq.StringArray
		inputAssetIDs         pq.ByteaArray
		inputAssetAliases     pq.StringArray
		inputAssetDefinitions pq.StringArray
		inputAssetTags        pq.StringArray
		inputAssetLocals      pq.BoolArray
		inputAmounts          pq.Int64Array
		inputAccountIDs       []sql.NullString
		inputAccountAliases   []sql.NullString
		inputAccountTags      []sql.NullString
		inputIssuancePrograms pq.ByteaArray
		inputReferenceDatas   pq.StringArray
		inputLocals           pq.BoolArray
		inputSpentOutputIDs   pq.ByteaArray
	)

	for _, annotatedTx := range annotatedTxs {
		for i, in := range annotatedTx.Inputs {
			inputTxHashes = append(inputTxHashes, annotatedTx.ID.Bytes())
			inputIndexes = append(inputIndexes, int64(i))
			inputTypes = append(inputTypes, in.Type)
			inputAssetIDs = append(inputAssetIDs, in.AssetID.Bytes())
			inputAssetAliases = append(inputAssetAliases, in.AssetAlias)
			inputAssetDefinitions = append(inputAssetDefinitions, string(*in.AssetDefinition))
			inputAssetTags = append(inputAssetTags, string(*in.AssetTags))
			inputAssetLocals = append(inputAssetLocals, bool(in.AssetIsLocal))
			inputAmounts = append(inputAmounts, int64(in.Amount))
			inputAccountIDs = append(inputAccountIDs, sql.NullString{String: in.AccountID, Valid: in.AccountID != ""})
			inputAccountAliases = append(inputAccountAliases, sql.NullString{String: in.AccountAlias, Valid: in.AccountAlias != ""})
			if in.AccountTags != nil {
				inputAccountTags = append(inputAccountTags, sql.NullString{String: string(*in.AccountTags), Valid: true})
			} else {
				inputAccountTags = append(inputAccountTags, sql.NullString{})
			}
			inputIssuancePrograms = append(inputIssuancePrograms, in.IssuanceProgram)
			inputReferenceDatas = append(inputReferenceDatas, string(*in.ReferenceData))
			inputLocals = append(inputLocals, bool(in.IsLocal))
			if in.SpentOutputID != nil {
				inputSpentOutputIDs = append(inputSpentOutputIDs, in.SpentOutputID.Bytes())
			} else {
				inputSpentOutputIDs = append(inputSpentOutputIDs, nil)
			}
		}
	}
	const insertQ = `
		INSERT INTO annotated_inputs (tx_hash, index, type,
			asset_id, asset_alias, asset_definition, asset_tags, asset_local,
			amount, account_id, account_alias, account_tags, issuance_program,
			reference_data, local, spent_output_id)
		SELECT unnest($1::bytea[]), unnest($2::integer[]), unnest($3::text[]), unnest($4::bytea[]),
		unnest($5::text[]), unnest($6::jsonb[]), unnest($7::jsonb[]), unnest($8::boolean[]),
		unnest($9::bigint[]), unnest($10::text[]), unnest($11::text[]), unnest($12::jsonb[]),
		unnest($13::bytea[]), unnest($14::jsonb[]), unnest($15::boolean[]), unnest($16::bytea[])
		ON CONFLICT (tx_hash, index) DO NOTHING;
	`
	_, err := ind.db.ExecContext(ctx, insertQ, inputTxHashes, inputIndexes, inputTypes, inputAssetIDs,
		inputAssetAliases, inputAssetDefinitions, pq.Array(inputAssetTags), inputAssetLocals,
		inputAmounts, pq.Array(inputAccountIDs), pq.Array(inputAccountAliases), pq.Array(inputAccountTags),
		inputIssuancePrograms, inputReferenceDatas, inputLocals, inputSpentOutputIDs)
	return errors.Wrap(err, "batch inserting annotated inputs")
}

func (ind *Indexer) insertAnnotatedOutputs(ctx context.Context, b *legacy.Block, annotatedTxs []*AnnotatedTx) error {
	var (
		outputIDs              pq.ByteaArray
		outputTxPositions      []uint32
		outputIndexes          []uint32
		outputTxHashes         pq.ByteaArray
		outputTypes            pq.StringArray
		outputPurposes         pq.StringArray
		outputAssetIDs         pq.ByteaArray
		outputAssetAliases     pq.StringArray
		outputAssetDefinitions pq.StringArray
		outputAssetTags        pq.StringArray
		outputAssetLocals      pq.BoolArray
		outputAmounts          pq.Int64Array
		outputAccountIDs       []sql.NullString
		outputAccountAliases   []sql.NullString
		outputAccountTags      []sql.NullString
		outputControlPrograms  pq.ByteaArray
		outputReferenceDatas   pq.StringArray
		outputLocals           pq.BoolArray
		prevoutIDs             pq.ByteaArray
	)
	for pos, tx := range b.Transactions {
		for _, inpID := range tx.Tx.InputIDs {
			if sp, err := tx.Spend(inpID); err == nil {
				prevoutIDs = append(prevoutIDs, sp.SpentOutputId.Bytes())
			}
		}

		for outIndex, out := range annotatedTxs[pos].Outputs {
			outputIDs = append(outputIDs, out.OutputID.Bytes())
			outputTxPositions = append(outputTxPositions, uint32(pos))
			outputIndexes = append(outputIndexes, uint32(outIndex))
			outputTxHashes = append(outputTxHashes, tx.ID.Bytes())
			outputTypes = append(outputTypes, out.Type)
			outputPurposes = append(outputPurposes, out.Purpose)
			outputAssetIDs = append(outputAssetIDs, out.AssetID.Bytes())
			outputAssetAliases = append(outputAssetAliases, out.AssetAlias)
			outputAssetDefinitions = append(outputAssetDefinitions, string(*out.AssetDefinition))
			outputAssetTags = append(outputAssetTags, string(*out.AssetTags))
			outputAssetLocals = append(outputAssetLocals, bool(out.AssetIsLocal))
			outputAmounts = append(outputAmounts, int64(out.Amount))
			outputAccountIDs = append(outputAccountIDs, sql.NullString{String: out.AccountID, Valid: out.AccountID != ""})
			outputAccountAliases = append(outputAccountAliases, sql.NullString{String: out.AccountAlias, Valid: out.AccountAlias != ""})
			if out.AccountTags != nil {
				outputAccountTags = append(outputAccountTags, sql.NullString{String: string(*out.AccountTags), Valid: true})
			} else {
				outputAccountTags = append(outputAccountTags, sql.NullString{})
			}
			outputControlPrograms = append(outputControlPrograms, out.ControlProgram)
			outputReferenceDatas = append(outputReferenceDatas, string(*out.ReferenceData))
			outputLocals = append(outputLocals, bool(out.IsLocal))
		}
	}

	// Insert all of the block's outputs at once.
	const insertQ = `
		WITH utxos AS (
			SELECT * FROM unnest($2::integer[], $3::integer[], $4::bytea[], $6::bytea[], $7::text[], $8::text[],
				$9::bytea[], $10::text[], $11::jsonb[], $12::jsonb[], $13::boolean[], $14::bigint[],
				$15::text[], $16::text[], $17::jsonb[], $18::bytea[], $19::jsonb[], $20::boolean[])
			AS t(tx_pos, output_index, tx_hash, output_id, type, purpose,
				asset_id, asset_alias, asset_definition, asset_tags, asset_local, amount,
				account_id, account_alias, account_tags, control_program, reference_data, local)
		)
		INSERT INTO annotated_outputs (block_height, tx_pos, output_index, tx_hash,
			timespan, output_id, type, purpose, asset_id, asset_alias, asset_definition,
			asset_tags, asset_local, amount, account_id, account_alias, account_tags,
			control_program, reference_data, local)
		SELECT $1, tx_pos, output_index, tx_hash,
		CASE WHEN type='retire' THEN int8range($5, $5) ELSE int8range($5, NULL) END,
		output_id, type, purpose, asset_id, asset_alias, asset_definition, asset_tags,
		asset_local, amount, account_id, account_alias, account_tags, control_program,
		reference_data, local
		FROM utxos
		ON CONFLICT (block_height, tx_pos, output_index) DO NOTHING;
	`
	_, err := ind.db.ExecContext(ctx, insertQ, b.Height, pq.Array(outputTxPositions),
		pq.Array(outputIndexes), outputTxHashes, b.TimestampMS, outputIDs, outputTypes,
		outputPurposes, outputAssetIDs, outputAssetAliases,
		outputAssetDefinitions, outputAssetTags, outputAssetLocals,
		outputAmounts, pq.Array(outputAccountIDs), pq.Array(outputAccountAliases),
		pq.Array(outputAccountTags), outputControlPrograms, outputReferenceDatas,
		outputLocals)
	if err != nil {
		return errors.Wrap(err, "batch inserting annotated outputs")
	}

	const updateQ = `
		UPDATE annotated_outputs SET timespan = INT8RANGE(LOWER(timespan), $1)
		WHERE (output_id) IN (SELECT unnest($2::bytea[]))
	`
	_, err = ind.db.ExecContext(ctx, updateQ, b.TimestampMS, prevoutIDs)
	return errors.Wrap(err, "updating spent annotated outputs")
}

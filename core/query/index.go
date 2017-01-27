package query

import (
	"context"
	"encoding/json"

	"github.com/lib/pq"

	"chain/database/pg"
	"chain/errors"
	"chain/protocol/bc"
)

const (
	// TxPinName is used to identify the pin associated
	// with the transaction block processor.
	TxPinName = "tx"
)

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
func (ind *Indexer) IndexTransactions(ctx context.Context, b *bc.Block) error {
	<-ind.pinStore.PinWaiter("asset", b.Height)

	err := ind.insertBlock(ctx, b)
	if err != nil {
		return err
	}

	txs, err := ind.insertAnnotatedTxs(ctx, b)
	if err != nil {
		return err
	}

	return ind.insertAnnotatedOutputs(ctx, b, txs)
}

func (ind *Indexer) insertBlock(ctx context.Context, b *bc.Block) error {
	const q = `
		INSERT INTO query_blocks (height, timestamp) VALUES($1, $2)
		ON CONFLICT (height) DO NOTHING
	`
	_, err := ind.db.Exec(ctx, q, b.Height, b.TimestampMS)
	return errors.Wrap(err, "inserting block timestamp")
}

func (ind *Indexer) insertAnnotatedTxs(ctx context.Context, b *bc.Block) ([]*AnnotatedTx, error) {
	var (
		hashes              = pq.ByteaArray(make([][]byte, 0, len(b.Transactions)))
		positions           = pg.Uint32s(make([]uint32, 0, len(b.Transactions)))
		annotatedTxs        = pq.StringArray(make([]string, 0, len(b.Transactions)))
		annotatedTxsDecoded = make([]*AnnotatedTx, 0, len(b.Transactions))
		outputIDs           = pq.ByteaArray(make([][]byte, 0))
	)
	for _, tx := range b.Transactions {
		for _, in := range tx.Inputs {
			if !in.IsIssuance() {
				outputIDs = append(outputIDs, in.SpentOutputID().Bytes())
			}
		}
	}
	outpoints, err := ind.loadOutpoints(ctx, outputIDs)
	if err != nil {
		return nil, err
	}
	for pos, tx := range b.Transactions {
		hashes = append(hashes, tx.Hash[:])
		positions = append(positions, uint32(pos))
		annotatedTxsDecoded = append(annotatedTxsDecoded, buildAnnotatedTransaction(tx, b, uint32(pos), outpoints))
	}

	for _, annotator := range ind.annotators {
		err = annotator(ctx, annotatedTxsDecoded)
		if err != nil {
			return nil, errors.Wrap(err, "adding external annotations")
		}
	}
	localAnnotator(ctx, annotatedTxsDecoded)

	for _, decoded := range annotatedTxsDecoded {
		b, err := json.Marshal(decoded)
		if err != nil {
			return nil, err
		}
		annotatedTxs = append(annotatedTxs, string(b))
	}

	// Save the annotated txs to the database.
	const insertQ = `
		INSERT INTO annotated_txs(block_height, tx_pos, tx_hash, data)
		SELECT $1, unnest($2::integer[]), unnest($3::bytea[]), unnest($4::jsonb[])
		ON CONFLICT (block_height, tx_pos) DO NOTHING;
	`
	_, err = ind.db.Exec(ctx, insertQ, b.Height, positions, hashes, annotatedTxs)
	if err != nil {
		return nil, errors.Wrap(err, "inserting annotated_txs to db")
	}
	return annotatedTxsDecoded, nil
}

func (ind *Indexer) loadOutpoints(ctx context.Context, outputIDs pq.ByteaArray) (map[bc.OutputID]*bc.Outpoint, error) {
	const q = `
		SELECT tx_hash, output_index
		FROM annotated_outputs
		WHERE output_id IN (SELECT unnest($1::bytea[]))
	`
	results := make(map[bc.OutputID]*bc.Outpoint)
	err := pg.ForQueryRows(ctx, ind.db, q, outputIDs, func(txHash bc.Hash, outputIndex uint32) {
		// We compute outid on the fly instead of receiving it from DB to save 40% of bandwidth.
		outid := bc.ComputeOutputID(txHash, outputIndex)
		results[outid] = &bc.Outpoint{
			Hash:  txHash,
			Index: outputIndex,
		}
	})
	if err != nil {
		return nil, errors.Wrap(err)
	}
	return results, nil
}

func (ind *Indexer) insertAnnotatedOutputs(ctx context.Context, b *bc.Block, annotatedTxs []*AnnotatedTx) error {
	var (
		outputTxPositions pg.Uint32s
		outputIndexes     pg.Uint32s
		outputTxHashes    pq.ByteaArray
		outputIDs         pq.ByteaArray
		outputData        pq.StringArray
		prevoutIDs        pq.ByteaArray
	)

	for pos, tx := range b.Transactions {
		for _, in := range tx.Inputs {
			if !in.IsIssuance() {
				prevoutID := in.SpentOutputID()
				prevoutIDs = append(prevoutIDs, prevoutID.Bytes())
			}
		}

		for outIndex, out := range annotatedTxs[pos].Outputs {
			if out.Type == "retire" {
				continue
			}

			outCopy := *out
			outCopy.TransactionID = tx.Hash[:]
			serializedData, err := json.Marshal(outCopy)
			if err != nil {
				return errors.Wrap(err, "serializing annotated output")
			}

			outputTxPositions = append(outputTxPositions, uint32(pos))
			outputIndexes = append(outputIndexes, uint32(outIndex))
			outputTxHashes = append(outputTxHashes, tx.Hash[:])
			outputIDs = append(outputIDs, outCopy.OutputID)
			outputData = append(outputData, string(serializedData))
		}
	}

	// Insert all of the block's outputs at once.
	const insertQ = `
		INSERT INTO annotated_outputs (block_height, tx_pos, output_index, tx_hash, output_id, data, timespan)
		SELECT $1, unnest($2::integer[]), unnest($3::integer[]), unnest($4::bytea[]), unnest($5::bytea[]),
		           unnest($6::jsonb[]),   int8range($7, NULL)
		ON CONFLICT (block_height, tx_pos, output_index) DO NOTHING;
	`
	_, err := ind.db.Exec(ctx, insertQ, b.Height, outputTxPositions,
		outputIndexes, outputTxHashes, outputIDs, outputData, b.TimestampMS)
	if err != nil {
		return errors.Wrap(err, "batch inserting annotated outputs")
	}

	const updateQ = `
		UPDATE annotated_outputs SET timespan = INT8RANGE(LOWER(timespan), $1)
		WHERE (output_id) IN (SELECT unnest($2::bytea[]))
	`
	_, err = ind.db.Exec(ctx, updateQ, b.TimestampMS, prevoutIDs)
	return errors.Wrap(err, "updating spent annotated outputs")
}

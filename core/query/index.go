package query

import (
	"encoding/json"
	"fmt"

	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/database/pg"
	"chain/errors"
	"chain/log"
)

// indexBlockCallback is registered as a block callback on the cos.FC. It
// saves all annotated transactions to the database and indexes them according
// to the Core's configured indexes.
func (i *Indexer) indexBlockCallback(ctx context.Context, b *bc.Block) {
	err := i.insertBlock(ctx, b)
	if err != nil {
		log.Fatal(ctx, err)
	}

	txs, err := i.insertAnnotatedTxs(ctx, b)
	if err != nil {
		log.Fatal(ctx, err)
	}

	err = i.insertAnnotatedOutputs(ctx, b, txs)
	if err != nil {
		log.Fatal(ctx, err)
	}

	// TODO(jackson): Build indexes
}

func (i *Indexer) insertBlock(ctx context.Context, b *bc.Block) error {
	const q = `
		INSERT INTO query_blocks (height, timestamp) VALUES($1, $2)
		ON CONFLICT (height) DO NOTHING
	`
	_, err := i.db.Exec(ctx, q, b.Height, b.TimestampMS)
	return errors.Wrap(err, "inserting block timestamp")
}

func (i *Indexer) insertAnnotatedTxs(ctx context.Context, b *bc.Block) ([]map[string]interface{}, error) {
	var (
		hashes              = pg.Strings(make([]string, 0, len(b.Transactions)))
		positions           = pg.Uint32s(make([]uint32, 0, len(b.Transactions)))
		annotatedTxs        = pg.Strings(make([]string, 0, len(b.Transactions)))
		annotatedTxsDecoded = make([]map[string]interface{}, 0, len(b.Transactions))
	)
	for pos, tx := range b.Transactions {
		hashes = append(hashes, tx.Hash.String())
		positions = append(positions, uint32(pos))

		// TODO(jackson): This is temporary until we have real
		// annotated transactions.
		b, err := json.Marshal(tx)
		if err != nil {
			return nil, err
		}
		annotatedTxs = append(annotatedTxs, string(b))

		var annotatedTx map[string]interface{}
		err = json.Unmarshal(b, &annotatedTx)
		if err != nil {
			return nil, err
		}
		annotatedTxsDecoded = append(annotatedTxsDecoded, annotatedTx)
	}

	// Save the annotated txs to the database.
	const insertQ = `
		INSERT INTO annotated_txs(block_height, tx_pos, tx_hash, data)
		SELECT $1, unnest($2::integer[]), unnest($3::text[]), unnest($4::jsonb[])
		ON CONFLICT (block_height, tx_pos) DO NOTHING;
	`
	_, err := i.db.Exec(ctx, insertQ, b.Height, positions, hashes, annotatedTxs)
	if err != nil {
		return nil, errors.Wrap(err, "inserting annotated_txs to db")
	}
	return annotatedTxsDecoded, nil
}

func (i *Indexer) insertAnnotatedOutputs(ctx context.Context, b *bc.Block, annotatedTxs []map[string]interface{}) error {
	var (
		outputTxPositions pg.Uint32s
		outputIndexes     pg.Uint32s
		outputTxHashes    pg.Strings
		outputData        pg.Strings
		prevoutHashes     pg.Strings
		prevoutIndexes    pg.Uint32s
	)

	for pos, tx := range b.Transactions {
		for _, in := range tx.Inputs {
			if !in.IsIssuance() {
				prevoutHashes = append(prevoutHashes, in.Outpoint().Hash.String())
				prevoutIndexes = append(prevoutIndexes, in.Outpoint().Index)
			}
		}

		outs, ok := annotatedTxs[pos]["outputs"].([]interface{})
		if !ok {
			return errors.Wrap(fmt.Errorf("bad outputs type %T", annotatedTxs[pos]["outputs"]))
		}
		for outIndex, out := range outs {
			txOut, ok := out.(map[string]interface{})
			if !ok {
				return errors.Wrap(fmt.Errorf("bad output type %T", out))
			}

			serializedData, err := json.Marshal(txOut)
			if err != nil {
				return errors.Wrap(err, "serializing annotated output")
			}

			outputTxPositions = append(outputTxPositions, uint32(pos))
			outputIndexes = append(outputIndexes, uint32(outIndex))
			outputTxHashes = append(outputTxHashes, tx.Hash.String())
			outputData = append(outputData, string(serializedData))
		}
	}

	// Insert all of the block's outputs at once.
	const insertQ = `
		INSERT INTO annotated_outputs (block_height, tx_pos, output_index, tx_hash, data, timespan)
		SELECT $1, unnest($2::integer[]), unnest($3::integer[]), unnest($4::text[]),
		           unnest($5::jsonb[]),   int8range($6, NULL)
		ON CONFLICT (block_height, tx_pos, output_index) DO NOTHING;
	`
	_, err := i.db.Exec(ctx, insertQ, b.Height, outputTxPositions,
		outputIndexes, outputTxHashes, outputData, b.TimestampMS)
	if err != nil {
		return errors.Wrap(err, "batch inserting annotated outputs")
	}

	const updateQ = `
		UPDATE annotated_outputs SET timespan = INT8RANGE(LOWER(timespan), $1)
		WHERE (tx_hash, output_index) IN (SELECT unnest($2::text[]), unnest($3::integer[]))
	`
	_, err = i.db.Exec(ctx, updateQ, b.TimestampMS, prevoutHashes, prevoutIndexes)
	return errors.Wrap(err, "updating spent annotated outputs")
}

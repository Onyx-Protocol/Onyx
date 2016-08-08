package query

import (
	"encoding/json"

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

	_, err = i.annotatedTxs(ctx, b)
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

func (i *Indexer) annotatedTxs(ctx context.Context, b *bc.Block) ([]map[string]interface{}, error) {
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
		INSERT INTO annotated_txs(block_height, block_index, tx_hash, data)
		SELECT $1, unnest($2::integer[]), unnest($3::text[]), unnest($4::jsonb[])
		ON CONFLICT (block_height, block_index) DO NOTHING;
	`
	_, err := i.db.Exec(ctx, insertQ, b.Height, positions, hashes, annotatedTxs)
	if err != nil {
		return nil, errors.Wrap(err, "inserting annotated_txs to db")
	}
	return annotatedTxsDecoded, nil
}

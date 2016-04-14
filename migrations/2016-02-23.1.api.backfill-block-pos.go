package main

import (
	"log"

	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/database/pg"
	"chain/database/sql"
	"chain/env"
)

// To avoid gaps in the backfill, please make sure you've migrated and deployed
// up to 7b40cd1 before running this migration.

func main() {
	// config vars
	var dbURL = env.String("DB_URL", "postgres:///api?sslmode=disable")
	var startBlock = env.Int("START_BLOCK", 0)
	var batchSize = env.Int("BATCH_SIZE", 10)

	env.Parse()

	sql.Register("schemadb", pg.SchemaDriver("backfill-block-pos"))
	db, err := sql.Open("schemadb", *dbURL)
	if err != nil {
		log.Fatal(err)
	}

	ctx := pg.NewContext(context.Background(), db)

	last := uint64(*startBlock)
	size := *batchSize
	for {
		nextLast, err := runBatch(ctx, last, size)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Processed blocks %d to %d", last+1, nextLast)

		if nextLast-last < uint64(size) {
			log.Println("No blocks remaining.")
			break
		}

		last = nextLast
	}
}

// runBatch runs the backfill for blocks whose height is greater than lastBlock,
// up to batchSize blocks. It returns:
// 1. The height of the last block that was processed, or lastBlock if no blocks
//    were affected. If lastBlock is returned, you should not run any subsequent
//    batches.
// 2. An error, if encountered.
func runBatch(ctx context.Context, lastBlock uint64, batchSize int) (uint64, error) {
	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer dbtx.Rollback(ctx)

	// Assemble per-transacton block information

	const blocksQ = `
		SELECT data
		FROM blocks
		WHERE HEIGHT > $1
		ORDER BY height
		LIMIT $2
	`

	rows, err := pg.FromContext(ctx).Query(ctx, blocksQ, lastBlock, batchSize)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var (
		blockHeights   []uint64
		txHashes       []string
		blockPositions []int32
	)

	for rows.Next() {
		b := new(bc.Block)
		err := rows.Scan(b)
		if err != nil {
			return 0, err
		}

		for i, tx := range b.Transactions {
			blockHeights = append(blockHeights, b.Height)
			txHashes = append(txHashes, tx.Hash.String())
			blockPositions = append(blockPositions, int32(i))
		}

		lastBlock = b.Height
	}

	if err := rows.Err(); err != nil {
		return 0, err
	}

	// Run actual backfill

	const insertQ = `
		UPDATE blocks_txs
		SET block_height = temp.block_height,
			block_pos = temp.block_pos
		FROM (
			SELECT unnest($1::bigint[]) as block_height,
				unnest($2::text[]) as tx_hash,
				unnest($3::int[]) as block_pos
		) AS temp
		WHERE blocks_txs.tx_hash = temp.tx_hash
	`

	_, err = pg.FromContext(ctx).Exec(
		ctx,
		insertQ,
		pg.Uint64s(blockHeights),
		pg.Strings(txHashes),
		pg.Int32s(blockPositions),
	)
	if err != nil {
		return 0, err
	}

	err = dbtx.Commit(ctx)
	if err != nil {
		return 0, err
	}

	return lastBlock, nil
}

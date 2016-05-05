// revalidate-bc validates the entire blockchain for a provided
// database or target.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	_ "github.com/lib/pq"
	"golang.org/x/net/context"

	"chain/cos"
	"chain/cos/bc"
	"chain/cos/memstore"
	"chain/database/pg"
	"chain/database/sql"
	"chain/env"
)

const (
	batchBlockCount = 50
)

const help = `
Usage:

	revalidate-bc [-t target] [-d url]

Command revalidate-bc revalidates the entire blockchain of a
database or target.

Either the database or the target flag must be specified,
but not both.
`

var (
	flagD = flag.String("d", "", "database")
	flagT = flag.String("t", "", "target")
	flagH = flag.Bool("h", false, "show help")
)

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

func main() {
	env.Parse()
	log.SetPrefix("appenv: ")
	log.SetFlags(0)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [-t target] [-d url]\n", os.Args[0])
	}
	flag.Parse()
	if *flagH || (*flagT == "") == (*flagD == "") {
		fmt.Println(strings.TrimSpace(help))
		fmt.Print("\nFlags:\n\n")
		flag.PrintDefaults()
		return
	}

	var dbURL string
	if *flagD != "" {
		dbURL = *flagD
	}
	if *flagT != "" {
		var err error
		dbURL, err = getTargetDBURL(*flagT)
		if err != nil {
		}
	}

	// Create a database connection.
	sql.Register("schemadb", pg.SchemaDriver("revalidate-bc"))
	db, err := sql.Open("schemadb", dbURL)
	if err != nil {
		fatalf("unable to get target DB_URL: %v\n", err)
	}
	defer db.Close()

	blocksValidated, err := RevalidateBlockchain(db)
	if err != nil {
		fatalf("error validating blockchain: %s\n", err)
	}
	fmt.Printf("Success: validated %d blocks\n", blocksValidated)
}

func RevalidateBlockchain(db *sql.DB) (blocksValidated uint64, err error) {
	dbCtx, cancel := context.WithCancel(pg.NewContext(context.Background(), db))
	blocks := streamBlocks(dbCtx)

	// Setup an FC backed with a memstore.
	// TODO(jackson): Don't keep everything in memory so that we can validate
	// larger blockchains in the future.
	ctx := context.Background()
	fc, err := cos.NewFC(ctx, memstore.New(), []*btcec.PublicKey{}, nil)
	if err != nil {
		fatalf("unable to construct FC: %s\n", err)
	}

	for b := range blocks {
		err = fc.AddBlock(ctx, b)
		if err != nil {
			cancel()
			return blocksValidated, fmt.Errorf("block %s, height %d: %s", b.Hash(), b.Height, err)
		}
		blocksValidated++
	}
	return blocksValidated, nil
}

func streamBlocks(ctx context.Context) <-chan *bc.Block {
	const q = `
		SELECT data FROM blocks WHERE height>=$1::bigint
		ORDER BY height ASC LIMIT $2
	`

	ch := make(chan *bc.Block, batchBlockCount)
	go func() {
		defer close(ch)
		var next uint64 = 0
		for {
			// Get a new page of blocks and send them out over the channel.
			var batch []*bc.Block
			err := pg.ForQueryRows(ctx, q, next, batchBlockCount, func(b bc.Block) {
				batch = append(batch, &b)
			})
			if err != nil {
				fatalf("error listing blocks from db: %s\n", err)
			}

			for _, b := range batch {
				select {
				case ch <- b:
				case <-ctx.Done():
					return
				}
			}

			// Check for an incomplete page, signalling current end of
			// the blockchain.
			if len(batch) != batchBlockCount {
				return
			}

			// Set the starting block height for the next iteration.
			next = batch[len(batch)-1].Height + 1
		}
	}()
	return ch
}

func getTargetDBURL(target string) (string, error) {
	out, err := exec.Command("appenv", "-t", target, "DB_URL").CombinedOutput()
	if err != nil {
		return "", errors.New(string(out))
	}
	return strings.TrimSpace(string(out)), nil
}

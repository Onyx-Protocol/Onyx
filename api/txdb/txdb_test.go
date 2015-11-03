package txdb

import (
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/fedchain/bc"
	"chain/fedchain/script"
	chainlog "chain/log"
)

func init() {
	chainlog.SetOutput(ioutil.Discard)

	u := "postgres:///api-test?sslmode=disable"
	if s := os.Getenv("DB_URL_TEST"); s != "" {
		u = s
	}

	db := pgtest.Open(u, "txdbtest", "../appdb/schema.sql")
	err := appdb.Init(db)
	if err != nil {
		log.Fatal(err)
	}
}

// Establish a context object with a new db transaction in which to
// run the given callback function.
func withContext(t *testing.T, sql string, fn func(*testing.T, context.Context)) {
	var dbtx pg.Tx
	if sql == "" {
		dbtx = pgtest.TxWithSQL(t)
	} else {
		dbtx = pgtest.TxWithSQL(t, sql)
	}
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)
	fn(t, ctx)
}

func mustParseHash(s string) bc.Hash {
	h, err := bc.ParseHash(s)
	if err != nil {
		panic(err)
	}
	return h
}

func TestPoolTxs(t *testing.T) {
	const fix = `
		INSERT INTO pool_txs (tx_hash, data)
		VALUES (
			'9e8cf364fc0446a1341dd021098a07983108c7bb853a8a33b466a292c4a8b248',
			decode('00000001000000000000000000000568656c6c6f', 'hex')
		);
	`
	withContext(t, fix, func(t *testing.T, ctx context.Context) {
		got, err := PoolTxs(ctx)
		if err != nil {
			t.Fatalf("err got = %v want nil", err)
		}

		want := []*bc.Tx{
			{
				Version:  1,
				Metadata: []byte("hello"),
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("txs:\ngot:  %v\nwant: %v", got, want)
		}
	})
}

func TestLatestBlock(t *testing.T) {
	const fix = `
		INSERT INTO blocks (block_hash, height, data)
		VALUES(
			'341fb89912be0110b527375998810c99ac96a317c63b071ccf33b7514cf0f0a5',
			1,
			decode('0000000100000000000000013132330000000000000000000000000000000000000000000000000000000000414243000000000000000000000000000000000000000000000000000000000058595a000000000000000000000000000000000000000000000000000000000000000000000000640f746573742d7369672d73637269707412746573742d6f75747075742d73637269707401000000010000000000000000000007746573742d7478', 'hex')
		);
	`
	withContext(t, fix, func(t *testing.T, ctx context.Context) {
		got, err := LatestBlock(ctx)
		if err != nil {
			t.Fatalf("err got = %v want nil", err)
		}

		want := &bc.Block{
			BlockHeader: bc.BlockHeader{
				Version:           bc.NewBlockVersion,
				Height:            1,
				PreviousBlockHash: [32]byte{'1', '2', '3'},
				TxRoot:            [32]byte{'A', 'B', 'C'},
				StateRoot:         [32]byte{'X', 'Y', 'Z'},
				Timestamp:         100,
				SignatureScript:   script.Script("test-sig-script"),
				OutputScript:      script.Script("test-output-script"),
			},
			Transactions: []*bc.Tx{
				{Version: 1, Metadata: []byte("test-tx")},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("latest block:\ngot:  %+v\nwant: %+v", got, want)
		}
	})
}

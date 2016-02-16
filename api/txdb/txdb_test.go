package txdb

import (
	"os"
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/fedchain/bc"
	"chain/fedchain/state"
)

func init() {
	u := "postgres:///api-test?sslmode=disable"
	if s := os.Getenv("DB_URL_TEST"); s != "" {
		u = s
	}

	ctx := context.Background()
	pgtest.Open(ctx, u, "txdbtest", "../appdb/schema.sql")
}

// Establish a context object with a new db transaction in which to
// run the given callback function.
func withContext(tb testing.TB, sql string, fn func(context.Context)) {
	var ctx context.Context
	if sql == "" {
		ctx = pgtest.NewContext(tb)
	} else {
		ctx = pgtest.NewContext(tb, sql)
	}
	defer pgtest.Finish(ctx)
	fn(ctx)
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
			'94fd998552e8ceceab82dd261f391bb2e4a1a08d8e9b624c5ca011d932d8b614',
			decode('01000000000000000000000000000568656c6c6f', 'hex')
		);
	`
	withContext(t, fix, func(ctx context.Context) {
		got, err := poolTxs(ctx)
		if err != nil {
			t.Fatalf("err got = %v want nil", err)
		}

		wantTx := bc.NewTx(bc.TxData{
			Version:  1,
			Metadata: []byte("hello"),
		})
		wantTx.Stored = true
		want := []*bc.Tx{wantTx}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("txs do not match")
			for _, tx := range got {
				t.Logf("\tgot %v", tx)
			}
			for _, tx := range want {
				t.Logf("\twant %v", tx)
			}
		}
	})
}

func TestGetTxs(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		tx := bc.NewTx(bc.TxData{Metadata: []byte("tx")})
		ok, err := insertTx(ctx, tx)
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("expected insertTx to be successful")
		}

		txs, err := GetTxs(ctx, tx.Hash.String())
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		tx.Stored = true
		if !reflect.DeepEqual(txs[tx.Hash.String()], tx) {
			t.Errorf("got:\n\t%+v\nwant:\n\t%+v", txs[tx.Hash.String()], tx)
		}

		_, gotErr := GetTxs(ctx, tx.Hash.String(), "nonexistent")
		if errors.Root(gotErr) != pg.ErrUserInputNotFound {
			t.Errorf("got err=%q want %q", errors.Root(gotErr), pg.ErrUserInputNotFound)
		}
	})
}

func TestInsertTx(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		tx := bc.NewTx(bc.TxData{Metadata: []byte("tx")})
		ok, err := insertTx(ctx, tx)
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("expected insertTx to be successful")
		}

		_, err = GetTxs(ctx, tx.Hash.String())
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}
	})
}

func TestLatestBlock(t *testing.T) {
	const fix = `
		INSERT INTO blocks (block_hash, height, data, header)
		VALUES
		('0000000000000000000000000000000000000000000000000000000000000000', 0, '', ''),
		(
			'72a7766e340ce61838d88fbd2f3bcf6064b2d6a21596028d031c6a9d24dbd4af',
			1,
			decode('0100000001000000000000003132330000000000000000000000000000000000000000000000000000000000414243000000000000000000000000000000000000000000000000000000000058595a000000000000000000000000000000000000000000000000000000000064000000000000000f746573742d7369672d73637269707412746573742d6f75747075742d73637269707401010000000000000000000000000007746573742d7478', 'hex'),
			''
		);
	`
	withContext(t, fix, func(ctx context.Context) {
		got, err := latestBlock(ctx)
		if err != nil {
			t.Fatalf("err got = %v want nil", err)
		}

		want := &bc.Block{
			BlockHeader: bc.BlockHeader{
				Version:           1,
				Height:            1,
				PreviousBlockHash: [32]byte{'1', '2', '3'},
				TxRoot:            [32]byte{'A', 'B', 'C'},
				StateRoot:         [32]byte{'X', 'Y', 'Z'},
				Timestamp:         100,
				SignatureScript:   []byte("test-sig-script"),
				OutputScript:      []byte("test-output-script"),
			},
			Transactions: []*bc.Tx{
				bc.NewTx(bc.TxData{Version: 1, Metadata: []byte("test-tx")}),
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("latest block:\ngot:  %+v\nwant: %+v", got, want)
		}
	})
}

func TestInsertBlock(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		blk := &bc.Block{
			BlockHeader: bc.BlockHeader{
				Version: 1,
				Height:  1,
			},
			Transactions: []*bc.Tx{
				bc.NewTx(bc.TxData{
					Metadata: []byte("a"),
				}),
				bc.NewTx(bc.TxData{
					Metadata: []byte("b"),
				}),
			},
		}
		_, err := insertBlock(ctx, blk)
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		// block in database
		_, err = GetBlock(ctx, blk.Hash().String())
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		// txs in database
		txs := blk.Transactions
		_, err = GetTxs(ctx, txs[0].Hash.String(), txs[1].Hash.String())
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}
	})
}

func TestGetBlock(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		blk := &bc.Block{
			BlockHeader: bc.BlockHeader{
				Version: 1,
				Height:  1,
			},
		}
		_, err := insertBlock(ctx, blk)
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		got, err := GetBlock(ctx, blk.Hash().String())
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		if !reflect.DeepEqual(got, blk) {
			t.Errorf("got:\n\t%+v\nwant:\n\t:%+v", got, blk)
		}

		_, gotErr := GetBlock(ctx, "nonexistent")
		if errors.Root(gotErr) != pg.ErrUserInputNotFound {
			t.Errorf("got err=%q want %q", errors.Root(gotErr), pg.ErrUserInputNotFound)
		}
	})
}

func TestListBlocks(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		blks := []*bc.Block{
			{BlockHeader: bc.BlockHeader{Height: 1}},
			{BlockHeader: bc.BlockHeader{Height: 0}},
		}
		for _, blk := range blks {
			_, err := insertBlock(ctx, blk)
			if err != nil {
				t.Log(errors.Stack(err))
				t.Fatal(err)
			}
		}
		cases := []struct {
			prev  string
			limit int
			want  []*bc.Block
		}{{
			prev:  "",
			limit: 50,
			want:  blks,
		}, {
			prev:  "1",
			limit: 50,
			want:  []*bc.Block{blks[1]},
		}, {
			prev:  "",
			limit: 1,
			want:  []*bc.Block{blks[0]},
		}, {
			prev:  "0",
			limit: 50,
			want:  nil,
		}}

		for _, c := range cases {
			got, err := ListBlocks(ctx, c.prev, c.limit)
			if err != nil {
				t.Log(errors.Stack(err))
				t.Errorf("ListBlocks(%q, %d) error = %q", c.prev, c.limit, err)
				continue
			}

			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("got ListBlocks(%q, %d):\n\t%+v\nwant:\n\t%+v", c.prev, c.limit, got, c.want)
			}
		}
	})
}

func TestRemoveBlockOutputs(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {

		out := &state.Output{
			TxOutput: bc.TxOutput{
				AssetAmount: bc.AssetAmount{AssetID: bc.AssetID{}, Amount: 5},
				Script:      []byte("a"),
				Metadata:    []byte("b"),
			},
			Outpoint: bc.Outpoint{},
		}
		err := insertBlockOutputs(ctx, []*state.Output{out})
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		out.Spent = true
		err = removeBlockSpentOutputs(ctx, []*state.Output{out})
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		gotOut, err := loadOutput(ctx, out.Outpoint)
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		if gotOut != nil {
			t.Fatal("expected out to be removed from database")
		}
	})
}

func TestInsertBlockOutputs(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		out := &state.Output{
			TxOutput: bc.TxOutput{
				AssetAmount: bc.AssetAmount{AssetID: bc.AssetID{}, Amount: 5},
				Script:      []byte("a"),
				Metadata:    []byte("b"),
			},
			Outpoint: bc.Outpoint{},
		}
		err := insertBlockOutputs(ctx, []*state.Output{out})
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		_, err = loadOutput(ctx, out.Outpoint)
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}
	})
}

// Helper function just for testing.
// In production, we ~never want to load a single output;
// we always load in batches.
func loadOutput(ctx context.Context, p bc.Outpoint) (*state.Output, error) {
	m, err := loadOutputs(ctx, []bc.Outpoint{p})
	return m[p], err
}

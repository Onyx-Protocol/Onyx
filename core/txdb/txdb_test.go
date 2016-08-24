package txdb

import (
	"context"
	"reflect"
	"testing"

	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/database/sql"
	"chain/errors"
	"chain/protocol/bc"
	"chain/testutil"
)

func mustParseHash(s string) bc.Hash {
	h, err := bc.ParseHash(s)
	if err != nil {
		panic(err)
	}
	return h
}

func TestPoolTxs(t *testing.T) {
	dbtx := pgtest.NewTx(t)
	ctx := context.Background()

	_, err := dbtx.Exec(ctx, `
		INSERT INTO pool_txs (tx_hash, data)
		VALUES (
			'70dd2c70c5c1e859c32bcc4415d15c70a95c977e0c35c115541f692f00ffbe9c',
			decode('07010200000000000568656c6c6f', 'hex')
		);
	`)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	got, err := dumpPoolTxs(ctx, dbtx)
	if err != nil {
		t.Fatalf("err got = %v want nil", err)
	}

	wantTx := bc.NewTx(bc.TxData{
		Version:       1,
		ReferenceData: []byte("hello"),
	})
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
}

func TestGetTxs(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	dbctx := pg.NewContext(context.Background(), db)
	pool := NewPool(pg.FromContext(dbctx).(*sql.DB))
	ctx := context.Background()

	tx := bc.NewTx(bc.TxData{ReferenceData: []byte("tx")})
	err := pool.Insert(ctx, tx)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
	poolTxs, err := pool.GetTxs(ctx, tx.Hash)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	if !reflect.DeepEqual(poolTxs[tx.Hash], tx) {
		t.Errorf("got:\n\t%+v\nwant:\n\t%+v", poolTxs[tx.Hash], tx)
	}

	nonexistentHash := mustParseHash("beefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeef")
	_, gotErr := pool.GetTxs(ctx, tx.Hash, nonexistentHash)
	if gotErr != nil {
		t.Errorf("got err=%v want nil", gotErr)
	}
}

func TestInsertTx(t *testing.T) {
	dbtx := pgtest.NewTx(t)
	ctx := context.Background()
	tx := bc.NewTx(bc.TxData{ReferenceData: []byte("tx")})
	ok, err := insertTx(ctx, dbtx, tx)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected insertTx to be successful")
	}

	_, err = getBlockchainTxs(ctx, dbtx, tx.Hash)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
}

func TestGetBlock(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	dbctx := pg.NewContext(context.Background(), db)
	pgtest.Exec(dbctx, t, `
		INSERT INTO blocks (block_hash, height, data, header)
		VALUES
		('0000000000000000000000000000000000000000000000000000000000000000', 0, '', ''),
		(
			'1f20d89dd393f452b4396589ed5d6f90465cb032aa3f9fe42a99d47c7089b0a3',
			1,
			decode('03010131323300000000000000000000000000000000000000000000000000000000006453414243000000000000000000000000000000000000000000000000000000000058595a000000000000000000000000000000000000000000000000000000000012746573742d6f75747075742d73637269707411010f746573742d7369672d73637269707401070102000000000007746573742d7478', 'hex'),
			''
		);
	`)
	store := NewStore(pg.FromContext(dbctx).(*sql.DB))
	ctx := context.Background()
	got, err := store.GetBlock(ctx, 1)
	if err != nil {
		t.Fatalf("err got = %v want nil", err)
	}
	want := &bc.Block{
		BlockHeader: bc.BlockHeader{
			Version:           1,
			Height:            1,
			PreviousBlockHash: [32]byte{'1', '2', '3'},
			TransactionsMerkleRoot: bc.Hash{
				'A', 'B', 'C', 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			},
			AssetsMerkleRoot: bc.Hash{
				'X', 'Y', 'Z', 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			},
			TimestampMS:      100,
			Witness:          [][]byte{[]byte("test-sig-script")},
			ConsensusProgram: []byte("test-output-script"),
		},
		Transactions: []*bc.Tx{
			bc.NewTx(bc.TxData{Version: 1, ReferenceData: []byte("test-tx")}),
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("latest block:\ngot:  %+v\nwant: %+v", got, want)
	}
}

func TestInsertBlock(t *testing.T) {
	dbtx := pgtest.NewTx(t)
	ctx := context.Background()
	blk := &bc.Block{
		BlockHeader: bc.BlockHeader{
			Version: 1,
			Height:  1,
		},
		Transactions: []*bc.Tx{
			bc.NewTx(bc.TxData{
				ReferenceData: []byte("a"),
			}),
			bc.NewTx(bc.TxData{
				ReferenceData: []byte("b"),
			}),
		},
	}
	err := insertBlock(ctx, dbtx, blk)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	// block in database
	_, err = getBlockByHash(ctx, dbtx, blk.Hash().String())
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	// txs in database
	txs := blk.Transactions
	_, err = getBlockchainTxs(ctx, dbtx, txs[0].Hash, txs[1].Hash)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
}

func TestGetBlockByHash(t *testing.T) {
	dbtx := pgtest.NewTx(t)
	ctx := context.Background()
	blk := &bc.Block{
		BlockHeader: bc.BlockHeader{
			Version: 1,
			Height:  1,
		},
	}
	err := insertBlock(ctx, dbtx, blk)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	got, err := getBlockByHash(ctx, dbtx, blk.Hash().String())
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, blk) {
		t.Errorf("got:\n\t%+v\nwant:\n\t:%+v", got, blk)
	}

	_, gotErr := getBlockByHash(ctx, dbtx, "nonexistent")
	if errors.Root(gotErr) != pg.ErrUserInputNotFound {
		t.Errorf("got err=%q want %q", errors.Root(gotErr), pg.ErrUserInputNotFound)
	}
}

func TestListBlocks(t *testing.T) {
	dbtx := pgtest.NewTx(t)
	ctx := context.Background()
	blks := []*bc.Block{
		{
			BlockHeader: bc.BlockHeader{
				Version: 1,
				Height:  2,
			},
		},
		{
			BlockHeader: bc.BlockHeader{
				Version: 1,
				Height:  1,
			},
		},
	}
	for _, blk := range blks {
		err := insertBlock(ctx, dbtx, blk)
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
		prev:  "2",
		limit: 50,
		want:  []*bc.Block{blks[1]},
	}, {
		prev:  "",
		limit: 1,
		want:  []*bc.Block{blks[0]},
	}, {
		prev:  "1",
		limit: 50,
		want:  nil,
	}}

	for _, c := range cases {
		got, err := listBlocks(ctx, dbtx, c.prev, c.limit)
		if err != nil {
			t.Log(errors.Stack(err))
			t.Errorf("ListBlocks(%q, %d) error = %q", c.prev, c.limit, err)
			continue
		}

		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("got ListBlocks(%q, %d):\n\t%+v\nwant:\n\t%+v", c.prev, c.limit, got, c.want)
		}
	}
}

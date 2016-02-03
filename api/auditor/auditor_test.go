package auditor

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"reflect"
	"testing"
	"time"

	"golang.org/x/net/context"

	"chain/api/asset"
	"chain/api/asset/assettest"
	"chain/api/txdb"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/fedchain"
	"chain/fedchain/bc"
	"chain/testutil"
)

func init() {
	fc := fedchain.New(&txdb.Store{}, nil)
	asset.ConnectFedchain(fc)

	u := "postgres:///api-test?sslmode=disable"
	if s := os.Getenv("DB_URL_TEST"); s != "" {
		u = s
	}

	ctx := context.Background()
	pgtest.Open(ctx, u, "auditortest", "../appdb/schema.sql")
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

func mustParseHash(str string) bc.Hash {
	hash, err := bc.ParseHash(str)
	if err != nil {
		panic(err)
	}
	return hash
}

func mustDecodeHex(str string) []byte {
	bytes, err := hex.DecodeString(str)
	if err != nil {
		panic(err)
	}
	return bytes
}

func TestListBlocks(t *testing.T) {
	const fix = `
		INSERT INTO blocks(block_hash, height, data, header)
		VALUES(
			'749b1570d9f67b0edd4a050710b766bc04d3a52cf1b8c3fac6269ac5053021bb',
			1,
			decode('0100000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000006400000000000000000001010000000000000000000000000003747831', 'hex'),
			''
		), (
			'a22aca4d61a90a36d5bc164925d264967f795c1caa5884997026c14977fffdf3',
			2,
			decode('010000000200000000000000b3431f1d6c5aa2746a08d933bab1c5e68df1b18f3a43010f6f247b839d89e174000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000006900000000000000000002010000000000000000000000000003747832010000000000000000000000000003747833', 'hex'),
			''
		);
	`
	withContext(t, fix, func(ctx context.Context) {
		cases := []struct {
			prev     string
			limit    int
			want     []ListBlocksItem
			wantLast string
		}{{
			prev:  "",
			limit: 50,
			want: []ListBlocksItem{{
				ID:      mustParseHash("a22aca4d61a90a36d5bc164925d264967f795c1caa5884997026c14977fffdf3"),
				Height:  2,
				Time:    time.Unix(105, 0).UTC(),
				TxCount: 2,
			}, {
				ID:      mustParseHash("749b1570d9f67b0edd4a050710b766bc04d3a52cf1b8c3fac6269ac5053021bb"),
				Height:  1,
				Time:    time.Unix(100, 0).UTC(),
				TxCount: 1,
			}},
			wantLast: "",
		}, {
			prev:  "2",
			limit: 50,
			want: []ListBlocksItem{{
				ID:      mustParseHash("749b1570d9f67b0edd4a050710b766bc04d3a52cf1b8c3fac6269ac5053021bb"),
				Height:  1,
				Time:    time.Unix(100, 0).UTC(),
				TxCount: 1,
			}},
			wantLast: "",
		}, {
			prev:  "",
			limit: 1,
			want: []ListBlocksItem{{
				ID:      mustParseHash("a22aca4d61a90a36d5bc164925d264967f795c1caa5884997026c14977fffdf3"),
				Height:  2,
				Time:    time.Unix(105, 0).UTC(),
				TxCount: 2,
			}},
			wantLast: "2",
		}, {
			prev:     "1",
			limit:    50,
			want:     nil,
			wantLast: "",
		}}
		for _, c := range cases {
			got, gotLast, err := ListBlocks(ctx, c.prev, c.limit)
			if err != nil {
				t.Errorf("ListBlocks(%v, %v) unexpected err = %q", c.prev, c.limit, err)
				continue
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("got ListBlocks(%v, %v) = %+v want %+v", c.prev, c.limit, got, c.want)
			}
			if gotLast != c.wantLast {
				t.Errorf("got ListBlocks(%v, %v) last = %q want %q", c.prev, c.limit, gotLast, c.wantLast)
			}
		}
	})
}

func TestGetBlockSummary(t *testing.T) {
	const fix = `
		INSERT INTO blocks(block_hash, height, data, header)
		VALUES(
			'a22aca4d61a90a36d5bc164925d264967f795c1caa5884997026c14977fffdf3',
			2,
			decode('010000000200000000000000b3431f1d6c5aa2746a08d933bab1c5e68df1b18f3a43010f6f247b839d89e174000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000006900000000000000000002010000000000000000000000000003747832010000000000000000000000000003747833', 'hex'),
			''
		);
	`
	withContext(t, fix, func(ctx context.Context) {
		got, err := GetBlockSummary(ctx, "a22aca4d61a90a36d5bc164925d264967f795c1caa5884997026c14977fffdf3")
		if err != nil {
			t.Fatal(err)
		}
		want := &BlockSummary{
			ID:      mustParseHash("a22aca4d61a90a36d5bc164925d264967f795c1caa5884997026c14977fffdf3"),
			Height:  2,
			Time:    time.Unix(105, 0).UTC(),
			TxCount: 2,
			TxIDs: []bc.Hash{
				mustParseHash("c4da6c3203bd0fc597be31244c419faf0b333c7eff6368de4bb3c3ccff2dcc9d"),
				mustParseHash("21f62c32964b7514afe6d1ef8b2ffb2dcc4c8c3c181f6f27b2f6868f32b0f9f5"),
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("got block header:\n\t%+v\nwant:\n\t%+v", got, want)
		}
	})
}

func TestGetTxIssuance(t *testing.T) {
	tx := bc.NewTx(bc.TxData{
		Inputs: []*bc.TxInput{{
			Previous:        bc.Outpoint{Index: bc.InvalidOutputIndex},
			Metadata:        []byte(`{"a":"b"}`),
			AssetDefinition: []byte(`{"c":"d"}`),
		}},
		Outputs: []*bc.TxOutput{{
			AssetAmount: bc.AssetAmount{AssetID: bc.AssetID([32]byte{0}), Amount: 5},
			Metadata:    []byte{2},
			Script:      []byte("addr0"),
		}, {
			AssetAmount: bc.AssetAmount{AssetID: bc.AssetID([32]byte{0}), Amount: 6},
			Script:      []byte("addr1"),
		}},
		Metadata: []byte{0},
	})

	withContext(t, "", func(ctx context.Context) {
		err := txdb.InsertTx(ctx, tx)
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		got, err := GetTx(ctx, tx.Hash.String())
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		want := &Tx{
			ID:       tx.Hash,
			BlockID:  nil,
			Metadata: []byte{0},
			Inputs: []*TxInput{{
				Type:     "issuance",
				AssetID:  bc.AssetID([32]byte{0}),
				Metadata: []byte(`{"a":"b"}`),
				AssetDef: []byte(`{"c":"d"}`),
			}},
			Outputs: []*TxOutput{{
				AssetID:  bc.AssetID([32]byte{0}),
				Amount:   5,
				Address:  []byte("addr0"),
				Script:   []byte("addr0"),
				Metadata: []byte{2},
			}, {
				AssetID: bc.AssetID([32]byte{0}),
				Amount:  6,
				Address: []byte("addr1"),
				Script:  []byte("addr1"),
			}},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("got:\n\t%+v\nwant:\n\t%+v", got, want)
		}
	})
}

func TestGetTxTransfer(t *testing.T) {
	prevTxs := []*bc.Tx{
		bc.NewTx(bc.TxData{
			Outputs: []*bc.TxOutput{{
				AssetAmount: bc.AssetAmount{AssetID: bc.AssetID([32]byte{1}), Amount: 5},
			}},
		}),
		bc.NewTx(bc.TxData{
			Outputs: []*bc.TxOutput{{}, {
				AssetAmount: bc.AssetAmount{AssetID: bc.AssetID([32]byte{2}), Amount: 6},
			}},
		}),
	}
	tx := bc.NewTx(bc.TxData{
		Inputs: []*bc.TxInput{{
			Previous: bc.Outpoint{Hash: prevTxs[0].Hash, Index: 0},
		}, {
			Previous: bc.Outpoint{Hash: prevTxs[1].Hash, Index: 1},
		}},
		Outputs: []*bc.TxOutput{{
			AssetAmount: bc.AssetAmount{AssetID: bc.AssetID([32]byte{1}), Amount: 5},
			Script:      []byte("addr0"),
		}, {
			AssetAmount: bc.AssetAmount{AssetID: bc.AssetID([32]byte{2}), Amount: 6},
			Script:      []byte("addr1"),
		}},
	})

	now := time.Now().UTC().Truncate(time.Second)
	blk := &bc.Block{
		BlockHeader: bc.BlockHeader{
			Height:    1,
			Timestamp: uint64(now.Unix()),
		},
		Transactions: append(prevTxs, tx),
	}

	withContext(t, "", func(ctx context.Context) {
		const q = `INSERT INTO txs (tx_hash, data) VALUES($1, $2)`
		_, err := txdb.InsertBlock(ctx, blk)
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		got, err := GetTx(ctx, tx.Hash.String())
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		var blkHash = blk.Hash()

		zero, one := uint32(0), uint32(1)
		five, six := uint64(5), uint64(6)
		h0, h1 := prevTxs[0].Hash, prevTxs[1].Hash

		want := &Tx{
			ID:          tx.Hash,
			BlockID:     &blkHash,
			BlockHeight: 1,
			BlockTime:   now,
			Inputs: []*TxInput{{
				Type:    "transfer",
				AssetID: bc.AssetID([32]byte{1}),
				Amount:  &five,
				TxID:    &h0,
				TxOut:   &zero,
			}, {
				Type:    "transfer",
				AssetID: bc.AssetID([32]byte{2}),
				Amount:  &six,
				TxID:    &h1,
				TxOut:   &one,
			}},
			Outputs: []*TxOutput{{
				AssetID: bc.AssetID([32]byte{1}),
				Amount:  5,
				Address: []byte("addr0"),
				Script:  []byte("addr0"),
			}, {
				AssetID: bc.AssetID([32]byte{2}),
				Amount:  6,
				Address: []byte("addr1"),
				Script:  []byte("addr1"),
			}},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("got:\n\t%+v\nwant:\n\t%+v", got, want)
		}
	})
}

func TestGetAssets(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	assettest.CreateGenesisBlockFixture(ctx, t)

	in0 := assettest.CreateIssuerNodeFixture(ctx, t, "", "in-0", nil, nil)

	asset0 := assettest.CreateAssetFixture(ctx, t, in0, "asset-0", "def-0")
	asset1 := assettest.CreateAssetFixture(ctx, t, in0, "asset-1", "def-1")

	def0 := []byte("{\n  \"s\": \"def-0\"\n}")
	defPtr0 := bc.HashAssetDefinition(def0).String()

	assettest.IssueAssetsFixture(ctx, t, asset0, 58, "")
	asset.MakeBlock(ctx, asset.BlockKey)
	assettest.IssueAssetsFixture(ctx, t, asset0, 12, "")
	assettest.IssueAssetsFixture(ctx, t, asset1, 10, "")

	got, err := GetAssets(ctx, []string{
		asset0.String(),
		asset1.String(),
		"other-asset-id",
	})
	if err != nil {
		testutil.FatalErr(t, err)
	}

	want := map[string]*Asset{
		asset0.String(): &Asset{
			ID:            asset0,
			DefinitionPtr: defPtr0,
			Definition:    def0,
			Issued:        58,
		},

		// Strictly speaking, asset1 should not be returned yet, since it has
		// not landed in a block, so we shouldn't return it. However, we are
		// including it here, since there is no easy way to know which asset
		// issuances have landed, and which haven't. We can fix this by always
		// writing asset definition pointers, even for issuances that have a
		// blank asset definition.
		asset1.String(): &Asset{
			ID:            asset1,
			DefinitionPtr: "",
			Definition:    nil,
			Issued:        0,
		},
	}

	if !reflect.DeepEqual(got, want) {
		g, err := json.MarshalIndent(got, "", "  ")
		if err != nil {
			testutil.FatalErr(t, err)
		}

		w, err := json.MarshalIndent(want, "", "  ")
		if err != nil {
			testutil.FatalErr(t, err)
		}

		t.Errorf("assets:\ngot:  %v\nwant: %v", string(g), string(w))
	}
}

func TestGetAsset(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	assettest.CreateGenesisBlockFixture(ctx, t)

	in0 := assettest.CreateIssuerNodeFixture(ctx, t, "", "in-0", nil, nil)

	asset0 := assettest.CreateAssetFixture(ctx, t, in0, "asset-0", "def-0")
	asset1 := assettest.CreateAssetFixture(ctx, t, in0, "asset-1", "def-1")

	def0 := []byte("{\n  \"s\": \"def-0\"\n}")
	defPtr0 := bc.HashAssetDefinition(def0).String()

	assettest.IssueAssetsFixture(ctx, t, asset0, 58, "")
	asset.MakeBlock(ctx, asset.BlockKey)
	assettest.IssueAssetsFixture(ctx, t, asset0, 12, "")
	assettest.IssueAssetsFixture(ctx, t, asset1, 10, "")

	examples := []struct {
		id      string
		wantErr error
		want    *Asset
	}{
		{
			id: asset0.String(),
			want: &Asset{
				ID:            asset0,
				DefinitionPtr: defPtr0,
				Definition:    def0,
				Issued:        58,
			},
		},

		// Blank definition
		{
			id: asset1.String(),
			want: &Asset{
				ID:            asset1,
				DefinitionPtr: "",
				Definition:    nil,
				Issued:        0,
			},
		},

		// Missing asset
		{
			id:      "other-asset-id",
			wantErr: pg.ErrUserInputNotFound,
		},
	}

	for _, ex := range examples {
		t.Log("Example", ex.id)

		got, err := GetAsset(ctx, ex.id)
		if errors.Root(err) != ex.wantErr {
			t.Fatalf("error:\ngot:  %v\nwant: %v", errors.Root(err), ex.wantErr)
		}

		if !reflect.DeepEqual(got, ex.want) {
			t.Errorf("got:\n\t%+v\nwant:\n\t%+v", got, ex.want)
		}
	}
}

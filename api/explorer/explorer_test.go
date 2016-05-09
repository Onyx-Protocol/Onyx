package explorer

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"chain/api/asset/assettest"
	"chain/api/generator"
	"chain/api/txbuilder"
	"chain/api/txdb"
	"chain/cos/bc"
	"chain/cos/patricia"
	"chain/cos/txscript"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/database/sql"
	"chain/errors"
	"chain/testutil"
)

func mustParseHash(str string) bc.Hash {
	hash, err := bc.ParseHash(str)
	if err != nil {
		panic(err)
	}
	return hash
}

func TestListBlocks(t *testing.T) {
	ctx := pgtest.NewContext(t)
	store := txdb.NewStore(pg.FromContext(ctx).(*sql.DB))
	pgtest.Exec(ctx, t, `
		INSERT INTO blocks(block_hash, height, data, header)
		VALUES(
			'9aa10b4210bf638c868c6696a91ad10eb4924818281394085d5768592cfcf742',
			1,
			decode('010000000100000000000000000000000000000000000000000000000000000000000000000000000000000000640000000000000000000107010000000000000000000000000003747831', 'hex'),
			''
		), (
			'c1b9e92a70a1ce5f837d0c9258318291924dbe3ebc29c5a74aa942db58e697a3',
			2,
			decode('010000000200000000000000b3431f1d6c5aa2746a08d933bab1c5e68df1b18f3a43010f6f247b839d89e1740069000000000000000000020701000000000000000000000000000374783207010000000000000000000000000003747833', 'hex'),
			''
		);
	`)

	cases := []struct {
		prev     string
		limit    int
		want     []ListBlocksItem
		wantLast string
	}{{
		prev:  "",
		limit: 50,
		want: []ListBlocksItem{{
			ID:      mustParseHash("c1b9e92a70a1ce5f837d0c9258318291924dbe3ebc29c5a74aa942db58e697a3"),
			Height:  2,
			Time:    time.Unix(105, 0).UTC(),
			TxCount: 2,
		}, {
			ID:      mustParseHash("9aa10b4210bf638c868c6696a91ad10eb4924818281394085d5768592cfcf742"),
			Height:  1,
			Time:    time.Unix(100, 0).UTC(),
			TxCount: 1,
		}},
		wantLast: "",
	}, {
		prev:  "2",
		limit: 50,
		want: []ListBlocksItem{{
			ID:      mustParseHash("9aa10b4210bf638c868c6696a91ad10eb4924818281394085d5768592cfcf742"),
			Height:  1,
			Time:    time.Unix(100, 0).UTC(),
			TxCount: 1,
		}},
		wantLast: "",
	}, {
		prev:  "",
		limit: 1,
		want: []ListBlocksItem{{
			ID:      mustParseHash("c1b9e92a70a1ce5f837d0c9258318291924dbe3ebc29c5a74aa942db58e697a3"),
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
		got, gotLast, err := ListBlocks(ctx, store, c.prev, c.limit)
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
}

func TestGetBlockSummary(t *testing.T) {
	ctx := pgtest.NewContext(t)
	store := txdb.NewStore(pg.FromContext(ctx).(*sql.DB))
	pgtest.Exec(ctx, t, `
		INSERT INTO blocks(block_hash, height, data, header)
		VALUES(
			'c1b9e92a70a1ce5f837d0c9258318291924dbe3ebc29c5a74aa942db58e697a3',
			2,
			decode('010000000200000000000000b3431f1d6c5aa2746a08d933bab1c5e68df1b18f3a43010f6f247b839d89e1740069000000000000000000020701000000000000000000000000000374783207010000000000000000000000000003747833', 'hex'),
			''
		);
	`)

	got, err := GetBlockSummary(ctx, store, "c1b9e92a70a1ce5f837d0c9258318291924dbe3ebc29c5a74aa942db58e697a3")
	if err != nil {
		t.Fatal(err)
	}
	want := &BlockSummary{
		ID:      mustParseHash("c1b9e92a70a1ce5f837d0c9258318291924dbe3ebc29c5a74aa942db58e697a3"),
		Height:  2,
		Time:    time.Unix(105, 0).UTC(),
		TxCount: 2,
		TxHashes: []bc.Hash{
			mustParseHash("c4bca11aefa0d71667eb50f7d775f33b7c0d8e435c09d07dbe2f71a78ec410c5"),
			mustParseHash("353e45c66cc76674225fc037eb2103617cbc900f4c8b8b487f1edac8dd9764c1"),
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got block header:\n\t%+v\nwant:\n\t%+v", got, want)
	}
}

func TestGetTxIssuance(t *testing.T) {
	ctx := pgtest.NewContext(t)
	assetID, sigScript := mockAssetIdAndSigScript()

	tx := bc.NewTx(bc.TxData{
		Inputs: []*bc.TxInput{{
			Previous:        bc.Outpoint{Index: bc.InvalidOutputIndex},
			SignatureScript: sigScript,
			Metadata:        []byte(`{"a":"b"}`),
			AssetDefinition: []byte(`{"c":"d"}`),
		}},
		Outputs: []*bc.TxOutput{{
			AssetAmount: bc.AssetAmount{AssetID: assetID, Amount: 5},
			Metadata:    []byte{2},
			Script:      []byte("addr0"),
		}, {
			AssetAmount: bc.AssetAmount{AssetID: assetID, Amount: 6},
			Script:      []byte("addr1"),
		}},
		Metadata: []byte{0},
	})

	now := time.Now().UTC().Truncate(time.Second)
	blk := &bc.Block{
		BlockHeader: bc.BlockHeader{
			Height:    1,
			Timestamp: uint64(now.Unix()),
		},
		Transactions: []*bc.Tx{tx},
	}

	store := txdb.NewStore(pg.FromContext(ctx).(*sql.DB)) // TODO(kr): use memstore

	err := store.ApplyTx(ctx, tx, nil)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	_, err = store.ApplyBlock(ctx, blk, nil, nil, patricia.NewTree(nil))
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	got, err := GetTx(ctx, store, tx.Hash.String())
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	bh := blk.Hash()

	want := &Tx{
		ID:          tx.Hash,
		BlockID:     &bh,
		BlockHeight: 1,
		BlockTime:   now,
		Metadata:    []byte{0},
		Inputs: []*TxInput{{
			Type:     "issuance",
			AssetID:  assetID,
			Metadata: []byte(`{"a":"b"}`),
			AssetDef: []byte(`{"c":"d"}`),
		}},
		Outputs: []*TxOutput{{
			AssetID:  assetID,
			Amount:   5,
			Address:  []byte("addr0"),
			Script:   []byte("addr0"),
			Metadata: []byte{2},
		}, {
			AssetID: assetID,
			Amount:  6,
			Address: []byte("addr1"),
			Script:  []byte("addr1"),
		}},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got:\n\t%+v\nwant:\n\t%+v", got, want)
	}
}

func TestGetTxTransfer(t *testing.T) {
	ctx := pgtest.NewContext(t)
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

	store := txdb.NewStore(pg.FromContext(ctx).(*sql.DB)) // TODO(kr): use memstore
	_, err := store.ApplyBlock(ctx, blk, nil, nil, patricia.NewTree(nil))
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	got, err := GetTx(ctx, store, tx.Hash.String())
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
			TxHash:  &h0,
			TxOut:   &zero,
		}, {
			Type:    "transfer",
			AssetID: bc.AssetID([32]byte{2}),
			Amount:  &six,
			TxHash:  &h1,
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
}

func TestGetAssets(t *testing.T) {
	ctx := pgtest.NewContext(t)
	store := txdb.NewStore(pg.FromContext(ctx).(*sql.DB)) // TODO(kr): use memstore
	_, err := assettest.InitializeSigningGenerator(ctx, store)
	if err != nil {
		t.Fatal(err)
	}

	in0 := assettest.CreateIssuerNodeFixture(ctx, t, "", "in-0", nil, nil)

	asset0 := assettest.CreateAssetFixture(ctx, t, in0, "asset-0", "def-0")
	asset1 := assettest.CreateAssetFixture(ctx, t, in0, "asset-1", "def-1")

	def0 := []byte("{\n  \"s\": \"def-0\"\n}")
	defPtr0 := bc.HashAssetDefinition(def0).String()

	assettest.IssueAssetsFixture(ctx, t, asset0, 58, "")

	_, err = generator.MakeBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assettest.IssueAssetsFixture(ctx, t, asset0, 12, "")
	assettest.IssueAssetsFixture(ctx, t, asset1, 10, "")

	got, err := GetAssets(ctx, store, []string{
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
	store := txdb.NewStore(pg.FromContext(ctx).(*sql.DB)) // TODO(kr): use memstore
	_, err := assettest.InitializeSigningGenerator(ctx, store)
	if err != nil {
		t.Fatal(err)
	}

	in0 := assettest.CreateIssuerNodeFixture(ctx, t, "", "in-0", nil, nil)

	asset0 := assettest.CreateAssetFixture(ctx, t, in0, "asset-0", "def-0")
	asset1 := assettest.CreateAssetFixture(ctx, t, in0, "asset-1", "def-1")

	def0 := []byte("{\n  \"s\": \"def-0\"\n}")
	defPtr0 := bc.HashAssetDefinition(def0).String()

	assettest.IssueAssetsFixture(ctx, t, asset0, 58, "")

	_, err = generator.MakeBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}

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

		got, err := GetAsset(ctx, store, ex.id)
		if errors.Root(err) != ex.wantErr {
			t.Fatalf("error:\ngot:  %v\nwant: %v", errors.Root(err), ex.wantErr)
		}

		if !reflect.DeepEqual(got, ex.want) {
			t.Errorf("got:\n\t%+v\nwant:\n\t%+v", got, ex.want)
		}
	}
}

func TestListUTXOsByAsset(t *testing.T) {
	ctx := pgtest.NewContext(t)
	store := txdb.NewStore(pg.FromContext(ctx).(*sql.DB)) // TODO(kr): use memstore
	_, err := assettest.InitializeSigningGenerator(ctx, store)
	if err != nil {
		t.Fatal(err)
	}

	projectID := assettest.CreateProjectFixture(ctx, t, "", "")
	issuerNodeID := assettest.CreateIssuerNodeFixture(ctx, t, projectID, "", nil, nil)
	managerNodeID := assettest.CreateManagerNodeFixture(ctx, t, projectID, "", nil, nil)
	assetID := assettest.CreateAssetFixture(ctx, t, issuerNodeID, "", "")
	accountID := assettest.CreateAccountFixture(ctx, t, managerNodeID, "", nil)

	tx := assettest.Issue(ctx, t, assetID, []*txbuilder.Destination{
		assettest.AccountDest(ctx, t, accountID, assetID, 1),
	})

	_, err = generator.MakeBlock(ctx)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	zero := uint32(0)

	want := []*TxOutput{{
		TxHash:   &tx.Hash,
		TxIndex:  &zero,
		AssetID:  assetID,
		Amount:   1,
		Address:  tx.Outputs[0].Script,
		Script:   tx.Outputs[0].Script,
		Metadata: []byte{},
	}}

	got, gotLast, err := ListUTXOsByAsset(ctx, store, assetID, "", 10000)
	if err != nil {
		t.Fatal("unexpected error: ", err)
	}

	if !reflect.DeepEqual(got, want) {
		gotStr, err := json.MarshalIndent(got, "", "  ")
		if err != nil {
			t.Fatal("unexpected error: ", err)
		}

		wantStr, err := json.MarshalIndent(want, "", "  ")
		if err != nil {
			t.Fatal("unexpected error: ", err)
		}

		t.Errorf("txs:\ngot:\n%s\nwant:\n%s", string(gotStr), string(wantStr))
	}

	// block height is unpredictable in this test file
	if !strings.HasSuffix(gotLast, "-0-0") {
		t.Errorf("last: got=%s should-end-with= -0-0", gotLast)
	}
}

func mockAssetIdAndSigScript() (bc.AssetID, []byte) {
	builder := txscript.NewScriptBuilder()
	builder.AddOp(txscript.OP_FALSE)
	script, err := builder.Script()
	if err != nil {
		panic(err)
	}

	redeemScript, err := txscript.RedeemScriptFromP2SHSigScript(script)
	if err != nil {
		panic(err)
	}
	pkScript := txscript.RedeemToPkScript(redeemScript)
	assetID := bc.ComputeAssetID(pkScript, [32]byte{})

	return assetID, script
}

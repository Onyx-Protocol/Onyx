package asset

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/testutil"
)

var (
	assetID1 = bc.AssetID{0x01}
	assetID2 = bc.AssetID{0x02}
	assetID3 = bc.AssetID{0x03}
)

func TestAssetDefinitions(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	createAssetDefFixture(ctx, t, assetID1, []byte("asset-1-def"))
	createAssetDefFixture(ctx, t, assetID2, []byte("asset-2-def"))

	examples := []struct {
		assetIDs []bc.AssetID
		want     map[bc.AssetID][]byte
	}{
		{
			[]bc.AssetID{assetID1},
			map[bc.AssetID][]byte{
				assetID1: []byte("asset-1-def"),
			},
		},
		{
			[]bc.AssetID{assetID1, assetID2, assetID3},
			map[bc.AssetID][]byte{
				assetID1: []byte("asset-1-def"),
				assetID2: []byte("asset-2-def"),
			},
		},
		{
			[]bc.AssetID{assetID3},
			map[bc.AssetID][]byte{},
		},
		{
			nil,
			map[bc.AssetID][]byte{},
		},
	}

	for _, ex := range examples {
		t.Log("Example:", ex.assetIDs)

		got, err := Definitions(ctx, ex.assetIDs)
		if err != nil {
			t.Fatal("unexpected error: ", err)
		}
		if !reflect.DeepEqual(got, ex.want) {
			t.Errorf("result:\ngot:  %v\nwant: %v", got, ex.want)
		}
	}
}

func TestInsertAssetDefinitions(t *testing.T) {
	defs := [][]byte{
		[]byte(`{"name": "asset 1"}`),
		[]byte(`{"name": "asset 2"}`),
	}

	var (
		hashes []string
		txs    []*bc.Tx
	)
	for _, d := range defs {
		hashes = append(hashes, bc.HashAssetDefinition(d).String())

		tx := bc.NewTx(bc.TxData{
			Inputs: []*bc.TxInput{
				bc.NewIssuanceInput(time.Now(), time.Now().Add(time.Hour), bc.Hash{}, 0, nil, d, nil, nil),
			},
		})
		txs = append(txs, tx)
	}

	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	block := &bc.Block{Transactions: txs}
	err := saveAssetDefinitions(ctx, block)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	var count int
	var checkQ = `SELECT COUNT(*) FROM asset_definitions`
	err = pg.QueryRow(ctx, checkQ).Scan(&count)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if count != len(defs) {
		t.Fatalf("result count got=%d, want=%d", count, len(defs))
	}

	for i := range defs {
		var got []byte
		const selectQ = `SELECT definition FROM asset_definitions WHERE hash=$1`
		err = pg.QueryRow(ctx, selectQ, hashes[i]).Scan(&got)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, defs[i]) {
			t.Fatalf("inserted definition %q want %q", got, defs[i])
		}
	}
}

func TestInsertAssetDefinitionsIdempotent(t *testing.T) {
	def := []byte("{'key': 'im totally json'}")
	hash := bc.HashAssetDefinition(def).String()

	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	block := &bc.Block{
		Transactions: []*bc.Tx{
			bc.NewTx(bc.TxData{Inputs: []*bc.TxInput{
				bc.NewIssuanceInput(time.Now(), time.Now().Add(time.Hour), bc.Hash{}, 0, nil, def, nil, nil),
			}}),
		},
	}
	err := saveAssetDefinitions(ctx, block)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	// Just do it again
	err = saveAssetDefinitions(ctx, block)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	var count int
	var checkQ = `
			SELECT COUNT(*) FROM asset_definitions
		`
	err = pg.QueryRow(ctx, checkQ).Scan(&count)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if count != 1 {
		t.Fatalf("checking results, want=1, got=%d", count)
	}

	var got []byte
	const selectQ = `SELECT definition FROM asset_definitions WHERE hash=$1`
	err = pg.QueryRow(ctx, selectQ, hash).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, def) {
		t.Fatalf("inserted definition %q want %q", got, def)
	}
}

func TestInsertAssetDefinitionsDuplicates(t *testing.T) {
	def := []byte("{'key': 'im totally json'}")
	hash := bc.HashAssetDefinition(def).String()

	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	now := time.Now()
	block := &bc.Block{
		Transactions: []*bc.Tx{
			bc.NewTx(bc.TxData{Inputs: []*bc.TxInput{
				bc.NewIssuanceInput(now, now.Add(time.Hour), bc.Hash{}, 0, nil, def, nil, nil),
				bc.NewIssuanceInput(now, now.Add(time.Hour), bc.Hash{}, 0, nil, def, nil, nil),
			}}),
		},
	}
	err := saveAssetDefinitions(ctx, block)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	var count int
	var checkQ = `
			SELECT COUNT(*) FROM asset_definitions
		`
	err = pg.QueryRow(ctx, checkQ).Scan(&count)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if count != 1 {
		t.Fatalf("checking results, want=1, got=%d", count)
	}

	var got []byte
	const selectQ = `SELECT definition FROM asset_definitions WHERE hash=$1`
	err = pg.QueryRow(ctx, selectQ, hash).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, def) {
		t.Fatalf("inserted definition %q want %q", got, def)
	}
}

func createAssetDefFixture(ctx context.Context, t *testing.T, assetID bc.AssetID, def []byte) {
	h := bc.HashAssetDefinition(def)

	const q1 = `
		INSERT INTO asset_definition_pointers (asset_id, asset_definition_hash)
		VALUES ($1, $2)
	`
	_, err := pg.Exec(ctx, q1, assetID, h)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	const q2 = `
		INSERT INTO asset_definitions (hash, definition)
		VALUES ($1, $2)
	`
	_, err = pg.Exec(ctx, q2, h, def)
	if err != nil {
		testutil.FatalErr(t, err)
	}
}

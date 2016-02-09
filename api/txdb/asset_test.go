package txdb

import (
	"bytes"
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/fedchain/bc"
	"chain/testutil"
)

func TestAssetDefinitions(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		createAssetDefFixture(ctx, t, "asset-1", []byte("asset-1-def"))
		createAssetDefFixture(ctx, t, "asset-2", []byte("asset-2-def"))

		examples := []struct {
			assetIDs []string
			want     map[string][]byte
		}{
			{
				[]string{"asset-1"},
				map[string][]byte{
					"asset-1": []byte("asset-1-def"),
				},
			},
			{
				[]string{"asset-1", "asset-2", "asset-3"},
				map[string][]byte{
					"asset-1": []byte("asset-1-def"),
					"asset-2": []byte("asset-2-def"),
				},
			},
			{
				[]string{"asset-3"},
				map[string][]byte{},
			},
			{
				nil,
				map[string][]byte{},
			},
		}

		for _, ex := range examples {
			t.Log("Example:", ex.assetIDs)

			got, err := AssetDefinitions(ctx, ex.assetIDs)
			if err != nil {
				t.Fatal("unexpected error: ", err)
			}

			if !reflect.DeepEqual(got, ex.want) {
				t.Errorf("result:\ngot:  %v\nwant: %v", got, ex.want)
			}
		}
	})
}

func TestInsertAssetDefinitionPointers(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		// These hash values are arbitrary.
		a0str := "a55e710000000000000000000000000000000000000000000000000000000000"
		def0str := "341fb89912be0110b527375998810c99ac96a317c63b071ccf33b7514cf0f0a5"
		a0 := bc.AssetID(mustParseHash(a0str))
		def0 := mustParseHash(def0str)
		adp0 := &bc.AssetDefinitionPointer{
			AssetID:        a0,
			DefinitionHash: def0,
		}

		a1str := "df03f294bd08930f542a42b91199a8afe1b45c28eeb058cc5e8c8d600e0dd42f"
		def1str := "5abad6dfb0de611046ebda5de05bfebc6a08d9a71831b43f2acd554bf54f3318"
		a1 := bc.AssetID(mustParseHash(a1str))
		def1 := mustParseHash(def1str)
		adp1 := &bc.AssetDefinitionPointer{
			AssetID:        a1,
			DefinitionHash: def1,
		}

		adps := make(map[bc.AssetID]*bc.AssetDefinitionPointer)
		adps[a0] = adp0
		adps[a1] = adp1

		err := insertAssetDefinitionPointers(ctx, adps)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}

		var resID string
		const checkQ = `
			SELECT asset_definition_hash FROM asset_definition_pointers WHERE asset_id=$1
		`
		err = pg.FromContext(ctx).QueryRow(ctx, checkQ, a0str).Scan(&resID)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		if resID != def0str {
			t.Fatalf("checking inputs, want=%s, got=%s", def0str, resID)
		}
		err = pg.FromContext(ctx).QueryRow(ctx, checkQ, a1str).Scan(&resID)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		if resID != def1str {
			t.Fatalf("checking inputs, want=%s, got=%s", def1str, resID)
		}
	})
}

func TestInsertAssetDefinitionPointersWithUpdate(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		// These hash values are arbitrary.
		a0str := "a55e710000000000000000000000000000000000000000000000000000000000"
		def0str := "341fb89912be0110b527375998810c99ac96a317c63b071ccf33b7514cf0f0a5"
		a0 := bc.AssetID(mustParseHash(a0str))
		def0 := mustParseHash(def0str)
		adp0 := &bc.AssetDefinitionPointer{
			AssetID:        a0,
			DefinitionHash: def0,
		}

		// a1 is the same as a0, so should overwrite.
		a1str := "a55e710000000000000000000000000000000000000000000000000000000000"
		def1str := "5abad6dfb0de611046ebda5de05bfebc6a08d9a71831b43f2acd554bf54f3318"
		a1 := bc.AssetID(mustParseHash(a1str))
		def1 := mustParseHash(def1str)
		adp1 := &bc.AssetDefinitionPointer{
			AssetID:        a1,
			DefinitionHash: def1,
		}

		adps := make(map[bc.AssetID]*bc.AssetDefinitionPointer)
		adps[a0] = adp0

		err := insertAssetDefinitionPointers(ctx, adps)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}

		delete(adps, a0)
		adps[a1] = adp1
		err = insertAssetDefinitionPointers(ctx, adps)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}

		var count int
		const checkQ = `
			SELECT COUNT(*) FROM asset_definition_pointers
		`
		err = pg.FromContext(ctx).QueryRow(ctx, checkQ).Scan(&count)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		if count != 1 {
			t.Fatalf("checking results, want=1, got=%d", count)
		}
	})
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
				{
					AssetDefinition: d,
					Previous:        bc.Outpoint{Index: bc.InvalidOutputIndex},
				},
			},
		})
		txs = append(txs, tx)
	}

	withContext(t, "", func(ctx context.Context) {
		block := &bc.Block{Transactions: txs}
		err := insertAssetDefinitions(ctx, block)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}

		var count int
		var checkQ = `SELECT COUNT(*) FROM asset_definitions`
		err = pg.FromContext(ctx).QueryRow(ctx, checkQ).Scan(&count)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		if count != len(defs) {
			t.Fatalf("result count got=%d, want=%d", count, len(defs))
		}

		for i := range defs {
			var got []byte
			const selectQ = `SELECT definition FROM asset_definitions WHERE hash=$1`
			err = pg.FromContext(ctx).QueryRow(ctx, selectQ, hashes[i]).Scan(&got)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, defs[i]) {
				t.Fatalf("inserted definition %q want %q", got, defs[i])
			}
		}
	})
}

func TestInsertAssetDefinitionsIdempotent(t *testing.T) {
	def := []byte("{'key': 'im totally json'}")
	hash := bc.HashAssetDefinition(def).String()

	withContext(t, "", func(ctx context.Context) {
		block := &bc.Block{
			Transactions: []*bc.Tx{
				bc.NewTx(bc.TxData{Inputs: []*bc.TxInput{
					{
						AssetDefinition: def,
						Previous:        bc.Outpoint{Index: bc.InvalidOutputIndex},
					},
				}}),
			},
		}
		err := insertAssetDefinitions(ctx, block)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		// Just do it again
		err = insertAssetDefinitions(ctx, block)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}

		var count int
		var checkQ = `
			SELECT COUNT(*) FROM asset_definitions
		`
		err = pg.FromContext(ctx).QueryRow(ctx, checkQ).Scan(&count)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		if count != 1 {
			t.Fatalf("checking results, want=1, got=%d", count)
		}

		var got []byte
		const selectQ = `SELECT definition FROM asset_definitions WHERE hash=$1`
		err = pg.FromContext(ctx).QueryRow(ctx, selectQ, hash).Scan(&got)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, def) {
			t.Fatalf("inserted definition %q want %q", got, def)
		}
	})
}

func TestInsertAssetDefinitionsDuplicates(t *testing.T) {
	def := []byte("{'key': 'im totally json'}")
	hash := bc.HashAssetDefinition(def).String()

	withContext(t, "", func(ctx context.Context) {
		block := &bc.Block{
			Transactions: []*bc.Tx{
				bc.NewTx(bc.TxData{Inputs: []*bc.TxInput{
					{
						AssetDefinition: def,
						Previous:        bc.Outpoint{Index: bc.InvalidOutputIndex},
					},
					{
						AssetDefinition: def, // duplicate
						Previous:        bc.Outpoint{Index: bc.InvalidOutputIndex},
					},
				}}),
			},
		}
		err := insertAssetDefinitions(ctx, block)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}

		var count int
		var checkQ = `
			SELECT COUNT(*) FROM asset_definitions
		`
		err = pg.FromContext(ctx).QueryRow(ctx, checkQ).Scan(&count)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		if count != 1 {
			t.Fatalf("checking results, want=1, got=%d", count)
		}

		var got []byte
		const selectQ = `SELECT definition FROM asset_definitions WHERE hash=$1`
		err = pg.FromContext(ctx).QueryRow(ctx, selectQ, hash).Scan(&got)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, def) {
			t.Fatalf("inserted definition %q want %q", got, def)
		}
	})
}

func createAssetDefFixture(ctx context.Context, t *testing.T, assetID string, def []byte) {
	h := bc.HashAssetDefinition(def)

	const q1 = `
		INSERT INTO asset_definition_pointers (asset_id, asset_definition_hash)
		VALUES ($1, $2)
	`
	_, err := pg.FromContext(ctx).Exec(ctx, q1, assetID, h)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	const q2 = `
		INSERT INTO asset_definitions (hash, definition)
		VALUES ($1, $2)
	`
	_, err = pg.FromContext(ctx).Exec(ctx, q2, h, def)
	if err != nil {
		testutil.FatalErr(t, err)
	}

}

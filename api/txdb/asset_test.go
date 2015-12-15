package txdb

import (
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/fedchain/bc"
)

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

		err := InsertAssetDefinitionPointers(ctx, adps)
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

		err := InsertAssetDefinitionPointers(ctx, adps)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}

		delete(adps, a0)
		adps[a1] = adp1
		err = InsertAssetDefinitionPointers(ctx, adps)
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
	withContext(t, "", func(ctx context.Context) {
		block := &bc.Block{
			Transactions: []*bc.Tx{
				bc.NewTx(bc.TxData{Inputs: []*bc.TxInput{
					{
						Metadata: []byte("{'key': 'im totally json'}"),
						Previous: bc.Outpoint{Index: bc.InvalidOutputIndex},
					},
				}}),
			},
		}
		err := InsertAssetDefinitions(ctx, block)
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
	})
}

func TestInsertAssetDefinitionsWithUpdate(t *testing.T) {
	withContext(t, "", func(ctx context.Context) {
		block := &bc.Block{
			Transactions: []*bc.Tx{
				bc.NewTx(bc.TxData{Inputs: []*bc.TxInput{
					{
						Metadata: []byte("{'key': 'im totally json'}"),
						Previous: bc.Outpoint{Index: bc.InvalidOutputIndex},
					},
				}}),
			},
		}
		err := InsertAssetDefinitions(ctx, block)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		// Just do it again
		err = InsertAssetDefinitions(ctx, block)
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
	})
}

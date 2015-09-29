package asset

import (
	"encoding/hex"
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/database/pg/pgtest"
)

func TestCreate(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, `
		INSERT INTO issuer_nodes (id, project_id, label, keyset)
		VALUES ('ag1', 'a1', 'foo', '{xpub661MyMwAqRbcGKBeRA9p52h7EueXnRWuPxLz4Zoo1ZCtX8CJR5hrnwvSkWCDf7A9tpEZCAcqex6KDuvzLxbxNZpWyH6hPgXPzji9myeqyHd}');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	asset, err := Create(ctx, "ag1", "fooAsset")
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	wantID := "7b97c04149f6f932a12b5e2f4149aa2024b48f5fdb1be383592f1f30af5aa8ab"
	if asset.Hash.String() != wantID {
		t.Errorf("got asset id = %v want %v", asset.Hash.String(), wantID)
	}

	wantRedeem := "51210371fe1fe0352f0cea91344d06c9d9b16e394e1945ee0f3063c2f9891d163f0f5551ae"
	if hex.EncodeToString(asset.RedeemScript) != wantRedeem {
		t.Errorf("got redeem script = %x want %v", asset.RedeemScript, wantRedeem)
	}

	if asset.Label != "fooAsset" {
		t.Errorf("got label = %v want %v", asset.Label, "fooAsset")
	}
}

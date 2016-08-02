package issuer

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/core/appdb"
	"chain/cos/bc"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/testutil"
)

func TestCreateAsset(t *testing.T) {
	dbtx := pgtest.NewTx(t)
	ctx := pg.NewContext(context.Background(), dbtx)
	pgtest.Exec(ctx, t, `
		ALTER SEQUENCE issuer_nodes_key_index_seq RESTART;
		ALTER SEQUENCE assets_key_index_seq RESTART;
	`)
	pgtest.Exec(ctx, t, fmt.Sprintf(`
		INSERT INTO issuer_nodes (id, project_id, label, keyset)
		VALUES ('in1', 'a1', 'foo', '{%s}');
	`, testutil.TestXPub.String()))

	clientToken := "a-client-provided-unique-token"
	definition := make(map[string]interface{})
	asset, err := CreateAsset(ctx, "in1", "fooAsset", bc.Hash{}, definition, &clientToken)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	wantID := "d214945af36ad07cc9738ecdbd5a4f2da59a5ffe1ba03e0bd44b663b2a88c695"
	if asset.Hash.String() != wantID {
		t.Errorf("got asset id = %v want %v", asset.Hash.String(), wantID)
	}

	wantRedeem := "5120ca7313d5998f6005cf5a9c29677c31adfc163f599412a6ba4e9bb19d361bf4f451ae"
	if hex.EncodeToString(asset.RedeemScript) != wantRedeem {
		t.Errorf("got redeem script = %x want %v", asset.RedeemScript, wantRedeem)
	}

	if asset.Label != "fooAsset" {
		t.Errorf("got label = %v want %v", asset.Label, "fooAsset")
	}

	wantIssuance := "76aa20d576c32879648a54df281c7839ff77a0e8315ed8fa3d34a3eb7dce22634f3d128800c0"
	if hex.EncodeToString(asset.IssuanceScript) != wantIssuance {
		t.Errorf("got issuance script=%x want=%v", asset.IssuanceScript, wantIssuance)
	}

	// Try to create the same asset again, and ensure that it returns the
	// original asset.
	newAsset, err := CreateAsset(ctx, "in1", "fooAsset2", bc.Hash{}, definition, &clientToken)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if !reflect.DeepEqual(asset, newAsset) {
		t.Errorf("got new asset = %#v want original asset with same client_token = %#v", newAsset, asset)
	}
}

func TestCreateDefs(t *testing.T) {
	fix := fmt.Sprintf(`
		INSERT INTO issuer_nodes (id, project_id, label, keyset)
		VALUES ('inode-0', 'proj-0', 'label-0', '{%s}');
	`, testutil.TestXPub.String())

	examples := []struct {
		def  map[string]interface{}
		want []byte
	}{
		// blank def
		{nil, nil},

		// empty JSON def
		{make(map[string]interface{}), []byte(`{}`)},

		// non-empty JSON def (whitespace matters)
		{map[string]interface{}{"foo": "bar"}, []byte(`{
  "foo": "bar"
}`,
		)},
	}

	for i, ex := range examples {
		clientToken := fmt.Sprintf("example-%d", i)

		dbtx := pgtest.NewTx(t)
		ctx := pg.NewContext(context.Background(), dbtx)
		pgtest.Exec(ctx, t, fix)
		gotCreated, err := CreateAsset(ctx, "inode-0", "label", bc.Hash{}, ex.def, &clientToken)
		if err != nil {
			t.Fatal("unexpected error: ", err)
		}

		if !bytes.Equal(gotCreated.Definition, ex.want) {
			t.Errorf("create result:\ngot:  %s\nwant: %s", gotCreated.Definition, ex.want)
		}

		gotFetch, err := appdb.AssetByID(ctx, gotCreated.Hash)
		if err != nil {
			t.Fatal("unexpected error: ", err)
		}

		if !bytes.Equal(gotFetch.Definition, ex.want) {
			t.Errorf("db fetch result:\ngot:  %s\nwant: %s", gotFetch.Definition, ex.want)
		}
	}
}

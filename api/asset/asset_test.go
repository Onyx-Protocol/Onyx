package asset

import (
	"bytes"
	"encoding/hex"
	"log"
	"os"
	"testing"

	"golang.org/x/net/context"

	"chain/api/appdb"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
)

func init() {
	u := "postgres:///api-test?sslmode=disable"
	if s := os.Getenv("DB_URL_TEST"); s != "" {
		u = s
	}

	db := pgtest.Open(u, "assettest", "../appdb/schema.sql")
	err := appdb.Init(db)
	if err != nil {
		log.Fatal(err)
	}
}

func TestOutputPkScript(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, `
		INSERT INTO applications (id, name) VALUES ('app-id-0', 'app-0');
		INSERT INTO keys (id, xpub) VALUES(
			'fda6bac8e1901cbc4813e729d3d766988b8b1ac7',
			'xpub661MyMwAqRbcGKBeRA9p52h7EueXnRWuPxLz4Zoo1ZCtX8CJR5hrnwvSkWCDf7A9tpEZCAcqex6KDuvzLxbxNZpWyH6hPgXPzji9myeqyHd'
		);
		INSERT INTO wallets (id, application_id, label, current_rotation)
			VALUES('w1', 'app-id-0', 'w1', 'rot1');
		INSERT INTO rotations (id, wallet_id, keyset)
			VALUES('rot1', 'w1', '{fda6bac8e1901cbc4813e729d3d766988b8b1ac7}');
		INSERT INTO buckets (id, wallet_id, key_index)
			VALUES('b1', 'w1', 0);
	`)
	defer dbtx.Rollback()

	// Test bucket output pk script (address creation)
	var (
		out = &Output{BucketID: "b1"}
		ctx = pg.NewContext(context.Background(), dbtx)
	)
	got, err := out.PkScript(ctx)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	want, _ := hex.DecodeString("a9140ac9c982fd389181752e5a414045dd424a10754b87")
	if !bytes.Equal(got, want) {
		t.Errorf("got pkscript = %x want %x", got, want)
	}

	// Test stringified address output
	out = &Output{Address: "32g4QsxVQrhZeXyXTUnfSByNBAdTfVUdVK"}
	got, err = out.PkScript(ctx)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	if !bytes.Equal(got, want) {
		t.Errorf("got pkscript = %x want %x", got, want)
	}
}

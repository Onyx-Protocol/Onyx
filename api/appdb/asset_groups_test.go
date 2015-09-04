package appdb

import (
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/database/pg/pgtest"
)

func TestCreateAssetGroup(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	id, err := CreateAssetGroup(ctx, "a1", "foo", []*Key{dummyXPub})
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if id == "" {
		t.Errorf("got empty asset group id")
	}
}

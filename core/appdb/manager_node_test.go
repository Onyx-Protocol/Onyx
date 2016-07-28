package appdb_test

import (
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/database/pg/pgtest"
)

func TestInsertManagerNode(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	newTestManagerNode(t, ctx, nil, "foo")
}

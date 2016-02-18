package asset_test

import (
	"os"
	"testing"

	"golang.org/x/net/context"

	. "chain/api/asset"
	"chain/api/txdb"
	"chain/database/pg/pgtest"
	"chain/fedchain"
	"chain/fedchain/bc"
)

func init() {
	Init(fedchain.New(txdb.NewStore(), nil), nil, true)
	u := "postgres:///api-test?sslmode=disable"
	if s := os.Getenv("DB_URL_TEST"); s != "" {
		u = s
	}

	ctx := context.Background()
	pgtest.Open(ctx, u, "assettest", "../appdb/schema.sql")
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

func mustParseHash(s string) [32]byte {
	h, err := bc.ParseHash(s)
	if err != nil {
		panic(err)
	}
	return h
}

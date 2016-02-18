package api

import (
	"os"

	"golang.org/x/net/context"

	"chain/api/asset"
	"chain/api/smartcontracts/orderbook"
	"chain/api/txdb"
	"chain/database/pg/pgtest"
	"chain/fedchain"
)

func init() {
	fc := fedchain.New(txdb.NewStore(), nil)
	asset.Init(fc, nil, true)
	orderbook.ConnectFedchain(fc)
	u := "postgres:///api-test?sslmode=disable"
	if s := os.Getenv("DB_URL_TEST"); s != "" {
		u = s
	}

	ctx := context.Background()
	pgtest.Open(ctx, u, "apitest", "appdb/schema.sql")
}

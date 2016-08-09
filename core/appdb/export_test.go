package appdb

import (
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg/pgtest"
)

func ResetSeqs(ctx context.Context, t testing.TB) {
	pgtest.Exec(ctx, t, `ALTER SEQUENCE assets_key_index_seq RESTART`)
	pgtest.Exec(ctx, t, `ALTER SEQUENCE issuer_nodes_key_index_seq RESTART`)
}

func KeyIndex(n int64) []uint32 {
	return keyIndex(n)
}

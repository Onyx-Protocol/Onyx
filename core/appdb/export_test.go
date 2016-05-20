package appdb

import (
	"testing"
	"time"

	"golang.org/x/net/context"

	"chain/database/pg/pgtest"
)

func ResetSeqs(ctx context.Context, t testing.TB) {
	addrIndexNext, addrIndexCap = 1, 100
	pgtest.Exec(ctx, t, `ALTER SEQUENCE address_index_seq RESTART`)
	pgtest.Exec(ctx, t, `ALTER SEQUENCE assets_key_index_seq RESTART`)
	pgtest.Exec(ctx, t, `ALTER SEQUENCE issuer_nodes_key_index_seq RESTART`)
	pgtest.Exec(ctx, t, `ALTER SEQUENCE manager_nodes_key_index_seq RESTART`)
}

func DeleteInvitation(ctx context.Context, invID string) error {
	return deleteInvitation(ctx, invID)
}

func KeyIndex(n int64) []uint32 {
	return keyIndex(n)
}

func CheckPassword(ctx context.Context, id, password string) error {
	return checkPassword(ctx, id, password)
}

func SetPasswordBCryptCost(cost int) {
	passwordBcryptCost = cost
}

func PWResetLifeTime() time.Duration {
	return pwResetLiftime
}

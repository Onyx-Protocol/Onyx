package appdb

import (
	"testing"
	"time"

	"golang.org/x/net/context"
)

func ResetSeqs(ctx context.Context, t testing.TB) {
	addrIndexNext, addrIndexCap = 1, 100
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

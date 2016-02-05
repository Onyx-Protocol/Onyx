package appdb

import "golang.org/x/net/context"

func ResetAddrIndex() {
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

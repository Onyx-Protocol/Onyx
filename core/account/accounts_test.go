package account

import (
	"bytes"
	"testing"

	"golang.org/x/net/context"

	"chain/core/signers"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/testutil"
)

var dummyXPub = testutil.TestXPub.String()

func resetSeqs(ctx context.Context, t testing.TB) {
	acpIndexNext, acpIndexCap = 1, 100
	pgtest.Exec(ctx, t, `ALTER SEQUENCE account_control_program_seq RESTART`)
	pgtest.Exec(ctx, t, `ALTER SEQUENCE signers_key_index_seq RESTART`)
}

func TestCreateControlProgram(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	resetSeqs(ctx, t)

	account, err := Create(ctx, []string{dummyXPub}, 1, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	got, err := CreateControlProgram(ctx, account.ID)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	want := []byte{
		118, 170, 32, 123, 19, 122, 245, 23, 227, 191, 180, 205, 25, 60,
		39, 118, 189, 113, 236, 68, 161, 78, 45, 172, 155, 195, 153, 145,
		125, 247, 89, 148, 224, 23, 127, 136, 0, 192,
	}

	if !bytes.Equal(got, want) {
		t.Errorf("got control program = %x want %x", got, want)
	}
}

func createTestAccount(ctx context.Context, t testing.TB) *signers.Signer {
	account, err := Create(ctx, []string{dummyXPub}, 1, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	return account
}

func createTestControlProgram(ctx context.Context, t testing.TB, accountID string) []byte {
	if accountID == "" {
		account := createTestAccount(ctx, t)
		accountID = account.ID
	}

	acp, err := CreateControlProgram(ctx, accountID)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	return acp
}

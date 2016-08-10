package account

import (
	"bytes"
	"reflect"
	"strconv"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"golang.org/x/net/context"

	"chain/cos/txscript"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/testutil"
)

var dummyXPub = testutil.TestXPub.String()

func TestCreateAccount(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)

	account, err := Create(ctx, []string{dummyXPub}, 1, nil, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// Verify that the account was defined.
	var id string
	var checkQ = `SELECT id FROM signers`
	err = pg.QueryRow(ctx, checkQ).Scan(&id)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if id != account.ID {
		t.Errorf("expected account %s to be recorded as %s", account.ID, id)
	}
}

func resetSeqs(ctx context.Context, t testing.TB) {
	acpIndexNext, acpIndexCap = 1, 100
	pgtest.Exec(ctx, t, `ALTER SEQUENCE account_control_program_seq RESTART`)
	pgtest.Exec(ctx, t, `ALTER SEQUENCE signers_key_index_seq RESTART`)
}

func TestCreateControlProgram(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	resetSeqs(ctx, t)

	account, err := Create(ctx, []string{dummyXPub}, 1, nil, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	got, err := CreateControlProgram(ctx, account.ID)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	want, err := txscript.ParseScriptString("OP_DUP OP_SHA3 OP_DATA_32 0x963e9956eabe4610b042a3b085a1dc648e7fd87298b51d0369e2f66446338739 OP_EQUALVERIFY 0 OP_CHECKPREDICATE")
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if !bytes.Equal(got, want) {
		t.Errorf("got control program = %x want %x", got, want)
	}
}

func createTestAccount(ctx context.Context, t testing.TB, tags map[string]interface{}) *Account {
	account, err := Create(ctx, []string{dummyXPub}, 1, tags, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	return account
}

func createTestControlProgram(ctx context.Context, t testing.TB, accountID string) []byte {
	if accountID == "" {
		account := createTestAccount(ctx, t, nil)
		accountID = account.ID
	}

	acp, err := CreateControlProgram(ctx, accountID)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	return acp
}

func TestSetTags(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	account := createTestAccount(ctx, t, nil)
	newTags := map[string]interface{}{"someTag": "taggityTag"}

	got, err := SetTags(ctx, account.ID, newTags)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	account.Tags = newTags
	if !reflect.DeepEqual(got, account) {
		t.Errorf("got SetTags=%v, want %v", got, account)
	}
}

func TestFind(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	tags := map[string]interface{}{"someTag": "taggityTag"}
	account := createTestAccount(ctx, t, tags)

	found, err := Find(ctx, account.ID)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if !reflect.DeepEqual(account, found) {
		t.Errorf("expected found account to be %v, instead found %v", account, found)
	}
}

func TestList(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)

	accounts := make(map[string]*Account)
	for i := 0; i < 3; i++ {
		tags := map[string]interface{}{"number": strconv.Itoa(i)}
		account := createTestAccount(ctx, t, tags)
		accounts[account.ID] = account
	}

	found, last, err := List(ctx, "", 3)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	for i, f := range found {
		for id, account := range accounts {
			if f.ID != id {
				continue
			}

			if !reflect.DeepEqual(f, account) {
				t.Fatalf("List(ctx, \"\", 3)=found; found[%d]=%v, want %v", i, spew.Sdump(f), spew.Sdump(account))
			}

			delete(accounts, id)
		}

		if i == len(found)-1 {
			if last != f.ID {
				t.Errorf("`last` doesn't match last ID. Got last=%s, want %s", last, f.ID)
			}
		}
	}

	// Make sure we used up everything in aMap
	if len(accounts) != 0 {
		t.Error("Didn't find all the assets.")
	}
}

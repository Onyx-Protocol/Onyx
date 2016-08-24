package account

import (
	"bytes"
	"context"
	"reflect"
	"strconv"
	"testing"

	"chain/core/signers"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/net/http/httpjson"
	"chain/protocol/vm"
	"chain/testutil"
)

var dummyXPub = testutil.TestXPub.String()

func TestCreateAccount(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)

	account, err := Create(ctx, []string{dummyXPub}, 1, "", nil, nil)
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

func TestCreateAccountReusedAlias(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	createTestAccount(ctx, t, "some-account", nil)

	_, err := Create(ctx, []string{dummyXPub}, 1, "some-account", nil, nil)
	if errors.Root(err) != httpjson.ErrBadRequest {
		t.Errorf("Expected %s when reusing an alias, got %v", httpjson.ErrBadRequest, err)
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

	account, err := Create(ctx, []string{dummyXPub}, 1, "", nil, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	got, err := CreateControlProgram(ctx, account.ID)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	want, err := vm.Compile("DUP SHA3 0x963e9956eabe4610b042a3b085a1dc648e7fd87298b51d0369e2f66446338739 EQUALVERIFY 0 CHECKPREDICATE")
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if !bytes.Equal(got, want) {
		t.Errorf("got control program = %x want %x", got, want)
	}
}

func createTestAccount(ctx context.Context, t testing.TB, alias string, tags map[string]interface{}) *Account {
	account, err := Create(ctx, []string{dummyXPub}, 1, alias, tags, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	return account
}

func createTestControlProgram(ctx context.Context, t testing.TB, accountID string) []byte {
	if accountID == "" {
		account := createTestAccount(ctx, t, "", nil)
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
	account := createTestAccount(ctx, t, "some-alias", nil)
	newTags := map[string]interface{}{"someTag": "taggityTag"}

	// first, set by ID
	got, err := SetTags(ctx, account.ID, "", newTags)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	account.Tags = newTags
	if !reflect.DeepEqual(got, account) {
		t.Errorf("got SetTags=%v, want %v", got, account)
	}

	newTags = map[string]interface{}{"someTag": "something different"}
	// next, set by alias
	got, err = SetTags(ctx, "", "some-alias", newTags)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	account.Tags = newTags
	if !reflect.DeepEqual(got, account) {
		t.Errorf("got SetTags=%v, want %v", got, account)
	}
}

func TestFindByID(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	tags := map[string]interface{}{"someTag": "taggityTag"}
	account := createTestAccount(ctx, t, "", tags)

	found, err := FindByID(ctx, account.ID)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if !reflect.DeepEqual(account, found) {
		t.Errorf("expected found account to be %v, instead found %v", account, found)
	}
}

func TestFindByAlias(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	tags := map[string]interface{}{"someTag": "taggityTag"}
	account := createTestAccount(ctx, t, "some-alias", tags)

	found, err := FindByAlias(ctx, "some-alias")
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if !reflect.DeepEqual(account, found) {
		t.Errorf("expected found account to be %v, instead found %v", account, found)
	}
}

func TestFindBatch(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)

	var accountIDs []string
	for i := 0; i < 3; i++ {
		tags := map[string]interface{}{"number": strconv.Itoa(i)}
		account := createTestAccount(ctx, t, "", tags)
		accountIDs = append(accountIDs, account.ID)
	}

	accs, err := FindBatch(ctx, accountIDs...)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	if len(accs) != len(accountIDs) {
		t.Errorf("got %d account IDs, want %d", len(accs), len(accountIDs))
	}
}

func TestArchiveByID(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	account := createTestAccount(ctx, t, "", nil)

	err := Archive(ctx, account.ID, "")
	if err != nil {
		testutil.FatalErr(t, err)
	}

	_, err = FindByID(ctx, account.ID)
	if errors.Root(err) != signers.ErrArchived {
		t.Errorf("expected %s when Finding an archived account, instead got %s", signers.ErrArchived, err)
	}
}

func TestArchiveByAlias(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	createTestAccount(ctx, t, "some-alias", nil)

	err := Archive(ctx, "", "some-alias")
	if err != nil {
		testutil.FatalErr(t, err)
	}

	_, err = FindByAlias(ctx, "some-alias")
	if errors.Root(err) != signers.ErrArchived {
		t.Errorf("expected %s when Finding an archived account, instead got %s", signers.ErrArchived, err)
	}
}

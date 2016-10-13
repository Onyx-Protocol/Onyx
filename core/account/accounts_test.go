package account

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/net/http/httpjson"
	"chain/protocol/prottest"
	"chain/protocol/vm"
	"chain/testutil"
)

var dummyXPub = testutil.TestXPub.String()

func TestCreateAccount(t *testing.T) {
	dbtx := pgtest.NewTx(t)
	ctx := pg.NewContext(context.Background(), dbtx)
	m := NewManager(prottest.NewChain(t))

	account, err := m.Create(ctx, []string{dummyXPub}, 1, "", nil, nil)
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

func TestCreateAccountIdempotency(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	var clientToken = "a-unique-client-token"
	m := NewManager(prottest.NewChain(t))

	account1, err := m.Create(ctx, []string{dummyXPub}, 1, "satoshi", nil, &clientToken)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	account2, err := m.Create(ctx, []string{dummyXPub}, 1, "satoshi", nil, &clientToken)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	if !reflect.DeepEqual(account1, account2) {
		t.Errorf("got=%#v, want=%#v", account2, account1)
	}
}

func TestCreateAccountReusedAlias(t *testing.T) {
	dbtx := pgtest.NewTx(t)
	ctx := pg.NewContext(context.Background(), dbtx)
	m := NewManager(prottest.NewChain(t))
	m.createTestAccount(ctx, t, "some-account", nil)

	_, err := m.Create(ctx, []string{dummyXPub}, 1, "some-account", nil, nil)
	if errors.Root(err) != httpjson.ErrBadRequest {
		t.Errorf("Expected %s when reusing an alias, got %v", httpjson.ErrBadRequest, err)
	}
}

func TestCreateControlProgram(t *testing.T) {
	// use pgtest.NewDB for deterministic postgres sequences
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	m := NewManager(prottest.NewChain(t))

	account, err := m.Create(ctx, []string{dummyXPub}, 1, "", nil, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	got, err := m.CreateControlProgram(ctx, account.ID, false)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	want, err := vm.Assemble("DUP TOALTSTACK SHA3 0x6dbfeed3d0cffddbda105bfe320072b067304af099c9cff0251d5446412e524a 1 1 CHECKMULTISIG VERIFY FROMALTSTACK 0 CHECKPREDICATE")
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if !bytes.Equal(got, want) {
		t.Errorf("got control program = %x want %x", got, want)
	}
}

func (m *Manager) createTestAccount(ctx context.Context, t testing.TB, alias string, tags map[string]interface{}) *Account {
	account, err := m.Create(ctx, []string{dummyXPub}, 1, alias, tags, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	return account
}

func (m *Manager) createTestControlProgram(ctx context.Context, t testing.TB, accountID string) []byte {
	if accountID == "" {
		account := m.createTestAccount(ctx, t, "", nil)
		accountID = account.ID
	}

	acp, err := m.CreateControlProgram(ctx, accountID, false)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	return acp
}

func TestFindByID(t *testing.T) {
	dbtx := pgtest.NewTx(t)
	ctx := pg.NewContext(context.Background(), dbtx)
	m := NewManager(prottest.NewChain(t))
	account := m.createTestAccount(ctx, t, "", nil)

	found, err := m.findByID(ctx, account.ID)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if !reflect.DeepEqual(account.Signer, found) {
		t.Errorf("expected found account to be %v, instead found %v", account, found)
	}
}

func TestFindByAlias(t *testing.T) {
	dbtx := pgtest.NewTx(t)
	ctx := pg.NewContext(context.Background(), dbtx)
	m := NewManager(prottest.NewChain(t))
	account := m.createTestAccount(ctx, t, "some-alias", nil)

	found, err := m.FindByAlias(ctx, "some-alias")
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if !reflect.DeepEqual(account.Signer, found) {
		t.Errorf("expected found account to be %v, instead found %v", account, found)
	}
}

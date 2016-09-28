package cursor

import (
	"context"
	"reflect"
	"testing"

	"chain/core/query"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
)

func TestInsertCursor(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	token := "test_token"
	alias := "test_cursor"
	cur := &Cursor{
		Alias: alias,
	}

	result, err := insertCursor(ctx, cur, &token)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	if result.ID == "" {
		t.Errorf("expected result.ID to be populated, but was empty")
	}

	// Verify that the cursor was created.
	var resultAlias string
	var checkQ = `SELECT alias FROM cursors`
	err = pg.QueryRow(ctx, checkQ).Scan(&resultAlias)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if resultAlias != alias {
		t.Errorf("expected new cursor with alias %s, got %s", alias, resultAlias)
	}
}

func TestInsertCursorRepeatToken(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	token := "test_token"
	alias := "test_cursor"
	cur := &Cursor{
		Alias: alias,
		Order: "desc",
	}

	result0, err := insertCursor(ctx, cur, &token)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	result1, err := insertCursor(ctx, cur, &token)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	if !reflect.DeepEqual(result0, result1) {
		t.Errorf("expected requests with matching tokens to yield matching results, instead got result0=%+v and result1=%+v",
			result0, result1)
	}
}

func TestInsertCursorDuplicateAlias(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	token0 := "test_token_0"
	token1 := "test_token_1"
	alias := "test_cursor"
	cur := &Cursor{
		Alias: alias,
	}

	_, err := insertCursor(ctx, cur, &token0)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	_, err = insertCursor(ctx, cur, &token1)
	if err.Error() != "non-unique alias: httpjson: bad request" {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestCursorIsBefore(t *testing.T) {
	cases := []struct {
		a       string
		b       string
		wantRes bool
		wantErr error
	}{
		{"1:1-2", "1:2-3", true, nil},
		{"1:1-2", "2:2-3", true, nil},
		{"2:1-2", "1:2-3", false, nil},
		{"not-a-cursor", "also, not a cursor", false, query.ErrBadAfter},
	}

	for _, c := range cases {
		res, err := isBefore(c.a, c.b)
		if errors.Root(err) != c.wantErr {
			t.Errorf("wanted err=%s, got %s", c.wantErr, err)
		}

		if res != c.wantRes {
			t.Errorf("wanted isBefore(%s, %s)=%t, got %t", c.a, c.b, c.wantRes, res)
		}
	}
}

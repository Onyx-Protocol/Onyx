package txfeed

import (
	"context"
	"reflect"
	"testing"

	"chain/core/query/filter"
	"chain/database/pg/pgtest"
	"chain/errors"
)

func TestInsertTxFeed(t *testing.T) {
	ctx := context.Background()
	db := pgtest.NewTx(t)
	token := "test_token"
	alias := "test_txfeed"
	feed := &TxFeed{
		Alias: &alias,
	}

	result, err := insertTxFeed(ctx, db, feed, &token)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	if result.ID == "" {
		t.Errorf("expected result.ID to be populated, but was empty")
	}

	// Verify that the txfeed was created.
	var resultAlias string
	var checkQ = `SELECT alias FROM txfeeds`
	err = db.QueryRow(ctx, checkQ).Scan(&resultAlias)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if resultAlias != alias {
		t.Errorf("expected new txfeed with alias %s, got %s", alias, resultAlias)
	}
}

func TestInsertTxFeedRepeatToken(t *testing.T) {
	ctx := context.Background()
	db := pgtest.NewTx(t)
	token := "test_token"
	alias := "test_txfeed"
	feed := &TxFeed{
		Alias: &alias,
	}

	result0, err := insertTxFeed(ctx, db, feed, &token)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	result1, err := insertTxFeed(ctx, db, feed, &token)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	if !reflect.DeepEqual(result0, result1) {
		t.Errorf("expected requests with matching tokens to yield matching results, instead got result0=%+v and result1=%+v",
			result0, result1)
	}
}

func TestInsertTxFeedDuplicateAlias(t *testing.T) {
	ctx := context.Background()
	db := pgtest.NewTx(t)
	token0 := "test_token_0"
	token1 := "test_token_1"
	alias := "test_txfeed"
	feed := &TxFeed{
		Alias: &alias,
	}

	_, err := insertTxFeed(ctx, db, feed, &token0)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	_, err = insertTxFeed(ctx, db, feed, &token1)
	if err.Error() != "non-unique alias: httpjson: bad request" {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestCreateTxFeedBadFilter(t *testing.T) {
	ctx := context.Background()
	tracker := &Tracker{DB: pgtest.NewTx(t)}
	token := "test_token_0"
	alias := "test_txfeed"
	fil := "lol i'm not a ~real~ filter"
	_, err := tracker.Create(ctx, alias, fil, "", &token)
	if errors.Root(err) != filter.ErrBadFilter {
		t.Errorf("expected ErrBadFilter, got %s", errors.Root(err))
	}
}

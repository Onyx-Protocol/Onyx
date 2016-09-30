package txconsumer

import (
	"context"
	"reflect"
	"testing"

	"chain/database/pg"
	"chain/database/pg/pgtest"
)

func TestInsertTxConsumer(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	token := "test_token"
	alias := "test_txconsumer"
	consumer := &TxConsumer{
		Alias: &alias,
	}

	result, err := insertTxConsumer(ctx, consumer, &token)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	if result.ID == "" {
		t.Errorf("expected result.ID to be populated, but was empty")
	}

	// Verify that the txconsumer was created.
	var resultAlias string
	var checkQ = `SELECT alias FROM txconsumers`
	err = pg.QueryRow(ctx, checkQ).Scan(&resultAlias)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if resultAlias != alias {
		t.Errorf("expected new txconsumer with alias %s, got %s", alias, resultAlias)
	}
}

func TestInsertTxConsumerRepeatToken(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	token := "test_token"
	alias := "test_txconsumer"
	consumer := &TxConsumer{
		Alias: &alias,
	}

	result0, err := insertTxConsumer(ctx, consumer, &token)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	result1, err := insertTxConsumer(ctx, consumer, &token)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	if !reflect.DeepEqual(result0, result1) {
		t.Errorf("expected requests with matching tokens to yield matching results, instead got result0=%+v and result1=%+v",
			result0, result1)
	}
}

func TestInsertTxConsumerDuplicateAlias(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	token0 := "test_token_0"
	token1 := "test_token_1"
	alias := "test_txconsumer"
	consumer := &TxConsumer{
		Alias: &alias,
	}

	_, err := insertTxConsumer(ctx, consumer, &token0)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	_, err = insertTxConsumer(ctx, consumer, &token1)
	if err.Error() != "non-unique alias: httpjson: bad request" {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

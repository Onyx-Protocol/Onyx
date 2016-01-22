package pg

import (
	"fmt"

	"golang.org/x/net/context"

	"chain/database/sql"
)

var savepointNameCounter = func() <-chan int {
	result := make(chan int)
	go func() {
		var n int
		for {
			n++
			result <- n
		}
	}()
	return result
}()

func genSavepointName() string {
	return fmt.Sprintf("savepoint%d", <-savepointNameCounter)
}

type nestedTx struct {
	tx            Tx
	savepointName string
	done          bool
}

func newNestedTx(ctx context.Context, tx Tx) (*nestedTx, context.Context, error) {
	savepointName := genSavepointName()
	result := &nestedTx{
		tx:            tx,
		savepointName: savepointName,
	}
	_, err := tx.Exec(ctx, fmt.Sprintf("SAVEPOINT %s", savepointName))
	if err != nil {
		return nil, nil, err
	}
	ctx = NewContext(ctx, result)
	return result, ctx, nil
}

func (tx *nestedTx) Commit(ctx context.Context) error {
	if tx.done {
		return sql.ErrTxDone
	}
	_, err := tx.Exec(ctx, fmt.Sprintf("RELEASE SAVEPOINT %s", tx.savepointName))
	if err == nil {
		tx.done = true
	}
	return err
}

func (tx *nestedTx) Rollback(ctx context.Context) error {
	if tx.done {
		return sql.ErrTxDone
	}
	_, err := tx.Exec(ctx, fmt.Sprintf("ROLLBACK TO SAVEPOINT %s", tx.savepointName))
	if err == nil {
		tx.done = true
	}
	return err
}

func (tx *nestedTx) Exec(ctx context.Context, q string, args ...interface{}) (sql.Result, error) {
	return tx.tx.Exec(ctx, q, args...)
}

func (tx *nestedTx) Query(ctx context.Context, q string, args ...interface{}) (*sql.Rows, error) {
	return tx.tx.Query(ctx, q, args...)
}

func (tx *nestedTx) QueryRow(ctx context.Context, q string, args ...interface{}) *sql.Row {
	return tx.tx.QueryRow(ctx, q, args...)
}

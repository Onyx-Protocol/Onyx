package account

import (
	"context"
	"reflect"
	"testing"

	"chain/database/pg/pgtest"
	"chain/protocol/bc"
	"chain/protocol/prottest"
	"chain/protocol/state"
	"chain/testutil"
)

func TestLoadAccountInfo(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	m := NewManager(db, prottest.NewChain(t), nil)
	ctx := context.Background()

	acc := m.createTestAccount(ctx, t, "", nil)
	acp := m.createTestControlProgram(ctx, t, acc.ID)

	to1 := bc.NewTxOutput(bc.AssetID{}, 0, acp, nil)
	to2 := bc.NewTxOutput(bc.AssetID{}, 0, []byte("notfound"), nil)

	outs := []*state.Output{{
		TxOutput: *to1,
	}, {
		TxOutput: *to2,
	}}

	got, err := m.loadAccountInfo(ctx, outs)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if !reflect.DeepEqual(got[0].AccountID, acc.ID) {
		t.Errorf("got account = %+v want %+v", got[0].AccountID, acc.ID)
	}
}

func TestDeleteUTXOs(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	m := NewManager(db, prottest.NewChain(t), nil)
	ctx := context.Background()

	assetID := bc.AssetID{}
	acp := m.createTestControlProgram(ctx, t, "")

	block1 := &bc.Block{Transactions: []*bc.Tx{
		bc.NewTx(bc.TxData{
			Outputs: []*bc.TxOutput{
				bc.NewTxOutput(assetID, 1, acp, nil),
			},
		}),
	}}
	err := m.indexAccountUTXOs(ctx, block1)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	block2 := &bc.Block{Transactions: []*bc.Tx{
		bc.NewTx(bc.TxData{
			Inputs: []*bc.TxInput{
				bc.NewSpendInput(block1.Transactions[0].Hash, 0, nil, assetID, 1, nil, nil),
			},
		}),
	}}
	err = m.indexAccountUTXOs(ctx, block2)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	var n int
	err = m.db.QueryRow(ctx, `SELECT count(*) FROM account_utxos`).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("count(account_utxos) = %d want 0", n)
	}
}

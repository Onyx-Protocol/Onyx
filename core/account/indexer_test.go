package account

import (
	"context"
	"testing"

	"chain/database/pg/pgtest"
	"chain/protocol/bc"
	"chain/protocol/bc/legacy"
	"chain/protocol/prottest"
	"chain/testutil"
)

func TestLoadAccountInfo(t *testing.T) {
	db := pgtest.NewTx(t)
	m := NewManager(db, prottest.NewChain(t), nil)
	ctx := context.Background()

	acc := m.createTestAccount(ctx, t, "", nil)
	acp := m.createTestControlProgram(ctx, t, acc.ID).controlProgram

	to1 := legacy.NewTxOutput(bc.AssetID{}, 0, acp, nil)
	to2 := legacy.NewTxOutput(bc.AssetID{}, 0, []byte("notfound"), nil)

	outs := []*rawOutput{{
		AssetAmount:    to1.AssetAmount,
		ControlProgram: to1.ControlProgram,
	}, {
		AssetAmount:    to2.AssetAmount,
		ControlProgram: to2.ControlProgram,
	}}

	got, err := m.loadAccountInfo(ctx, outs)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if !testutil.DeepEqual(got[0].AccountID, acc.ID) {
		t.Errorf("got account = %+v want %+v", got[0].AccountID, acc.ID)
	}
}

func TestDeleteUTXOs(t *testing.T) {
	db := pgtest.NewTx(t)
	m := NewManager(db, prottest.NewChain(t), nil)
	ctx := context.Background()

	assetID := bc.AssetID{}
	acp := m.createTestControlProgram(ctx, t, "").controlProgram
	tx := legacy.NewTx(legacy.TxData{
		Outputs: []*legacy.TxOutput{
			legacy.NewTxOutput(assetID, 1, acp, nil),
		},
	})

	block1 := &legacy.Block{Transactions: []*legacy.Tx{tx}}
	err := m.indexAccountUTXOs(ctx, block1)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	err = m.deleteSpentOutputs(ctx, block1)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	out0 := tx.Entries[*tx.ResultIds[0]].(*bc.Output)
	block2 := &legacy.Block{Transactions: []*legacy.Tx{
		legacy.NewTx(legacy.TxData{
			Inputs: []*legacy.TxInput{
				legacy.NewSpendInput(nil, *out0.Source.Ref, assetID, 1, out0.Source.Position, acp, *out0.Data, nil),
			},
		}),
	}}
	err = m.indexAccountUTXOs(ctx, block2)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	err = m.deleteSpentOutputs(ctx, block2)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	var n int
	err = m.db.QueryRowContext(ctx, `SELECT count(*) FROM account_utxos`).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("count(account_utxos) = %d want 0", n)
	}
}

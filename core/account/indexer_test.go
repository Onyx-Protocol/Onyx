package account

import (
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/cos/state"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/testutil"
)

func TestLoadAccountInfo(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	acc := createTestAccount(ctx, t, nil)
	acp := createTestControlProgram(ctx, t, acc.ID)

	to1 := bc.NewTxOutput(bc.AssetID{}, 0, acp, nil)
	to2 := bc.NewTxOutput(bc.AssetID{}, 0, []byte("notfound"), nil)

	outs := []*state.Output{{
		TxOutput: *to1,
	}, {
		TxOutput: *to2,
	}}

	got, err := loadAccountInfo(ctx, outs)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if !reflect.DeepEqual(got[0].AccountID, acc.ID) {
		t.Errorf("got account = %+v want %+v", got[0].AccountID, acc.ID)
	}
}

func TestDeleteUTXOs(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	assetID := bc.AssetID{}
	acp := createTestControlProgram(ctx, t, "")

	tx := bc.NewTx(bc.TxData{
		Outputs: []*bc.TxOutput{
			bc.NewTxOutput(assetID, 1, acp, nil),
		},
	})

	err := addAccountData(ctx, tx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	block := &bc.Block{Transactions: []*bc.Tx{
		bc.NewTx(bc.TxData{
			Inputs: []*bc.TxInput{
				bc.NewSpendInput(tx.Hash, 0, nil, assetID, 1, nil, nil),
			},
		}),
	}}

	indexAccountUTXOs(ctx, block)

	var n int
	err = pg.QueryRow(ctx, `SELECT count(*) FROM account_utxos`).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("count(account_utxos) = %d want 0", n)
	}
}

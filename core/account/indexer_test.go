package account

import (
	"context"
	"testing"
	"time"

	"chain/core/pin"
	"chain/core/query"
	"chain/crypto/ed25519/chainkd"
	"chain/database/pg/pgtest"
	"chain/protocol/bc"
	"chain/protocol/prottest"
	"chain/testutil"
)

func TestUsedExpiredAccountControlPrograms(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	c := prottest.NewChain(t)
	pins := pin.NewStore(db)
	m := NewManager(db, c, pins)
	ctx := context.Background()

	account, err := m.Create(ctx, []chainkd.XPub{testutil.TestXPub}, 1, "", nil, "")
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// Create an expired account control program.
	acp, err := m.CreateControlProgram(ctx, account.ID, false, time.Now().Add(-5*time.Minute))
	if err != nil {
		testutil.FatalErr(t, err)
	}
	fakeOutput := bc.NewTxOutput(bc.AssetID{}, 100, acp, nil)

	// Make a fake account utxo.
	b := &bc.Block{
		BlockHeader: bc.BlockHeader{Height: 2},
		Transactions: []*bc.Tx{{
			TxHashes: bc.TxHashes{Results: make([]bc.ResultInfo, 1)},
			TxData:   bc.TxData{Outputs: []*bc.TxOutput{fakeOutput}},
		}},
	}
	err = m.indexAccountUTXOs(ctx, b)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// The expire control programs routine requires that the account
	// and query tx processors to run first. Create fake pins indiciating
	// that they've already run.
	err = pins.CreatePin(ctx, PinName, 2)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	err = pins.CreatePin(ctx, query.TxPinName, 2)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// Delete expired control programs. Our control program should
	// not be deleted because there's an existing account UTXO.
	err = m.expireControlPrograms(ctx, b)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	const q = `
		SELECT COUNT(*) FROM account_control_programs
		WHERE control_program = $1::bytea
	`
	var count int
	err = db.QueryRow(ctx, q, acp).Scan(&count)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	if count != 1 {
		t.Fatal("Expected account control program to not be deleted, but it was.")
	}
}

func TestLoadAccountInfo(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	m := NewManager(db, prottest.NewChain(t), nil)
	ctx := context.Background()

	acc := m.createTestAccount(ctx, t, "", nil)
	acp := m.createTestControlProgram(ctx, t, acc.ID)

	to1 := bc.NewTxOutput(bc.AssetID{}, 0, acp, nil)
	to2 := bc.NewTxOutput(bc.AssetID{}, 0, []byte("notfound"), nil)

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
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	m := NewManager(db, prottest.NewChain(t), nil)
	ctx := context.Background()

	assetID := bc.AssetID{}
	acp := m.createTestControlProgram(ctx, t, "")
	tx := bc.NewTx(bc.TxData{
		Outputs: []*bc.TxOutput{
			bc.NewTxOutput(assetID, 1, acp, nil),
		},
	})

	block1 := &bc.Block{Transactions: []*bc.Tx{tx}}
	err := m.indexAccountUTXOs(ctx, block1)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	block2 := &bc.Block{Transactions: []*bc.Tx{
		bc.NewTx(bc.TxData{
			Inputs: []*bc.TxInput{
				bc.NewSpendInput(nil, tx.Results[0].SourceID, assetID, 1, tx.Results[0].SourcePos, acp, tx.Results[0].RefDataHash, nil),
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

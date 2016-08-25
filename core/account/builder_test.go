package account_test

import (
	"context"
	"database/sql"
	"reflect"
	"testing"
	"time"

	"chain/core/account"
	"chain/core/asset/assettest"
	"chain/core/txbuilder"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/protocol/bc"
	"chain/testutil"
)

func TestAccountSourceReserve(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	c, g, err := assettest.InitializeSigningGenerator(ctx, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	accID := assettest.CreateAccountFixture(ctx, t, nil, 0, "", nil)
	asset := assettest.CreateAssetFixture(ctx, t, nil, 0, nil, "", nil)
	out := assettest.IssueAssetsFixture(ctx, t, c, asset, 2, accID)

	// Make a block so that account UTXOs are available to spend.
	_, err = g.MakeBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assetAmount1 := bc.AssetAmount{
		AssetID: asset,
		Amount:  1,
	}
	source := assettest.NewAccountSpendAction(assetAmount1, accID, nil, nil, nil)

	gotTxIns, gotTxOuts, _, err := source.Build(ctx)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	wantTxIns := []*bc.TxInput{bc.NewSpendInput(out.Hash, out.Index, nil, out.AssetID, out.Amount, out.ControlProgram, nil)}

	if !reflect.DeepEqual(gotTxIns, wantTxIns) {
		t.Errorf("build txins\ngot:\n\t%+v\nwant:\n\t%+v", gotTxIns, wantTxIns)
	}

	if len(gotTxOuts) != 1 {
		t.Errorf("expected 1 change output")
	}

	if gotTxOuts[0].Amount != 1 {
		t.Errorf("expected change amount to be 1")
	}

	if !programInAccount(ctx, t, gotTxOuts[0].ControlProgram, accID) {
		t.Errorf("expected change control program to belong to account")
	}
}

func TestAccountSourceUTXOReserve(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	c, g, err := assettest.InitializeSigningGenerator(ctx, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	accID := assettest.CreateAccountFixture(ctx, t, nil, 0, "", nil)
	asset := assettest.CreateAssetFixture(ctx, t, nil, 0, nil, "", nil)
	out := assettest.IssueAssetsFixture(ctx, t, c, asset, 2, accID)

	// Make a block so that account UTXOs are available to spend.
	_, err = g.MakeBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	source := &account.SpendUTXOAction{
		Params: struct {
			TxHash bc.Hash       `json:"transaction_id"`
			TxOut  uint32        `json:"position"`
			TTL    time.Duration `json:"reservation_ttl"`
		}{out.Hash, out.Index, time.Minute},
	}

	gotTxIns, _, _, err := source.Build(ctx)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	wantTxIns := []*bc.TxInput{bc.NewSpendInput(out.Hash, out.Index, nil, out.AssetID, out.Amount, out.ControlProgram, nil)}

	if !reflect.DeepEqual(gotTxIns, wantTxIns) {
		t.Errorf("build txins\ngot:\n\t%+v\nwant:\n\t%+v", gotTxIns, wantTxIns)
	}
}

func TestAccountSourceReserveIdempotency(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	c, g, err := assettest.InitializeSigningGenerator(ctx, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	var (
		accID        = assettest.CreateAccountFixture(ctx, t, nil, 0, "", nil)
		asset        = assettest.CreateAssetFixture(ctx, t, nil, 0, nil, "", nil)
		_            = assettest.IssueAssetsFixture(ctx, t, c, asset, 2, accID)
		_            = assettest.IssueAssetsFixture(ctx, t, c, asset, 2, accID)
		assetAmount1 = bc.AssetAmount{
			AssetID: asset,
			Amount:  1,
		}

		// An idempotency key that both reservations should use.
		clientToken1 = "a-unique-idempotency-key"
		clientToken2 = "another-unique-idempotency-key"
		wantSrc      = assettest.NewAccountSpendAction(assetAmount1, accID, nil, nil, nil)
		gotSrc       = assettest.NewAccountSpendAction(assetAmount1, accID, nil, nil, nil)
		separateSrc  = assettest.NewAccountSpendAction(assetAmount1, accID, nil, nil, nil)
	)
	wantSrc.ClientToken = &clientToken1
	gotSrc.ClientToken = &clientToken1
	separateSrc.ClientToken = &clientToken2

	// Make a block so that account UTXOs are available to spend.
	_, err = g.MakeBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	reserveFunc := func(source txbuilder.Action) []*bc.TxInput {
		got, _, _, err := source.Build(ctx)
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 result utxo")
		}
		return got
	}

	var (
		got      = reserveFunc(gotSrc)
		want     = reserveFunc(wantSrc)
		separate = reserveFunc(separateSrc)
	)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("reserve result\ngot:\n\t%+v\nwant:\n\t%+v", got, want)
	}

	// The third reservation attempt should be distinct and not the same as the first two.
	if reflect.DeepEqual(separate, want) {
		t.Errorf("reserve result\ngot:\n\t%+v\ndo not want:\n\t%+v", separate, want)
	}
}

func TestAccountSourceWithTxHash(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	c, g, err := assettest.InitializeSigningGenerator(ctx, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	var (
		acc      = assettest.CreateAccountFixture(ctx, t, nil, 0, "", nil)
		asset    = assettest.CreateAssetFixture(ctx, t, nil, 0, nil, "", nil)
		assetAmt = bc.AssetAmount{AssetID: asset, Amount: 1}
		utxos    = 4
		srcTxs   []bc.Hash
	)

	for i := 0; i < utxos; i++ {
		o := assettest.IssueAssetsFixture(ctx, t, c, asset, 1, acc)
		srcTxs = append(srcTxs, o.Outpoint.Hash)
	}

	// Make a block so that account UTXOs are available to spend.
	_, err = g.MakeBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < utxos; i++ {
		theTxHash := srcTxs[i]
		source := assettest.NewAccountSpendAction(assetAmt, acc, &theTxHash, nil, nil)

		gotRes, _, _, err := source.Build(ctx)
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		if len(gotRes) != 1 {
			t.Fatalf("expected 1 result utxo")
		}

		got := gotRes[0].Outpoint()
		want := bc.Outpoint{Hash: theTxHash, Index: 0}
		if got != want {
			t.Errorf("reserved utxo outpoint got=%v want=%v", got, want)
		}
	}
}

func programInAccount(ctx context.Context, t testing.TB, program []byte, account string) bool {
	const q = `SELECT signer_id=$1 FROM account_control_programs WHERE control_program=$2`
	var in bool
	err := pg.QueryRow(ctx, q, account, program).Scan(&in)
	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		testutil.FatalErr(t, err)
	}
	return in
}

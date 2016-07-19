package appdb_test

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"golang.org/x/net/context"

	. "chain/core/appdb"
	"chain/core/asset"
	"chain/core/asset/assettest"
	"chain/core/asset/nodetxlog"
	"chain/core/generator"
	"chain/core/txbuilder"
	"chain/cos/bc"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
)

func TestWriteManagerTx(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	accounts := []string{"account-1", "account-2"}
	_, err := WriteManagerTx(ctx, "tx1", []byte(`{}`), "mnode-1", time.Now(), accounts)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	txs, _, err := ManagerTxs(ctx, "mnode-1", "", 100)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	if len(txs) != 1 {
		t.Fatal("expected manager tx to be in db")
	}

	for _, acc := range accounts {
		txs, _, err = AccountTxs(ctx, acc, time.Time{}, time.Now(), "", 100)
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		if len(txs) != 1 {
			t.Errorf("expected account tx to be in db for %s", acc)
		}
	}
}

func TestWriteIssuerTx(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	assets := []string{"asset-1", "asset-2"}
	_, err := WriteIssuerTx(ctx, "tx1", []byte(`{}`), "inode-1", time.Now(), assets)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	txs, _, err := IssuerTxs(ctx, "inode-1", "", 100)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	if len(txs) != 1 {
		t.Fatal("expected issuer tx to be in db")
	}

	for _, asset := range assets {
		txs, _, err = AssetTxs(ctx, asset, "", 100)
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}

		if len(txs) != 1 {
			t.Fatal("expected asset tx to be in db")
		}
	}
}

func TestManagerTxs(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	mn0 := assettest.CreateManagerNodeFixture(ctx, t, "", "x", nil, nil)
	mtx := assettest.ManagerTxFixture(ctx, t, "tx0", []byte(`{"outputs":"boop"}`), mn0, nil)

	txs, last, err := ManagerTxs(ctx, mn0, "", 1)
	if err != nil {
		t.Fatalf("unexpected err %v", err)
	}

	if len(txs) != 1 {
		t.Fatalf("want len(txs)=1 got=%d", len(txs))
	}

	if last != mtx {
		t.Fatalf("got last tx=%v want %v", last, mtx)
	}

	if string(*txs[0]) != `{"outputs":"boop"}` {
		t.Fatalf("want={outputs: boop}, got=%v", *txs[0])
	}
}

func TestManagerTxsLimit(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	mn0 := assettest.CreateManagerNodeFixture(ctx, t, "", "x", nil, nil)
	assettest.ManagerTxFixture(ctx, t, "tx0", []byte(`{"outputs":"boop"}`), mn0, nil)
	assettest.ManagerTxFixture(ctx, t, "tx1", []byte(`{"outputs":"coop"}`), mn0, nil)
	mtx2 := assettest.ManagerTxFixture(ctx, t, "tx2", []byte(`{"outputs":"doop"}`), mn0, nil)
	assettest.ManagerTxFixture(ctx, t, "tx3", []byte(`{"outputs":"foop"}`), mn0, nil)

	txs, last, err := ManagerTxs(ctx, mn0, "", 2)
	if err != nil {
		t.Fatalf("unexpected err %v", err)
	}

	if len(txs) != 2 {
		t.Log(txs)
		t.Fatalf("want len(txs)=2 got=%d", len(txs))
	}

	if last != mtx2 {
		t.Fatalf("got last tx=%v want %v", last, mtx2)
	}

	if string(*txs[0]) != `{"outputs":"foop"}` {
		t.Fatalf("want={outputs: foop}, got=%v", *txs[0])
	}

	if string(*txs[1]) != `{"outputs":"doop"}` {
		t.Fatalf("want={outputs: doop}, got=%v", *txs[1])
	}
}

func TestAccountTxs(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	mn0 := assettest.CreateManagerNodeFixture(ctx, t, "", "x", nil, nil)
	acc0 := assettest.CreateAccountFixture(ctx, t, mn0, "foo", nil)
	mtx := assettest.ManagerTxFixture(ctx, t, "tx0", []byte(`{"outputs":"boop"}`), mn0, []string{acc0})

	txs, last, err := AccountTxs(ctx, acc0, time.Time{}, time.Now(), "", 1)
	if err != nil {
		t.Fatalf("unexpected err %v", err)
	}

	if len(txs) != 1 {
		t.Fatalf("want len(txs)=1 got=%d", len(txs))
	}

	if last != mtx {
		t.Fatalf("got last tx=%v want %v", last, mtx)
	}
}

func TestAccountTxsLimit(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	mn0 := assettest.CreateManagerNodeFixture(ctx, t, "", "x", nil, nil)
	acc0 := assettest.CreateAccountFixture(ctx, t, mn0, "foo", nil)
	assettest.ManagerTxFixture(ctx, t, "tx0", []byte(`{"outputs":"boop"}`), mn0, []string{acc0})
	assettest.ManagerTxFixture(ctx, t, "tx1", []byte(`{"outputs":"coop"}`), mn0, []string{acc0})
	mtx2 := assettest.ManagerTxFixture(ctx, t, "tx2", []byte(`{"outputs":"doop"}`), mn0, []string{acc0})
	assettest.ManagerTxFixture(ctx, t, "tx3", []byte(`{"outputs":"foop"}`), mn0, []string{acc0})

	txs, last, err := AccountTxs(ctx, acc0, time.Time{}, time.Now(), "", 2)
	if err != nil {
		t.Fatalf("unexpected err %v", err)
	}

	if len(txs) != 2 {
		t.Log(txs)
		t.Fatalf("want len(txs)=2 got=%d", len(txs))
	}

	if last != mtx2 {
		t.Fatalf("got last tx=%v want %v", last, mtx2)
	}

	if string(*txs[0]) != `{"outputs":"foop"}` {
		t.Fatalf("want={outputs: foop}, got=%v", *txs[0])
	}

	if string(*txs[1]) != `{"outputs":"doop"}` {
		t.Fatalf("want={outputs: doop}, got=%v", *txs[1])
	}
}

func TestAccountTxsTimeLimit(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	_, err := assettest.InitializeSigningGenerator(ctx, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	assetID := assettest.CreateAssetFixture(ctx, t, "", "", "")
	mnodeID := assettest.CreateManagerNodeFixture(ctx, t, "", "", nil, nil)
	acct1ID := assettest.CreateAccountFixture(ctx, t, mnodeID, "", nil)
	acct2ID := assettest.CreateAccountFixture(ctx, t, mnodeID, "", nil)

	assettest.IssueAssetsFixture(ctx, t, assetID, 100, acct1ID)

	srcs := func(n uint64) []*txbuilder.Source {
		return []*txbuilder.Source{
			asset.NewAccountSource(ctx, &bc.AssetAmount{AssetID: assetID, Amount: n}, acct1ID, nil, nil, nil),
		}
	}
	dests := func(n uint64) []*txbuilder.Destination {
		return []*txbuilder.Destination{
			assettest.AccountDest(ctx, t, acct2ID, assetID, n),
		}
	}

	// Don't include this transfer in the output
	assettest.Transfer(ctx, t, srcs(1), dests(1))

	_, err = generator.MakeBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	startTime := time.Now()

	// Do include this transfer in the output
	assettest.Transfer(ctx, t, srcs(2), dests(2))

	// Do include this transfer in the output too
	assettest.Transfer(ctx, t, srcs(4), dests(4))

	_, err = generator.MakeBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	endTime := time.Now()

	// Don't include this transfer in the output
	assettest.Transfer(ctx, t, srcs(8), dests(8))

	_, err = generator.MakeBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	txs, _, err := AccountTxs(ctx, acct1ID, startTime, endTime, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(txs) != 2 {
		t.Fatalf("expected 2 txs, got %d", len(txs))
	}

	var sum uint64
	for _, tx := range txs {
		var nodeTx nodetxlog.NodeTx
		err = json.Unmarshal([]byte(*tx), &nodeTx)
		if err != nil {
			t.Fatal(err)
		}
		for _, o := range nodeTx.Outputs {
			if o.AccountID == acct2ID {
				sum += o.Amount
			}
		}
	}
	if sum != 6 {
		t.Errorf("expected transfers of 6 units, got a transfer of %d", sum)
	}
}

func TestIssuerTxs(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	in0 := assettest.CreateIssuerNodeFixture(ctx, t, "", "in-0", nil, nil)
	in1 := assettest.CreateIssuerNodeFixture(ctx, t, "", "in-1", nil, nil)

	itx0 := assettest.IssuerTxFixture(ctx, t, "tx-id-0", []byte(`{"transaction_id": "tx-id-0"}`), in0, []string{"asset-id-0"})
	itx1 := assettest.IssuerTxFixture(ctx, t, "tx-id-1", []byte(`{"transaction_id": "tx-id-1"}`), in1, []string{"asset-id-1"})
	assettest.IssuerTxFixture(ctx, t, "tx-id-2", []byte(`{"transaction_id": "tx-id-2"}`), in0, []string{"asset-id-0"})

	examples := []struct {
		inodeID  string
		wantAct  []*json.RawMessage
		wantLast string
	}{
		{
			in0,
			stringsToRawJSON(
				`{"transaction_id": "tx-id-2"}`,
				`{"transaction_id": "tx-id-0"}`,
			),
			itx0,
		},
		{
			in1,
			stringsToRawJSON(
				`{"transaction_id": "tx-id-1"}`,
			),
			itx1,
		},
	}

	for _, ex := range examples {
		t.Log("Example", ex.inodeID)

		gotAct, gotLast, err := IssuerTxs(ctx, ex.inodeID, "", 50)
		if err != nil {
			t.Fatal("unexpected error", err)
		}

		if !reflect.DeepEqual(gotAct, ex.wantAct) {
			t.Errorf("txs:\ngot:  %v\nwant: %v", gotAct, ex.wantAct)
		}

		if gotLast != ex.wantLast {
			t.Errorf("last got = %v want %v", gotLast, ex.wantLast)
		}
	}
}

func TestAssetTxs(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	in0 := assettest.CreateIssuerNodeFixture(ctx, t, "", "in-0", nil, nil)
	in1 := assettest.CreateIssuerNodeFixture(ctx, t, "", "in-1", nil, nil)

	itx0 := assettest.IssuerTxFixture(ctx, t, "tx-id-0", []byte(`{"transaction_id": "tx-id-0"}`), in0, []string{"asset-id-0"})
	itx1 := assettest.IssuerTxFixture(ctx, t, "tx-id-1", []byte(`{"transaction_id": "tx-id-1"}`), in1, []string{"asset-id-1"})
	assettest.IssuerTxFixture(ctx, t, "tx-id-2", []byte(`{"transaction_id": "tx-id-2"}`), in0, []string{"asset-id-0"})

	stringsToRawJSON := func(strs ...string) []*json.RawMessage {
		var res []*json.RawMessage
		for _, s := range strs {
			b := json.RawMessage([]byte(s))
			res = append(res, &b)
		}
		return res
	}

	examples := []struct {
		assetID  string
		wantAct  []*json.RawMessage
		wantLast string
	}{
		{
			"asset-id-0",
			stringsToRawJSON(`{"transaction_id": "tx-id-2"}`, `{"transaction_id": "tx-id-0"}`),
			itx0,
		},
		{
			"asset-id-1",
			stringsToRawJSON(`{"transaction_id": "tx-id-1"}`),
			itx1,
		},
	}

	for _, ex := range examples {
		t.Log("Example", ex.assetID)

		gotAct, gotLast, err := AssetTxs(ctx, ex.assetID, "", 50)
		if err != nil {
			t.Fatal("unexpected error", err)
		}

		if !reflect.DeepEqual(gotAct, ex.wantAct) {
			t.Errorf("txs:\ngot:  %v\nwant: %v", gotAct, ex.wantAct)
		}

		if gotLast != ex.wantLast {
			t.Errorf("last got = %v want %v", gotLast, ex.wantLast)
		}
	}
}

func TestManagerTx(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	mn0 := assettest.CreateManagerNodeFixture(ctx, t, "", "x", nil, nil)
	assettest.ManagerTxFixture(ctx, t, "tx0", []byte(`{"outputs":"boop"}`), mn0, nil)

	txs, err := ManagerTx(ctx, mn0, "tx0")
	if err != nil {
		t.Fatalf("unexpected err %v", err)
	}

	if string(*txs) != `{"outputs":"boop"}` {
		t.Fatalf("want={outputs: boop}, got=%s", *txs)
	}

	_, err = ManagerTx(ctx, mn0, "txDoesNotExist")
	if errors.Root(err) != pg.ErrUserInputNotFound {
		t.Fatalf("want=%v got=%v", pg.ErrUserInputNotFound, err)
	}
}

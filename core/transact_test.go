package core

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/coretest"
	"chain/core/generator"
	"chain/core/pin"
	"chain/core/query"
	"chain/core/txbuilder"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/bc/bctest"
	"chain/protocol/bc/legacy"
	"chain/protocol/prottest"
	"chain/testutil"
)

func TestAccountTransferSpendChange(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := context.Background()
	c := prottest.NewChain(t)
	g := generator.New(c, nil, db)
	pinStore := pin.NewStore(db)
	assets := asset.NewRegistry(db, c, pinStore)
	accounts := account.NewManager(db, c, pinStore)
	coretest.CreatePins(ctx, t, pinStore)
	accounts.IndexAccounts(query.NewIndexer(db, c, pinStore))
	go accounts.ProcessBlocks(ctx)

	acc := coretest.CreateAccount(ctx, t, accounts, "", nil)

	assetID := coretest.CreateAsset(ctx, t, assets, nil, "", nil)
	assetAmt := bc.AssetAmount{
		AssetId: &assetID,
		Amount:  100,
	}

	source := txbuilder.Action(assets.NewIssueAction(assetAmt, nil))
	dest := accounts.NewControlAction(assetAmt, acc, nil)

	tmpl, err := txbuilder.Build(ctx, nil, []txbuilder.Action{source, dest}, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	coretest.SignTxTemplate(t, ctx, tmpl, &testutil.TestXPrv)
	err = txbuilder.FinalizeTx(ctx, c, g, tmpl.Transaction)
	if err != nil {
		t.Fatal(err)
	}
	b := prottest.MakeBlock(t, c, g.PendingTxs())
	if len(b.Transactions) != 1 {
		t.Errorf("len(b.Transactions) = %d, want 1", len(b.Transactions))
	}
	<-pinStore.PinWaiter(account.PinName, c.Height())

	// Add a new source, spending the change output produced above.
	source = accounts.NewSpendAction(assetAmt, acc, nil, nil)
	tmpl, err = txbuilder.Build(ctx, nil, []txbuilder.Action{source, dest}, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	coretest.SignTxTemplate(t, ctx, tmpl, &testutil.TestXPrv)
	err = txbuilder.FinalizeTx(ctx, c, g, tmpl.Transaction)
	if err != nil {
		t.Fatal(err)
	}
	b = prottest.MakeBlock(t, c, g.PendingTxs())
	if len(b.Transactions) != 1 {
		t.Errorf("len(b.Transactions) = %d, want 1", len(b.Transactions))
	}
}

func TestRecordSubmittedTxs(t *testing.T) {
	ctx := context.Background()
	dbtx := pgtest.NewTx(t)

	testCases := []struct {
		hash   bc.Hash
		height uint64
		want   uint64
	}{
		{hash: bc.NewHash([32]byte{0x01}), height: 2, want: 2},
		{hash: bc.NewHash([32]byte{0x02}), height: 3, want: 3},
		{hash: bc.NewHash([32]byte{0x01}), height: 3, want: 2},
	}

	for i, tc := range testCases {
		got, err := recordSubmittedTx(ctx, dbtx, tc.hash, tc.height)
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Errorf("%d: got %d want %d for hash %x", i, got, tc.want, tc.hash.Bytes())
		}
	}
}

type submitterFunc func(context.Context, *legacy.Tx) error

func (f submitterFunc) Submit(ctx context.Context, tx *legacy.Tx) error {
	return f(ctx, tx)
}

func TestWaitForTxInBlock(t *testing.T) {
	c := prottest.NewChain(t)
	submittedTx := bctest.NewIssuanceTx(t, prottest.Initial(t, c).Hash())
	a := &API{
		chain: c,
		submitter: submitterFunc(func(context.Context, *legacy.Tx) error {
			return nil
		}),
	}

	// Start a goroutine waiting for submittedTx to appear in a block.
	heightFound := make(chan uint64)
	go func() {
		h, err := a.waitForTxInBlock(context.Background(), submittedTx, 1)
		if err != nil {
			t.Fatal(err)
		}
		heightFound <- h
		close(heightFound)
	}()

	// Make a block with some transactions but not the transaction
	// that we're looking for.
	_ = prottest.MakeBlock(t, c, []*legacy.Tx{
		bctest.NewIssuanceTx(t, prottest.Initial(t, c).Hash()),
		bctest.NewIssuanceTx(t, prottest.Initial(t, c).Hash()),
	})
	// Make a block with a few transactions, including the one
	// we're waiting for.
	b := prottest.MakeBlock(t, c, []*legacy.Tx{
		bctest.NewIssuanceTx(t, prottest.Initial(t, c).Hash()),
		submittedTx, // bingo
		bctest.NewIssuanceTx(t, prottest.Initial(t, c).Hash()),
	})

	// Make sure that the goroutine found the tx and at the right height.
	h := <-heightFound
	if h != b.Height {
		t.Errorf("got height %d, wanted height %d", h, b.Height)
	}
}

func TestInsufficient(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := context.Background()
	c := prottest.NewChain(t)
	g := generator.New(c, nil, db)
	pinStore := pin.NewStore(db)
	assets := asset.NewRegistry(db, c, pinStore)
	accounts := account.NewManager(db, c, pinStore)
	coretest.CreatePins(ctx, t, pinStore)
	accounts.IndexAccounts(query.NewIndexer(db, c, pinStore))
	go accounts.ProcessBlocks(ctx)

	accID := coretest.CreateAccount(ctx, t, accounts, "", nil)
	assetID := coretest.CreateAsset(ctx, t, assets, nil, "", nil)
	assetAmt := bc.AssetAmount{AssetId: &assetID, Amount: 100}

	source1 := txbuilder.Action(assets.NewIssueAction(assetAmt, nil))
	dest1 := accounts.NewControlAction(assetAmt, accID, nil)
	tmpl, err := txbuilder.Build(ctx, nil, []txbuilder.Action{source1, dest1}, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	coretest.SignTxTemplate(t, ctx, tmpl, &testutil.TestXPrv)
	err = txbuilder.FinalizeTx(ctx, c, g, tmpl.Transaction)
	if err != nil {
		t.Fatal(err)
	}
	prottest.MakeBlock(t, c, g.PendingTxs())
	<-pinStore.PinWaiter(account.PinName, c.Height())

	source2 := accounts.NewSpendAction(assetAmt, accID, nil, nil)
	dest2, _ := txbuilder.DecodeRetireAction([]byte(fmt.Sprintf(`{"asset_id":"%x","amount":100}`, assetID.Bytes())))
	tmpl, err = txbuilder.Build(ctx, nil, []txbuilder.Action{source2, dest2}, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	coretest.SignTxTemplate(t, ctx, tmpl, &testutil.TestXPrv)
	err = txbuilder.FinalizeTx(ctx, c, g, tmpl.Transaction)
	if err != nil {
		t.Fatal(err)
	}
	prottest.MakeBlock(t, c, g.PendingTxs())
	<-pinStore.PinWaiter(account.PinName, c.Height())

	assetAmt.Amount = 10
	source3 := accounts.NewSpendAction(assetAmt, accID, nil, nil)
	dest3, _ := txbuilder.DecodeRetireAction([]byte(fmt.Sprintf(`{"asset_id":"%x","amount":10}`, assetID.Bytes())))
	_, err = txbuilder.Build(ctx, nil, []txbuilder.Action{source3, dest3}, time.Now().Add(time.Minute))
	switch errors.Root(err) {
	case nil:
		t.Error("got no error, want ErrAction (containing ErrInsufficient)")
	case txbuilder.ErrAction:
		actionErrs := errors.Data(err)["actions"].([]error)
		if errors.Root(actionErrs[0]) != account.ErrInsufficient {
			t.Errorf("got error %s, want ErrInsufficient", actionErrs[0])
		}
	default:
		t.Errorf("got error %s, want ErrAction (containing ErrInsufficient)", err)
	}
}

func TestWaitForTxInBlockResubmits(t *testing.T) {
	const timesToResubmit = 5

	c := prottest.NewChain(t)
	orig := bctest.NewIssuanceTx(t, prottest.Initial(t, c).Hash())
	a := &API{chain: c}

	// Record every time the Submit function is called.
	var wg sync.WaitGroup
	wg.Add(timesToResubmit)
	a.submitter = submitterFunc(func(_ context.Context, tx *legacy.Tx) error {
		if orig.ID != tx.ID {
			t.Errorf("got tx %s, want tx %s", tx.ID, orig.ID)
		}
		wg.Done()
		return nil
	})

	// Start a goroutine waiting for orig to appear in a block.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go a.waitForTxInBlock(ctx, orig, 1)

	// Make n blocks but never include the transaction
	// that we're looking for. The tx should be resubmitted
	// to the generator each time.
	for i := 0; i < timesToResubmit; i++ {
		prottest.MakeBlock(t, c, []*legacy.Tx{})
	}

	done := make(chan struct{})
	go func() {
		// Wait until the submitter records that the transaction has been
		// re-submitted to the generator all n times.
		wg.Wait()
		close(done)
	}()

	select {
	case <-time.After(2 * time.Second):
		t.Error("timed out waiting for tx to be resubmitted")
	case <-done:
		return
	}
}

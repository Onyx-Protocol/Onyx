package account_test

import (
	"context"
	"database/sql"
	"reflect"
	"testing"
	"time"

	"chain-stealth/core/account"
	"chain-stealth/core/asset"
	"chain-stealth/core/confidentiality"
	"chain-stealth/core/coretest"
	"chain-stealth/core/pin"
	"chain-stealth/core/query"
	"chain-stealth/core/txbuilder"
	"chain-stealth/database/pg"
	"chain-stealth/database/pg/pgtest"
	"chain-stealth/protocol/bc"
	"chain-stealth/protocol/mempool"
	"chain-stealth/protocol/prottest"
	"chain-stealth/testutil"
)

func TestAccountSourceReserve(t *testing.T) {
	var (
		_, db    = pgtest.NewDB(t, pgtest.SchemaPath)
		ctx      = context.Background()
		c        = prottest.NewChain(t)
		p        = mempool.New()
		pinStore = pin.NewStore(db)
		conf     = &confidentiality.Storage{DB: db}
		accounts = account.NewManager(db, c, pinStore, conf)
		assets   = asset.NewRegistry(db, c, pinStore, conf)
		indexer  = query.NewIndexer(db, c, pinStore)

		accID = coretest.CreateAccount(ctx, t, accounts, "", nil)
		asset = coretest.CreateAsset(ctx, t, assets, nil, "", nil)
		out   = coretest.IssueAssets(ctx, t, c, p, assets, accounts, asset, 2, accID)
	)

	coretest.CreatePins(ctx, t, pinStore)
	// Make a block so that account UTXOs are available to spend.
	assets.IndexAssets(indexer)
	accounts.IndexAccounts(indexer)
	go accounts.ProcessBlocks(ctx)
	prottest.MakeBlock(t, c, p.Dump(ctx))
	<-pinStore.PinWaiter(account.PinName, c.Height())

	assetAmount1 := bc.AssetAmount{
		AssetID: asset,
		Amount:  1,
	}
	source := accounts.NewSpendAction(assetAmount1, accID, nil, nil)

	builder := txbuilder.NewTemplateBuilder()
	err := source.Build(ctx, time.Now().Add(time.Minute), builder)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	tpl, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}

	if len(tpl.Transaction.Inputs) != 1 {
		t.Errorf("got %d inputs want %d", len(tpl.Transaction.Inputs), 1)
	}
	in, ok := tpl.Transaction.Inputs[0].TypedInput.(*bc.SpendInput)
	if !ok {
		t.Errorf("got %#v, want a spend input", tpl.Transaction.Inputs[0])
	}
	if in.Outpoint != out.Outpoint {
		t.Errorf("for input's prevout got %#v; want %#v", in.Outpoint, out.Outpoint)
	}
	if len(tpl.Transaction.Outputs) != 1 {
		t.Fatal("expected 1 change output")
	}
	if !programInAccount(ctx, t, db, tpl.Transaction.Outputs[0].Program(), accID) {
		t.Errorf("expected change control program to belong to account")
	}
	// TODO(jackson): decrypt the change output and verify its amount is 1
}

func TestAccountSourceUTXOReserve(t *testing.T) {
	var (
		_, db    = pgtest.NewDB(t, pgtest.SchemaPath)
		ctx      = context.Background()
		c        = prottest.NewChain(t)
		p        = mempool.New()
		pinStore = pin.NewStore(db)
		conf     = &confidentiality.Storage{DB: db}
		accounts = account.NewManager(db, c, pinStore, conf)
		assets   = asset.NewRegistry(db, c, pinStore, conf)
		indexer  = query.NewIndexer(db, c, pinStore)

		accID = coretest.CreateAccount(ctx, t, accounts, "", nil)
		asset = coretest.CreateAsset(ctx, t, assets, nil, "", nil)
		out   = coretest.IssueAssets(ctx, t, c, p, assets, accounts, asset, 2, accID)
	)

	coretest.CreatePins(ctx, t, pinStore)
	// Make a block so that account UTXOs are available to spend.
	assets.IndexAssets(indexer)
	accounts.IndexAccounts(indexer)
	go accounts.ProcessBlocks(ctx)
	prottest.MakeBlock(t, c, p.Dump(ctx))
	<-pinStore.PinWaiter(account.PinName, c.Height())

	source := accounts.NewSpendUTXOAction(out.Outpoint)

	builder := txbuilder.NewTemplateBuilder()
	err := source.Build(ctx, time.Now().Add(time.Minute), builder)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	tpl, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}

	if len(tpl.Transaction.Inputs) != 1 {
		t.Errorf("got %d inputs want %d", len(tpl.Transaction.Inputs), 1)
	}
	in, ok := tpl.Transaction.Inputs[0].TypedInput.(*bc.SpendInput)
	if !ok {
		t.Errorf("got %#v, want a spend input", tpl.Transaction.Inputs[0])
	}
	if in.Outpoint != out.Outpoint {
		t.Errorf("for input's prevout got %#v; want %#v", in.Outpoint, out.Outpoint)
	}
}

func TestAccountSourceReserveIdempotency(t *testing.T) {
	var (
		_, db    = pgtest.NewDB(t, pgtest.SchemaPath)
		ctx      = context.Background()
		c        = prottest.NewChain(t)
		p        = mempool.New()
		pinStore = pin.NewStore(db)
		conf     = &confidentiality.Storage{DB: db}
		accounts = account.NewManager(db, c, pinStore, conf)
		assets   = asset.NewRegistry(db, c, pinStore, conf)
		indexer  = query.NewIndexer(db, c, pinStore)

		accID        = coretest.CreateAccount(ctx, t, accounts, "", nil)
		asset        = coretest.CreateAsset(ctx, t, assets, nil, "", nil)
		_            = coretest.IssueAssets(ctx, t, c, p, assets, accounts, asset, 2, accID)
		_            = coretest.IssueAssets(ctx, t, c, p, assets, accounts, asset, 2, accID)
		assetAmount1 = bc.AssetAmount{
			AssetID: asset,
			Amount:  1,
		}

		// An idempotency key that both reservations should use.
		clientToken1 = "a-unique-idempotency-key"
		clientToken2 = "another-unique-idempotency-key"
		wantSrc      = accounts.NewSpendAction(assetAmount1, accID, nil, &clientToken1)
		gotSrc       = accounts.NewSpendAction(assetAmount1, accID, nil, &clientToken1)
		separateSrc  = accounts.NewSpendAction(assetAmount1, accID, nil, &clientToken2)
	)

	coretest.CreatePins(ctx, t, pinStore)
	// Make a block so that account UTXOs are available to spend.
	assets.IndexAssets(indexer)
	accounts.IndexAccounts(indexer)
	go accounts.ProcessBlocks(ctx)
	prottest.MakeBlock(t, c, p.Dump(ctx))
	<-pinStore.PinWaiter(account.PinName, c.Height())

	reserveFunc := func(source txbuilder.Action) []*bc.TxInput {
		builder := txbuilder.NewTemplateBuilder()
		err := source.Build(ctx, time.Now().Add(time.Minute), builder)
		if err != nil {
			testutil.FatalErr(t, err)
		}
		tpl, err := builder.Build()
		if err != nil {
			t.Fatal(err)
		}
		if len(tpl.Transaction.Inputs) != 1 {
			t.Fatalf("got %d result utxo, expected 1 result utxo", len(tpl.Transaction.Inputs))
		}
		return tpl.Transaction.Inputs
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

func programInAccount(ctx context.Context, t testing.TB, db pg.DB, program []byte, account string) bool {
	const q = `SELECT signer_id=$1 FROM account_control_programs WHERE control_program=$2`
	var in bool
	err := db.QueryRow(ctx, q, account, program).Scan(&in)
	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		testutil.FatalErr(t, err)
	}
	return in
}

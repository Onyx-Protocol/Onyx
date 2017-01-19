package core

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/coretest"
	"chain/core/generator"
	"chain/core/leader"
	"chain/core/pb"
	"chain/core/pin"
	"chain/core/query"
	"chain/core/txbuilder"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/prottest"
	"chain/protocol/vm"
	"chain/testutil"
)

func TestBuildFinal(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := context.Background()
	c := prottest.NewChain(t)
	g := generator.New(c, nil, db)
	pinStore := pin.NewStore(db)
	accounts := account.NewManager(db, c, pinStore)
	assets := asset.NewRegistry(db, c, pinStore)
	coretest.CreatePins(ctx, t, pinStore)
	accounts.IndexAccounts(query.NewIndexer(db, c, pinStore))
	go accounts.ProcessBlocks(ctx)

	acc := coretest.CreateAccount(ctx, t, accounts, "", nil)

	assetID := coretest.CreateAsset(ctx, t, assets, nil, "", nil)
	assetAmt := bc.AssetAmount{
		AssetID: assetID,
		Amount:  100,
	}

	sources := txbuilder.Action(assets.NewIssueAction(assetAmt, nil))
	dests := accounts.NewControlAction(assetAmt, acc, nil)

	tmpl, err := txbuilder.Build(ctx, nil, []txbuilder.Action{sources, dests}, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	coretest.SignTxTemplate(t, ctx, tmpl, &testutil.TestXPrv)

	txdata, err := bc.NewTxDataFromBytes(tmpl.RawTransaction)
	if err != nil {
		t.Fatal(err)
	}

	err = txbuilder.FinalizeTx(ctx, c, g, bc.NewTx(*txdata))
	if err != nil {
		t.Fatal(err)
	}

	// Make a block so that UTXOs from the above tx are available to spend.
	prottest.MakeBlock(t, c, g.PendingTxs())
	<-pinStore.PinWaiter(account.PinName, c.Height())

	sources = accounts.NewSpendAction(assetAmt, acc, nil, nil)
	tmpl, err = txbuilder.Build(ctx, nil, []txbuilder.Action{sources, dests}, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	// deep-copy tmpl via protobuf
	tmplProto, err := proto.Marshal(tmpl)
	if err != nil {
		t.Fatal(err)
	}
	tmpl2 := new(pb.TxTemplate)
	err = proto.Unmarshal(tmplProto, tmpl2)
	if err != nil {
		t.Fatal(err)
	}

	tmpl.AllowAdditionalActions = true
	tmpl2.AllowAdditionalActions = false

	coretest.SignTxTemplate(t, ctx, tmpl, nil)
	coretest.SignTxTemplate(t, ctx, tmpl2, nil)

	prog1 := tmpl.SigningInstructions[0].WitnessComponents[0].GetSignature().Program
	insts1, err := vm.ParseProgram(prog1)
	if err != nil {
		t.Fatal(err)
	}
	if len(insts1) != 23 {
		t.Fatalf("expected 23 instructions in sigwitness program 1, got %d (%x)", len(insts1), prog1)
	}
	if insts1[0].Op != vm.OP_MAXTIME {
		t.Fatalf("sigwitness program1 opcode 0 is %02x, expected %02x", insts1[0].Op, vm.OP_MAXTIME)
	}
	if insts1[2].Op != vm.OP_LESSTHANOREQUAL {
		t.Fatalf("sigwitness program1 opcode 2 is %02x, expected %02x", insts1[2].Op, vm.OP_LESSTHANOREQUAL)
	}
	if insts1[3].Op != vm.OP_VERIFY {
		t.Fatalf("sigwitness program1 opcode 3 is %02x, expected %02x", insts1[3].Op, vm.OP_VERIFY)
	}
	for i, op := range []vm.Op{vm.OP_FALSE, vm.OP_OUTPOINT, vm.OP_ROT, vm.OP_NUMEQUAL, vm.OP_VERIFY, vm.OP_EQUAL, vm.OP_VERIFY} {
		if insts1[i+5].Op != op {
			t.Fatalf("sigwitness program 1 opcode %d is %02x, expected %02x", i+5, insts1[i+5].Op, op)
		}
	}
	for i, op := range []vm.Op{vm.OP_REFDATAHASH, vm.OP_EQUAL, vm.OP_VERIFY, vm.OP_FALSE, vm.OP_FALSE} {
		if insts1[i+13].Op != op {
			t.Fatalf("sigwitness program 1 opcode %d is %02x, expected %02x", i+13, insts1[i+13].Op, op)
		}
	}
	if insts1[22].Op != vm.OP_CHECKOUTPUT {
		t.Fatalf("sigwitness program1 opcode 18 is %02x, expected %02x", insts1[18].Op, vm.OP_CHECKOUTPUT)
	}

	prog2 := tmpl2.SigningInstructions[0].WitnessComponents[0].GetSignature().Program
	insts2, err := vm.ParseProgram(prog2)
	if err != nil {
		t.Fatal(err)
	}
	if len(insts2) != 3 {
		t.Fatalf("expected 3 instructions in sigwitness program 2, got %d (%x)", len(insts2), prog2)
	}
	if insts2[1].Op != vm.OP_TXSIGHASH {
		t.Fatalf("sigwitness program2 opcode 1 is %02x, expected %02x", insts2[1].Op, vm.OP_TXSIGHASH)
	}
	if insts2[2].Op != vm.OP_EQUAL {
		t.Fatalf("sigwitness program2 opcode 2 is %02x, expected %02x", insts2[2].Op, vm.OP_EQUAL)
	}
}

func TestAccountTransfer(t *testing.T) {
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
		AssetID: assetID,
		Amount:  100,
	}

	sources := txbuilder.Action(assets.NewIssueAction(assetAmt, nil))
	dests := accounts.NewControlAction(assetAmt, acc, nil)
	tmpl, err := txbuilder.Build(ctx, nil, []txbuilder.Action{sources, dests}, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	coretest.SignTxTemplate(t, ctx, tmpl, &testutil.TestXPrv)

	txdata, err := bc.NewTxDataFromBytes(tmpl.RawTransaction)
	if err != nil {
		t.Fatal(err)
	}

	err = txbuilder.FinalizeTx(ctx, c, g, bc.NewTx(*txdata))
	if err != nil {
		t.Fatal(err)
	}

	// Make a block so that UTXOs from the above tx are available to spend.
	prottest.MakeBlock(t, c, g.PendingTxs())
	<-pinStore.PinWaiter(account.PinName, c.Height())

	// new source
	sources = accounts.NewSpendAction(assetAmt, acc, nil, nil)
	tmpl, err = txbuilder.Build(ctx, nil, []txbuilder.Action{sources, dests}, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	coretest.SignTxTemplate(t, ctx, tmpl, &testutil.TestXPrv)
	txdata, err = bc.NewTxDataFromBytes(tmpl.RawTransaction)
	if err != nil {
		t.Fatal(err)
	}

	err = txbuilder.FinalizeTx(ctx, c, g, bc.NewTx(*txdata))
	if err != nil {
		t.Fatal(err)
	}
}

func TestTransfer(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := context.Background()
	c := prottest.NewChain(t)
	g := generator.New(c, nil, db)
	pinStore := pin.NewStore(db)
	coretest.CreatePins(ctx, t, pinStore)
	handler := &Handler{
		Chain:     c,
		Submitter: g,
		Assets:    asset.NewRegistry(db, c, pinStore),
		Accounts:  account.NewManager(db, c, pinStore),
		Indexer:   query.NewIndexer(db, c, pinStore),
		DB:        db,
	}
	handler.Assets.IndexAssets(handler.Indexer)
	handler.Accounts.IndexAccounts(handler.Indexer)
	go handler.Accounts.ProcessBlocks(ctx)
	handler.Indexer.RegisterAnnotator(handler.Accounts.AnnotateTxs)
	handler.Indexer.RegisterAnnotator(handler.Assets.AnnotateTxs)

	// TODO(jackson): Replace this with a mock leader.
	var wg sync.WaitGroup
	wg.Add(1)
	go leader.Run(db, ":1999", func(ctx context.Context) {
		wg.Done()
	})
	wg.Wait()

	assetAlias := "some-asset"
	account1Alias := "first-account"
	account2Alias := "second-account"

	assetID := coretest.CreateAsset(ctx, t, handler.Assets, nil, assetAlias, nil)
	account1ID := coretest.CreateAccount(ctx, t, handler.Accounts, account1Alias, nil)
	account2ID := coretest.CreateAccount(ctx, t, handler.Accounts, account2Alias, nil)

	// Preface: issue some asset for account1ID to transfer to account2ID
	issueAssetAmount := bc.AssetAmount{
		AssetID: assetID,
		Amount:  100,
	}
	txTemplate, err := txbuilder.Build(ctx, nil, []txbuilder.Action{
		handler.Assets.NewIssueAction(issueAssetAmount, nil),
		handler.Accounts.NewControlAction(issueAssetAmount, account1ID, nil),
	}, time.Now().Add(time.Minute))
	if err != nil {
		testutil.FatalErr(t, err)
	}

	coretest.SignTxTemplate(t, ctx, txTemplate, nil)

	txdata, err := bc.NewTxDataFromBytes(txTemplate.RawTransaction)
	if err != nil {
		t.Fatal(err)
	}

	err = txbuilder.FinalizeTx(ctx, c, g, bc.NewTx(*txdata))
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// Make a block so that UTXOs from the above tx are available to spend.
	prottest.MakeBlock(t, c, g.PendingTxs())
	<-pinStore.PinWaiter(account.PinName, c.Height())

	// Now transfer
	buildResult, err := handler.BuildTxs(ctx, &pb.BuildTxsRequest{
		Requests: []*pb.BuildTxsRequest_Request{{
			Actions: []*pb.Action{{
				Action: &pb.Action_SpendAccount_{
					SpendAccount: &pb.Action_SpendAccount{
						Asset:   &pb.AssetIdentifier{Identifier: &pb.AssetIdentifier_AssetId{AssetId: assetID[:]}},
						Amount:  100,
						Account: &pb.AccountIdentifier{Identifier: &pb.AccountIdentifier_AccountId{AccountId: account1ID}},
					},
				},
			}, {
				Action: &pb.Action_ControlAccount_{
					ControlAccount: &pb.Action_ControlAccount{
						Asset:   &pb.AssetIdentifier{Identifier: &pb.AssetIdentifier_AssetId{AssetId: assetID[:]}},
						Amount:  100,
						Account: &pb.AccountIdentifier{Identifier: &pb.AccountIdentifier_AccountId{AccountId: account2ID}},
					},
				},
			}},
		}},
	})
	if err != nil {
		testutil.FatalErr(t, err)
	}
	inspectTxResps(t, buildResult)

	txTemplate = buildResult.Responses[0].Template

	if len(txTemplate.SigningInstructions) != 1 {
		t.Errorf("expected template.signing_instructions in result to have length 1, got %d", len(txTemplate.SigningInstructions))
	}

	coretest.SignTxTemplate(t, ctx, txTemplate, &testutil.TestXPrv)
	_, err = handler.submitSingle(ctx, txTemplate, "none")
	if err != nil && errors.Root(err) != context.DeadlineExceeded {
		testutil.FatalErr(t, err)
	}

	// Make a block so that UTXOs from the above tx are available to spend.
	prottest.MakeBlock(t, c, g.PendingTxs())
	<-pinStore.PinWaiter(account.PinName, c.Height())

	// Now transfer back using aliases.
	buildResult, err = handler.BuildTxs(ctx, &pb.BuildTxsRequest{
		Requests: []*pb.BuildTxsRequest_Request{{
			Actions: []*pb.Action{{
				Action: &pb.Action_SpendAccount_{
					SpendAccount: &pb.Action_SpendAccount{
						Asset:   &pb.AssetIdentifier{Identifier: &pb.AssetIdentifier_AssetAlias{AssetAlias: assetAlias}},
						Amount:  100,
						Account: &pb.AccountIdentifier{Identifier: &pb.AccountIdentifier_AccountAlias{AccountAlias: account2Alias}},
					},
				},
			}, {
				Action: &pb.Action_ControlAccount_{
					ControlAccount: &pb.Action_ControlAccount{
						Asset:   &pb.AssetIdentifier{Identifier: &pb.AssetIdentifier_AssetAlias{AssetAlias: assetAlias}},
						Amount:  100,
						Account: &pb.AccountIdentifier{Identifier: &pb.AccountIdentifier_AccountAlias{AccountAlias: account1Alias}},
					},
				},
			}},
		}},
	})
	if err != nil {
		testutil.FatalErr(t, err)
	}
	inspectTxResps(t, buildResult)

	txTemplate = buildResult.Responses[0].Template

	if len(txTemplate.SigningInstructions) != 1 {
		t.Errorf("expected template.signing_instructions in result to have length 1, got %d", len(txTemplate.SigningInstructions))
	}

	coretest.SignTxTemplate(t, ctx, txTemplate, &testutil.TestXPrv)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	_, err = handler.submitSingle(ctx, txTemplate, "none")
	if err != nil && errors.Root(err) != context.DeadlineExceeded {
		testutil.FatalErr(t, err)
	}
}

func inspectTxResps(t testing.TB, resp *pb.TxsResponse) {
	var erred bool
	for _, r := range resp.Responses {
		if r.Error != nil {
			t.Errorf("%+v\n", r.Error)
			erred = true
		}
	}
	if erred {
		t.Fatal()
	}
}

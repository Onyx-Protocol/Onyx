package core

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/config"
	"chain/core/coretest"
	"chain/core/leader"
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
	pinStore := pin.NewStore(db)
	accounts := account.NewManager(db, c, pinStore)
	assets := asset.NewRegistry(db, c, pinStore)
	coretest.CreatePins(ctx, t, pinStore)
	accounts.IndexAccounts(query.NewIndexer(db, c, pinStore))
	go accounts.ProcessBlocks(ctx)

	acc, err := accounts.Create(ctx, []string{testutil.TestXPub.String()}, 1, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	assetID := coretest.CreateAsset(ctx, t, assets, nil, "", nil)
	assetAmt := bc.AssetAmount{
		AssetID: assetID,
		Amount:  100,
	}

	sources := txbuilder.Action(assets.NewIssueAction(assetAmt, nil))
	dests := accounts.NewControlAction(assetAmt, acc.ID, nil)

	tmpl, err := txbuilder.Build(ctx, nil, []txbuilder.Action{sources, dests}, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	coretest.SignTxTemplate(t, ctx, tmpl, &testutil.TestXPrv)
	err = txbuilder.FinalizeTx(ctx, c, bc.NewTx(*tmpl.Transaction))
	if err != nil {
		t.Fatal(err)
	}

	// Make a block so that UTXOs from the above tx are available to spend.
	prottest.MakeBlock(t, c)
	<-pinStore.PinWaiter(account.PinName, c.Height())

	sources = accounts.NewSpendAction(assetAmt, acc.ID, nil, nil)
	tmpl, err = txbuilder.Build(ctx, nil, []txbuilder.Action{sources, dests}, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	// deep-copy tmpl via json
	tmplJSON, err := json.Marshal(tmpl)
	if err != nil {
		t.Fatal(err)
	}
	var tmpl2 txbuilder.Template
	err = json.Unmarshal(tmplJSON, &tmpl2)
	if err != nil {
		t.Fatal(err)
	}

	tmpl.AllowAdditional = true
	tmpl2.AllowAdditional = false

	coretest.SignTxTemplate(t, ctx, tmpl, nil)
	coretest.SignTxTemplate(t, ctx, &tmpl2, nil)

	prog1 := tmpl.SigningInstructions[0].WitnessComponents[0].(*txbuilder.SignatureWitness).Program
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

	prog2 := tmpl2.SigningInstructions[0].WitnessComponents[0].(*txbuilder.SignatureWitness).Program
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
	pinStore := pin.NewStore(db)
	assets := asset.NewRegistry(db, c, pinStore)
	accounts := account.NewManager(db, c, pinStore)
	coretest.CreatePins(ctx, t, pinStore)
	accounts.IndexAccounts(query.NewIndexer(db, c, pinStore))
	go accounts.ProcessBlocks(ctx)

	acc, err := accounts.Create(ctx, []string{testutil.TestXPub.String()}, 1, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	assetID := coretest.CreateAsset(ctx, t, assets, nil, "", nil)
	assetAmt := bc.AssetAmount{
		AssetID: assetID,
		Amount:  100,
	}

	sources := txbuilder.Action(assets.NewIssueAction(assetAmt, nil))
	dests := accounts.NewControlAction(assetAmt, acc.ID, nil)
	tmpl, err := txbuilder.Build(ctx, nil, []txbuilder.Action{sources, dests}, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	coretest.SignTxTemplate(t, ctx, tmpl, &testutil.TestXPrv)
	err = txbuilder.FinalizeTx(ctx, c, bc.NewTx(*tmpl.Transaction))
	if err != nil {
		t.Fatal(err)
	}

	// Make a block so that UTXOs from the above tx are available to spend.
	prottest.MakeBlock(t, c)
	<-pinStore.PinWaiter(account.PinName, c.Height())

	// new source
	sources = accounts.NewSpendAction(assetAmt, acc.ID, nil, nil)
	tmpl, err = txbuilder.Build(ctx, nil, []txbuilder.Action{sources, dests}, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	coretest.SignTxTemplate(t, ctx, tmpl, &testutil.TestXPrv)
	err = txbuilder.FinalizeTx(ctx, c, bc.NewTx(*tmpl.Transaction))
	if err != nil {
		t.Fatal(err)
	}
}

func TestMux(t *testing.T) {
	// Handler calls handleJSON, which panics
	// if the function signature is not of the right form.
	// So call Handler here and rescue any panic
	// to check for this case.
	defer func() {
		if err := recover(); err != nil {
			t.Fatal("unexpected panic:", err)
		}
	}()
	(&Handler{Config: &config.Config{}}).init()
}

func TestTransfer(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := context.Background()
	c := prottest.NewChain(t)
	pinStore := pin.NewStore(db)
	coretest.CreatePins(ctx, t, pinStore)
	handler := &Handler{
		Chain:    c,
		Assets:   asset.NewRegistry(db, c, pinStore),
		Accounts: account.NewManager(db, c, pinStore),
		Indexer:  query.NewIndexer(db, c, pinStore),
		DB:       db,
	}
	handler.Assets.IndexAssets(handler.Indexer)
	handler.Accounts.IndexAccounts(handler.Indexer)
	go handler.Accounts.ProcessBlocks(ctx)
	handler.Indexer.RegisterAnnotator(handler.Accounts.AnnotateTxs)
	handler.Indexer.RegisterAnnotator(handler.Assets.AnnotateTxs)
	handler.init()

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

	assetIDStr := assetID.String()

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
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	coretest.SignTxTemplate(t, ctx, txTemplate, nil)

	err = txbuilder.FinalizeTx(ctx, c, bc.NewTx(*txTemplate.Transaction))
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	// Make a block so that UTXOs from the above tx are available to spend.
	prottest.MakeBlock(t, c)
	<-pinStore.PinWaiter(account.PinName, c.Height())

	// Now transfer
	buildReqFmt := `
		{"actions": [
			{"type": "spend_account", "asset_id": "%s", "amount": 100, "account_id": "%s"},
			{"type": "control_account", "asset_id": "%s", "amount": 100, "account_id": "%s"}
		]}
	`
	buildReqStr := fmt.Sprintf(buildReqFmt, assetIDStr, account1ID, assetIDStr, account2ID)
	var buildReq buildRequest
	err = json.Unmarshal([]byte(buildReqStr), &buildReq)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	buildResult, err := handler.build(ctx, []*buildRequest{&buildReq})
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
	jsonResult, err := json.MarshalIndent(buildResult, "", "  ")
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	var parsedResult []map[string]interface{}
	err = json.Unmarshal(jsonResult, &parsedResult)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
	if len(parsedResult) != 1 {
		t.Errorf("expected build result to have length 1, got %d", len(parsedResult))
	}
	toSign := inspectTemplate(t, parsedResult[0], account2ID)
	txTemplate, err = toTxTemplate(ctx, toSign)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
	coretest.SignTxTemplate(t, ctx, txTemplate, &testutil.TestXPrv)
	_, err = handler.submitSingle(ctx, txTemplate, "none")
	if err != nil && errors.Root(err) != context.DeadlineExceeded {
		testutil.FatalErr(t, err)
	}

	// Now transfer back using aliases.
	buildReqFmt = `
		{"actions": [
			{"type": "spend_account", "params": {"asset_alias": "%s", "amount": 100, "account_alias": "%s"}},
			{"type": "control_account", "params": {"asset_alias": "%s", "amount": 100, "account_alias": "%s"}}
		]}
	`
	buildReqStr = fmt.Sprintf(buildReqFmt, assetAlias, account2Alias, assetAlias, account1Alias)
	err = json.Unmarshal([]byte(buildReqStr), &buildReq)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	buildResult, err = handler.build(ctx, []*buildRequest{&buildReq})
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
	jsonResult, err = json.MarshalIndent(buildResult, "", "  ")
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	err = json.Unmarshal(jsonResult, &parsedResult)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
	if len(parsedResult) != 1 {
		t.Errorf("expected build result to have length 1, got %d", len(parsedResult))
	}
	toSign = inspectTemplate(t, parsedResult[0], account2ID)
	txTemplate, err = toTxTemplate(ctx, toSign)
	coretest.SignTxTemplate(t, ctx, txTemplate, &testutil.TestXPrv)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
	_, err = handler.submitSingle(ctx, txTemplate, "none")
	if err != nil && errors.Root(err) != context.DeadlineExceeded {
		testutil.FatalErr(t, err)
	}
}

// expects inp to be a map, with one input member
func inspectTemplate(t *testing.T, inp map[string]interface{}, expectedReceiverAccountID string) map[string]interface{} {
	member, ok := inp["signing_instructions"]
	if !ok {
		t.Errorf("expected template.signing_instructions in result")
	}
	parsedInputs, ok := member.([]interface{})
	if !ok {
		t.Errorf("expected template.signing_instructions in result to be a list")
	}
	if len(parsedInputs) != 1 {
		t.Errorf("expected template.signing_instructions in result to have length 1, got %d", len(parsedInputs))
	}

	return inp
}

func toTxTemplate(ctx context.Context, inp map[string]interface{}) (*txbuilder.Template, error) {
	jsonInp, err := json.Marshal(inp)
	if err != nil {
		return nil, err
	}
	tpl := new(txbuilder.Template)
	err = json.Unmarshal(jsonInp, tpl)
	return tpl, err
}

package core

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/asset/assettest"
	"chain/core/query"
	"chain/core/txbuilder"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/prottest"
	"chain/testutil"
)

func TestAccountTransfer(t *testing.T) {
	dbtx := pgtest.NewTx(t)
	ctx := pg.NewContext(context.Background(), dbtx)
	c := prottest.NewChain(t)
	asset.Init(c, nil)
	account.Init(c, nil)

	acc, err := account.Create(ctx, []string{testutil.TestXPub.String()}, 1, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	assetID := assettest.CreateAssetFixture(ctx, t, nil, 1, nil, "", nil)
	assetAmt := bc.AssetAmount{
		AssetID: assetID,
		Amount:  100,
	}

	sources := txbuilder.Action(assettest.NewIssueAction(assetAmt, nil))
	dests := assettest.NewAccountControlAction(assetAmt, acc.ID, nil)

	tmpl, err := txbuilder.Build(ctx, nil, []txbuilder.Action{sources, dests}, nil, bc.Millis(time.Now().Add(time.Minute)))
	if err != nil {
		t.Fatal(err)
	}

	assettest.SignTxTemplate(t, tmpl, testutil.TestXPrv)
	_, err = txbuilder.FinalizeTx(ctx, c, tmpl)
	if err != nil {
		t.Fatal(err)
	}

	// Make a block so that UTXOs from the above tx are available to spend.
	prottest.MakeBlock(ctx, t, c)

	// new source
	sources = assettest.NewAccountSpendAction(assetAmt, acc.ID, nil, nil, nil, nil)
	tmpl, err = txbuilder.Build(ctx, nil, []txbuilder.Action{sources, dests}, nil, bc.Millis(time.Now().Add(time.Minute)))
	if err != nil {
		t.Fatal(err)
	}

	assettest.SignTxTemplate(t, tmpl, testutil.TestXPrv)
	_, err = txbuilder.FinalizeTx(ctx, c, tmpl)
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
	Handler("", "", nil, nil, nil, nil, &Config{})
}

func TestTransfer(t *testing.T) {
	dbtx := pgtest.NewTx(t)
	ctx := pg.NewContext(context.Background(), dbtx)
	c := prottest.NewChain(t)
	asset.Init(c, nil)
	account.Init(c, nil)

	ind := query.NewIndexer(dbtx, c)
	asset.Init(c, ind)
	account.Init(c, ind)

	assetAlias := "some-asset"
	account1Alias := "first-account"
	account2Alias := "second-account"

	assetID := assettest.CreateAssetFixture(ctx, t, nil, 1, nil, assetAlias, nil)
	account1ID := assettest.CreateAccountFixture(ctx, t, nil, 0, account1Alias, nil)
	account2ID := assettest.CreateAccountFixture(ctx, t, nil, 0, account2Alias, nil)

	assetIDStr := assetID.String()

	// Preface: issue some asset for account1ID to transfer to account2ID
	issueAssetAmount := bc.AssetAmount{
		AssetID: assetID,
		Amount:  100,
	}
	issueDest := assettest.NewAccountControlAction(issueAssetAmount, account1ID, nil)
	txTemplate, err := txbuilder.Build(
		ctx,
		nil,
		[]txbuilder.Action{assettest.NewIssueAction(issueAssetAmount, nil), issueDest},
		nil,
		bc.Millis(time.Now().Add(time.Minute)),
	)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	assettest.SignTxTemplate(t, txTemplate, nil)

	_, err = txbuilder.FinalizeTx(ctx, c, txTemplate)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	// Make a block so that UTXOs from the above tx are available to spend.
	prottest.MakeBlock(ctx, t, c)

	// Now transfer
	buildReqFmt := `
		{"actions": [
			{"type": "spend_account_unspent_output_selector", "params": {"asset_id": "%s", "amount": 100, "account_id": "%s"}},
			{"type": "control_account", "params": {"asset_id": "%s", "amount": 100, "account_id": "%s"}}
		]}
	`
	buildReqStr := fmt.Sprintf(buildReqFmt, assetIDStr, account1ID, assetIDStr, account2ID)
	var buildReq buildRequest
	err = json.Unmarshal([]byte(buildReqStr), &buildReq)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	buildResult, err := build(ctx, []*buildRequest{&buildReq})
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
	assettest.SignTxTemplate(t, txTemplate, testutil.TestXPrv)
	_, err = submitSingle(ctx, c, submitSingleArg{tpl: txTemplate, wait: time.Millisecond})
	if err != nil && err != context.DeadlineExceeded {
		testutil.FatalErr(t, err)
	}

	// Now transfer back using aliases.
	buildReqFmt = `
		{"actions": [
			{"type": "spend_account_unspent_output_selector", "params": {"asset_alias": "%s", "amount": 100, "account_alias": "%s"}},
			{"type": "control_account", "params": {"asset_alias": "%s", "amount": 100, "account_alias": "%s"}}
		]}
	`
	buildReqStr = fmt.Sprintf(buildReqFmt, assetAlias, account2Alias, assetAlias, account1Alias)
	err = json.Unmarshal([]byte(buildReqStr), &buildReq)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	buildResult, err = build(ctx, []*buildRequest{&buildReq})
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
	assettest.SignTxTemplate(t, txTemplate, testutil.TestXPrv)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
	_, err = submitSingle(ctx, c, submitSingleArg{tpl: txTemplate, wait: time.Millisecond})
	if err != nil && err != context.DeadlineExceeded {
		testutil.FatalErr(t, err)
	}
}

// expects inp to be a map, with one input member
func inspectTemplate(t *testing.T, inp map[string]interface{}, expectedReceiverAccountID string) map[string]interface{} {
	member, ok := inp["inputs"]
	if !ok {
		t.Errorf("expected template.inputs in result")
	}
	parsedInputs, ok := member.([]interface{})
	if !ok {
		t.Errorf("expected template.inputs in result to be a list")
	}
	if len(parsedInputs) != 1 {
		t.Errorf("expected template.inputs in result to have length 1, got %d", len(parsedInputs))
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

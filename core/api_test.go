package core

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"golang.org/x/net/context"

	"chain/core/accounts"
	"chain/core/asset"
	"chain/core/asset/assettest"
	"chain/core/issuer"
	"chain/core/txbuilder"
	"chain/cos/bc"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/net/http/authn"
	"chain/testutil"
)

func TestAccountTransfer(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	_, _, err := assettest.InitializeSigningGenerator(ctx, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	acc, err := accounts.Create(ctx, []string{testutil.TestXPub.String()}, 1, nil)
	if err != nil {
		t.Fatal(err)
	}

	assetID := assettest.CreateAssetFixture(ctx, t, "", "", "")
	assetAmt := bc.AssetAmount{
		AssetID: assetID,
		Amount:  100,
	}
	sources := issuer.NewIssueSource(ctx, assetAmt, nil, nil)
	dests, err := accounts.NewDestination(ctx, &assetAmt, acc.ID, nil)
	if err != nil {
		t.Fatal(err)
	}

	tmpl, err := txbuilder.Build(ctx, nil, []*txbuilder.Source{sources}, []*txbuilder.Destination{dests}, nil, time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	assettest.SignTxTemplate(t, tmpl, testutil.TestXPrv)
	_, err = asset.FinalizeTx(ctx, tmpl)
	if err != nil {
		t.Fatal(err)
	}

	// new source
	sources = accounts.NewSource(ctx, &assetAmt, acc.ID, nil, nil, nil)
	tmpl, err = txbuilder.Build(ctx, nil, []*txbuilder.Source{sources}, []*txbuilder.Destination{dests}, nil, time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	assettest.SignTxTemplate(t, tmpl, testutil.TestXPrv)
	_, err = asset.FinalizeTx(ctx, tmpl)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMux(t *testing.T) {
	// Handler calls httpjson.HandleFunc, which panics
	// if the function signature is not of the right form.
	// So call Handler here and rescue any panic
	// to check for this case.
	defer func() {
		if err := recover(); err != nil {
			t.Fatal("unexpected panic:", err)
		}
	}()
	Handler("", nil, nil, nil, nil, nil, nil)
}

func TestLogin(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	uid := assettest.CreateUserFixture(ctx, t, "foo@bar.com", "abracadabra", "developer")
	ctx = authn.NewContext(ctx, uid)

	tok, err := login(ctx)
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	// Verify that the token is valid
	gotUID, err := authenticateToken(ctx, tok.ID, tok.Secret)
	if err != nil {
		t.Errorf("authenticate token err = %v want nil", err)
	}
	if gotUID != uid {
		t.Errorf("authenticated user ID = %v want %v", gotUID, uid)
	}
}

func TestIssue(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	fc, _, err := assettest.InitializeSigningGenerator(ctx, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	asset.Init(fc, true)
	accounts.Init(fc)

	userID := assettest.CreateUserFixture(ctx, t, "", "", "")
	projectID := assettest.CreateProjectFixture(ctx, t, "")
	issuerNodeID := assettest.CreateIssuerNodeFixture(ctx, t, projectID, "", nil, nil)
	assetID := assettest.CreateAssetFixture(ctx, t, issuerNodeID, "", "")
	account1ID := assettest.CreateAccountFixture(ctx, t, nil, 0)

	ctx = authn.NewContext(ctx, userID)

	reqDestFmt := `
		{"asset_id": "%s",
		 "amount": 100,
		 "account_id": "%s"}
	`
	reqDestStr := fmt.Sprintf(reqDestFmt, assetID.String(), account1ID)
	var reqDest Destination
	err = json.Unmarshal([]byte(reqDestStr), &reqDest)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
	result, err := issueAsset(ctx, assetID.String(), []*Destination{&reqDest})
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
	jsonResult, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
	var parsedResult map[string]interface{}
	err = json.Unmarshal(jsonResult, &parsedResult)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
	inspectTemplate(t, parsedResult, account1ID)
}

func TestTransfer(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	fc, _, err := assettest.InitializeSigningGenerator(ctx, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	asset.Init(fc, true)
	accounts.Init(fc)

	userID := assettest.CreateUserFixture(ctx, t, "", "", "")
	projectID := assettest.CreateProjectFixture(ctx, t, "")
	issuerNodeID := assettest.CreateIssuerNodeFixture(ctx, t, projectID, "", nil, nil)
	assetID := assettest.CreateAssetFixture(ctx, t, issuerNodeID, "", "")
	account1ID := assettest.CreateAccountFixture(ctx, t, nil, 0)
	account2ID := assettest.CreateAccountFixture(ctx, t, nil, 0)

	assetIDStr := assetID.String()

	ctx = authn.NewContext(ctx, userID)

	// Preface: issue some asset for account1ID to transfer to account2ID
	issueAssetAmount := bc.AssetAmount{
		AssetID: assetID,
		Amount:  100,
	}
	issueDest, err := accounts.NewDestination(ctx, &issueAssetAmount, account1ID, nil)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
	txTemplate, err := issuer.Issue(ctx, issueAssetAmount, []*txbuilder.Destination{issueDest})
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	assettest.SignTxTemplate(t, txTemplate, nil)

	_, err = asset.FinalizeTx(ctx, txTemplate)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	// Now transfer
	buildReqFmt := `
		{"inputs": [{"asset_id": "%s", "amount": 100, "account_id": "%s", "type": "account"}],
		 "outputs": [{"asset_id": "%s", "amount": 100, "account_id": "%s", "type": "account"}]}
	`
	buildReqStr := fmt.Sprintf(buildReqFmt, assetIDStr, account1ID, assetIDStr, account2ID)
	var buildReq BuildRequest
	err = json.Unmarshal([]byte(buildReqStr), &buildReq)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
	buildResult, err := build(ctx, []*BuildRequest{&buildReq})
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
	assettest.SignTxTemplate(t, txTemplate, testutil.TestXPrv)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
	_, err = submitSingle(ctx, submitSingleArg{tpl: txTemplate, wait: time.Millisecond})
	if err != nil && err != context.DeadlineExceeded {
		testutil.FatalErr(t, err)
	}
}

// expects inp to have one "template" member, with one input and one output
func inspectTemplate(t *testing.T, inp map[string]interface{}, expectedReceiverAccountID string) map[string]interface{} {
	member, ok := inp["template"]
	if !ok {
		t.Errorf("expected \"template\" in result")
	}
	parsedTemplate, ok := member.(map[string]interface{})
	if !ok {
		t.Errorf("expected \"template\" in result to be a map")
	}
	member, ok = parsedTemplate["inputs"]
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
	return parsedTemplate
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

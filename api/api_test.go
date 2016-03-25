package api

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"golang.org/x/net/context"

	"chain/api/asset"
	"chain/api/asset/assettest"
	"chain/api/issuer"
	"chain/api/smartcontracts/orderbook"
	"chain/api/txbuilder"
	"chain/api/txdb"
	"chain/cos"
	"chain/cos/bc"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/net/http/authn"
	chaintest "chain/testutil"
)

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
	Handler("", nil)
}

func TestLogin(t *testing.T) {
	ctx := pgtest.NewContext(t)
	uid := assettest.CreateUserFixture(ctx, t, "foo@bar.com", "abracadabra")
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
	ctx := apiTest(t)

	_, err := assettest.InitializeSigningGenerator(ctx)
	if err != nil {
		t.Fatal(err)
	}

	userID := assettest.CreateUserFixture(ctx, t, "", "")
	projectID := assettest.CreateProjectFixture(ctx, t, userID, "")
	issuerNodeID := assettest.CreateIssuerNodeFixture(ctx, t, projectID, "", nil, nil)
	managerNodeID := assettest.CreateManagerNodeFixture(ctx, t, projectID, "", nil, nil)
	assetID := assettest.CreateAssetFixture(ctx, t, issuerNodeID, "", "")
	account1ID := assettest.CreateAccountFixture(ctx, t, managerNodeID, "", nil)

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
	inspectTemplate(t, parsedResult, managerNodeID, account1ID)
}

func TestTransfer(t *testing.T) {
	ctx := apiTest(t)

	_, err := assettest.InitializeSigningGenerator(ctx)
	if err != nil {
		t.Fatal(err)
	}

	userID := assettest.CreateUserFixture(ctx, t, "", "")
	projectID := assettest.CreateProjectFixture(ctx, t, userID, "")
	issuerNodeID := assettest.CreateIssuerNodeFixture(ctx, t, projectID, "", nil, nil)
	managerNodeID := assettest.CreateManagerNodeFixture(ctx, t, projectID, "", nil, nil)
	assetID := assettest.CreateAssetFixture(ctx, t, issuerNodeID, "", "")
	account1ID := assettest.CreateAccountFixture(ctx, t, managerNodeID, "", nil)
	account2ID := assettest.CreateAccountFixture(ctx, t, managerNodeID, "", nil)

	assetIDStr := assetID.String()

	ctx = authn.NewContext(ctx, userID)

	// Preface: issue some asset for account1ID to transfer to account2ID
	issueAssetAmount := &bc.AssetAmount{
		AssetID: assetID,
		Amount:  100,
	}
	issueDest, err := asset.NewAccountDestination(ctx, issueAssetAmount, account1ID, nil)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
	txTemplate, err := issuer.Issue(ctx, assetID, []*txbuilder.Destination{issueDest})
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

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
	toSign := inspectTemplate(t, parsedResult[0], managerNodeID, account2ID)
	txTemplate, err = toTxTemplate(ctx, toSign)
	assettest.SignTxTemplate(t, txTemplate, chaintest.TestXPrv)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
	_, err = submitSingle(ctx, submitSingleArg{tpl: txTemplate, wait: time.Millisecond})
	if err != nil && err != context.DeadlineExceeded {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
}

// expects inp to have one "template" member, with one input and one output
func inspectTemplate(t *testing.T, inp map[string]interface{}, expectedReceiverManagerNodeID, expectedReceiverAccountID string) map[string]interface{} {
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

func apiTest(t testing.TB) context.Context {
	ctx := pgtest.NewContext(t)

	fc, err := cos.NewFC(ctx, txdb.NewStore(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	asset.Init(fc, true)
	orderbook.Connect(fc)

	return ctx
}

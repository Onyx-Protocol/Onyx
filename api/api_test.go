package api

import (
	"encoding/json"
	"fmt"
	"testing"

	"golang.org/x/net/context"

	"chain/api/asset"
	"chain/api/asset/assettest"
	"chain/api/txbuilder"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/fedchain/bc"
	"chain/net/http/authn"
	chaintest "chain/testutil"
)

const testUserFixture = `
	INSERT INTO users (id, email, password_hash) VALUES (
		'sample-user-id-0',
		'foo@bar.com',
		'$2a$08$WF7tWRx/26m9Cp2kQBQEwuKxCev9S4TSzWdmtNmHSvan4UhEw0Er.'::bytea -- plaintext: abracadabra
	);
`

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
	Handler("")
}

func TestLogin(t *testing.T) {
	ctx := pgtest.NewContext(t, testUserFixture)
	defer pgtest.Finish(ctx)
	ctx = authn.NewContext(ctx, "sample-user-id-0")

	tok, err := login(ctx)
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	// Verify that the token is valid
	uid, err := authenticateToken(ctx, tok.ID, tok.Secret)
	if err != nil {
		t.Errorf("authenticate token err = %v want nil", err)
	}
	if uid != "sample-user-id-0" {
		t.Errorf("authenticated user ID = %v want sample-user-id-0", uid)
	}
}

func TestIssue(t *testing.T) {
	ctx := assettest.NewContextWithGenesisBlock(t)
	defer pgtest.Finish(ctx)

	userID := assettest.CreateUserFixture(ctx, t, "", "")
	projectID := assettest.CreateProjectFixture(ctx, t, userID, "")
	issuerNodeID := assettest.CreateIssuerNodeFixture(ctx, t, projectID, "", nil, nil)
	managerNodeID := assettest.CreateManagerNodeFixture(ctx, t, projectID, "", nil, nil)
	assetID := assettest.CreateAssetFixture(ctx, t, issuerNodeID, "")
	account1ID := assettest.CreateAccountFixture(ctx, t, managerNodeID, "", nil)

	ctx = authn.NewContext(ctx, userID)

	reqDestFmt := `
		{"asset_id": "%s",
		 "amount": 100,
		 "account_id": "%s"}
	`
	reqDestStr := fmt.Sprintf(reqDestFmt, assetID.String(), account1ID)
	var reqDest Destination
	err := json.Unmarshal([]byte(reqDestStr), &reqDest)
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
	ctx := assettest.NewContextWithGenesisBlock(t)
	defer pgtest.Finish(ctx)

	userID := assettest.CreateUserFixture(ctx, t, "", "")
	projectID := assettest.CreateProjectFixture(ctx, t, userID, "")
	issuerNodeID := assettest.CreateIssuerNodeFixture(ctx, t, projectID, "", nil, nil)
	managerNodeID := assettest.CreateManagerNodeFixture(ctx, t, projectID, "", nil, nil)
	assetID := assettest.CreateAssetFixture(ctx, t, issuerNodeID, "")
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
	txTemplate, err := asset.Issue(ctx, assetIDStr, []*txbuilder.Destination{issueDest})
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
	err = assettest.SignTxTemplate(txTemplate, chaintest.TestXPrv)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
	signedTemplate, err := toRequestTemplate(txTemplate)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
	_, err = submitSingle(ctx, signedTemplate)
	if err != nil {
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
	member, ok = parsedTemplate["output_receivers"]
	if !ok {
		t.Errorf("expected template.output_receivers in result")
	}
	parsedReceivers, ok := member.([]interface{})
	if !ok {
		t.Errorf("expected template.output_receivers to be a list")
	}
	if len(parsedReceivers) != 1 {
		t.Errorf("expected template.output_receivers in result to have length 1, got %d", len(parsedReceivers))
	}
	member = parsedReceivers[0]
	parsedReceiver, ok := member.(map[string]interface{})
	if !ok {
		t.Errorf("expected template.output_receivers in result to be a list of maps")
	}
	member, ok = parsedReceiver["account_id"]
	if !ok {
		t.Errorf("expected template.output_receivers[0].account_id in result")
	}
	receiverAccountID, ok := member.(string)
	if !ok {
		t.Errorf("expected template.output_receivers[0].account_id in result to be a string")
	}
	if receiverAccountID != expectedReceiverAccountID {
		t.Errorf("expected template.output_receivers[0].account_id in result to be %s, got %s", expectedReceiverAccountID, receiverAccountID)
	}
	member, ok = parsedReceiver["manager_node_id"]
	if !ok {
		t.Errorf("expected template.output_receivers[0].manager_node_id in result")
	}
	receiverManagerNodeID, ok := member.(string)
	if !ok {
		t.Errorf("expected template.output_receivers[0].manager_node_id in result to be a string")
	}
	if receiverManagerNodeID != expectedReceiverManagerNodeID {
		t.Errorf("expected template.output_receivers[0].manager_node_id in result to be %s, got %s", expectedReceiverManagerNodeID, receiverManagerNodeID)
	}
	member, ok = parsedReceiver["type"]
	if !ok {
		t.Errorf("expected template.output_receivers[0].type in result")
	}
	receiverType, ok := member.(string)
	if !ok {
		t.Errorf("expected template.output_receivers[0].type in result to be a string")
	}
	if receiverType != "account" {
		t.Errorf("expected template.output_receivers[0].type in result to be account, got %s", receiverType)
	}
	return parsedTemplate
}

func toTxTemplate(ctx context.Context, inp map[string]interface{}) (*txbuilder.Template, error) {
	jsonInp, err := json.Marshal(inp)
	if err != nil {
		return nil, err
	}
	var tpl Template
	err = json.Unmarshal(jsonInp, &tpl)
	if err != nil {
		return nil, err
	}
	return tpl.parse(ctx)
}

func toRequestTemplate(inp *txbuilder.Template) (*Template, error) {
	jsonInp, err := json.Marshal(inp)
	if err != nil {
		return nil, err
	}
	var tpl Template
	err = json.Unmarshal(jsonInp, &tpl)
	return &tpl, err
}

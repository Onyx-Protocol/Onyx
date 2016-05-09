package voting

import (
	"reflect"
	"testing"
	"time"

	"chain/api/asset"
	"chain/api/asset/assettest"
	"chain/api/generator"
	"chain/api/txbuilder"
	"chain/api/txdb"
	"chain/cos/bc"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/database/sql"
	"chain/testutil"
)

// TestAuthenticateEndToEnd tests building and submitting a register-to-vote
// transaction with voting right authentication from beginning to end.
func TestAuthenticateEndToEnd(t *testing.T) {
	ctx := pgtest.NewContext(t)
	store := txdb.NewStore(pg.FromContext(ctx).(*sql.DB)) // TODO(kr): use memstore
	fc, err := assettest.InitializeSigningGenerator(ctx, store)
	if err != nil {
		t.Fatal(err)
	}
	Connect(fc)

	var (
		accountID   = assettest.CreateAccountFixture(ctx, t, "", "", nil)
		holderAddr  = assettest.CreateAddressFixture(ctx, t, accountID)
		right       = createVotingRightFixture(ctx, t, holderAddr.PKScript)
		token       = createVotingTokenFixture(ctx, t, right.AssetID, holderAddr.PKScript, 100)
		rightAmount = bc.AssetAmount{AssetID: right.AssetID, Amount: 1}
		tokenAmount = bc.AssetAmount{AssetID: token.AssetID, Amount: 100}
	)

	// Build an authentication transaction.
	rightReserver, rightReceiver, err := RightAuthentication(ctx, right)
	if err != nil {
		t.Fatal(err)
	}
	tokenReserver, tokenReceiver, err := TokenRegistration(ctx, token, rightReceiver.PKScript())
	if err != nil {
		t.Fatal(err)
	}

	sources := []*txbuilder.Source{
		{Reserver: rightReserver, AssetAmount: rightAmount},
		{Reserver: tokenReserver, AssetAmount: tokenAmount},
	}
	destinations := []*txbuilder.Destination{
		{Receiver: rightReceiver, AssetAmount: rightAmount},
		{Receiver: tokenReceiver, AssetAmount: tokenAmount},
	}
	tmpl, err := txbuilder.Build(ctx, nil, sources, destinations, nil, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	assettest.SignTxTemplate(t, tmpl, testutil.TestXPrv)

	authTx, err := asset.FinalizeTx(ctx, tmpl)
	if err != nil {
		t.Fatal(err)
	}

	// Make a block to ensure that the resulting tx gets indexed.
	_, err = generator.MakeBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	gotRight, err := FindRightForOutpoint(ctx, bc.Outpoint{Hash: authTx.Hash, Index: 0})
	if err != nil {
		t.Fatal(err)
	}
	if gotRight.AccountID == nil || *gotRight.AccountID != accountID {
		t.Errorf("account ID, got=%#v want=%s", gotRight.AccountID, accountID)
	}
	if !reflect.DeepEqual(gotRight.rightScriptData, right.rightScriptData) {
		t.Errorf("script data, got=%#v want=%#v", gotRight.rightScriptData, right.rightScriptData)
	}

	gotToken, err := FindTokenForAsset(ctx, token.AssetID, token.Right)
	if err != nil {
		t.Fatal(err)
	}
	wantToken := token.tokenScriptData
	wantToken.State = stateRegistered
	if !reflect.DeepEqual(gotToken.tokenScriptData, wantToken) {
		t.Errorf("token data, got=%#v want=%#v", gotToken.tokenScriptData, wantToken)
	}
}

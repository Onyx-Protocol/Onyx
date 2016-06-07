package voting

import (
	"reflect"
	"testing"
	"time"

	"golang.org/x/net/context"

	"chain/core/asset"
	"chain/core/asset/assettest"
	"chain/core/generator"
	"chain/core/txbuilder"
	"chain/cos/bc"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/testutil"
)

// TestAuthenticateEndToEnd tests building and submitting a register-to-vote
// transaction with voting right authentication from beginning to end.
func TestAuthenticateEndToEnd(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	fc, err := assettest.InitializeSigningGenerator(ctx, nil, nil)
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
	tokenReserver, tokenDestinations, err := TokenRegistration(ctx, token, rightReceiver.PKScript(), []Registration{
		{ID: []byte{0x4e, 0x61, 0x51, 0x34, 0x3d}, Amount: 100},
	})
	if err != nil {
		t.Fatal(err)
	}

	sources := []*txbuilder.Source{
		{Reserver: rightReserver, AssetAmount: rightAmount},
		{Reserver: tokenReserver, AssetAmount: tokenAmount},
	}
	destinations := append([]*txbuilder.Destination{
		{Receiver: rightReceiver, AssetAmount: rightAmount},
	}, tokenDestinations...)
	tmpl, err := txbuilder.Build(ctx, nil, sources, destinations, nil, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	assettest.SignTxTemplate(t, tmpl, testutil.TestXPrv)

	tx, err := asset.FinalizeTx(ctx, tmpl)
	if err != nil {
		t.Fatal(err)
	}

	// Make a block to ensure that the resulting tx gets indexed.
	_, err = generator.MakeBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	gotRight, err := GetCurrentHolder(ctx, right.AssetID)
	if err != nil {
		t.Fatal(err)
	}
	if gotRight.AccountID == nil || *gotRight.AccountID != accountID {
		t.Errorf("account ID, got=%#v want=%s", gotRight.AccountID, accountID)
	}
	if !reflect.DeepEqual(gotRight.rightScriptData, right.rightScriptData) {
		t.Errorf("script data, got=%#v want=%#v", gotRight.rightScriptData, right.rightScriptData)
	}

	gotToken, err := FindTokenForOutpoint(ctx, bc.Outpoint{Hash: tx.Hash, Index: 1})
	if err != nil {
		t.Fatal(err)
	}
	wantToken := token.tokenScriptData
	wantToken.RegistrationID = []byte{0x4e, 0x61, 0x51, 0x34, 0x3d}
	wantToken.State = stateRegistered
	if !reflect.DeepEqual(gotToken.tokenScriptData, wantToken) {
		t.Errorf("token data, got=%#v want=%#v", gotToken.tokenScriptData, wantToken)
	}
}

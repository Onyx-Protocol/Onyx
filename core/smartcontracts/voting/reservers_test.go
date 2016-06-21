package voting

import (
	"encoding/json"
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/core/appdb"
	"chain/core/asset/assettest"
	"chain/core/txbuilder"
	"chain/cos/bc"
	"chain/cos/hdkey"
	"chain/cos/txscript"
	"chain/database/pg"
	"chain/database/pg/pgtest"
)

func TestRightsReserver(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	var (
		accountID = assettest.CreateAccountFixture(ctx, t, "", "", nil)
		assetID   = assettest.CreateAssetFixture(ctx, t, "", "", "")
		address   = assettest.CreateAddressFixture(ctx, t, accountID)
	)

	prev := &Right{
		AssetID:   assetID,
		Ordinal:   0,
		Outpoint:  bc.Outpoint{Hash: exampleHash, Index: 1},
		AccountID: &accountID,
		rightScriptData: rightScriptData{
			AdminScript:    []byte{0x01, 0x01},
			HolderScript:   address.PKScript,
			OwnershipChain: exampleHash2,
			Delegatable:    true,
		},
	}
	reserver, _, err := RightAuthentication(ctx, prev)
	if err != nil {
		t.Fatal(err)
	}

	got, err := reserver.Reserve(ctx, &bc.AssetAmount{AssetID: assetID, Amount: 1}, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := &txbuilder.ReserveResult{
		Items: []*txbuilder.ReserveResultItem{
			{
				TxInput: &bc.TxInput{
					Previous:    bc.Outpoint{Hash: exampleHash, Index: 1},
					AssetAmount: bc.AssetAmount{AssetID: assetID, Amount: 1},
					PrevScript:  prev.PKScript(),
				},
				TemplateInput: &txbuilder.Input{
					AssetAmount: bc.AssetAmount{AssetID: assetID, Amount: 1},
					SigComponents: []*txbuilder.SigScriptComponent{
						{
							Type: "signature",
							Signatures: txbuilder.InputSigs(
								hdkey.Derive(address.Keys, appdb.ReceiverPath(address, address.Index)),
							),
						},
						{
							Type:   "script",
							Script: txscript.AddDataToScript(nil, address.RedeemScript),
						},
						{
							Type:   "script",
							Script: txscript.AddDataToScript([]byte{txscript.OP_1}, rightsHoldingContract),
						},
					},
				},
			},
		},
	}
	if !reflect.DeepEqual(got, want) {
		gotJSON, err := json.MarshalIndent(got, "", "  ")
		if err != nil {
			t.Fatal(err)
		}

		wantJSON, err := json.MarshalIndent(want, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		t.Errorf("got=%s\n\nwant=%s\n\n", gotJSON, wantJSON)
	}
}

func TestTokenReserver(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	var (
		assetID      = assettest.CreateAssetFixture(ctx, t, "", "", "")
		rightAssetID = assettest.CreateAssetFixture(ctx, t, "", "", "")
	)

	prev := &Token{
		AssetID:  assetID,
		Outpoint: bc.Outpoint{Hash: exampleHash, Index: 1},
		Amount:   300,
		tokenScriptData: tokenScriptData{
			Right:       rightAssetID,
			AdminScript: []byte{txscript.OP_1},
			State:       stateDistributed,
			Vote:        2,
		},
	}
	rightData := rightScriptData{}
	reserver, _, err := TokenRegistration(ctx, prev, rightData.PKScript(), []Registration{
		{ID: []byte{0xc0, 0x01}, Amount: 300},
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := reserver.Reserve(ctx, &bc.AssetAmount{AssetID: assetID, Amount: 300}, 0)
	if err != nil {
		t.Fatal(err)
	}

	var sigscript []byte
	sigscript = txscript.AddInt64ToScript(sigscript, 300)
	sigscript = txscript.AddDataToScript(sigscript, []byte{0xc0, 0x01})
	sigscript = append(sigscript, txscript.OP_1)
	sigscript = txscript.AddDataToScript(sigscript, rightData.PKScript())
	sigscript = append(sigscript, txscript.OP_2)
	sigscript = txscript.AddDataToScript(sigscript, tokenHoldingContract)

	want := &txbuilder.ReserveResult{
		Items: []*txbuilder.ReserveResultItem{
			{
				TxInput: &bc.TxInput{
					Previous:    bc.Outpoint{Hash: exampleHash, Index: 1},
					AssetAmount: bc.AssetAmount{AssetID: assetID, Amount: 300},
					PrevScript:  prev.PKScript(),
				},
				TemplateInput: &txbuilder.Input{
					AssetAmount: bc.AssetAmount{AssetID: assetID, Amount: 300},
					SigComponents: []*txbuilder.SigScriptComponent{
						{
							Type:   "script",
							Script: sigscript,
						},
					},
				},
			},
		},
	}
	if !reflect.DeepEqual(got, want) {
		gotJSON, err := json.MarshalIndent(got, "", "  ")
		if err != nil {
			t.Fatal(err)
		}

		wantJSON, err := json.MarshalIndent(want, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		t.Errorf("got=%s\n\nwant=%s\n\n", gotJSON, wantJSON)
	}
}

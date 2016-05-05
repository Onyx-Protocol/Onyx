package voting

import (
	"encoding/json"
	"reflect"
	"testing"

	"chain/api/appdb"
	"chain/api/asset/assettest"
	"chain/api/txbuilder"
	"chain/cos/bc"
	"chain/cos/hdkey"
	"chain/cos/txscript"
	"chain/database/pg/pgtest"
)

func TestRightsReserver(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	var (
		accountID = assettest.CreateAccountFixture(ctx, t, "", "", nil)
		assetID   = assettest.CreateAssetFixture(ctx, t, "", "", "")
		address   = assettest.CreateAddressFixture(ctx, t, accountID)
	)

	prev := &RightWithUTXO{
		UTXO:         bc.Outpoint{Hash: exampleHash, Index: 1},
		Outpoint:     bc.Outpoint{Hash: exampleHash, Index: 1},
		BlockHeight:  2,
		BlockTxIndex: 43,
		AssetID:      assetID,
		AccountID:    &accountID,
		rightScriptData: rightScriptData{
			AdminScript:    []byte{0x01, 0x01},
			HolderScript:   address.PKScript,
			OwnershipChain: exampleHash2,
			Deadline:       10000,
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
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

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
			OptionCount: 3,
			State:       stateDistributed,
			SecretHash:  exampleHash2,
			Vote:        2,
		},
	}
	rightData := rightScriptData{}
	reserver, _, err := TokenRegistration(ctx, prev, rightData.PKScript())
	if err != nil {
		t.Fatal(err)
	}

	got, err := reserver.Reserve(ctx, &bc.AssetAmount{AssetID: assetID, Amount: 300}, 0)
	if err != nil {
		t.Fatal(err)
	}

	var sigscript []byte
	sigscript = txscript.AddDataToScript(sigscript, rightData.PKScript())
	sigscript = append(sigscript, txscript.OP_1)
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

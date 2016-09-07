package core

import (
	"context"
	"testing"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/asset/assettest"
	"chain/core/mockhsm"
	"chain/core/txbuilder"
	"chain/crypto/ed25519/hd25519"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/protocol/bc"
	"chain/protocol/prottest"
	"chain/testutil"
)

func TestMockHSM(t *testing.T) {
	dbtx := pgtest.NewTx(t)
	ctx := pg.NewContext(context.Background(), dbtx)
	c := prottest.NewChain(t)
	asset.Init(c, nil)
	account.Init(c, nil)
	mockhsm := mockhsm.New(dbtx)
	xpub1, err := mockhsm.CreateKey(ctx, "")
	if err != nil {
		t.Fatal(err)
	}

	acct1, err := account.Create(ctx, []string{xpub1.String()}, 1, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, xpub2, err := hd25519.NewXKeys(nil)
	if err != nil {
		t.Fatal(err)
	}

	acct2, err := account.Create(ctx, []string{xpub2.String()}, 1, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	assetDef1 := map[string]interface{}{"foo": 1}
	assetDef2 := map[string]interface{}{"foo": 2}

	asset1ID := assettest.CreateAssetFixture(ctx, t, []string{testutil.TestXPub.String()}, 1, assetDef1, "", nil)
	asset2ID := assettest.CreateAssetFixture(ctx, t, []string{testutil.TestXPub.String()}, 1, assetDef2, "", nil)

	issueSrc1 := txbuilder.Action(assettest.NewIssueAction(bc.AssetAmount{AssetID: asset1ID, Amount: 100}, nil))
	issueSrc2 := txbuilder.Action(assettest.NewIssueAction(bc.AssetAmount{AssetID: asset2ID, Amount: 200}, nil))
	issueDest1 := assettest.NewAccountControlAction(bc.AssetAmount{AssetID: asset1ID, Amount: 100}, acct1.ID, nil)
	issueDest2 := assettest.NewAccountControlAction(bc.AssetAmount{AssetID: asset2ID, Amount: 200}, acct2.ID, nil)
	tmpl, err := txbuilder.Build(ctx, nil, []txbuilder.Action{issueSrc1, issueSrc2, issueDest1, issueDest2}, nil)
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

	xferSrc1 := assettest.NewAccountSpendAction(bc.AssetAmount{AssetID: asset1ID, Amount: 10}, acct1.ID, nil, nil, nil, nil)
	xferSrc2 := assettest.NewAccountSpendAction(bc.AssetAmount{AssetID: asset2ID, Amount: 20}, acct2.ID, nil, nil, nil, nil)
	xferDest1 := assettest.NewAccountControlAction(bc.AssetAmount{AssetID: asset2ID, Amount: 20}, acct1.ID, nil)
	xferDest2 := assettest.NewAccountControlAction(bc.AssetAmount{AssetID: asset1ID, Amount: 10}, acct2.ID, nil)
	tmpl, err = txbuilder.Build(ctx, nil, []txbuilder.Action{xferSrc1, xferSrc2, xferDest1, xferDest2}, nil)
	if err != nil {
		t.Fatal(err)
	}

	a := &api{hsm: mockhsm}
	outTmpls := a.mockhsmSignTemplates(ctx, []*txbuilder.Template{tmpl})
	if len(outTmpls) != 1 {
		t.Fatalf("expected 1 output template, got %d", len(outTmpls))
	}
	outTmpl, ok := outTmpls[0].(*txbuilder.Template)
	if !ok {
		t.Fatalf("expected a *txbuilder.Template, got %T (%v)", outTmpls[0], outTmpls[0])
	}
	if len(outTmpl.Inputs) != 2 {
		t.Fatalf("expected 2 inputs, got %d", len(outTmpl.Inputs))
	}

	inspectInput(t, outTmpl.Inputs[0], true)
	inspectInput(t, outTmpl.Inputs[1], false)
}

func inspectInput(t *testing.T, inp *txbuilder.Input, expectSig bool) {
	if len(inp.WitnessComponents) != 1 {
		t.Fatalf("len(inp.WitnessComponents) is %d, want 1", len(inp.WitnessComponents))
	}
	s, ok := inp.WitnessComponents[0].(*txbuilder.SignatureWitness)
	if !ok {
		t.Fatalf("inp.WitnessComponents[0] has type %T, want *txbuilder.SignatureWitness", inp.WitnessComponents[0])
	}
	if len(s.Sigs) != 1 {
		t.Fatalf("len(s.Sigs) is %d, want 1", len(s.Sigs))
	}
	if expectSig {
		if len(s.Sigs[0]) == 0 {
			t.Errorf("expected a signature in s.Sigs[0]")
		}
	} else {
		if len(s.Sigs[0]) != 0 {
			t.Errorf("expected no signature in s.Sigs[0], got %x", s.Sigs[0])
		}
	}
}

package core

import (
	"context"
	"testing"
	"time"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/coretest"
	"chain/core/mockhsm"
	"chain/core/pin"
	"chain/core/query"
	"chain/core/txbuilder"
	"chain/crypto/ed25519/chainkd"
	"chain/database/pg/pgtest"
	"chain/protocol/bc"
	"chain/protocol/prottest"
	"chain/testutil"
)

func TestMockHSM(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := context.Background()
	c := prottest.NewChain(t)
	pinStore := pin.NewStore(db)
	assets := asset.NewRegistry(db, c, pinStore)
	accounts := account.NewManager(db, c, pinStore)
	coretest.CreatePins(ctx, t, pinStore)
	accounts.IndexAccounts(query.NewIndexer(db, c, pinStore))
	go accounts.ProcessBlocks(ctx)
	mockhsm := mockhsm.New(db)

	xpub1, err := mockhsm.XCreate(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	acct1, err := accounts.Create(ctx, []string{xpub1.XPub.String()}, 1, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, xpub2, err := chainkd.NewXKeys(nil)
	if err != nil {
		t.Fatal(err)
	}
	acct2, err := accounts.Create(ctx, []string{xpub2.String()}, 1, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	assetDef1 := map[string]interface{}{"foo": 1}
	assetDef2 := map[string]interface{}{"foo": 2}

	asset1ID := coretest.CreateAsset(ctx, t, assets, assetDef1, "", nil)
	asset2ID := coretest.CreateAsset(ctx, t, assets, assetDef2, "", nil)

	issueSrc1 := txbuilder.Action(assets.NewIssueAction(bc.AssetAmount{AssetID: asset1ID, Amount: 100}, nil))
	issueSrc2 := txbuilder.Action(assets.NewIssueAction(bc.AssetAmount{AssetID: asset2ID, Amount: 200}, nil))
	issueDest1 := accounts.NewControlAction(bc.AssetAmount{AssetID: asset1ID, Amount: 100}, acct1.ID, nil)
	issueDest2 := accounts.NewControlAction(bc.AssetAmount{AssetID: asset2ID, Amount: 200}, acct2.ID, nil)
	tmpl, err := txbuilder.Build(ctx, nil, []txbuilder.Action{issueSrc1, issueSrc2, issueDest1, issueDest2}, time.Now().Add(time.Minute))
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

	xferSrc1 := accounts.NewSpendAction(bc.AssetAmount{AssetID: asset1ID, Amount: 10}, acct1.ID, nil, nil)
	xferSrc2 := accounts.NewSpendAction(bc.AssetAmount{AssetID: asset2ID, Amount: 20}, acct2.ID, nil, nil)
	xferDest1 := accounts.NewControlAction(bc.AssetAmount{AssetID: asset2ID, Amount: 20}, acct1.ID, nil)
	xferDest2 := accounts.NewControlAction(bc.AssetAmount{AssetID: asset1ID, Amount: 10}, acct2.ID, nil)
	tmpl, err = txbuilder.Build(ctx, nil, []txbuilder.Action{xferSrc1, xferSrc2, xferDest1, xferDest2}, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	h := &Handler{HSM: mockhsm}
	outTmpls := h.mockhsmSignTemplates(ctx, struct {
		Txs   []*txbuilder.Template `json:"transactions"`
		XPubs []string              `json:"xpubs"`
	}{[]*txbuilder.Template{tmpl}, []string{xpub1.XPub.String()}})
	if len(outTmpls) != 1 {
		t.Fatalf("expected 1 output template, got %d", len(outTmpls))
	}
	outTmpl, ok := outTmpls[0].(*txbuilder.Template)
	if !ok {
		t.Fatalf("expected a *txbuilder.Template, got %T (%v)", outTmpls[0], outTmpls[0])
	}
	if len(outTmpl.SigningInstructions) != 2 {
		t.Fatalf("expected 2 signing instructions, got %d", len(outTmpl.SigningInstructions))
	}

	inspectSigInst(t, outTmpl.SigningInstructions[0], true)
	inspectSigInst(t, outTmpl.SigningInstructions[1], false)
}

func inspectSigInst(t *testing.T, si *txbuilder.SigningInstruction, expectSig bool) {
	if len(si.WitnessComponents) != 1 {
		t.Fatalf("len(si.WitnessComponents) is %d, want 1", len(si.WitnessComponents))
	}
	s, ok := si.WitnessComponents[0].(*txbuilder.SignatureWitness)
	if !ok {
		t.Fatalf("si.WitnessComponents[0] has type %T, want *txbuilder.SignatureWitness", si.WitnessComponents[0])
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

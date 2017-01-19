package core

import (
	"context"
	"testing"
	"time"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/coretest"
	"chain/core/generator"
	"chain/core/mockhsm"
	"chain/core/pb"
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
	g := generator.New(c, nil, db)
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
	acct1, err := accounts.Create(ctx, []chainkd.XPub{xpub1.XPub}, 1, "", nil, "")
	if err != nil {
		t.Fatal(err)
	}

	_, xpub2, err := chainkd.NewXKeys(nil)
	if err != nil {
		t.Fatal(err)
	}
	acct2, err := accounts.Create(ctx, []chainkd.XPub{xpub2}, 1, "", nil, "")
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

	txdata, err := bc.NewTxDataFromBytes(tmpl.RawTransaction)
	if err != nil {
		t.Fatal(err)
	}

	err = txbuilder.FinalizeTx(ctx, c, g, bc.NewTx(*txdata))
	if err != nil {
		t.Fatal(err)
	}

	// Make a block so that UTXOs from the above tx are available to spend.
	prottest.MakeBlock(t, c, g.PendingTxs())
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
	outTmpls, _ := h.SignTxs(ctx, &pb.SignTxsRequest{
		Transactions: []*pb.TxTemplate{tmpl},
		Xpubs:        [][]byte{xpub1.XPub[:]},
	})
	if len(outTmpls.Responses) != 1 {
		t.Fatalf("expected 1 output template, got %d", len(outTmpls.Responses))
	}
	outTmpl := outTmpls.Responses[0].Template
	if len(outTmpl.SigningInstructions) != 2 {
		t.Fatalf("expected 2 signing instructions, got %d", len(outTmpl.SigningInstructions))
	}

	inspectSigInst(t, outTmpl.SigningInstructions[0], true)
	inspectSigInst(t, outTmpl.SigningInstructions[1], false)
}

func inspectSigInst(t *testing.T, si *pb.TxTemplate_SigningInstruction, expectSig bool) {
	if len(si.WitnessComponents) != 1 {
		t.Fatalf("len(si.WitnessComponents) is %d, want 1", len(si.WitnessComponents))
	}
	_, ok := si.WitnessComponents[0].Component.(*pb.TxTemplate_WitnessComponent_Signature)
	if !ok {
		t.Fatalf("si.WitnessComponents[0] has type %T, want *pb.TxTemplate_WitnessComponent_Signature", si.WitnessComponents[0].Component)
	}
	s := si.WitnessComponents[0].GetSignature()
	if len(s.Signatures) != 1 {
		t.Fatalf("len(s.Sigs) is %d, want 1", len(s.Signatures))
	}
	if expectSig {
		if len(s.Signatures[0]) == 0 {
			t.Errorf("expected a signature in s.Sigs[0]")
		}
	} else {
		if len(s.Signatures[0]) != 0 {
			t.Errorf("expected no signature in s.Sigs[0], got %x", s.Signatures[0])
		}
	}
}

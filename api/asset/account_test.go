package asset_test

import (
	"reflect"
	"testing"
	"time"

	"chain/api/appdb"
	. "chain/api/asset"
	"chain/api/asset/assettest"
	"chain/api/txbuilder"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
	"chain/fedchain/bc"
	"chain/testutil"
)

func TestAccountSourceReserve(t *testing.T) {
	ctx := assettest.NewContextWithGenesisBlock(t)
	defer pgtest.Finish(ctx)

	accID := assettest.CreateAccountFixture(ctx, t, "", "", nil)
	op := assettest.CreateAccountUTXOFixture(ctx, t, accID, [32]byte{255}, 2, false)

	assetAmount1 := &bc.AssetAmount{
		AssetID: [32]byte{255},
		Amount:  1,
	}
	source := NewAccountSource(ctx, assetAmount1, accID)

	got, err := source.Reserver.Reserve(ctx, assetAmount1, time.Minute)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	want := &txbuilder.ReserveResult{
		Items: []*txbuilder.ReserveResultItem{{
			TxInput: &bc.TxInput{
				Previous: op,
			},
			TemplateInput: nil,
		}},
		Change: &txbuilder.Destination{
			AssetAmount: bc.AssetAmount{AssetID: [32]byte{255}, Amount: 1},
		},
	}

	if len(got.Items) != 1 {
		t.Fatalf("expected 1 result utxo")
	}

	// generated address can change based on test ordering, so ignore in comparison
	got.Items[0].TemplateInput = nil

	ar, ok := got.Change.Receiver.(*AccountReceiver)
	if !ok {
		t.Fatalf("expected change destination to have AccountReceiver")
	}

	if ar.Addr().AccountID != accID {
		t.Errorf("got receiver addr account = %v want %v", ar.Addr().AccountID, accID)
	}

	// clear out to not compare generated addresses
	got.Change.Receiver = nil

	if !reflect.DeepEqual(got, want) {
		t.Errorf("reserve result\ngot:\n\t%+v\nwant:\n\t%+v", got, want)
		t.Errorf("reserve item\ngot:\n\t%+v\nwant:\n\t%+v", got.Items[0], want.Items[0])
		t.Errorf("reserve txin\ngot:\n\t%+v\nwant:\n\t%+v", got.Items[0].TxInput, want.Items[0].TxInput)
		t.Errorf("reserve change\ngot:\n\t%+v\nwant:\n\t%+v", got.Change, want.Change)
	}
}

func TestAccountDestinationPKScript(t *testing.T) {
	ctx := pgtest.NewContext(t, ``)
	defer pgtest.Finish(ctx)

	acc := assettest.CreateAccountFixture(ctx, t, "", "", nil)

	// Test account output pk script (address creation)
	dest, err := NewAccountDestination(ctx, &bc.AssetAmount{Amount: 1}, acc, nil)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
	got := dest.PKScript()

	receiver := dest.Receiver
	accountReceiver, ok := receiver.(*AccountReceiver)
	if !ok {
		t.Log(errors.Stack(err))
		t.Fatal("receiver is not an AccountReceiver")
	}
	addr := accountReceiver.Addr()
	want, _, err := hdkey.Scripts(addr.Keys, appdb.ReceiverPath(addr, addr.Index), addr.SigsRequired)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
	testutil.ExpectScriptEqual(t, got, want, "AccountDestination pk script")
}

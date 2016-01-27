package asset

import (
	"reflect"
	"testing"
	"time"

	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/fedchain/bc"
)

func TestBuildTrade(t *testing.T) {
	ctx := pgtest.NewContext(t, `
		INSERT INTO projects (id, name) VALUES ('proj-id-0', 'proj-0');
		INSERT INTO manager_nodes (id, project_id, label, current_rotation)
			VALUES('mn1', 'proj-id-0', 'mn1', 'rot1');
		INSERT INTO rotations (id, manager_node_id, keyset)
			VALUES('rot1', 'mn1', '{xpub661MyMwAqRbcGKBeRA9p52h7EueXnRWuPxLz4Zoo1ZCtX8CJR5hrnwvSkWCDf7A9tpEZCAcqex6KDuvzLxbxNZpWyH6hPgXPzji9myeqyHd}');
		INSERT INTO accounts (id, manager_node_id, key_index, next_address_index)
			VALUES('acc1', 'mn1', 0, 1);
		INSERT INTO accounts (id, manager_node_id, key_index, next_address_index)
			VALUES('acc2', 'mn1', 1, 1);
		INSERT INTO addresses (id, manager_node_id, account_id, keyset, key_index, redeem_script, pk_script)
			VALUES('a2', 'mn1', 'acc2', '{xpub661MyMwAqRbcGKBeRA9p52h7EueXnRWuPxLz4Zoo1ZCtX8CJR5hrnwvSkWCDf7A9tpEZCAcqex6KDuvzLxbxNZpWyH6hPgXPzji9myeqyHd}', 0, '', '\x01');
		INSERT INTO utxos
			(tx_hash, index, asset_id, amount, addr_index, account_id, manager_node_id, confirmed, block_hash, block_height)
		VALUES
			('1000000000000000000000000000000000000000000000000000000000000000', 0, 'fe00000000000000000000000000000000000000000000000000000000000000', 5, 1, 'acc1', 'mn1', TRUE, 'bh1', 1),
			('2000000000000000000000000000000000000000000000000000000000000000', 0, 'ff00000000000000000000000000000000000000000000000000000000000000', 2, 0, 'acc2', 'mn1', TRUE, 'bh1', 1);
	`)
	defer pgtest.Finish(ctx)

	unsignedTx := &bc.TxData{
		Version: 1,
		Inputs:  []*bc.TxInput{{Previous: bc.Outpoint{Hash: [32]byte{16}, Index: 0}}},
		Outputs: []*bc.TxOutput{{AssetAmount: bc.AssetAmount{AssetID: [32]byte{255}, Amount: 2}}},
	}

	tpl := &TxTemplate{
		Unsigned:   unsignedTx,
		Inputs:     []*Input{{}},
		BlockChain: "sandbox",
	}
	assetAmount1 := &bc.AssetAmount{
		AssetID: [32]byte{255},
		Amount:  2,
	}
	source := NewAccountSource(ctx, assetAmount1, "acc2")
	sources := []*Source{source}

	assetAmount2 := &bc.AssetAmount{
		AssetID: [32]byte{254},
		Amount:  5,
	}
	dest, err := NewScriptDestination(ctx, assetAmount2, []byte{}, false, nil)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
	dests := []*Destination{dest}

	got, err := Build(ctx, tpl, sources, dests, []byte{}, time.Hour*24)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	wantHash := "1000000000000000000000000000000000000000000000000000000000000000"
	if got.Unsigned.Inputs[0].Previous.Hash.String() != wantHash {
		t.Errorf("got txin[0].hash = %s want %s", got.Unsigned.Inputs[0].Previous.Hash.String(), wantHash)
	}

	wantHash = "2000000000000000000000000000000000000000000000000000000000000000"
	if got.Unsigned.Inputs[1].Previous.Hash.String() != wantHash {
		t.Errorf("got txin[1].hash = %s want %s", got.Unsigned.Inputs[1].Previous.Hash.String(), wantHash)
	}
}

func TestBuildTransfer(t *testing.T) {
	ctx := pgtest.NewContext(t, `
		INSERT INTO projects (id, name) VALUES ('proj-id-0', 'proj-0');
		INSERT INTO manager_nodes (id, project_id, label, current_rotation)
			VALUES('mn1', 'proj-id-0', 'mn1', 'rot1');
		INSERT INTO rotations (id, manager_node_id, keyset)
			VALUES('rot1', 'mn1', '{xpub661MyMwAqRbcGKBeRA9p52h7EueXnRWuPxLz4Zoo1ZCtX8CJR5hrnwvSkWCDf7A9tpEZCAcqex6KDuvzLxbxNZpWyH6hPgXPzji9myeqyHd}');
		INSERT INTO accounts (id, manager_node_id, key_index, next_address_index)
			VALUES('acc1', 'mn1', 0, 1);
		INSERT INTO addresses (id, manager_node_id, account_id, keyset, key_index, redeem_script, pk_script)
			VALUES('a1', 'mn1', 'acc1', '{xpub661MyMwAqRbcGKBeRA9p52h7EueXnRWuPxLz4Zoo1ZCtX8CJR5hrnwvSkWCDf7A9tpEZCAcqex6KDuvzLxbxNZpWyH6hPgXPzji9myeqyHd}', 0, '', '\x01');
		INSERT INTO utxos (tx_hash, index, asset_id, amount, addr_index, account_id, manager_node_id, confirmed, block_hash, block_height)
			VALUES ('246c6aa1e5cc2bd1132a37cbc267e2031558aee26a8956e21b749d72920331a7', 0, 'ff00000000000000000000000000000000000000000000000000000000000000', 6, 0, 'acc1', 'mn1', TRUE, 'bh1', 1);
	`)
	defer pgtest.Finish(ctx)

	assetAmount := &bc.AssetAmount{
		AssetID: [32]byte{255},
		Amount:  5,
	}
	source := NewAccountSource(ctx, assetAmount, "acc1")
	sources := []*Source{source}

	dest, err := NewScriptDestination(ctx, assetAmount, []byte{}, false, nil)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
	dests := []*Destination{dest}

	_, err = Build(ctx, nil, sources, dests, []byte{}, time.Minute)

	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
}

func TestCombine(t *testing.T) {
	unsigned1 := &bc.TxData{
		Version: 1,
		Inputs:  []*bc.TxInput{{Previous: bc.Outpoint{Hash: bc.Hash{}, Index: 0}}},
		Outputs: []*bc.TxOutput{{AssetAmount: bc.AssetAmount{AssetID: [32]byte{254}, Amount: 5}}},
	}

	unsigned2 := &bc.TxData{
		Version: 1,
		Inputs:  []*bc.TxInput{{Previous: bc.Outpoint{Hash: bc.Hash{}, Index: 0}}},
		Outputs: []*bc.TxOutput{{AssetAmount: bc.AssetAmount{AssetID: [32]byte{255}, Amount: 6}}},
	}

	combined := &bc.TxData{
		Version: 1,
		Inputs: []*bc.TxInput{
			{Previous: bc.Outpoint{Hash: bc.Hash{}, Index: 0}},
			{Previous: bc.Outpoint{Hash: bc.Hash{}, Index: 0}},
		},
		Outputs: []*bc.TxOutput{
			{AssetAmount: bc.AssetAmount{AssetID: [32]byte{254}, Amount: 5}},
			{AssetAmount: bc.AssetAmount{AssetID: [32]byte{255}, Amount: 6}},
		},
	}

	scriptDest1, err := NewScriptDestination(nil, &bc.AssetAmount{Amount: 1}, []byte{}, false, nil)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	scriptDest2, err := NewScriptDestination(nil, &bc.AssetAmount{Amount: 2}, []byte{}, false, nil)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	tpl1 := &TxTemplate{
		Unsigned:   unsigned1,
		Inputs:     []*Input{{}},
		OutRecvs:   []Receiver{scriptDest1.Receiver},
		BlockChain: "sandbox",
	}

	tpl2 := &TxTemplate{
		Unsigned:   unsigned2,
		Inputs:     []*Input{{}},
		OutRecvs:   []Receiver{scriptDest2.Receiver},
		BlockChain: "sandbox",
	}

	got, err := combine(tpl1, tpl2)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	want := &TxTemplate{
		Unsigned:   combined,
		Inputs:     []*Input{{}, {}},
		OutRecvs:   []Receiver{scriptDest1.Receiver, scriptDest2.Receiver},
		BlockChain: "sandbox",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("combine:\ngot: \t%+v\nwant:\t%+v", got, want)
	}
}

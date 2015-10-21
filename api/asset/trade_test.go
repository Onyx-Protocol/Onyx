package asset

import (
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/api/utxodb"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/fedchain/bc"
)

func TestTrade(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, `
		INSERT INTO projects (id, name) VALUES ('proj-id-0', 'proj-0');
		INSERT INTO manager_nodes (id, project_id, label, current_rotation)
			VALUES('mn1', 'proj-id-0', 'mn1', 'rot1');
		INSERT INTO rotations (id, manager_node_id, keyset)
			VALUES('rot1', 'mn1', '{xpub661MyMwAqRbcGKBeRA9p52h7EueXnRWuPxLz4Zoo1ZCtX8CJR5hrnwvSkWCDf7A9tpEZCAcqex6KDuvzLxbxNZpWyH6hPgXPzji9myeqyHd}');
		INSERT INTO accounts (id, manager_node_id, key_index, next_address_index)
			VALUES('acc1', 'mn1', 0, 1);
		INSERT INTO accounts (id, manager_node_id, key_index, next_address_index)
			VALUES('acc2', 'mn1', 1, 1);
		INSERT INTO addresses (id, manager_node_id, account_id, keyset, key_index, address, redeem_script, pk_script)
			VALUES('a2', 'mn1', 'acc2', '{xpub661MyMwAqRbcGKBeRA9p52h7EueXnRWuPxLz4Zoo1ZCtX8CJR5hrnwvSkWCDf7A9tpEZCAcqex6KDuvzLxbxNZpWyH6hPgXPzji9myeqyHd}', 0, 'a2', '', '');
		INSERT INTO utxos
			(txid, index, asset_id, amount, addr_index, account_id, manager_node_id)
		VALUES
			('1000000000000000000000000000000000000000000000000000000000000000', 0, 'fe00000000000000000000000000000000000000000000000000000000000000', 5, 1, 'acc1', 'mn1'),
			('2000000000000000000000000000000000000000000000000000000000000000', 0, 'ff00000000000000000000000000000000000000000000000000000000000000', 2, 0, 'acc2', 'mn1');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	unsignedTx := &bc.Tx{
		Version: 1,
		Inputs:  []*bc.TxInput{{Previous: bc.Outpoint{Hash: [32]byte{16}, Index: 0}}},
		Outputs: []*bc.TxOutput{{AssetID: [32]byte{255}, Value: 2}},
	}

	tpl := &Tx{
		Unsigned:   unsignedTx,
		Inputs:     []*Input{{ManagerNodeID: "mn1"}},
		BlockChain: "sandbox",
	}
	inputs := []utxodb.Input{{
		AccountID: "acc2",
		AssetID:   "ff00000000000000000000000000000000000000000000000000000000000000",
		Amount:    2,
	}}
	outputs := []*Output{{
		Address: "32g4QsxVQrhZeXyXTUnfSByNBAdTfVUdVK",
		AssetID: "fe00000000000000000000000000000000000000000000000000000000000000",
		Amount:  5,
	}}

	got, err := Trade(ctx, tpl, inputs, outputs)
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

func TestCombine(t *testing.T) {
	unsigned1 := &bc.Tx{
		Version: 1,
		Inputs:  []*bc.TxInput{{Previous: bc.Outpoint{Hash: bc.Hash{}, Index: 0}}},
		Outputs: []*bc.TxOutput{{AssetID: [32]byte{254}, Value: 5}},
	}

	unsigned2 := &bc.Tx{
		Version: 1,
		Inputs:  []*bc.TxInput{{Previous: bc.Outpoint{Hash: bc.Hash{}, Index: 0}}},
		Outputs: []*bc.TxOutput{{AssetID: [32]byte{255}, Value: 6}},
	}

	combined := &bc.Tx{
		Version: 1,
		Inputs: []*bc.TxInput{
			{Previous: bc.Outpoint{Hash: bc.Hash{}, Index: 0}},
			{Previous: bc.Outpoint{Hash: bc.Hash{}, Index: 0}},
		},
		Outputs: []*bc.TxOutput{
			{AssetID: [32]byte{254}, Value: 5},
			{AssetID: [32]byte{255}, Value: 6},
		},
	}

	tpl1 := &Tx{
		Unsigned:   unsigned1,
		Inputs:     []*Input{{ManagerNodeID: "mn1"}},
		OutRecvs:   []*utxodb.Receiver{{ManagerNodeID: "mn1"}},
		BlockChain: "sandbox",
	}

	tpl2 := &Tx{
		Unsigned:   unsigned2,
		Inputs:     []*Input{{ManagerNodeID: "mn2"}},
		OutRecvs:   []*utxodb.Receiver{{ManagerNodeID: "mn2"}},
		BlockChain: "sandbox",
	}

	got, err := combine(tpl1, tpl2)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	want := &Tx{
		Unsigned:   combined,
		Inputs:     []*Input{{ManagerNodeID: "mn1"}, {ManagerNodeID: "mn2"}},
		OutRecvs:   []*utxodb.Receiver{{ManagerNodeID: "mn1"}, {ManagerNodeID: "mn2"}},
		BlockChain: "sandbox",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("combine:\ngot: \t%+v\nwant:\t%+v", got, want)
	}
}

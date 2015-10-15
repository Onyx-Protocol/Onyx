package asset

import (
	"bytes"
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/api/utxodb"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/fedchain-sandbox/wire"
)

func TestTrade(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, `
		INSERT INTO projects (id, name) VALUES ('proj-id-0', 'proj-0');
		INSERT INTO manager_nodes (id, project_id, label, current_rotation)
			VALUES('w1', 'proj-id-0', 'w1', 'rot1');
		INSERT INTO rotations (id, manager_node_id, keyset)
			VALUES('rot1', 'w1', '{xpub661MyMwAqRbcGKBeRA9p52h7EueXnRWuPxLz4Zoo1ZCtX8CJR5hrnwvSkWCDf7A9tpEZCAcqex6KDuvzLxbxNZpWyH6hPgXPzji9myeqyHd}');
		INSERT INTO accounts (id, manager_node_id, key_index, next_address_index)
			VALUES('b1', 'w1', 0, 1);
		INSERT INTO accounts (id, manager_node_id, key_index, next_address_index)
			VALUES('b2', 'w1', 1, 1);
		INSERT INTO addresses (id, manager_node_id, account_id, keyset, key_index, address, redeem_script, pk_script)
			VALUES('a2', 'w1', 'b2', '{xpub661MyMwAqRbcGKBeRA9p52h7EueXnRWuPxLz4Zoo1ZCtX8CJR5hrnwvSkWCDf7A9tpEZCAcqex6KDuvzLxbxNZpWyH6hPgXPzji9myeqyHd}', 0, 'a2', '', '');
		INSERT INTO utxos
			(txid, index, asset_id, amount, addr_index, account_id, manager_node_id)
		VALUES
			('0000000000000000000000000000000000000000000000000000000000000001', 0, 'AZZR3GkaeC3kbTx37ip8sDPb3AYtdQYrEx', 5, 1, 'b1', 'w1'),
			('0000000000000000000000000000000000000000000000000000000000000002', 0, 'AdihbprwmmjfCqJbM4PUrncQHuM4kAvGbo', 2, 0, 'b2', 'w1');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)
	utxoDB = utxodb.New(sqlUTXODB{})

	outAsset, _ := wire.NewHash20FromStr("AdihbprwmmjfCqJbM4PUrncQHuM4kAvGbo")
	unsignedTx := &wire.MsgTx{
		Version: 1,
		TxIn: []*wire.TxIn{
			wire.NewTxIn(wire.NewOutPoint((*wire.Hash32)(&[32]byte{1}), 0), []byte{}),
		},
		TxOut: []*wire.TxOut{wire.NewTxOut(outAsset, 2, []byte{})},
	}
	var buf bytes.Buffer
	unsignedTx.Serialize(&buf)

	tpl := &Tx{
		Unsigned:   buf.Bytes(),
		Inputs:     []*Input{{WalletID: "w1"}},
		BlockChain: "sandbox",
	}
	inputs := []utxodb.Input{{
		BucketID: "b2",
		AssetID:  "AdihbprwmmjfCqJbM4PUrncQHuM4kAvGbo",
		Amount:   2,
	}}
	outputs := []*Output{{
		Address: "32g4QsxVQrhZeXyXTUnfSByNBAdTfVUdVK",
		AssetID: "AZZR3GkaeC3kbTx37ip8sDPb3AYtdQYrEx",
		Amount:  5,
	}}

	got, err := Trade(ctx, tpl, inputs, outputs)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	gotWire := wire.NewMsgTx()
	gotWire.Deserialize(bytes.NewReader(got.Unsigned))

	wantHash := "0000000000000000000000000000000000000000000000000000000000000001"
	if gotWire.TxIn[0].PreviousOutPoint.Hash.String() != wantHash {
		t.Errorf("got txin[0].hash = %s want %s", gotWire.TxIn[0].PreviousOutPoint.Hash.String(), wantHash)
	}

	wantHash = "0000000000000000000000000000000000000000000000000000000000000002"
	if gotWire.TxIn[1].PreviousOutPoint.Hash.String() != wantHash {
		t.Errorf("got txin[1].hash = %s want %s", gotWire.TxIn[1].PreviousOutPoint.Hash.String(), wantHash)
	}
}

func TestCombine(t *testing.T) {
	asset1, _ := wire.NewHash20FromStr("AZZR3GkaeC3kbTx37ip8sDPb3AYtdQYrEx")
	asset2, _ := wire.NewHash20FromStr("AdihbprwmmjfCqJbM4PUrncQHuM4kAvGbo")
	unsigned1 := &wire.MsgTx{
		Version: 1,
		TxIn: []*wire.TxIn{
			wire.NewTxIn(wire.NewOutPoint(new(wire.Hash32), 0), []byte{}),
		},
		TxOut: []*wire.TxOut{wire.NewTxOut(asset1, 5, []byte{})},
	}

	unsigned2 := &wire.MsgTx{
		Version: 1,
		TxIn: []*wire.TxIn{
			wire.NewTxIn(wire.NewOutPoint(new(wire.Hash32), 0), []byte{}),
		},
		TxOut: []*wire.TxOut{wire.NewTxOut(asset2, 6, []byte{})},
	}

	combined := &wire.MsgTx{
		Version: 1,
		TxIn: []*wire.TxIn{
			wire.NewTxIn(wire.NewOutPoint(new(wire.Hash32), 0), []byte{}),
			wire.NewTxIn(wire.NewOutPoint(new(wire.Hash32), 0), []byte{}),
		},
		TxOut: []*wire.TxOut{
			wire.NewTxOut(asset1, 5, []byte{}),
			wire.NewTxOut(asset2, 6, []byte{}),
		},
	}

	var buf bytes.Buffer
	unsigned1.Serialize(&buf)

	tpl1 := &Tx{
		Unsigned:   buf.Bytes(),
		Inputs:     []*Input{{WalletID: "w1"}},
		BlockChain: "sandbox",
	}

	buf = bytes.Buffer{}
	unsigned2.Serialize(&buf)
	tpl2 := &Tx{
		Unsigned:   buf.Bytes(),
		Inputs:     []*Input{{WalletID: "w2"}},
		BlockChain: "sandbox",
	}

	got, err := combine(tpl1, tpl2)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	buf = bytes.Buffer{}
	combined.Serialize(&buf)
	want := &Tx{
		Unsigned:   buf.Bytes(),
		Inputs:     []*Input{{WalletID: "w1"}, {WalletID: "w2"}},
		BlockChain: "sandbox",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("combine:\ngot: \t%+v\nwant:\t%+v", got, want)
	}
}

package nodetxlog

import (
	"chain/api/appdb"
	"chain/errors"
	"chain/fedchain/bc"
	"reflect"
	"testing"
	"time"
)

func TestGenerateNodeTxTransfer(t *testing.T) {
	asset0 := bc.AssetID{}
	asset1 := bc.AssetID([32]byte{1})
	txTime := time.Now()
	tx := &bc.Tx{
		Hash: bc.Hash{},
		TxData: bc.TxData{
			Inputs: []*bc.TxInput{{
				Previous: bc.Outpoint{Hash: bc.Hash([32]byte{255}), Index: 1},
				Metadata: []byte("input"),
			}},
			Outputs: []*bc.TxOutput{{
				AssetAmount: bc.AssetAmount{AssetID: bc.AssetID([32]byte{1}), Amount: 987},
				Script:      []byte{1, 1},
				Metadata:    []byte("output"),
			}},
			Metadata: []byte("tx"),
		},
	}
	ins := []*appdb.ActUTXO{{
		AccountID: "acc-0",
		AssetID:   asset0.String(),
		Amount:    987,
		Script:    []byte{0, 1},
	}}
	outs := []*appdb.ActUTXO{{
		AccountID: "acc-1",
	}}
	assetMap := map[string]*appdb.ActAsset{
		asset0.String(): &appdb.ActAsset{Label: "asset0"},
		asset1.String(): &appdb.ActAsset{Label: "asset1"},
	}
	accountMap := map[string]*appdb.ActAccount{
		"acc-0": &appdb.ActAccount{
			ManagerNodeID: "mnode-0",
			Label:         "foo",
		},
		"acc-1": &appdb.ActAccount{
			ManagerNodeID: "mnode-1",
			Label:         "bar",
		},
	}
	got, err := generateNodeTx(tx, ins, outs, assetMap, accountMap, txTime)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	one := uint32(1)

	want := &nodeTx{
		ID:   bc.Hash{},
		Time: txTime,
		Inputs: []nodeTxInput{{
			Type:         "transfer",
			TxID:         (*bc.Hash)(&[32]byte{255}),
			TxOut:        &one,
			AssetID:      asset0,
			AssetLabel:   "asset0",
			Amount:       987,
			AccountID:    "acc-0",
			AccountLabel: "foo",
			Address:      []byte{0, 1},
			Script:       []byte{0, 1},
			Metadata:     []byte("input"),
			mNodeID:      "mnode-0",
		}},
		Outputs: []nodeTxOutput{{
			AssetID:      asset1,
			AssetLabel:   "asset1",
			Amount:       987,
			Address:      []byte{1, 1},
			Script:       []byte{1, 1},
			AccountID:    "acc-1",
			AccountLabel: "bar",
			Metadata:     []byte("output"),
			mNodeID:      "mnode-1",
		}},
		Metadata: []byte("tx"),
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got:\n\t%+v\nwant:\n\t%+v", got, want)
	}
}

func TestGenerateNodeTxIssuance(t *testing.T) {
	asset1 := bc.AssetID([32]byte{1})
	txTime := time.Now()
	tx := &bc.Tx{
		Hash: bc.Hash{},
		TxData: bc.TxData{
			Inputs: []*bc.TxInput{{
				Previous:        bc.Outpoint{Index: bc.InvalidOutputIndex},
				AssetDefinition: []byte(`{"name": "asset 1"}`),
			}},
			Outputs: []*bc.TxOutput{{
				AssetAmount: bc.AssetAmount{AssetID: bc.AssetID([32]byte{1}), Amount: 543},
			}},
		},
	}
	ins := []*appdb.ActUTXO{nil}
	outs := []*appdb.ActUTXO{{
		AccountID: "acc-1",
	}}
	assetMap := map[string]*appdb.ActAsset{
		asset1.String(): &appdb.ActAsset{Label: "asset1"},
	}
	accountMap := map[string]*appdb.ActAccount{
		"acc-1": &appdb.ActAccount{
			ManagerNodeID: "mnode-1",
			Label:         "bar",
		},
	}
	got, err := generateNodeTx(tx, ins, outs, assetMap, accountMap, txTime)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	want := &nodeTx{
		ID:   bc.Hash{},
		Time: txTime,
		Inputs: []nodeTxInput{{
			Type:            "issuance",
			AssetID:         asset1,
			AssetLabel:      "asset1",
			AssetDefinition: []byte(`{"name": "asset 1"}`),
			Amount:          543,
		}},
		Outputs: []nodeTxOutput{{
			AssetID:      asset1,
			AssetLabel:   "asset1",
			Amount:       543,
			AccountID:    "acc-1",
			AccountLabel: "bar",
			mNodeID:      "mnode-1",
		}},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got:\n\t%+v\nwant:\n\t%+v", got, want)
	}
}

func TestFilterAccounts(t *testing.T) {
	startTx := &nodeTx{
		Inputs: []nodeTxInput{
			{mNodeID: "mnode-1", AccountID: "acc-1", AccountLabel: "foo"},
			{mNodeID: "mnode-2", AccountID: "acc-2", AccountLabel: "bar"},
		},
		Outputs: []nodeTxOutput{
			{mNodeID: "mnode-1", AccountID: "acc-3", AccountLabel: "baz"},
		},
	}
	cases := []struct {
		mnode string
		want  *nodeTx
	}{
		{
			mnode: "mnode-1",
			want: &nodeTx{
				Inputs: []nodeTxInput{
					{mNodeID: "mnode-1", AccountID: "acc-1", AccountLabel: "foo"},
					{mNodeID: "mnode-2", AccountID: "", AccountLabel: ""},
				},
				Outputs: []nodeTxOutput{
					{mNodeID: "mnode-1", AccountID: "acc-3", AccountLabel: "baz"},
				},
			},
		},
		{
			mnode: "mnode-2",
			want: &nodeTx{
				Inputs: []nodeTxInput{
					{mNodeID: "mnode-1", AccountID: "", AccountLabel: ""},
					{mNodeID: "mnode-2", AccountID: "acc-2", AccountLabel: "bar"},
				},
				Outputs: []nodeTxOutput{
					{mNodeID: "mnode-1", AccountID: "", AccountLabel: ""},
				},
			},
		},
		{
			mnode: "",
			want: &nodeTx{
				Inputs: []nodeTxInput{
					{mNodeID: "mnode-1", AccountID: "", AccountLabel: ""},
					{mNodeID: "mnode-2", AccountID: "", AccountLabel: ""},
				},
				Outputs: []nodeTxOutput{
					{mNodeID: "mnode-1", AccountID: "", AccountLabel: ""},
				},
			},
		},
	}

	for _, c := range cases {
		got := filterAccounts(startTx, c.mnode)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("filterAccounts(%s)\ngot:\n\t%+v\nwant:\n\t%+v", c.mnode, got, c.want)
		}
	}
}

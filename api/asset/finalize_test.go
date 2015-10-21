package asset

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/btcsuite/btcd/btcec"
	"golang.org/x/net/context"

	"chain/api/utxodb"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/fedchain/bc"
)

func mustDecodeHex(data string) []byte {
	h, err := hex.DecodeString(data)
	if err != nil {
		panic(err)
	}
	return h
}

func TestFinalizeTx(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, `
		INSERT INTO projects (id, name) VALUES ('app-id-0', 'app-0');
		INSERT INTO issuer_nodes (id, project_id, label, keyset, key_index)
			VALUES ('in1', 'app-id-0', 'foo', '{xpub661MyMwAqRbcGKBeRA9p52h7EueXnRWuPxLz4Zoo1ZCtX8CJR5hrnwvSkWCDf7A9tpEZCAcqex6KDuvzLxbxNZpWyH6hPgXPzji9myeqyHd}', 0);
		INSERT INTO assets (id, issuer_node_id, key_index, keyset, redeem_script, label)
		VALUES('ff00000000000000000000000000000000000000000000000000000000000000', 'in1', 0, '{}', '', 'foo');
		INSERT INTO manager_nodes (id, project_id, label, key_index)
			VALUES ('mn1', 'app-id-0', 'mnode1', 0);
		INSERT INTO accounts (id, manager_node_id, key_index, label) VALUES('acc1', 'mn1', 0, 'x');
		INSERT INTO addresses
			(id, manager_node_id, account_id, keyset, key_index, address, redeem_script, pk_script)
			VALUES ('a1', 'mn1', 'acc1', '{}', 0, '32g4QsxVQrhZeXyXTUnfSByNBAdTfVUdVK', '', '');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	outscript := mustDecodeHex("a9140ac9c982fd389181752e5a414045dd424a10754b87")
	unsigned := &bc.Tx{
		Version: 1,
		Inputs:  []*bc.TxInput{{Previous: bc.IssuanceOutpoint}},
		Outputs: []*bc.TxOutput{{AssetID: [32]byte{255}, Value: 5, Script: outscript}},
	}
	sigHash, _ := bc.ParseHash("78e437f627019fc270bbe9ed309291d0a5f6bf98bfae0f750538ba56646f7327")

	tpl := &Tx{
		Unsigned: unsigned,
		Inputs: []*Input{{
			RedeemScript:  []byte{},
			SignatureData: sigHash,
			Sigs: []*Signature{{
				XPub:           "xpub661MyMwAqRbcGiDB8FQvHnDAZyaGUyzm3qN1Q3NDJz1PgAWCfyi9WRCS7Z9HyM5QNEh45fMyoaBMqjfoWPdnktcN8chJYB57D2Y7QtNmadr",
				DerivationPath: []uint32{0, 0, 0, 0},
				DER:            mustDecodeHex("3044022004da5732f6c988b9e2882f5ca4f569b9525d313940e0372d6a84fef73be78f8f02204656916481dc573d771ec42923a8f5af31ae634241a4cb30ea5b359363cf064d"),
			}},
		}},
		OutRecvs: []*utxodb.Receiver{nil}, // pays to external party
	}

	tx, err := FinalizeTx(ctx, tpl)
	if err != nil {
		t.Fatal(withStack(err))
	}

	want := "053159ce7fc94d40d867de1de5b5529948b8ee88d2a9e98faf9aa187f79e9c6b"
	if tx.Hash().String() != want {
		t.Errorf("got tx hash = %v want %v", tx.Hash().String(), want)
	}
}

func TestCheckSig(t *testing.T) {
	cases := []struct {
		pubkey []byte
		data   []byte
		sig    []byte
		valid  bool
	}{{
		pubkey: mustDecodeHex("03e64d680d79157cc7d13437c86abcd6bdcf347f919a3b53a8fe0e50272ac1d474"),
		data:   mustDecodeHex("78e437f627019fc270bbe9ed309291d0a5f6bf98bfae0f750538ba56646f7327"),
		sig:    mustDecodeHex("3044022004da5732f6c988b9e2882f5ca4f569b9525d313940e0372d6a84fef73be78f8f02204656916481dc573d771ec42923a8f5af31ae634241a4cb30ea5b359363cf064d"),
		valid:  true,
	}, {
		pubkey: mustDecodeHex("03e64d680d79157cc7d13437c86abcd6bdcf347f919a3b53a8fe0e50272ac1d474"),
		data:   mustDecodeHex("78e437f627019fc270bbe9ed309291d0a5f6bf98bfae0f750538ba56646f7327"),
		sig:    mustDecodeHex("3044"),
		valid:  false,
	}}

	for _, c := range cases {
		key, _ := btcec.ParsePubKey(c.pubkey, btcec.S256())
		var hash bc.Hash
		copy(hash[:], c.data)
		err := checkSig(key, hash[:], c.sig)
		if (err == nil) != c.valid {
			t.Error("invalid signature")
		}
	}
}

func TestIsIssuance(t *testing.T) {
	asset1 := bc.AssetID{0}
	asset2 := bc.AssetID{1}
	cases := []struct {
		tx   *bc.Tx
		want bool
	}{{ // issuance input
		tx: &bc.Tx{
			Inputs:  []*bc.TxInput{{Previous: bc.IssuanceOutpoint}},
			Outputs: []*bc.TxOutput{{AssetID: asset1, Value: 5}},
		},
		want: true,
	}, { // no outputs
		tx: &bc.Tx{
			Inputs:  []*bc.TxInput{{Previous: bc.IssuanceOutpoint}},
			Outputs: []*bc.TxOutput{},
		},
		want: false,
	}, { // different asset ids on outputs
		tx: &bc.Tx{
			Inputs: []*bc.TxInput{{Previous: bc.IssuanceOutpoint}},
			Outputs: []*bc.TxOutput{
				{AssetID: asset1, Value: 5},
				{AssetID: asset2, Value: 5},
			},
		},
		want: false,
	}, { // too many inputs
		tx: &bc.Tx{
			Inputs: []*bc.TxInput{
				{Previous: bc.IssuanceOutpoint},
				{Previous: bc.IssuanceOutpoint},
			},
			Outputs: []*bc.TxOutput{{AssetID: asset1, Value: 5}},
		},
		want: false,
	}, { // wrong previous outpoint index
		tx: &bc.Tx{
			Inputs: []*bc.TxInput{
				{Previous: bc.Outpoint{Hash: bc.Hash{}, Index: 1}},
			},
			Outputs: []*bc.TxOutput{{AssetID: asset1, Value: 5}},
		},
		want: false,
	}, { // empty txin
		tx:   &bc.Tx{},
		want: false,
	}}

	for _, c := range cases {
		got := isIssuance(c.tx)
		if got != c.want {
			t.Errorf("got isIssuance(%+v) = %v want %v", c.tx, got, c.want)
		}
	}
}

func TestIssued(t *testing.T) {
	asset := [32]byte{255}
	outs := []*bc.TxOutput{{AssetID: asset, Value: 2}, {AssetID: asset, Value: 3}}

	gotAsset, gotAmt := issued(outs)
	if !bytes.Equal(gotAsset[:], asset[:]) {
		t.Errorf("got asset = %q want %q", gotAsset.String(), bc.AssetID(asset).String())
	}

	if gotAmt != 5 {
		t.Errorf("got amt = %d want %d", gotAmt, 5)
	}
}

func withStack(err error) string {
	s := err.Error()
	for _, frame := range errors.Stack(err) {
		s += "\n" + frame.String()
	}
	return s
}

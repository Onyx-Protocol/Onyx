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
	"chain/fedchain-sandbox/wire"
)

func mustDecodeHex(data string) []byte {
	h, err := hex.DecodeString(data)
	if err != nil {
		panic(err)
	}
	return h
}

func TestFinalizeTx(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)
	utxoDB = utxodb.New(sqlUTXODB{})

	tpl := &Tx{
		Unsigned: mustDecodeHex("010000000101000000000000000000000000000000000000000000000000000000000000000000000000ffffffff0187849ccdeaa558af265aafdfb6aa17903b2fc6997b0000000000000017a9140ac9c982fd389181752e5a414045dd424a10754b8700000000"),
		Inputs: []*Input{{
			RedeemScript:  []byte{},
			SignatureData: mustDecodeHex("78e437f627019fc270bbe9ed309291d0a5f6bf98bfae0f750538ba56646f7327"),
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

	want := "2b7c01a96523a1368cc25d179a15b460cf1f959c09b41a69ad1562652bab97ee"
	if tx.TxSha().String() != want {
		t.Errorf("got tx hash = %v want %v", tx.TxSha().String(), want)
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
		err := checkSig(key, c.data, c.sig)
		if (err == nil) != c.valid {
			t.Error("invalid signature")
		}
	}
}

func TestIsIssuance(t *testing.T) {
	asset1, _ := wire.NewHash20FromStr("AU8RjUUysqep9wXcZKqtTty1BssV6TcX7p")
	asset2, _ := wire.NewHash20FromStr("AZZR3GkaeC3kbTx37ip8sDPb3AYtdQYrEx")
	cases := []struct {
		raw  []byte
		tx   *wire.MsgTx
		want bool
	}{{ // issuance input
		tx: &wire.MsgTx{
			TxIn:  []*wire.TxIn{wire.NewTxIn(wire.NewOutPoint(new(wire.Hash32), 0), []byte{})},
			TxOut: []*wire.TxOut{wire.NewTxOut(asset1, 5, []byte{})},
		},
		want: true,
	}, { // no outputs
		tx: &wire.MsgTx{
			TxIn:  []*wire.TxIn{wire.NewTxIn(wire.NewOutPoint(new(wire.Hash32), 0), []byte{})},
			TxOut: []*wire.TxOut{},
		},
		want: false,
	}, { // different asset ids on outputs
		tx: &wire.MsgTx{
			TxIn: []*wire.TxIn{wire.NewTxIn(wire.NewOutPoint(new(wire.Hash32), 0), []byte{})},
			TxOut: []*wire.TxOut{
				wire.NewTxOut(asset1, 5, []byte{}),
				wire.NewTxOut(asset2, 5, []byte{}),
			},
		},
		want: false,
	}, { // too many inputs
		tx: &wire.MsgTx{
			TxIn: []*wire.TxIn{
				wire.NewTxIn(wire.NewOutPoint(new(wire.Hash32), 0), []byte{}),
				wire.NewTxIn(wire.NewOutPoint(new(wire.Hash32), 0), []byte{}),
			},
			TxOut: []*wire.TxOut{wire.NewTxOut(asset1, 5, []byte{})},
		},
		want: false,
	}, { // wrong previous outpoint hash
		tx: &wire.MsgTx{
			TxIn: []*wire.TxIn{
				wire.NewTxIn(wire.NewOutPoint((*wire.Hash32)(&[32]byte{1}), 0), []byte{}),
			},
			TxOut: []*wire.TxOut{wire.NewTxOut(asset1, 5, []byte{})},
		},
		want: false,
	}, { // empty txin
		tx:   &wire.MsgTx{},
		want: false,
	}}

	for _, c := range cases {
		got := isIssuance(c.tx)
		if got != c.want {
			t.Errorf("got isIssuance(%x) = %v want %v", c.raw, got, c.want)
		}
	}
}

func TestIssued(t *testing.T) {
	hash, _ := wire.NewHash20FromStr("AU8RjUUysqep9wXcZKqtTty1BssV6TcX7p")
	outs := []*wire.TxOut{
		wire.NewTxOut(hash, 2, []byte{}),
		wire.NewTxOut(hash, 3, []byte{}),
	}

	gotAsset, gotAmt := issued(outs)
	if !bytes.Equal(gotAsset[:], hash[:]) {
		t.Errorf("got asset = %q want %q", gotAsset.String(), hash.String())
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

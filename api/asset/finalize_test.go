package asset

import (
	"encoding/hex"
	"testing"

	"github.com/btcsuite/btcd/btcec"
	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
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
		INSERT INTO keys (id, xpub)
		VALUES(
			'f990c45d75c2b80537d3363f3f91a4c756f51fdc',
			'xpub661MyMwAqRbcGiDB8FQvHnDAZyaGUyzm3qN1Q3NDJz1PgAWCfyi9WRCS7Z9HyM5QNEh45fMyoaBMqjfoWPdnktcN8chJYB57D2Y7QtNmadr'
		);
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	tpl := &Tx{
		Unsigned: mustDecodeHex("010000000100000000000000000000000000000000000000000000000000000000000000000000000000ffffffff0187849ccdeaa558af265aafdfb6aa17903b2fc6997b0000000000000017a9140ac9c982fd389181752e5a414045dd424a10754b8700000000"),
		Inputs: []*Input{{
			RedeemScript:  []byte{},
			SignatureData: mustDecodeHex("78e437f627019fc270bbe9ed309291d0a5f6bf98bfae0f750538ba56646f7327"),
			Sigs: []*Signature{{
				XPubHash:       "f990c45d75c2b80537d3363f3f91a4c756f51fdc",
				DerivationPath: []uint32{0, 0, 0, 0},
				DER:            mustDecodeHex("3044022004da5732f6c988b9e2882f5ca4f569b9525d313940e0372d6a84fef73be78f8f02204656916481dc573d771ec42923a8f5af31ae634241a4cb30ea5b359363cf064d"),
			}},
		}},
	}

	tx, err := FinalizeTx(ctx, tpl)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	want := "e3f197e2e73242092f449c06d3c4c8911e1c884d8308ef01dc90f6abfb5da64d"
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

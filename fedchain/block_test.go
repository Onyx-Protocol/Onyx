package fedchain_test

import (
	"encoding/hex"
	"os"
	"reflect"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/btcsuite/btcd/btcec"

	"chain/api/asset/assettest"
	"chain/api/generator"
	"chain/database/pg/pgtest"
	"chain/errors"
	. "chain/fedchain"
	"chain/fedchain/bc"
	"chain/testutil"
)

func init() {
	u := "postgres:///api-test?sslmode=disable"
	if s := os.Getenv("DB_URL_TEST"); s != "" {
		u = s
	}

	ctx := context.Background()
	pgtest.Open(ctx, u, "fedchaintest", "../api/appdb/schema.sql")
}

func TestIdempotentUpsert(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	fc, err := assettest.InitializeSigningGenerator(ctx)
	if err != nil {
		t.Fatal(err)
	}

	pubkey, err := testutil.TestXPub.ECPubKey()
	if err != nil {
		t.Fatal(err)
	}

	// InitializeSigningGenerator added a genesis block.  Calling
	// UpsertGenesisBlock again should be a no-op, not produce an error.
	_, err = fc.UpsertGenesisBlock(ctx, []*btcec.PublicKey{pubkey}, 1)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGenerateBlock(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	fc, err := assettest.InitializeSigningGenerator(ctx)
	if err != nil {
		t.Fatal(err)
	}

	latestBlock, err := fc.LatestBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	txs := []*bc.Tx{
		bc.NewTx(bc.TxData{
			Version: 1,
			Inputs: []*bc.TxInput{{
				Previous: bc.Outpoint{
					Hash:  mustParseHash("92b34025babea306bdf67cfe9a2576d8475ea9476caeb1fbdea43bf3d56d011a"),
					Index: bc.InvalidOutputIndex,
				},
				SignatureScript: mustDecodeHex("004830450221009037e1d39b7d59d24eba8012baddd5f4ab886a51b46f52b7c479ddfa55eeb5c5022076008409243475b25dfba6db85e15cf3d74561a147375941e4830baa69769b51012551210210b002870438af79b829bc22c4505e14779ef0080c411ad497d7a0846ee0af6f51ae"),
				AssetDefinition: []byte(`{
"key": "clam"
}`),
			}},
			Outputs: []*bc.TxOutput{{
				AssetAmount: bc.AssetAmount{
					AssetID: mustParseHash("25fbb43a93c290fde3997d92c416d3cc7ff40a13aa309d051406978635085c8d"),
					Amount:  50,
				},
				Script: mustDecodeHex("a9145881cd104f8d64635751ac0f3c0decf9150c110687"),
			}},
		}),
		bc.NewTx(bc.TxData{
			Version: 1,
			Inputs: []*bc.TxInput{{
				Previous: bc.Outpoint{
					Hash:  mustParseHash("92b34025babea306bdf67cfe9a2576d8475ea9476caeb1fbdea43bf3d56d011a"),
					Index: bc.InvalidOutputIndex,
				},
				SignatureScript: mustDecodeHex("00483045022100f3bcffcfd6a1ce9542b653500386cd0ee7b9c86c59390ca0fc0238c0ebe3f1d6022065ac468a51a016842660c3a616c99a9aa5109a3bad1877ba3e0f010f3972472e012551210210b002870438af79b829bc22c4505e14779ef0080c411ad497d7a0846ee0af6f51ae"),
				AssetDefinition: []byte(`{
"key": "clam"
}`),
			}},
			Outputs: []*bc.TxOutput{{
				AssetAmount: bc.AssetAmount{
					AssetID: mustParseHash("25fbb43a93c290fde3997d92c416d3cc7ff40a13aa309d051406978635085c8d"),
					Amount:  50,
				},
				Script: mustDecodeHex("a914c171e443e05b953baa7b7d834028ed91e47b4d0b87"),
			}},
		}),
	}
	for _, tx := range txs {
		err := fc.ApplyTx(ctx, tx)
		if err != nil {
			t.Log(errors.Stack(err))
			t.Fatal(err)
		}
	}

	now := time.Now()
	got, _, err := fc.GenerateBlock(ctx, now)
	if err != nil {
		t.Fatalf("err got = %v want nil", err)
	}

	want := &bc.Block{
		BlockHeader: bc.BlockHeader{
			Version:           bc.NewBlockVersion,
			Height:            1,
			PreviousBlockHash: latestBlock.Hash(),
			TxRoot:            mustParseHash("221e04fdea661d26dbaef32df7b40fd93d97e359dcb9113c0fab763291a97a75"),
			Timestamp:         uint64(now.Unix()),
			OutputScript:      latestBlock.OutputScript,
		},
		Transactions: txs,
	}
	for _, wanttx := range want.Transactions {
		wanttx.Stored = true
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("generated block:\ngot:  %+v\nwant: %+v", got, want)
	}
}

func TestIsSignedByTrustedHost(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	fc, err := assettest.InitializeSigningGenerator(ctx)
	if err != nil {
		t.Fatal(err)
	}

	latestBlock, err := fc.LatestBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	privKey, err := testutil.TestXPrv.ECPrivKey()
	if err != nil {
		t.Fatal(err)
	}

	keys := []*btcec.PrivateKey{privKey}

	block := &bc.Block{}
	err = generator.GetAndAddBlockSignatures(ctx, block, latestBlock)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
	sig := block.SignatureScript

	cases := []struct {
		desc        string
		sigScript   []byte
		trustedKeys []*btcec.PublicKey
		want        bool
	}{{
		desc:        "empty sig",
		sigScript:   nil,
		trustedKeys: privToPub(keys),
		want:        false,
	}, {
		desc:        "wrong trusted keys",
		sigScript:   sig,
		trustedKeys: privToPub([]*btcec.PrivateKey{newPrivKey(t)}),
		want:        false,
	}, {
		desc:        "one-of-one trusted keys",
		sigScript:   sig,
		trustedKeys: privToPub(keys),
		want:        true,
	}, {
		desc:        "one-of-two trusted keys",
		sigScript:   sig,
		trustedKeys: privToPub(append(keys, newPrivKey(t))),
		want:        true,
	}}

	for _, c := range cases {
		block.SignatureScript = c.sigScript
		got := IsSignedByTrustedHost(block, c.trustedKeys)

		if got != c.want {
			t.Errorf("%s: got %v want %v", c.desc, got, c.want)
		}
	}
}

func privToPub(privs []*btcec.PrivateKey) []*btcec.PublicKey {
	var public []*btcec.PublicKey
	for _, priv := range privs {
		public = append(public, priv.PubKey())
	}
	return public
}

func newPrivKey(t *testing.T) *btcec.PrivateKey {
	key, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func mustParseHash(s string) [32]byte {
	h, err := bc.ParseHash(s)
	if err != nil {
		panic(err)
	}
	return h
}

func mustDecodeHex(s string) []byte {
	data, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return data
}

package asset_test

import (
	"log"
	"reflect"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/btcsuite/btcd/btcec"

	"chain/api/appdb"
	. "chain/api/asset"
	"chain/api/asset/assettest"
	"chain/api/txbuilder"
	"chain/api/utxodb"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/fedchain-sandbox/hdkey"
	"chain/fedchain/bc"
	"chain/fedchain/txscript"
	"chain/testutil"
)

func TestTransferConfirmed(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)
	info, err := bootdb(ctx)
	if err != nil {
		t.Fatal(err)
	}

	_, err = issue(ctx, t, info, info.acctA.ID, 10)
	if err != nil {
		t.Fatal(err)
	}

	dumpState(ctx, t)
	MakeBlock(ctx, BlockKey)
	dumpState(ctx, t)

	_, err = transfer(ctx, t, info, info.acctA.ID, info.acctB.ID, 10)
	if err != nil {
		testutil.FatalErr(t, err)
	}
}

func TestGenSpendApply(t *testing.T) {
	// 1. Start with an output in pool.
	// 2. Generate a block.
	// 3. Spend the output.
	// 4. Apply the block.
	// Output should stay spent!

	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	info, err := bootdb(ctx)
	if err != nil {
		t.Fatal(err)
	}
	issueTx, err := issue(ctx, t, info, info.acctA.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("issued %v", issueTx.Hash)

	block, err := GenerateBlock(ctx, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	err = SignBlock(block, BlockKey)
	if err != nil {
		t.Fatal(err)
	}

	_, err = transfer(ctx, t, info, info.acctA.ID, info.acctB.ID, 10)
	if err != nil {
		t.Fatal(err)
	}

	err = ApplyBlock(ctx, block)
	if err != nil {
		t.Fatal(err)
	}

	inputs := []utxodb.Source{{
		AssetID:   info.asset.Hash,
		AccountID: info.acctA.ID,
		Amount:    10,
	}}
	reserved, _, err := UTXODB().Reserve(ctx, inputs, 2*time.Minute)
	if err != nil && errors.Root(err) != utxodb.ErrInsufficient {
		t.Fatal(err)
	}
	if len(reserved) > 0 {
		t.Fatalf("want %v to stay spent after landing block", issueTx.Hash)
	}
}

func BenchmarkTransferWithBlocks(b *testing.B) {
	withContext(b, "", func(ctx context.Context) {
		info, err := bootdb(ctx)
		if err != nil {
			b.Fatal(err)
		}

		for i := 0; i < b.N; i++ {
			tx, err := issue(ctx, b, info, info.acctA.ID, 10)
			if err != nil {
				b.Fatal(err)
			}
			b.Logf("finalized %v", tx.Hash)

			tx, err = transfer(ctx, b, info, info.acctA.ID, info.acctB.ID, 10)
			if err != nil {
				b.Fatal(err)
			}
			b.Logf("finalized %v", tx.Hash)

			if i%10 == 0 {
				MakeBlock(ctx, BlockKey)
			}
		}
	})
}

func dumpState(ctx context.Context, t *testing.T) {
	t.Log("pool")
	dumpTab(ctx, t, `
		SELECT tx_hash, index, script FROM utxos_status
		WHERE NOT confirmed
	`)
	t.Log("blockchain")
	dumpTab(ctx, t, `
		SELECT tx_hash, index, script FROM utxos_status
		WHERE confirmed
	`)
}

func dumpTab(ctx context.Context, t *testing.T, q string) {
	rows, err := pg.FromContext(ctx).Query(ctx, q)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var hash bc.Hash
		var index int32
		var script []byte
		err = rows.Scan(&hash, &index, &script)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("hash: %s index: %d pkscript: %x", hash, index, script)
	}
	if rows.Err() != nil {
		t.Fatal(rows.Err())
	}
}

func TestGenerateBlock(t *testing.T) {
	const fix = `
		INSERT INTO blocks (block_hash, height, data, header)
		VALUES(
			'c33576c0fa9ee9b3b71dcfb7872835400df327501da144fdf770ef751c08376d',
			11,
			decode('010000000b0000000000000095a00b5cd11f577a461e6bb884899ee0aa1662088097b644af7a50d76e1a243f0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000003895f5600000000000001010000000195a00b5cd11f577a461e6bb884899ee0aa1662088097b644af7a50d76e1a243fffffffff7000483045022100c80b4deb9aae29da4e8768a5fbe0ac6ccca1d020f4b924005cc066f09b18e14e02206acd491a84eda9c15bed01a7648b191974a2ef47f7cefda3bde06092cd144e68012551210210b002870438af79b829bc22c4505e14779ef0080c411ad497d7a0846ee0af6f51ae00000125fbb43a93c290fde3997d92c416d3cc7ff40a13aa309d051406978635085c8d320000000000000017a9144bbae13b661c0cd9cf89271fd96dadb65e7a80378700000000000000000000', 'hex'),
			''
		);

		INSERT INTO pool_txs (tx_hash, data, sort_id)
		VALUES (
			'87a15e9b5707faac3fb7f573faf5f64b60696e584e158b8a494c12a26149a313',
			decode('010000000192b34025babea306bdf67cfe9a2576d8475ea9476caeb1fbdea43bf3d56d011affffffff70004830450221009037e1d39b7d59d24eba8012baddd5f4ab886a51b46f52b7c479ddfa55eeb5c5022076008409243475b25dfba6db85e15cf3d74561a147375941e4830baa69769b51012551210210b002870438af79b829bc22c4505e14779ef0080c411ad497d7a0846ee0af6f51ae00137b0a2020226b6579223a2022636c616d220a7d0125fbb43a93c290fde3997d92c416d3cc7ff40a13aa309d051406978635085c8d320000000000000017a9145881cd104f8d64635751ac0f3c0decf9150c11068700000000000000000000', 'hex'),
			1
		), (
			'c9667fd0c400df50b1b0629b0c2f47a72cad0a38a5c5a9cfc669ca3155f79791',
			decode('010000000192b34025babea306bdf67cfe9a2576d8475ea9476caeb1fbdea43bf3d56d011affffffff7000483045022100f3bcffcfd6a1ce9542b653500386cd0ee7b9c86c59390ca0fc0238c0ebe3f1d6022065ac468a51a016842660c3a616c99a9aa5109a3bad1877ba3e0f010f3972472e012551210210b002870438af79b829bc22c4505e14779ef0080c411ad497d7a0846ee0af6f51ae00137b0a2020226b6579223a2022636c616d220a7d0125fbb43a93c290fde3997d92c416d3cc7ff40a13aa309d051406978635085c8d320000000000000017a914c171e443e05b953baa7b7d834028ed91e47b4d0b8700000000000000000000', 'hex'),
			2
		);
	`

	withContext(t, fix, func(ctx context.Context) {
		now := time.Now()
		got, err := GenerateBlock(ctx, now)
		if err != nil {
			t.Fatalf("err got = %v want nil", err)
		}

		want := &bc.Block{
			BlockHeader: bc.BlockHeader{
				Version:           bc.NewBlockVersion,
				Height:            12,
				PreviousBlockHash: mustParseHash("c33576c0fa9ee9b3b71dcfb7872835400df327501da144fdf770ef751c08376d"),
				Timestamp:         uint64(now.Unix()),
			},
			Transactions: []*bc.Tx{
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
			},
		}
		for _, wanttx := range want.Transactions {
			wanttx.Stored = true
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("generated block:\ngot:  %+v\nwant: %+v", got, want)
		}
	})
}

func TestSignBlock(t *testing.T) {
	ctx := context.Background()

	key := newPrivKey(t)

	outscript, err := GenerateBlockScript([]*btcec.PublicKey{key.PubKey()}, 1)
	if err != nil {
		t.Log(errors.Stack(err))
		log.Fatal(err)
	}

	block := &bc.Block{}
	err = SignBlock(block, key)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	engine, err := txscript.NewEngineForBlock(ctx, outscript, block, txscript.StandardVerifyFlags)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	err = engine.Execute()
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}
}

func TestIsSignedByTrustedHost(t *testing.T) {
	keys := []*btcec.PrivateKey{newPrivKey(t)}

	block := &bc.Block{}
	err := SignBlock(block, keys[0])
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

func BenchmarkGenerateBlock(b *testing.B) {
	for i := 0; i < b.N; i++ {
		benchGenBlock(b)
	}
}

func benchGenBlock(b *testing.B) {
	const fix = `
		INSERT INTO blocks (block_hash, height, data, header)
		VALUES(
			'341fb89912be0110b527375998810c99ac96a317c63b071ccf33b7514cf0f0a5',
			1,
			decode('0000000100000000000000013132330000000000000000000000000000000000000000000000000000000000414243000000000000000000000000000000000000000000000000000000000058595a000000000000000000000000000000000000000000000000000000000000000000000000640f746573742d7369672d73637269707412746573742d6f75747075742d73637269707401000000010000000000000000000007746573742d7478', 'hex'),
			''
		);

		INSERT INTO pool_txs (tx_hash, data, sort_id)
		VALUES (
			'37383ebfffe807d694343a9004a42f605592e0dc7f7d5de76857fb46a7050410',
			decode('0000000101341fb89912be0110b527375998810c99ac96a317c63b071ccf33b7514cf0f0a5ffffffff6f00473045022100c561a9b4854742bc36c805513b872b2c0a1a367da24710eadd4f3fbc3b1ab41302207cf9eec4e5db694831fe43cf193f23d869291025ac6062199dd6b8998e93e15825512103623fb1fe38ce7e43cf407ec99b061c6d2da0278e80ce094393875c5b94f1ed9051ae0001df03f294bd08930f542a42b91199a8afe1b45c28eeb058cc5e8c8d600e0dd42f0000000000000001000000000000000000000474782d31', 'hex'),
			1
		), (
			'5b3864897b701f217ae956c7ce2bbfb9ac415da38430b7d56acd104ca9b03ed6',
			decode('0000000101341fb89912be0110b527375998810c99ac96a317c63b071ccf33b7514cf0f0a5ffffffff6e0046304402206ac2db5b49c8f9059d7ecad4f08a1d29e851e321720f590f5426cfbb19840d4402206aacef503d7c3cd065a17c2553b372ca2de0613eba3debc70896c9ab6545029b25512103b050bdde9880d9e8634f12798748cb26e9435a778305f3ae1ddba759d6479b2a51ae00015abad6dfb0de611046ebda5de05bfebc6a08d9a71831b43f2acd554bf54f33180000000000000001000000000000000000000474782d32', 'hex'),
			2
		);
	`

	b.StopTimer()
	withContext(b, fix, func(ctx context.Context) {
		now := time.Now()
		b.StartTimer()
		_, err := GenerateBlock(ctx, now)
		b.StopTimer()
		if err != nil {
			b.Fatal(err)
		}
	})
}

type clientInfo struct {
	asset          *appdb.Asset
	acctA          *appdb.Account
	acctB          *appdb.Account
	privKeyIssuer  *hdkey.XKey
	privKeyManager *hdkey.XKey
}

// TODO(kr): refactor this into new package api/apiutil
// and consume it from cmd/bootdb.
func bootdb(ctx context.Context) (*clientInfo, error) {
	key, err := testutil.TestXPrv.ECPrivKey()
	if err != nil {
		return nil, err
	}
	BlockKey = key
	_, err = UpsertGenesisBlock(ctx)
	if err != nil {
		return nil, err
	}

	u, err := appdb.CreateUser(ctx, "user@example.com", "password")
	if err != nil {
		return nil, err
	}

	proj, err := appdb.CreateProject(ctx, "proj", u.ID)
	if err != nil {
		return nil, err
	}

	manPub, manPriv, err := NewKey()
	if err != nil {
		return nil, err
	}
	manager, err := appdb.InsertManagerNode(ctx, proj.ID, "manager", []*hdkey.XKey{manPub}, []*hdkey.XKey{manPriv}, 0, 1)
	if err != nil {
		return nil, err
	}

	acctA, err := appdb.CreateAccount(ctx, manager.ID, "label", nil)
	if err != nil {
		return nil, err
	}

	acctB, err := appdb.CreateAccount(ctx, manager.ID, "label", nil)
	if err != nil {
		return nil, err
	}

	issPub, issPriv, err := NewKey()
	if err != nil {
		return nil, err
	}
	issuer, err := appdb.InsertIssuerNode(ctx, proj.ID, "issuer", []*hdkey.XKey{issPub}, []*hdkey.XKey{issPriv}, 1)
	if err != nil {
		return nil, err
	}

	asset, err := Create(ctx, issuer.ID, "label", map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	info := &clientInfo{
		asset:          asset,
		acctA:          acctA,
		acctB:          acctB,
		privKeyIssuer:  issPriv,
		privKeyManager: manPriv,
	}
	return info, nil
}

func issue(ctx context.Context, t testing.TB, info *clientInfo, destAcctID string, amount uint64) (*bc.Tx, error) {
	assetID := info.asset.Hash
	assetAmount := &bc.AssetAmount{
		AssetID: info.asset.Hash,
		Amount:  amount,
	}
	issueDest, err := NewAccountDestination(ctx, assetAmount, destAcctID, nil)
	if err != nil {
		return nil, err
	}
	issueTx, err := Issue(ctx, assetID.String(), []*txbuilder.Destination{issueDest})
	if err != nil {
		return nil, err
	}
	assettest.SignTxTemplate(t, issueTx, info.privKeyIssuer)
	return FinalizeTx(ctx, issueTx)
}

func transfer(ctx context.Context, t testing.TB, info *clientInfo, srcAcctID, destAcctID string, amount uint64) (*bc.Tx, error) {
	assetAmount := &bc.AssetAmount{
		AssetID: info.asset.Hash,
		Amount:  amount,
	}
	source := NewAccountSource(ctx, assetAmount, srcAcctID)
	sources := []*txbuilder.Source{source}

	dest, err := NewAccountDestination(ctx, assetAmount, destAcctID, nil)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	dests := []*txbuilder.Destination{dest}

	xferTx, err := txbuilder.Build(ctx, nil, sources, dests, []byte{}, time.Minute)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	assettest.SignTxTemplate(t, xferTx, info.privKeyManager)

	tx, err := FinalizeTx(ctx, xferTx)
	return tx, errors.Wrap(err)
}

func TestUpsertGenesisBlock(t *testing.T) {
	ctx := pgtest.NewContext(t)
	defer pgtest.Finish(ctx)

	key, err := testutil.TestXPrv.ECPrivKey()
	if err != nil {
		t.Fatal(err)
	}
	BlockKey = key

	b, err := UpsertGenesisBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	n := pgtest.Count(ctx, t, pg.FromContext(ctx), "blocks")
	if n != 1 {
		t.Fatalf("count(*) FROM blocks = %d want 1", n)
	}
	var got bc.Hash
	err = pg.FromContext(ctx).QueryRow(ctx, `SELECT block_hash FROM blocks`).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	want := b.Hash()
	if got != want {
		t.Errorf("block hash = %v want %v", got, want)
	}
}

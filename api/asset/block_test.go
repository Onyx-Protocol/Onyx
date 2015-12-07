package asset

import (
	"reflect"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/btcsuite/btcd/btcec"

	"chain/api/appdb"
	"chain/api/utxodb"
	"chain/database/pg"
	"chain/fedchain-sandbox/hdkey"
	"chain/fedchain/bc"
)

func TestTransferConfirmed(t *testing.T) {
	const genesisBlock = `
		INSERT INTO blocks (block_hash, height, data)
		VALUES(
			'341fb89912be0110b527375998810c99ac96a317c63b071ccf33b7514cf0f0a5',
			1,
			decode('0000000100000000000000013132330000000000000000000000000000000000000000000000000000000000414243000000000000000000000000000000000000000000000000000000000058595a000000000000000000000000000000000000000000000000000000000000000000000000640f746573742d7369672d73637269707412746573742d6f75747075742d73637269707401000000010000000000000000000007746573742d7478', 'hex')
		);
	`
	withContext(t, genesisBlock, func(ctx context.Context) {
		u, err := appdb.CreateUser(ctx, "user@example.com", "password")
		if err != nil {
			t.Fatal(err)
		}

		proj, err := appdb.CreateProject(ctx, "proj", u.ID)
		if err != nil {
			t.Fatal(err)
		}

		manPub, manPriv, err := newKey()
		if err != nil {
			t.Fatal(err)
		}
		manager, err := appdb.InsertManagerNode(ctx, proj.ID, "manager", []*hdkey.XKey{manPub}, []*hdkey.XKey{manPriv})
		if err != nil {
			t.Fatal(err)
		}

		acctA, err := appdb.CreateAccount(ctx, manager.ID, "label")
		if err != nil {
			t.Fatal(err)
		}

		acctB, err := appdb.CreateAccount(ctx, manager.ID, "label")
		if err != nil {
			t.Fatal(err)
		}

		issPub, issPriv, err := newKey()
		if err != nil {
			t.Fatal(err)
		}
		issuer, err := appdb.InsertIssuerNode(ctx, proj.ID, "issuer", []*hdkey.XKey{issPub}, []*hdkey.XKey{issPriv})
		if err != nil {
			t.Fatal(err)
		}

		asset, err := Create(ctx, issuer.ID, "label", map[string]interface{}{})
		if err != nil {
			t.Fatal(err)
		}

		issueOuts := []*Output{{
			AssetID:   asset.Hash.String(),
			AccountID: acctA.ID,
			Amount:    10,
		}}
		issueTx, err := Issue(ctx, asset.Hash.String(), issueOuts)
		if err != nil {
			t.Fatal(err)
		}
		signTx(t, issueTx, issPriv)
		_, err = FinalizeTx(ctx, issueTx)
		if err != nil {
			t.Fatal(err)
		}

		dumpState(ctx, t)
		makeBlock(ctx)
		dumpState(ctx, t)

		inputs := []utxodb.Input{{
			AssetID:   asset.Hash.String(),
			AccountID: acctA.ID,
			Amount:    10,
		}}
		outputs := []*Output{{
			AssetID:   asset.Hash.String(),
			AccountID: acctB.ID,
			Amount:    10,
		}}
		xferTx, err := Transfer(ctx, inputs, outputs)
		if err != nil {
			t.Fatal(err)
		}

		signTx(t, xferTx, manPriv)
		_, err = FinalizeTx(ctx, xferTx)
		if err != nil {
			t.Fatal(err)
		}
	})
}

func BenchmarkTransferWithBlocks(b *testing.B) {
	const genesisBlock = `
		INSERT INTO blocks (block_hash, height, data)
		VALUES(
			'341fb89912be0110b527375998810c99ac96a317c63b071ccf33b7514cf0f0a5',
			1,
			decode('0000000100000000000000013132330000000000000000000000000000000000000000000000000000000000414243000000000000000000000000000000000000000000000000000000000058595a000000000000000000000000000000000000000000000000000000000000000000000000640f746573742d7369672d73637269707412746573742d6f75747075742d73637269707401000000010000000000000000000007746573742d7478', 'hex')
		);
	`
	withContext(b, genesisBlock, func(ctx context.Context) {
		u, err := appdb.CreateUser(ctx, "user@example.com", "password")
		if err != nil {
			b.Fatal(err)
		}

		proj, err := appdb.CreateProject(ctx, "proj", u.ID)
		if err != nil {
			b.Fatal(err)
		}

		manPub, manPriv, err := newKey()
		if err != nil {
			b.Fatal(err)
		}
		manager, err := appdb.InsertManagerNode(ctx, proj.ID, "manager", []*hdkey.XKey{manPub}, []*hdkey.XKey{manPriv})
		if err != nil {
			b.Fatal(err)
		}

		acctA, err := appdb.CreateAccount(ctx, manager.ID, "label")
		if err != nil {
			b.Fatal(err)
		}

		acctB, err := appdb.CreateAccount(ctx, manager.ID, "label")
		if err != nil {
			b.Fatal(err)
		}

		issPub, issPriv, err := newKey()
		if err != nil {
			b.Fatal(err)
		}
		issuer, err := appdb.InsertIssuerNode(ctx, proj.ID, "issuer", []*hdkey.XKey{issPub}, []*hdkey.XKey{issPriv})
		if err != nil {
			b.Fatal(err)
		}

		asset, err := Create(ctx, issuer.ID, "label", map[string]interface{}{})
		if err != nil {
			b.Fatal(err)
		}

		for i := 0; i < b.N; i++ {
			issueOuts := []*Output{{
				AssetID:   asset.Hash.String(),
				AccountID: acctA.ID,
				Amount:    10,
			}}
			issueTx, err := Issue(ctx, asset.Hash.String(), issueOuts)
			if err != nil {
				b.Fatal(err)
			}
			signTx(b, issueTx, issPriv)
			tx, err := FinalizeTx(ctx, issueTx)
			if err != nil {
				b.Fatal(err)
			}
			b.Logf("finalized %v", tx.Hash())

			inputs := []utxodb.Input{{
				AssetID:   asset.Hash.String(),
				AccountID: acctA.ID,
				Amount:    10,
			}}
			outputs := []*Output{{
				AssetID:   asset.Hash.String(),
				AccountID: acctB.ID,
				Amount:    10,
			}}
			xferTx, err := Transfer(ctx, inputs, outputs)
			if err != nil {
				b.Fatal(err)
			}

			signTx(b, xferTx, manPriv)
			tx, err = FinalizeTx(ctx, xferTx)
			if err != nil {
				b.Fatal(err)
			}
			b.Logf("finalized %v", tx.Hash())

			if i%10 == 0 {
				makeBlock(ctx)
			}
		}
	})
}

func signTx(tb testing.TB, tx *Tx, priv *hdkey.XKey) {
	for _, input := range tx.Inputs {
		for _, sig := range input.Sigs {
			key, err := derive(priv, sig.DerivationPath)
			if err != nil {
				tb.Fatal(err)
			}
			dat, err := key.Sign(input.SignatureData[:])
			if err != nil {
				tb.Fatal(err)
			}
			sig.DER = append(dat.Serialize(), 1) // append hashtype SIGHASH_ALL
		}
	}
}

func dumpState(ctx context.Context, t *testing.T) {
	t.Log("pool")
	dumpTab(ctx, t, `SELECT tx_hash, index, script FROM pool_outputs`)
	t.Log("blockchain")
	dumpTab(ctx, t, `SELECT txid, index, script FROM utxos`)
}

func dumpTab(ctx context.Context, t *testing.T, q string) {
	rows, err := pg.FromContext(ctx).Query(q)
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
		t.Logf("hash: %x index: %d pkscript: %x", hash, index, script)
	}
	if rows.Err() != nil {
		t.Fatal(rows.Err())
	}
}

func TestGenerateBlock(t *testing.T) {
	const fix = `
		INSERT INTO blocks (block_hash, height, data)
		VALUES(
			'92b34025babea306bdf67cfe9a2576d8475ea9476caeb1fbdea43bf3d56d011a',
			11,
			decode('00000001000000000000000b95a00b5cd11f577a461e6bb884899ee0aa1662088097b644af7a50d76e1a243f0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000565f8903000001000000010195a00b5cd11f577a461e6bb884899ee0aa1662088097b644af7a50d76e1a243fffffffff7000483045022100c80b4deb9aae29da4e8768a5fbe0ac6ccca1d020f4b924005cc066f09b18e14e02206acd491a84eda9c15bed01a7648b191974a2ef47f7cefda3bde06092cd144e68012551210210b002870438af79b829bc22c4505e14779ef0080c411ad497d7a0846ee0af6f51ae00000125fbb43a93c290fde3997d92c416d3cc7ff40a13aa309d051406978635085c8d000000000000003217a9144bbae13b661c0cd9cf89271fd96dadb65e7a80378700000000000000000000', 'hex')
		);

		INSERT INTO pool_txs (tx_hash, data, sort_id)
		VALUES (
			'd8d804a9fae1dc447779eb9826116f32f22c83bef4ef228d6423e99a546deebd',
			decode('000000010192b34025babea306bdf67cfe9a2576d8475ea9476caeb1fbdea43bf3d56d011affffffff70004830450221009037e1d39b7d59d24eba8012baddd5f4ab886a51b46f52b7c479ddfa55eeb5c5022076008409243475b25dfba6db85e15cf3d74561a147375941e4830baa69769b51012551210210b002870438af79b829bc22c4505e14779ef0080c411ad497d7a0846ee0af6f51ae00137b0a2020226b6579223a2022636c616d220a7d0125fbb43a93c290fde3997d92c416d3cc7ff40a13aa309d051406978635085c8d000000000000003217a9145881cd104f8d64635751ac0f3c0decf9150c11068700000000000000000000', 'hex'),
			1
		), (
			'27764579c4cf0395c91c6941011b3e9a627b02f29b259e8f6bc5ca9c50c5f256',
			decode('000000010192b34025babea306bdf67cfe9a2576d8475ea9476caeb1fbdea43bf3d56d011affffffff7000483045022100f3bcffcfd6a1ce9542b653500386cd0ee7b9c86c59390ca0fc0238c0ebe3f1d6022065ac468a51a016842660c3a616c99a9aa5109a3bad1877ba3e0f010f3972472e012551210210b002870438af79b829bc22c4505e14779ef0080c411ad497d7a0846ee0af6f51ae00137b0a2020226b6579223a2022636c616d220a7d0125fbb43a93c290fde3997d92c416d3cc7ff40a13aa309d051406978635085c8d000000000000003217a914c171e443e05b953baa7b7d834028ed91e47b4d0b8700000000000000000000', 'hex'),
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
				PreviousBlockHash: mustParseHash("92b34025babea306bdf67cfe9a2576d8475ea9476caeb1fbdea43bf3d56d011a"),
				Timestamp:         uint64(now.Unix()),
			},
			Transactions: []*bc.Tx{
				{
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
						Value:   50,
						AssetID: mustParseHash("25fbb43a93c290fde3997d92c416d3cc7ff40a13aa309d051406978635085c8d"),
						Script:  mustDecodeHex("a9145881cd104f8d64635751ac0f3c0decf9150c110687"),
					}},
				},
				{
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
						Value:   50,
						AssetID: mustParseHash("25fbb43a93c290fde3997d92c416d3cc7ff40a13aa309d051406978635085c8d"),
						Script:  mustDecodeHex("a914c171e443e05b953baa7b7d834028ed91e47b4d0b87"),
					}},
				},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("generated block:\ngot:  %+v\nwant: %+v", got, want)
		}
	})
}

func derive(xkey *hdkey.XKey, path []uint32) (*btcec.PrivateKey, error) {
	// The only error has a uniformly distributed probability of 1/2^127
	// We've decided to ignore this chance.
	key := &xkey.ExtendedKey
	for _, p := range path {
		key, _ = key.Child(p)
	}
	return key.ECPrivKey()
}

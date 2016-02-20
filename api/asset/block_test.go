package asset_test

import (
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec"

	"golang.org/x/net/context"

	"chain/api/appdb"
	. "chain/api/asset"
	"chain/api/asset/assettest"
	"chain/api/generator"
	"chain/api/issuer"
	"chain/api/txbuilder"
	"chain/api/txdb"
	"chain/api/utxodb"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/fedchain"
	"chain/fedchain-sandbox/hdkey"
	"chain/fedchain/bc"
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
	_, err = generator.MakeBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}
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

	block, prevBlock, err := FC().GenerateBlock(ctx, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	err = generator.GetAndAddBlockSignatures(ctx, block, prevBlock)
	if err != nil {
		t.Fatal(err)
	}

	_, err = transfer(ctx, t, info, info.acctA.ID, info.acctB.ID, 10)
	if err != nil {
		t.Fatal(err)
	}

	err = FC().AddBlock(ctx, block)
	if err != nil {
		t.Fatal(err)
	}

	inputs := []utxodb.Source{{
		AssetID:   info.asset.Hash,
		AccountID: info.acctA.ID,
		Amount:    10,
	}}
	reserved, _, err := utxodb.Reserve(ctx, inputs, 2*time.Minute)
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
				_, err = generator.MakeBlock(ctx)
				if err != nil {
					b.Fatal(err)
				}
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
		_, _, err := FC().GenerateBlock(ctx, now)
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
	_, err := assettest.InitializeSigningGenerator(ctx)
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
	manager, err := appdb.InsertManagerNode(ctx, proj.ID, "manager", []*hdkey.XKey{manPub}, []*hdkey.XKey{manPriv}, 0, 1, nil)
	if err != nil {
		return nil, err
	}

	acctA, err := appdb.CreateAccount(ctx, manager.ID, "label", nil, nil)
	if err != nil {
		return nil, err
	}

	acctB, err := appdb.CreateAccount(ctx, manager.ID, "label", nil, nil)
	if err != nil {
		return nil, err
	}

	issPub, issPriv, err := NewKey()
	if err != nil {
		return nil, err
	}
	iNode, err := appdb.InsertIssuerNode(ctx, proj.ID, "issuer", []*hdkey.XKey{issPub}, []*hdkey.XKey{issPriv}, 1, nil)
	if err != nil {
		return nil, err
	}

	asset, err := issuer.CreateAsset(ctx, iNode.ID, "label", map[string]interface{}{}, nil)
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
	issueTx, err := issuer.Issue(ctx, assetID.String(), []*txbuilder.Destination{issueDest})
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
	source := NewAccountSource(ctx, assetAmount, srcAcctID, nil)
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

	pubkey, err := testutil.TestXPub.ECPubKey()
	if err != nil {
		t.Fatal(err)
	}

	fc, err := fedchain.New(ctx, txdb.NewStore(), nil)
	if err != nil {
		t.Fatal(err)
	}

	b, err := fc.UpsertGenesisBlock(ctx, []*btcec.PublicKey{pubkey}, 1)
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

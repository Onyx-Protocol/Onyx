package txbuilder_test

import (
	"testing"
	"time"

	"golang.org/x/net/context"

	"chain/core/account"
	"chain/core/account/utxodb"
	"chain/core/asset"
	"chain/core/asset/assettest"
	"chain/core/generator"
	. "chain/core/txbuilder"
	"chain/core/txdb"
	"chain/cos"
	"chain/cos/bc"
	"chain/crypto/ed25519"
	"chain/crypto/ed25519/hd25519"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/database/sql"
	"chain/errors"
	"chain/testutil"
)

// TestConflictingTxsInPool tests creating conflicting transactions, and
// ensures that they both make it into the tx pool. Then, when a block
// lands, only one of the txs should be confirmed.
//
// Conflicting txs are created by building a tx template with only a
// source, and then building two different txs with that same source,
// but destinations w/ different addresses.
func TestConflictingTxsInPool(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	info, fc, g, err := bootdb(ctx, t)
	if err != nil {
		t.Fatal(err)
	}

	_, err = issue(ctx, t, fc, info, info.acctA.ID, 10)
	if err != nil {
		t.Fatal(err)
	}

	dumpState(ctx, t)
	_, err = g.MakeBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	dumpState(ctx, t)

	assetAmount := bc.AssetAmount{
		AssetID: info.asset.AssetID,
		Amount:  10,
	}
	spendAction := assettest.NewAccountSpendAction(assetAmount, info.acctA.ID, nil, nil, nil)
	spendAction.Params.TTL = time.Millisecond
	dest1 := assettest.NewAccountControlAction(assetAmount, info.acctB.ID, nil)

	// Build the first tx
	firstTemplate, err := Build(ctx, nil, []Action{spendAction, dest1}, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	assettest.SignTxTemplate(t, firstTemplate, info.privKeyAccounts)
	tx, err := FinalizeTx(ctx, fc, firstTemplate)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// Build the second tx
	secondTemplate, err := Build(ctx, &tx.TxData, nil, []byte("test"))
	secondTemplate.Inputs = firstTemplate.Inputs
	ComputeSigHashes(secondTemplate)
	assettest.SignTxTemplate(t, secondTemplate, info.privKeyAccounts)
	_, err = FinalizeTx(ctx, fc, secondTemplate)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// Make a block, which should reject one of the txs.
	dumpState(ctx, t)
	b, err := g.MakeBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	dumpState(ctx, t)
	if len(b.Transactions) != 1 {
		t.Errorf("got block.Transactions = %#v\n, want exactly one tx", b.Transactions)
	}
}

func TestTransferConfirmed(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)

	info, fc, g, err := bootdb(ctx, t)
	if err != nil {
		t.Fatal(err)
	}

	_, err = issue(ctx, t, fc, info, info.acctA.ID, 10)
	if err != nil {
		t.Fatal(err)
	}

	dumpState(ctx, t)
	_, err = g.MakeBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	dumpState(ctx, t)

	_, err = transfer(ctx, t, fc, info, info.acctA.ID, info.acctB.ID, 10)
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

	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)

	info, fc, g, err := bootdb(ctx, t)
	if err != nil {
		t.Fatal(err)
	}
	issueTx, err := issue(ctx, t, fc, info, info.acctA.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("issued %v", issueTx.Hash)

	block, prevBlock, err := fc.GenerateBlock(ctx, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	err = g.GetAndAddBlockSignatures(ctx, block, prevBlock)
	if err != nil {
		t.Fatal(err)
	}

	_, err = transfer(ctx, t, fc, info, info.acctA.ID, info.acctB.ID, 10)
	if err != nil {
		t.Fatal(err)
	}

	err = fc.AddBlock(ctx, block)
	if err != nil {
		t.Fatal(err)
	}

	inputs := []utxodb.Source{{
		AssetID:   info.asset.AssetID,
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
	_, db := pgtest.NewDB(b, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	info, fc, g, err := bootdb(ctx, b)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		tx, err := issue(ctx, b, fc, info, info.acctA.ID, 10)
		if err != nil {
			b.Fatal(err)
		}
		b.Logf("finalized %v", tx.Hash)

		tx, err = transfer(ctx, b, fc, info, info.acctA.ID, info.acctB.ID, 10)
		if err != nil {
			b.Fatal(err)
		}
		b.Logf("finalized %v", tx.Hash)

		if i%10 == 0 {
			_, err = g.MakeBlock(ctx)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func dumpState(ctx context.Context, t *testing.T) {
	t.Log("pool")
	dumpTab(ctx, t, `
		SELECT tx_hash, data FROM pool_txs
	`)
	t.Log("blockchain")
	dumpTab(ctx, t, `
		SELECT blocks_txs.tx_hash, txs.data FROM blocks_txs
		INNER JOIN txs ON blocks_txs.tx_hash = txs.tx_hash
	`)
}

func dumpTab(ctx context.Context, t *testing.T, q string) {
	rows, err := pg.Query(ctx, q)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var hash bc.Hash
		var tx bc.TxData
		err = rows.Scan(&hash, &tx)
		if err != nil {
			t.Fatal(err)
		}
		for index, o := range tx.Outputs {
			t.Logf("hash: %s index: %d pkscript: %x", hash, index, o.ControlProgram)
		}
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
	b.StopTimer()
	_, db := pgtest.NewDB(b, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)
	pgtest.Exec(ctx, b, `
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
	`)
	_, fc, _, err := bootdb(ctx, b)
	if err != nil {
		testutil.FatalErr(b, err)
	}
	now := time.Now()
	b.StartTimer()
	_, _, err = fc.GenerateBlock(ctx, now)
	b.StopTimer()
	if err != nil {
		b.Fatal(err)
	}
}

type clientInfo struct {
	asset           *asset.Asset
	acctA           *account.Account
	acctB           *account.Account
	privKeyAsset    *hd25519.XPrv
	privKeyAccounts *hd25519.XPrv
}

// TODO(kr): refactor this into new package core/coreutil
// and consume it from cmd/corectl.
func bootdb(ctx context.Context, t testing.TB) (*clientInfo, *cos.FC, *generator.Generator, error) {
	store, pool := txdb.New(pg.FromContext(ctx).(*sql.DB))
	fc, g, err := assettest.InitializeSigningGenerator(ctx, store, pool)
	if err != nil {
		return nil, nil, nil, err
	}

	dbtx, ctx, err := pg.Begin(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	defer dbtx.Rollback(ctx)

	accPriv, accPub, err := hd25519.NewXKeys(nil)
	if err != nil {
		return nil, nil, nil, err
	}

	acctA, err := account.Create(ctx, []string{accPub.String()}, 1, "", nil, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	acctB, err := account.Create(ctx, []string{accPub.String()}, 1, "", nil, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	genesis, err := store.GetBlock(ctx, 1)
	if err != nil {
		return nil, nil, nil, err
	}

	assetPriv, assetPub, err := hd25519.NewXKeys(nil)
	if err != nil {
		return nil, nil, nil, err
	}

	asset, err := asset.Define(ctx, []string{assetPub.String()}, 1, nil, genesis.Hash(), "", nil, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	err = dbtx.Commit(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	info := &clientInfo{
		asset:           asset,
		acctA:           acctA,
		acctB:           acctB,
		privKeyAsset:    assetPriv,
		privKeyAccounts: accPriv,
	}
	return info, fc, g, nil
}

func issue(ctx context.Context, t testing.TB, fc *cos.FC, info *clientInfo, destAcctID string, amount uint64) (*bc.Tx, error) {
	assetAmount := bc.AssetAmount{
		AssetID: info.asset.AssetID,
		Amount:  amount,
	}
	issueDest := assettest.NewAccountControlAction(assetAmount, destAcctID, nil)
	issueTx, err := Build(
		ctx,
		nil,
		[]Action{assettest.NewIssueAction(assetAmount, nil), issueDest},
		nil,
	)
	if err != nil {
		return nil, err
	}
	assettest.SignTxTemplate(t, issueTx, info.privKeyAsset)
	return FinalizeTx(ctx, fc, issueTx)
}

func transfer(ctx context.Context, t testing.TB, fc *cos.FC, info *clientInfo, srcAcctID, destAcctID string, amount uint64) (*bc.Tx, error) {
	assetAmount := bc.AssetAmount{
		AssetID: info.asset.AssetID,
		Amount:  amount,
	}
	source := assettest.NewAccountSpendAction(assetAmount, srcAcctID, nil, nil, nil)
	dest := assettest.NewAccountControlAction(assetAmount, destAcctID, nil)

	xferTx, err := Build(ctx, nil, []Action{source, dest}, []byte{})
	if err != nil {
		return nil, errors.Wrap(err)
	}

	assettest.SignTxTemplate(t, xferTx, info.privKeyAccounts)

	tx, err := FinalizeTx(ctx, fc, xferTx)
	return tx, errors.Wrap(err)
}

func TestUpsertGenesisBlock(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := pg.NewContext(context.Background(), db)

	pubkey := testutil.TestPub

	store, pool := txdb.New(pg.FromContext(ctx).(*sql.DB))
	fc, err := cos.NewFC(ctx, store, pool, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	b, err := fc.UpsertGenesisBlock(ctx, []ed25519.PublicKey{pubkey}, 1, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	var n int64
	err = pg.QueryRow(ctx, "SELECT count(*) FROM blocks").Scan(&n)
	if err != nil {
		t.Fatal("Count:", err)
	} else if n != 1 {
		t.Fatalf("count(*) FROM blocks = %d want 1", n)
	}

	var got bc.Hash
	err = pg.QueryRow(ctx, `SELECT block_hash FROM blocks`).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	want := b.Hash()
	if got != want {
		t.Errorf("block hash = %v want %v", got, want)
	}
}

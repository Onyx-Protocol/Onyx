package txbuilder_test

import (
	"context"
	"testing"
	"time"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/coretest"
	"chain/core/pin"
	"chain/core/query"
	. "chain/core/txbuilder"
	"chain/crypto/ed25519/chainkd"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/database/sql"
	"chain/errors"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/mempool"
	"chain/protocol/memstore"
	"chain/protocol/prottest"
	"chain/protocol/state"
	"chain/testutil"
)

func TestSighashCheck(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := context.Background()
	info, err := bootdb(ctx, db, t)
	if err != nil {
		t.Fatal(err)
	}

	_, err = issue(ctx, t, info, info.acctA.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	_, err = issue(ctx, t, info, info.acctB.ID, 10)
	if err != nil {
		t.Fatal(err)
	}

	prottest.MakeBlock(t, info.Chain)
	<-info.pinStore.PinWaiter(account.PinName, info.Chain.Height())

	assetAmount := bc.AssetAmount{
		AssetID: info.asset.AssetID,
		Amount:  1,
	}
	spendAction1 := info.NewSpendAction(assetAmount, info.acctA.ID, nil, nil)
	controlAction1 := info.NewControlAction(assetAmount, info.acctB.ID, nil)

	tpl1, err := Build(ctx, nil, []Action{spendAction1, controlAction1}, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	tpl1.AllowAdditional = true
	coretest.SignTxTemplate(t, ctx, tpl1, &info.privKeyAccounts)
	err = CheckTxSighashCommitment(bc.NewTx(*tpl1.Transaction))
	if err == nil {
		t.Error("unexpected success from checkTxSighashCommitment")
	}

	spendAction2a := info.NewSpendAction(assetAmount, info.acctB.ID, nil, nil)
	controlAction2 := info.NewControlAction(assetAmount, info.acctA.ID, nil)

	tpl2a, err := Build(ctx, tpl1.Transaction, []Action{spendAction2a, controlAction2}, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	coretest.SignTxTemplate(t, ctx, tpl2a, &info.privKeyAccounts)
	err = CheckTxSighashCommitment(bc.NewTx(*tpl2a.Transaction))
	if err != nil {
		t.Errorf("unexpected failure from checkTxSighashCommitment (case 1): %v", err)
	}

	issueAction2b := info.NewIssueAction(assetAmount, nil)
	tpl2b, err := Build(ctx, tpl1.Transaction, []Action{issueAction2b, controlAction2}, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	coretest.SignTxTemplate(t, ctx, tpl2b, &info.privKeyAsset)
	err = CheckTxSighashCommitment(bc.NewTx(*tpl2b.Transaction))
	if err != nil {
		t.Errorf("unexpected failure from checkTxSighashCommitment (case 2): %v", err)
	}
}

// TestConflictingTxsInPool tests creating conflicting transactions, and
// ensures that they both make it into the tx pool. Then, when a block
// lands, only one of the txs should be confirmed.
//
// Conflicting txs are created by building a tx template with only a
// source, and then building two different txs with that same source,
// but destinations w/ different addresses.
func TestConflictingTxsInPool(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := context.Background()
	info, err := bootdb(ctx, db, t)
	if err != nil {
		t.Fatal(err)
	}

	_, err = issue(ctx, t, info, info.acctA.ID, 10)
	if err != nil {
		t.Fatal(err)
	}

	dumpBlocks(ctx, t, db)
	prottest.MakeBlock(t, info.Chain)
	dumpBlocks(ctx, t, db)
	<-info.pinStore.PinWaiter(account.PinName, info.Chain.Height())

	assetAmount := bc.AssetAmount{
		AssetID: info.asset.AssetID,
		Amount:  10,
	}
	spendAction := info.NewSpendAction(assetAmount, info.acctA.ID, nil, nil)
	dest1 := info.NewControlAction(assetAmount, info.acctB.ID, nil)

	// Build the first tx
	firstTemplate, err := Build(ctx, nil, []Action{spendAction, dest1}, time.Now().Add(time.Minute))
	if err != nil {
		testutil.FatalErr(t, err)
	}
	unsignedTx := *firstTemplate.Transaction
	coretest.SignTxTemplate(t, ctx, firstTemplate, &info.privKeyAccounts)
	tx := bc.NewTx(*firstTemplate.Transaction)
	err = FinalizeTx(ctx, info.Chain, tx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// Slighly tweak the first tx so it has a different hash, but
	// still consumes the same UTXOs.
	unsignedTx.MaxTime++
	secondTemplate := &Template{
		Transaction:         &unsignedTx,
		SigningInstructions: firstTemplate.SigningInstructions,
		Local:               true,
	}
	secondTemplate.SigningInstructions[0].WitnessComponents[0].(*SignatureWitness).Program = nil
	secondTemplate.SigningInstructions[0].WitnessComponents[0].(*SignatureWitness).Sigs = nil
	coretest.SignTxTemplate(t, ctx, secondTemplate, &info.privKeyAccounts)
	err = FinalizeTx(ctx, info.Chain, bc.NewTx(*secondTemplate.Transaction))
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// Make a block, which should reject one of the txs.
	dumpBlocks(ctx, t, db)
	b := prottest.MakeBlock(t, info.Chain)
	<-info.pinStore.PinWaiter(account.PinName, info.Chain.Height())

	dumpBlocks(ctx, t, db)
	if len(b.Transactions) != 1 {
		t.Errorf("got block.Transactions = %#v\n, want exactly one tx", b.Transactions)
	}
}

func TestTransferConfirmed(t *testing.T) {
	_, db := pgtest.NewDB(t, pgtest.SchemaPath)
	ctx := context.Background()

	info, err := bootdb(ctx, db, t)
	if err != nil {
		t.Fatal(err)
	}

	_, err = issue(ctx, t, info, info.acctA.ID, 10)
	if err != nil {
		t.Fatal(err)
	}

	dumpBlocks(ctx, t, db)
	prottest.MakeBlock(t, info.Chain)
	dumpBlocks(ctx, t, db)

	<-info.pinStore.PinWaiter(account.PinName, info.Chain.Height())

	_, err = transfer(ctx, t, info, info.acctA.ID, info.acctB.ID, 10)
	if err != nil {
		testutil.FatalErr(t, err)
	}
}

func BenchmarkTransferWithBlocks(b *testing.B) {
	_, db := pgtest.NewDB(b, pgtest.SchemaPath)
	ctx := context.Background()
	info, err := bootdb(ctx, db, b)
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
			prottest.MakeBlock(b, info.Chain)
		}
	}
}

func dumpTab(ctx context.Context, t *testing.T, db pg.DB, q string) {
	rows, err := db.Query(ctx, q)
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

func dumpBlocks(ctx context.Context, t *testing.T, db pg.DB) {
	rows, err := db.Query(ctx, `SELECT height, block_hash FROM blocks`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var height uint64
		var hash bc.Hash
		err = rows.Scan(&height, &hash)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("height:%d hash:%v", height, hash)
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
	const tx1hex = `0000000101341fb89912be0110b527375998810c99ac96a317c63b071ccf33b7514cf0f0a5ffffffff6f00473045022100c561a9b4854742bc36c805513b872b2c0a1a367da24710eadd4f3fbc3b1ab41302207cf9eec4e5db694831fe43cf193f23d869291025ac6062199dd6b8998e93e15825512103623fb1fe38ce7e43cf407ec99b061c6d2da0278e80ce094393875c5b94f1ed9051ae0001df03f294bd08930f542a42b91199a8afe1b45c28eeb058cc5e8c8d600e0dd42f0000000000000001000000000000000000000474782d31`
	const tx2hex = `0000000101341fb89912be0110b527375998810c99ac96a317c63b071ccf33b7514cf0f0a5ffffffff6e0046304402206ac2db5b49c8f9059d7ecad4f08a1d29e851e321720f590f5426cfbb19840d4402206aacef503d7c3cd065a17c2553b372ca2de0613eba3debc70896c9ab6545029b25512103b050bdde9880d9e8634f12798748cb26e9435a778305f3ae1ddba759d6479b2a51ae00015abad6dfb0de611046ebda5de05bfebc6a08d9a71831b43f2acd554bf54f33180000000000000001000000000000000000000474782d32`

	b.StopTimer()

	ctx := context.Background()
	store, pool := memstore.New(), mempool.New()
	c := prottest.NewChainWithStorage(b, store, pool)
	initialBlock, err := c.GetBlock(ctx, 1)
	if err != nil {
		testutil.FatalErr(b, err)
	}

	var tx1, tx2 bc.Tx
	err = tx1.UnmarshalText([]byte(tx1hex))
	if err != nil {
		b.Fatal(err)
	}
	err = tx2.UnmarshalText([]byte(tx2hex))
	if err != nil {
		b.Fatal(err)
	}
	err = pool.Insert(ctx, &tx1)
	if err != nil {
		b.Fatal(err)
	}
	err = pool.Insert(ctx, &tx2)
	if err != nil {
		b.Fatal(err)
	}

	now := time.Now()
	b.StartTimer()
	_, _, err = c.GenerateBlock(ctx, initialBlock, state.Empty(), now)
	b.StopTimer()
	if err != nil {
		b.Fatal(err)
	}
}

type testInfo struct {
	*asset.Registry
	*account.Manager
	*protocol.Chain
	pinStore        *pin.Store
	asset           *asset.Asset
	acctA           *account.Account
	acctB           *account.Account
	privKeyAsset    chainkd.XPrv
	privKeyAccounts chainkd.XPrv
}

// TODO(kr): refactor this into new package core/coreutil
// and consume it from cmd/corectl.
func bootdb(ctx context.Context, db *sql.DB, t testing.TB) (*testInfo, error) {
	c := prottest.NewChain(t)
	pinStore := pin.NewStore(db)
	coretest.CreatePins(ctx, t, pinStore)
	indexer := query.NewIndexer(db, c, pinStore)
	assets := asset.NewRegistry(db, c, pinStore)
	accounts := account.NewManager(db, c, pinStore)
	assets.IndexAssets(indexer)
	accounts.IndexAccounts(indexer)
	go accounts.ProcessBlocks(ctx)

	accPriv, accPub, err := chainkd.NewXKeys(nil)
	if err != nil {
		return nil, err
	}

	acctA, err := accounts.Create(ctx, []string{accPub.String()}, 1, "", nil, nil)
	if err != nil {
		return nil, err
	}
	acctB, err := accounts.Create(ctx, []string{accPub.String()}, 1, "", nil, nil)
	if err != nil {
		return nil, err
	}

	assetPriv, assetPub, err := chainkd.NewXKeys(nil)
	if err != nil {
		return nil, err
	}
	asset, err := assets.Define(ctx, []string{assetPub.String()}, 1, nil, "", nil, nil)
	if err != nil {
		return nil, err
	}

	info := &testInfo{
		Chain:           c,
		Registry:        assets,
		Manager:         accounts,
		pinStore:        pinStore,
		asset:           asset,
		acctA:           acctA,
		acctB:           acctB,
		privKeyAsset:    assetPriv,
		privKeyAccounts: accPriv,
	}
	return info, nil
}

func issue(ctx context.Context, t testing.TB, info *testInfo, destAcctID string, amount uint64) (*bc.Tx, error) {
	assetAmount := bc.AssetAmount{
		AssetID: info.asset.AssetID,
		Amount:  amount,
	}
	issueTx, err := Build(ctx, nil, []Action{
		info.Registry.NewIssueAction(assetAmount, nil),
		info.Manager.NewControlAction(assetAmount, destAcctID, nil),
	}, time.Now().Add(time.Minute))
	if err != nil {
		return nil, err
	}
	coretest.SignTxTemplate(t, ctx, issueTx, &info.privKeyAsset)
	tx := bc.NewTx(*issueTx.Transaction)
	return tx, FinalizeTx(ctx, info.Chain, tx)
}

func transfer(ctx context.Context, t testing.TB, info *testInfo, srcAcctID, destAcctID string, amount uint64) (*bc.Tx, error) {
	assetAmount := bc.AssetAmount{
		AssetID: info.asset.AssetID,
		Amount:  amount,
	}
	source := info.NewSpendAction(assetAmount, srcAcctID, nil, nil)
	dest := info.NewControlAction(assetAmount, destAcctID, nil)

	xferTx, err := Build(ctx, nil, []Action{source, dest}, time.Now().Add(time.Minute))
	if err != nil {
		return nil, errors.Wrap(err)
	}

	coretest.SignTxTemplate(t, ctx, xferTx, &info.privKeyAccounts)

	tx := bc.NewTx(*xferTx.Transaction)
	err = FinalizeTx(ctx, info.Chain, tx)
	return tx, errors.Wrap(err)
}

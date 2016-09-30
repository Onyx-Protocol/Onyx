package txbuilder_test

import (
	"context"
	"testing"
	"time"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/asset/assettest"
	. "chain/core/txbuilder"
	"chain/crypto/ed25519/chainkd"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/protocol"
	"chain/protocol/bc"
	"chain/protocol/prottest"
	"chain/protocol/state"
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
	dbtx := pgtest.NewTx(t)
	ctx := pg.NewContext(context.Background(), dbtx)
	info, c, err := bootdb(ctx, t)
	if err != nil {
		t.Fatal(err)
	}

	_, err = issue(ctx, t, c, info, info.acctA.ID, 10)
	if err != nil {
		t.Fatal(err)
	}

	dumpState(ctx, t)
	prottest.MakeBlock(ctx, t, c)
	dumpState(ctx, t)

	assetAmount := bc.AssetAmount{
		AssetID: info.asset.AssetID,
		Amount:  10,
	}
	spendAction := assettest.NewAccountSpendAction(assetAmount, info.acctA.ID, nil, nil, nil)
	dest1 := assettest.NewAccountControlAction(assetAmount, info.acctB.ID, nil)

	// Build the first tx
	firstTemplate, err := Build(ctx, nil, []Action{spendAction, dest1}, nil, time.Now().Add(time.Minute))
	if err != nil {
		testutil.FatalErr(t, err)
	}
	firstTemplate.AllowAdditional = true
	assettest.SignTxTemplate(t, ctx, firstTemplate, &info.privKeyAccounts)
	tx := bc.NewTx(*firstTemplate.Transaction)
	err = FinalizeTx(ctx, c, tx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// Build the second tx
	secondTemplate, err := Build(ctx, &tx.TxData, nil, []byte("test"), time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	secondTemplate.SigningInstructions = firstTemplate.SigningInstructions
	secondTemplate.SigningInstructions[0].WitnessComponents[0].(*SignatureWitness).Sigs[0] = nil

	assettest.SignTxTemplate(t, ctx, secondTemplate, &info.privKeyAccounts)
	err = FinalizeTx(ctx, c, bc.NewTx(*secondTemplate.Transaction))
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// Make a block, which should reject one of the txs.
	dumpState(ctx, t)
	b := prottest.MakeBlock(ctx, t, c)

	dumpState(ctx, t)
	if len(b.Transactions) != 1 {
		t.Errorf("got block.Transactions = %#v\n, want exactly one tx", b.Transactions)
	}
}

func TestTransferConfirmed(t *testing.T) {
	dbtx := pgtest.NewTx(t)
	ctx := pg.NewContext(context.Background(), dbtx)

	info, c, err := bootdb(ctx, t)
	if err != nil {
		t.Fatal(err)
	}

	_, err = issue(ctx, t, c, info, info.acctA.ID, 10)
	if err != nil {
		t.Fatal(err)
	}

	dumpState(ctx, t)
	prottest.MakeBlock(ctx, t, c)
	dumpState(ctx, t)

	_, err = transfer(ctx, t, c, info, info.acctA.ID, info.acctB.ID, 10)
	if err != nil {
		testutil.FatalErr(t, err)
	}
}

func BenchmarkTransferWithBlocks(b *testing.B) {
	dbtx := pgtest.NewTx(b)
	ctx := pg.NewContext(context.Background(), dbtx)
	info, c, err := bootdb(ctx, b)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		tx, err := issue(ctx, b, c, info, info.acctA.ID, 10)
		if err != nil {
			b.Fatal(err)
		}
		b.Logf("finalized %v", tx.Hash)

		tx, err = transfer(ctx, b, c, info, info.acctA.ID, info.acctB.ID, 10)
		if err != nil {
			b.Fatal(err)
		}
		b.Logf("finalized %v", tx.Hash)

		if i%10 == 0 {
			prottest.MakeBlock(ctx, b, c)
		}
	}
}

func dumpState(ctx context.Context, t *testing.T) {
	t.Log("pool")
	dumpTab(ctx, t, `
		SELECT tx_hash, data FROM pool_txs
	`)
	t.Log("blockchain")
	dumpBlocks(ctx, t)
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

func dumpBlocks(ctx context.Context, t *testing.T) {
	rows, err := pg.Query(ctx, `SELECT height, block_hash FROM blocks`)
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
	b.StopTimer()
	dbtx := pgtest.NewTx(b)
	ctx := pg.NewContext(context.Background(), dbtx)
	pgtest.Exec(ctx, b, `
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
	_, c, err := bootdb(ctx, b)
	if err != nil {
		testutil.FatalErr(b, err)
	}

	initialBlock, err := c.GetBlock(ctx, 1)
	if err != nil {
		testutil.FatalErr(b, err)
	}

	now := time.Now()
	b.StartTimer()
	_, _, err = c.GenerateBlock(ctx, initialBlock, state.Empty(), now)
	b.StopTimer()
	if err != nil {
		b.Fatal(err)
	}
}

type clientInfo struct {
	asset           *asset.Asset
	acctA           *account.Account
	acctB           *account.Account
	privKeyAsset    chainkd.XPrv
	privKeyAccounts chainkd.XPrv
}

// TODO(kr): refactor this into new package core/coreutil
// and consume it from cmd/corectl.
func bootdb(ctx context.Context, t testing.TB) (*clientInfo, *protocol.Chain, error) {
	c := prottest.NewChain(t)
	asset.Init(c, nil)
	account.Init(c, nil)

	accPriv, accPub, err := chainkd.NewXKeys(nil)
	if err != nil {
		return nil, nil, err
	}

	acctA, err := account.Create(ctx, []string{accPub.String()}, 1, "", nil, nil)
	if err != nil {
		return nil, nil, err
	}

	acctB, err := account.Create(ctx, []string{accPub.String()}, 1, "", nil, nil)
	if err != nil {
		return nil, nil, err
	}

	initialBlock, err := c.GetBlock(ctx, 1)
	if err != nil {
		return nil, nil, err
	}

	assetPriv, assetPub, err := chainkd.NewXKeys(nil)
	if err != nil {
		return nil, nil, err
	}

	asset, err := asset.Define(ctx, []string{assetPub.String()}, 1, nil, initialBlock.Hash(), "", nil, nil)
	if err != nil {
		return nil, nil, err
	}

	info := &clientInfo{
		asset:           asset,
		acctA:           acctA,
		acctB:           acctB,
		privKeyAsset:    assetPriv,
		privKeyAccounts: accPriv,
	}
	return info, c, nil
}

func issue(ctx context.Context, t testing.TB, c *protocol.Chain, info *clientInfo, destAcctID string, amount uint64) (*bc.Tx, error) {
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
		time.Now().Add(time.Minute),
	)
	if err != nil {
		return nil, err
	}
	assettest.SignTxTemplate(t, ctx, issueTx, &info.privKeyAsset)
	tx := bc.NewTx(*issueTx.Transaction)
	return tx, FinalizeTx(ctx, c, tx)
}

func transfer(ctx context.Context, t testing.TB, c *protocol.Chain, info *clientInfo, srcAcctID, destAcctID string, amount uint64) (*bc.Tx, error) {
	assetAmount := bc.AssetAmount{
		AssetID: info.asset.AssetID,
		Amount:  amount,
	}
	source := assettest.NewAccountSpendAction(assetAmount, srcAcctID, nil, nil, nil)
	dest := assettest.NewAccountControlAction(assetAmount, destAcctID, nil)

	xferTx, err := Build(ctx, nil, []Action{source, dest}, []byte{}, time.Now().Add(time.Minute))
	if err != nil {
		return nil, errors.Wrap(err)
	}

	assettest.SignTxTemplate(t, ctx, xferTx, &info.privKeyAccounts)

	tx := bc.NewTx(*xferTx.Transaction)
	err = FinalizeTx(ctx, c, tx)
	return tx, errors.Wrap(err)
}

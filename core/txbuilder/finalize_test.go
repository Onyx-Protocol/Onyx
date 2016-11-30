package txbuilder_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/coretest"
	"chain/core/generator"
	"chain/core/pb"
	"chain/core/pin"
	"chain/core/query"
	. "chain/core/txbuilder"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/database/sql"
	"chain/errors"
	"chain/protocol"
	"chain/protocol/bc"
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

	if err != nil {
		t.Fatal(err)
	}
	g := generator.New(info.Chain, nil, db)
	_, err = issue(ctx, t, info, g, info.acctA, 10)
	if err != nil {
		t.Fatal(err)
	}
	_, err = issue(ctx, t, info, g, info.acctB, 10)
	if err != nil {
		t.Fatal(err)
	}

	prottest.MakeBlock(t, info.Chain, g.PendingTxs())
	<-info.pinStore.PinWaiter(account.PinName, info.Chain.Height())

	assetAmount := bc.AssetAmount{
		AssetID: info.asset,
		Amount:  1,
	}
	spendAction1 := info.NewSpendAction(assetAmount, info.acctA, nil, nil)
	controlAction1 := info.NewControlAction(assetAmount, info.acctB, nil)

	tpl1, err := Build(ctx, nil, []Action{spendAction1, controlAction1}, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	tpl1.AllowAdditionalActions = true
	coretest.SignTxTemplate(t, ctx, tpl1, nil)

	txdata1, err := bc.NewTxDataFromBytes(tpl1.RawTransaction)
	if err != nil {
		t.Fatal(err)
	}

	err = CheckTxSighashCommitment(bc.NewTx(*txdata1))
	if err == nil {
		t.Error("unexpected success from checkTxSighashCommitment")
	}

	spendAction2a := info.NewSpendAction(assetAmount, info.acctB, nil, nil)
	controlAction2 := info.NewControlAction(assetAmount, info.acctA, nil)

	tpl2a, err := Build(ctx, txdata1, []Action{spendAction2a, controlAction2}, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	coretest.SignTxTemplate(t, ctx, tpl2a, nil)

	txdata2a, err := bc.NewTxDataFromBytes(tpl2a.RawTransaction)
	if err != nil {
		t.Fatal(err)
	}
	err = CheckTxSighashCommitment(bc.NewTx(*txdata2a))
	if err != nil {
		t.Errorf("unexpected failure from checkTxSighashCommitment (case 1): %v", err)
	}

	issueAction2b := info.NewIssueAction(assetAmount, nil)
	tpl2b, err := Build(ctx, txdata1, []Action{issueAction2b, controlAction2}, time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	coretest.SignTxTemplate(t, ctx, tpl2b, nil)
	txdata2b, err := bc.NewTxDataFromBytes(tpl2a.RawTransaction)
	if err != nil {
		t.Fatal(err)
	}
	err = CheckTxSighashCommitment(bc.NewTx(*txdata2b))
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

	g := generator.New(info.Chain, nil, db)
	_, err = issue(ctx, t, info, g, info.acctA, 10)
	if err != nil {
		t.Fatal(err)
	}

	dumpBlocks(ctx, t, db)
	prottest.MakeBlock(t, info.Chain, g.PendingTxs())
	dumpBlocks(ctx, t, db)
	<-info.pinStore.PinWaiter(account.PinName, info.Chain.Height())

	assetAmount := bc.AssetAmount{
		AssetID: info.asset,
		Amount:  10,
	}
	spendAction := info.NewSpendAction(assetAmount, info.acctA, nil, nil)
	dest1 := info.NewControlAction(assetAmount, info.acctB, nil)

	// Build the first tx
	firstTemplate, err := Build(ctx, nil, []Action{spendAction, dest1}, time.Now().Add(time.Minute))
	if err != nil {
		testutil.FatalErr(t, err)
	}
	unsignedTx, err := bc.NewTxDataFromBytes(firstTemplate.RawTransaction)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	coretest.SignTxTemplate(t, ctx, firstTemplate, nil)

	signedTx, err := bc.NewTxDataFromBytes(firstTemplate.RawTransaction)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	tx := bc.NewTx(*signedTx)
	err = FinalizeTx(ctx, info.Chain, g, tx)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	var buf bytes.Buffer
	_, err = unsignedTx.WriteTo(&buf)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// Slightly tweak the first tx so it has a different hash, but
	// still consumes the same UTXOs.
	unsignedTx.MaxTime++
	secondTemplate := &pb.TxTemplate{
		RawTransaction:      buf.Bytes(),
		SigningInstructions: firstTemplate.SigningInstructions,
		Local:               true,
	}
	secondTemplate.SigningInstructions[0].WitnessComponents[0].GetSignature().Program = nil
	secondTemplate.SigningInstructions[0].WitnessComponents[0].GetSignature().Signatures = nil
	coretest.SignTxTemplate(t, ctx, secondTemplate, nil)
	signedTx2, err := bc.NewTxDataFromBytes(secondTemplate.RawTransaction)
	err = FinalizeTx(ctx, info.Chain, g, bc.NewTx(*signedTx2))
	if err != nil {
		testutil.FatalErr(t, err)
	}

	// Make a block, which should reject one of the txs.
	dumpBlocks(ctx, t, db)
	b := prottest.MakeBlock(t, info.Chain, g.PendingTxs())
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

	g := generator.New(info.Chain, nil, db)
	_, err = issue(ctx, t, info, g, info.acctA, 10)
	if err != nil {
		t.Fatal(err)
	}

	dumpBlocks(ctx, t, db)
	prottest.MakeBlock(t, info.Chain, g.PendingTxs())
	dumpBlocks(ctx, t, db)

	<-info.pinStore.PinWaiter(account.PinName, info.Chain.Height())

	_, err = transfer(ctx, t, info, g, info.acctA, info.acctB, 10)
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

	g := generator.New(info.Chain, nil, db)
	for i := 0; i < b.N; i++ {
		tx, err := issue(ctx, b, info, g, info.acctA, 10)
		if err != nil {
			b.Fatal(err)
		}
		b.Logf("finalized %v", tx.Hash)
		prottest.MakeBlock(b, info.Chain, g.PendingTxs())
		<-info.pinStore.PinWaiter(account.PinName, info.Chain.Height())

		tx, err = transfer(ctx, b, info, g, info.acctA, info.acctB, 10)
		if err != nil {
			b.Fatal(err)
		}
		b.Logf("finalized %v", tx.Hash)

		if i%10 == 0 {
			prottest.MakeBlock(b, info.Chain, g.PendingTxs())
		}
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
	const tx1hex = ("07" + // serflags
		"01" + // transaction version
		"0a" + // common fields extensible string length
		"b0bbdcc705" + // common fields, mintime
		"ffbfdcc705" + // common fields, maxtime
		"00" + // common witness extensible string length
		"01" + // inputs count
		"01" + // input 0, asset version
		"4c" + // input 0, input commitment length prefix
		"01" + // input 0, input commitment, "spend" type
		"dd385f6fe25d91d8c1bd0fa58951ad56b0c5229dcc01f61d9f9e8b9eb92d3292" + // input 0, spend input commitment, outpoint tx hash
		"00" + // input 0, spend input commitment, outpoint index
		"29" + // input 0, spend input commitment, output commitment length prefix
		"0000000000000000000000000000000000000000000000000000000000000000" + // input 0, spend input commitment, output commitment, asset id
		"80a094a58d1d" + // input 0, spend input commitment, output commitment, amount
		"01" + // input 0, spend input commitment, output commitment, vm version
		"0101" + // input 0, spend input commitment, output commitment, control program
		"05696e707574" + // input 0, reference data
		"01" + // input 0, input witness length prefix
		"00" + // input 0, input witness, number of args
		"02" + // outputs count
		"01" + // output 0, asset version
		"29" + // output 0, output commitment length
		"9ed3e85a8c2d3717b5c94bd2db2ab9cab56955b2c4fb4696f345ca97aaab82d6" + // output 0, output commitment, asset id
		"80e0a596bb11" + // output 0, output commitment, amount
		"01" + // output 0, output commitment, vm version
		"0101" + // output 0, output commitment, control program
		"00" + // output 0, reference data
		"00" + // output 0, output witness
		"01" + // output 1, asset version
		"29" + // output 1, output commitment length
		"9ed3e85a8c2d3717b5c94bd2db2ab9cab56955b2c4fb4696f345ca97aaab82d6" + // output 1, output commitment, asset id
		"80c0ee8ed20b" + // output 1, output commitment, amount
		"01" + // output 1, vm version
		"0102" + // output 1, output commitment, control program
		"00" + // output 1, reference data
		"00" + // output 1, output witness
		"0c646973747269627574696f6e") // reference data

	const tx2hex = ("07" + // serflags
		"01" + // transaction version
		"02" + // common fields extensible string length
		"00" + // common fields, mintime
		"00" + // common fields, maxtime
		"00" + // common witness extensible string length
		"00" + // inputs count
		"00" + // outputs count
		"00") // reference data

	b.StopTimer()

	ctx := context.Background()
	c := prottest.NewChainWithStorage(b, memstore.New())
	g := generator.New(c, nil, pgtest.NewTx(b))
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
	err = g.Submit(ctx, &tx1)
	if err != nil {
		b.Fatal(err)
	}
	err = g.Submit(ctx, &tx2)
	if err != nil {
		b.Fatal(err)
	}

	now := time.Now()
	b.StartTimer()
	_, _, err = c.GenerateBlock(ctx, initialBlock, state.Empty(), now, g.PendingTxs())
	b.StopTimer()
	if err != nil {
		b.Fatal(err)
	}
}

type testInfo struct {
	*asset.Registry
	*account.Manager
	*protocol.Chain
	pinStore *pin.Store
	asset    bc.AssetID
	acctA    string
	acctB    string
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

	acctA := coretest.CreateAccount(ctx, t, accounts, "", nil)
	acctB := coretest.CreateAccount(ctx, t, accounts, "", nil)

	asset := coretest.CreateAsset(ctx, t, assets, nil, "", nil)

	info := &testInfo{
		Chain:    c,
		Registry: assets,
		Manager:  accounts,
		pinStore: pinStore,
		asset:    asset,
		acctA:    acctA,
		acctB:    acctB,
	}
	return info, nil
}

func issue(ctx context.Context, t testing.TB, info *testInfo, s Submitter, destAcctID string, amount uint64) (*bc.Tx, error) {
	assetAmount := bc.AssetAmount{
		AssetID: info.asset,
		Amount:  amount,
	}
	issueTx, err := Build(ctx, nil, []Action{
		info.Registry.NewIssueAction(assetAmount, nil),
		info.Manager.NewControlAction(assetAmount, destAcctID, nil),
	}, time.Now().Add(time.Minute))
	if err != nil {
		return nil, err
	}
	coretest.SignTxTemplate(t, ctx, issueTx, nil)

	txdata, err := bc.NewTxDataFromBytes(issueTx.RawTransaction)
	if err != nil {
		return nil, err
	}

	tx := bc.NewTx(*txdata)
	return tx, FinalizeTx(ctx, info.Chain, s, tx)
}

func transfer(ctx context.Context, t testing.TB, info *testInfo, s Submitter, srcAcctID, destAcctID string, amount uint64) (*bc.Tx, error) {
	assetAmount := bc.AssetAmount{
		AssetID: info.asset,
		Amount:  amount,
	}
	source := info.NewSpendAction(assetAmount, srcAcctID, nil, nil)
	dest := info.NewControlAction(assetAmount, destAcctID, nil)

	xferTx, err := Build(ctx, nil, []Action{source, dest}, time.Now().Add(time.Minute))
	if err != nil {
		return nil, errors.Wrap(err)
	}

	coretest.SignTxTemplate(t, ctx, xferTx, nil)
	txdata, err := bc.NewTxDataFromBytes(xferTx.RawTransaction)
	if err != nil {
		return nil, err
	}

	tx := bc.NewTx(*txdata)
	err = FinalizeTx(ctx, info.Chain, s, tx)
	return tx, errors.Wrap(err)
}

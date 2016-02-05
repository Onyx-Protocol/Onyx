package appdb_test

import (
	"encoding/hex"
	"encoding/json"
	"reflect"
	"sort"
	"testing"
	"time"

	"golang.org/x/net/context"

	. "chain/api/appdb"
	"chain/api/asset"
	"chain/api/asset/assettest"
	"chain/api/txbuilder"
	"chain/api/txdb"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
	"chain/fedchain/bc"
	"chain/metrics"
	"chain/testutil"
)

// Copied from $CHAIN/api/asset/utxodb.go.
// Delete this and use fedchain.ApplyTx when it exists.
func applyTx(ctx context.Context, tx *bc.Tx, outRecs []txbuilder.Receiver) error {
	defer metrics.RecordElapsed(time.Now())

	_ = pg.FromContext(ctx).(pg.Tx) // panics if not in a db transaction

	err := txdb.InsertTx(ctx, tx)
	if err != nil {
		return errors.Wrap(err, "insert into txs")
	}

	err = txdb.InsertPoolTx(ctx, tx)
	if err != nil {
		return errors.Wrap(err, "insert into pool_txs")
	}

	_, err = insertUTXOs(ctx, tx.Hash, tx.Outputs, outRecs)
	if err != nil {
		return errors.Wrap(err, "insert outputs")
	}

	var deleted []bc.Outpoint
	for _, in := range tx.Inputs {
		if in.IsIssuance() {
			continue
		}
		deleted = append(deleted, in.Previous)
	}
	err = txdb.InsertPoolInputs(ctx, deleted)
	if err != nil {
		return errors.Wrap(err, "delete")
	}

	return err
}

// Copied from $CHAIN/api/asset/utxodb.go.
// Delete this when we delete applyTx above.
func insertUTXOs(ctx context.Context, hash bc.Hash, txouts []*bc.TxOutput, receivers []txbuilder.Receiver) (inserted []*txdb.Output, err error) {
	if len(txouts) != len(receivers) {
		return nil, errors.New("length mismatch")
	}
	defer metrics.RecordElapsed(time.Now())

	var utxoInserters []txbuilder.UTXOInserter

	for i, txOutput := range txouts {
		receiver := receivers[i]
		outpoint := &bc.Outpoint{
			Hash:  hash,
			Index: uint32(i),
		}
		utxoInserters, err = receiver.AccumulateUTXO(ctx, outpoint, txOutput, utxoInserters)
		if err != nil {
			return nil, errors.Wrap(err, "accumulate utxo inserter")
		}
	}

	for _, utxoInserter := range utxoInserters {
		theseInsertions, err := utxoInserter.InsertUTXOs(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "insert utxos")
		}
		inserted = append(inserted, theseInsertions...)
	}

	return inserted, nil
}

func TestGetActUTXOs(t *testing.T) {
	ctx := assettest.NewContextWithGenesisBlock(t)
	defer pgtest.Finish(ctx)
	mn0 := assettest.CreateManagerNodeFixture(ctx, t, "", "manager-0", nil, nil)
	mn2 := assettest.CreateManagerNodeFixture(ctx, t, "", "manager-2", nil, nil)
	mn3 := assettest.CreateManagerNodeFixture(ctx, t, "", "manager-3", nil, nil)
	mn4 := assettest.CreateManagerNodeFixture(ctx, t, "", "manager-4", nil, nil)
	acc0 := assettest.CreateAccountFixture(ctx, t, mn0, "account-0", nil)
	acc1 := assettest.CreateAccountFixture(ctx, t, mn0, "account-1", nil)
	acc2 := assettest.CreateAccountFixture(ctx, t, mn2, "account-2", nil)
	acc3 := assettest.CreateAccountFixture(ctx, t, mn3, "account-3", nil)
	acc4 := assettest.CreateAccountFixture(ctx, t, mn4, "account-4", nil)

	asset0 := assettest.CreateAssetFixture(ctx, t, "", "asset-0", "")
	asset1 := assettest.CreateAssetFixture(ctx, t, "", "asset-1", "")

	out0 := assettest.IssueAssetsFixture(ctx, t, asset0, 1, acc0)
	out1 := assettest.IssueAssetsFixture(ctx, t, asset0, 2, acc1)
	asset.MakeBlock(ctx, asset.BlockKey)

	out2 := assettest.IssueAssetsFixture(ctx, t, asset1, 3, acc2)
	dest0 := assettest.AccountDestinationFixture(ctx, t, asset0, 3, acc3)
	dest1 := assettest.AccountDestinationFixture(ctx, t, asset1, 3, acc4)

	tx := bc.NewTx(bc.TxData{
		Inputs: []*bc.TxInput{
			{Previous: out0.Outpoint},
			{Previous: out1.Outpoint},
			{Previous: out2.Outpoint},
		},
		Outputs: []*bc.TxOutput{
			{
				AssetAmount: bc.AssetAmount{AssetID: asset0, Amount: 3},
				Script:      dest0.Receiver.PKScript(),
			},
			{
				AssetAmount: bc.AssetAmount{AssetID: asset1, Amount: 3},
				Script:      dest1.Receiver.PKScript(),
			},
		},
	})

	err := applyTx(ctx, tx, []txbuilder.Receiver{dest0.Receiver, dest1.Receiver})
	if err != nil {
		testutil.FatalErr(t, err)
	}

	gotIns, gotOuts, err := GetActUTXOs(ctx, tx)
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	wantIns := []*ActUTXO{
		{
			AssetID:       asset0.String(),
			Amount:        1,
			ManagerNodeID: mn0,
			AccountID:     acc0,
			Script:        out0.Script,
		},
		{
			AssetID:       asset0.String(),
			Amount:        2,
			ManagerNodeID: mn0,
			AccountID:     acc1,
			Script:        out1.Script,
		},
		{
			AssetID:       asset1.String(),
			Amount:        3,
			ManagerNodeID: mn2,
			AccountID:     acc2,
			Script:        out2.Script,
		},
	}

	wantOuts := []*ActUTXO{
		{
			AssetID:       asset0.String(),
			Amount:        3,
			ManagerNodeID: mn3,
			AccountID:     acc3,
			Script:        dest0.Receiver.PKScript(),
		},
		{
			AssetID:       asset1.String(),
			Amount:        3,
			ManagerNodeID: mn4,
			AccountID:     acc4,
			Script:        dest1.Receiver.PKScript(),
		},
	}

	if !reflect.DeepEqual(gotIns, wantIns) {
		t.Errorf("inputs:\ngot:  %v\nwant: %v", gotIns, wantIns)
	}

	if !reflect.DeepEqual(gotOuts, wantOuts) {
		t.Errorf("outputs:\ngot:  %v\nwant: %v", gotOuts, wantOuts)
	}
}

func TestGetActUTXOsIssuance(t *testing.T) {
	ctx := assettest.NewContextWithGenesisBlock(t)
	defer pgtest.Finish(ctx)

	mn0 := assettest.CreateManagerNodeFixture(ctx, t, "", "manager-0", nil, nil)
	acc0 := assettest.CreateAccountFixture(ctx, t, mn0, "account-0", nil)
	asset0 := assettest.CreateAssetFixture(ctx, t, "", "asset-0", "")

	dest0 := assettest.AccountDestinationFixture(ctx, t, asset0, 1, acc0)

	tx := bc.NewTx(bc.TxData{
		Inputs: []*bc.TxInput{{Previous: bc.Outpoint{Index: bc.InvalidOutputIndex}}},
		Outputs: []*bc.TxOutput{{
			AssetAmount: bc.AssetAmount{AssetID: asset0, Amount: 1},
			Script:      dest0.Receiver.PKScript(),
		}},
	})

	err := applyTx(ctx, tx, []txbuilder.Receiver{dest0.Receiver})
	if err != nil {
		testutil.FatalErr(t, err)
	}

	gotIns, gotOuts, err := GetActUTXOs(ctx, tx)
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	var wantIns []*ActUTXO
	wantOuts := []*ActUTXO{
		{
			AssetID:       asset0.String(),
			Amount:        1,
			ManagerNodeID: mn0,
			AccountID:     acc0,
			Script:        dest0.Receiver.PKScript(),
		},
	}

	if !reflect.DeepEqual(gotIns, wantIns) {
		t.Errorf("inputs:\ngot:  %v\nwant: %v", gotIns, wantIns)
	}

	if !reflect.DeepEqual(gotOuts, wantOuts) {
		t.Errorf("outputs:\ngot:  %v\nwant: %v", gotOuts, wantOuts)
	}
}

func TestGetActAssets(t *testing.T) {
	ctx := assettest.NewContextWithGenesisBlock(t)
	defer pgtest.Finish(ctx)

	proj0 := assettest.CreateProjectFixture(ctx, t, "", "proj0")
	in0 := assettest.CreateIssuerNodeFixture(ctx, t, proj0, "in-0", nil, nil)
	in1 := assettest.CreateIssuerNodeFixture(ctx, t, proj0, "in-1", nil, nil)
	asset0 := assettest.CreateAssetFixture(ctx, t, in0, "asset-0", "")
	asset1 := assettest.CreateAssetFixture(ctx, t, in0, "asset-1", "")
	asset2 := assettest.CreateAssetFixture(ctx, t, in1, "asset-2", "")

	examples := []struct {
		assetIDs []string
		want     []*ActAsset
	}{
		{
			[]string{asset0.String(), asset2.String()},
			[]*ActAsset{
				{ID: asset0.String(), Label: "asset-0", IssuerNodeID: in0, ProjID: proj0},
				{ID: asset2.String(), Label: "asset-2", IssuerNodeID: in1, ProjID: proj0},
			},
		},
		{
			[]string{asset1.String()},
			[]*ActAsset{
				{ID: asset1.String(), Label: "asset-1", IssuerNodeID: in0, ProjID: proj0},
			},
		},
	}

	for _, ex := range examples {
		t.Log("Example:", ex.assetIDs)

		got, err := GetActAssets(ctx, ex.assetIDs)
		if err != nil {
			t.Fatal("unexpected error", err)
		}

		sort.Sort(byAssetID(got))
		sort.Sort(byAssetID(ex.want))

		if !reflect.DeepEqual(got, ex.want) {
			t.Errorf("assets:\ngot:  %v\nwant: %v", got, ex.want)
			t.Log("got:")
			for _, a := range got {
				t.Log(a)
			}
			t.Log("want:")
			for _, a := range ex.want {
				t.Log(a)
			}
		}
	}
}

type byAssetID []*ActAsset

func (a byAssetID) Len() int           { return len(a) }
func (a byAssetID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byAssetID) Less(i, j int) bool { return a[i].ID < a[j].ID }

func TestGetActAccounts(t *testing.T) {
	ctx := assettest.NewContextWithGenesisBlock(t)
	defer pgtest.Finish(ctx)

	proj0 := assettest.CreateProjectFixture(ctx, t, "", "proj0")
	mn0 := assettest.CreateManagerNodeFixture(ctx, t, proj0, "in-0", nil, nil)
	mn1 := assettest.CreateManagerNodeFixture(ctx, t, proj0, "in-1", nil, nil)
	acc0 := assettest.CreateAccountFixture(ctx, t, mn0, "asset-0", nil)
	acc1 := assettest.CreateAccountFixture(ctx, t, mn0, "asset-1", nil)
	acc2 := assettest.CreateAccountFixture(ctx, t, mn1, "asset-2", nil)

	examples := []struct {
		accountIDs []string
		want       []*ActAccount
	}{
		{
			[]string{acc0, acc2},
			[]*ActAccount{
				{ID: acc0, Label: "asset-0", ManagerNodeID: mn0, ProjID: proj0},
				{ID: acc2, Label: "asset-2", ManagerNodeID: mn1, ProjID: proj0},
			},
		},
		{
			[]string{acc1},
			[]*ActAccount{
				{ID: acc1, Label: "asset-1", ManagerNodeID: mn0, ProjID: proj0},
			},
		},
	}

	for _, ex := range examples {
		t.Log("Example:", ex.accountIDs)

		got, err := GetActAccounts(ctx, ex.accountIDs)
		if err != nil {
			t.Fatal("unexpected error", err)
		}

		if !reflect.DeepEqual(got, ex.want) {
			t.Errorf("accounts:\ngot:  %v\nwant: %v", got, ex.want)
		}
	}
}

func mustHashFromStr(s string) bc.Hash {
	h, err := bc.ParseHash(s)
	if err != nil {
		panic(err)
	}
	return h
}

func stringsToRawJSON(strs ...string) []*json.RawMessage {
	var res []*json.RawMessage
	for _, s := range strs {
		b := json.RawMessage([]byte(s))
		res = append(res, &b)
	}
	return res
}

func withStack(err error) string {
	s := err.Error()
	for _, frame := range errors.Stack(err) {
		s += "\n" + frame.String()
	}
	return s
}

func mustDecodeHex(str string) []byte {
	bytes, err := hex.DecodeString(str)
	if err != nil {
		panic(err)
	}
	return bytes
}

package issuer

import (
	"encoding/hex"
	"reflect"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"golang.org/x/net/context"

	"chain/core/asset"
	"chain/core/txbuilder"
	"chain/cos"
	"chain/cos/bc"
	"chain/cos/mempool"
	"chain/cos/memstore"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/testutil"
)

func TestIssue(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	store := memstore.New()
	fc, err := cos.NewFC(ctx, store, mempool.New(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Unix(233400000, 0)
	b, err := fc.UpsertGenesisBlock(ctx, nil, 0, now)
	if err != nil {
		t.Fatal(err)
	}

	assetID := createAssetFixture(ctx, t, []string{testutil.TestXPub.String()}, 1, nil, b.Hash())
	amount := uint64(123)
	assetAmount := bc.AssetAmount{AssetID: assetID, Amount: amount}
	outScript := mustDecodeHex("a9140ac9c982fd389181752e5a414045dd424a10754b87")
	dest := txbuilder.NewScriptDestination(ctx, &assetAmount, outScript, nil)
	resp, err := Issue(ctx, assetAmount, []*txbuilder.Destination{dest})
	if err != nil {
		t.Fatal(err)
	}

	ic := resp.Unsigned.Inputs[0].InputCommitment.(*bc.IssuanceInputCommitment)

	assetObj, err := asset.Find(ctx, assetID)
	if err != nil {
		t.Fatal(err)
	}

	minTime := time.Unix(0, int64(ic.MinTimeMS)*int64(time.Millisecond))
	maxTime := time.Unix(0, int64(ic.MaxTimeMS)*int64(time.Millisecond))
	want := &bc.TxData{
		Version: 1,
		Inputs: []*bc.TxInput{
			bc.NewIssuanceInput(minTime, maxTime, b.Hash(), amount, assetObj.IssuanceProgram, nil, nil, nil),
		},
		Outputs: []*bc.TxOutput{
			bc.NewTxOutput(assetID, amount, outScript, nil),
		},
	}
	if !reflect.DeepEqual(resp.Unsigned, want) {
		t.Errorf("got tx:\n%s\nwant tx:\n%s", spew.Sdump(resp.Unsigned), spew.Sdump(want))
	}
}

func mustDecodeHex(str string) []byte {
	d, err := hex.DecodeString(str)
	if err != nil {
		panic(err)
	}
	return d
}

func createAssetFixture(ctx context.Context, t testing.TB, keys []string, quorum int, def map[string]interface{}, genesisHash bc.Hash) bc.AssetID {
	if quorum == 0 {
		quorum = len(keys)
	}

	asset, err := asset.Define(ctx, keys, quorum, def, genesisHash, nil) // xpubs []string, quorum int, definition map[string]interface{}, genesisHash bc.Hash, clientToken *string
	if err != nil {
		testutil.FatalErr(t, err)
	}

	return asset.AssetID
}

package txbuilder

import (
	"encoding/hex"
	"reflect"
	"testing"
	"time"

	"golang.org/x/net/context"

	"chain/core/txdb"
	"chain/cos/bc"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/database/sql"
	"chain/errors"
	"chain/testutil"
)

type testRecv struct {
	script []byte
}

func (tr *testRecv) PKScript() []byte { return tr.script }

type testReserver struct{}

func (tr *testReserver) Reserve(ctx context.Context, assetAmt *bc.AssetAmount, ttl time.Duration) (*ReserveResult, error) {
	return &ReserveResult{
		Items: []*ReserveResultItem{{
			TxInput: &bc.TxInput{
				Previous:    bc.Outpoint{Hash: [32]byte{255}, Index: 0},
				AssetAmount: *assetAmt,
				PrevScript:  []byte{},
			},
			TemplateInput: &Input{
				SigScriptSuffix: []byte("redeem"),
			},
		}},
		Change: []*Destination{{
			AssetAmount: *assetAmt,
			Receiver:    &testRecv{script: []byte("change")},
		}},
	}, nil
}

func TestBuild(t *testing.T) {
	ctx := pgtest.NewContext(t)
	store := txdb.NewStore(pg.FromContext(ctx).(*sql.DB))

	err := store.ApplyTx(ctx, &bc.Tx{Hash: [32]byte{255}, TxData: bc.TxData{
		Outputs: []*bc.TxOutput{{
			AssetAmount: bc.AssetAmount{AssetID: [32]byte{1}, Amount: 5},
			Script:      []byte{},
		}},
	}}, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	sources := []*Source{{
		AssetAmount: bc.AssetAmount{AssetID: [32]byte{1}, Amount: 5},
		Reserver:    &testReserver{},
	}}
	dests := []*Destination{{
		AssetAmount: bc.AssetAmount{AssetID: [32]byte{2}, Amount: 6},
		Receiver:    &testRecv{script: []byte("dest")},
	}}

	got, err := Build(ctx, nil, sources, dests, nil, time.Minute)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	want := &Template{
		BlockChain: "sandbox",
		Unsigned: &bc.TxData{
			Version: 1,
			Inputs: []*bc.TxInput{{
				Previous:    bc.Outpoint{Hash: [32]byte{255}, Index: 0},
				AssetAmount: bc.AssetAmount{AssetID: [32]byte{1}, Amount: 5},
				PrevScript:  []byte{},
			}},
			Outputs: []*bc.TxOutput{
				{
					AssetAmount: bc.AssetAmount{AssetID: [32]byte{2}, Amount: 6},
					Script:      []byte("dest"),
				},
				{
					AssetAmount: bc.AssetAmount{AssetID: [32]byte{1}, Amount: 5},
					Script:      []byte("change"),
				},
			},
		},
		Inputs: []*Input{{
			SigScriptSuffix: []byte("redeem"),
			Sigs:            []*Signature{},
			SigComponents:   []*SigScriptComponent{},
		}},
	}

	ComputeSigHashes(ctx, want)
	if !reflect.DeepEqual(got.Unsigned, want.Unsigned) {
		t.Errorf("got tx:\n\t%#v\nwant tx:\n\t%#v", got.Unsigned, want.Unsigned)
		t.Errorf("got tx inputs:\n\t%#v\nwant tx inputs:\n\t%#v", got.Unsigned.Inputs, want.Unsigned.Inputs)
		t.Errorf("got tx outputs:\n\t%#v\nwant tx outputs:\n\t%#v", got.Unsigned.Outputs, want.Unsigned.Outputs)
	}

	if !reflect.DeepEqual(got.Inputs, want.Inputs) {
		t.Errorf("got inputs:\n\t%#v\nwant inputs:\n\t%#v", got.Inputs, want.Inputs)
	}
}

func TestCombine(t *testing.T) {
	unsigned1 := &bc.TxData{
		Version: 1,
		Inputs:  []*bc.TxInput{{Previous: bc.Outpoint{Hash: bc.Hash{}, Index: 0}}},
		Outputs: []*bc.TxOutput{{
			AssetAmount: bc.AssetAmount{AssetID: [32]byte{254}, Amount: 5},
		}},
	}

	unsigned2 := &bc.TxData{
		Version: 1,
		Inputs:  []*bc.TxInput{{Previous: bc.Outpoint{Hash: bc.Hash{}, Index: 0}}},
		Outputs: []*bc.TxOutput{{
			AssetAmount: bc.AssetAmount{AssetID: [32]byte{255}, Amount: 6},
		}},
	}

	combined := &bc.TxData{
		Version: 1,
		Inputs: []*bc.TxInput{
			{Previous: bc.Outpoint{Hash: bc.Hash{}, Index: 0}},
			{Previous: bc.Outpoint{Hash: bc.Hash{}, Index: 0}},
		},
		Outputs: []*bc.TxOutput{
			{AssetAmount: bc.AssetAmount{AssetID: [32]byte{254}, Amount: 5}},
			{AssetAmount: bc.AssetAmount{AssetID: [32]byte{255}, Amount: 6}},
		},
	}

	tpl1 := &Template{
		Unsigned:   unsigned1,
		Inputs:     []*Input{{}},
		BlockChain: "sandbox",
	}

	tpl2 := &Template{
		Unsigned:   unsigned2,
		Inputs:     []*Input{{}},
		BlockChain: "sandbox",
	}

	got, err := combine(tpl1, tpl2)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	want := &Template{
		Unsigned:   combined,
		Inputs:     []*Input{{}, {}},
		BlockChain: "sandbox",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("combine:\ngot: \t%#v\nwant:\t%#v", got, want)
	}
}

func TestAssembleSignatures(t *testing.T) {
	outscript := mustDecodeHex("a9140ac9c982fd389181752e5a414045dd424a10754b87")
	unsigned := &bc.TxData{
		Version: 1,
		Inputs:  []*bc.TxInput{{Previous: bc.Outpoint{Index: bc.InvalidOutputIndex}}},
		Outputs: []*bc.TxOutput{{
			AssetAmount: bc.AssetAmount{AssetID: [32]byte{255}, Amount: 5},
			Script:      outscript,
		}},
	}
	sigHash, _ := bc.ParseHash("78e437f627019fc270bbe9ed309291d0a5f6bf98bfae0f750538ba56646f7327")

	tpl := &Template{
		Unsigned: unsigned,
		Inputs: []*Input{{
			SignatureData: sigHash,
			Sigs: []*Signature{{
				XPub:           "xpub661MyMwAqRbcGiDB8FQvHnDAZyaGUyzm3qN1Q3NDJz1PgAWCfyi9WRCS7Z9HyM5QNEh45fMyoaBMqjfoWPdnktcN8chJYB57D2Y7QtNmadr",
				DerivationPath: []uint32{0, 0, 0, 0},
				DER:            mustDecodeHex("3044022004da5732f6c988b9e2882f5ca4f569b9525d313940e0372d6a84fef73be78f8f02204656916481dc573d771ec42923a8f5af31ae634241a4cb30ea5b359363cf064d"),
			}},
		}},
	}

	tx, err := AssembleSignatures(tpl)
	if err != nil {
		t.Fatal(withStack(err))
	}

	want := "3d8cc3226186daa9f510d47dc737378633a9005baf091d3f02827672dc895c94"
	if got := tx.WitnessHash().String(); got != want {
		t.Errorf("got tx witness hash = %v want %v", got, want)
	}
}

func withStack(err error) string {
	s := err.Error()
	for _, frame := range errors.Stack(err) {
		s += "\n" + frame.String()
	}
	return s
}

func mustDecodeHex(str string) []byte {
	data, err := hex.DecodeString(str)
	if err != nil {
		panic(err)
	}
	return data
}

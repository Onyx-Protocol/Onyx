package txbuilder

import (
	"encoding/hex"
	"reflect"
	"testing"
	"time"

	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/cos/mempool"
	"chain/database/pg"
	"chain/database/pg/pgtest"
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
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	pool := mempool.New()

	err := pool.Insert(ctx, &bc.Tx{
		Hash: [32]byte{255},
		TxData: bc.TxData{
			Outputs: []*bc.TxOutput{
				bc.NewTxOutput([32]byte{1}, 5, nil, nil),
			},
		},
	})
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
				bc.NewTxOutput([32]byte{2}, 6, []byte("dest"), nil),
				bc.NewTxOutput([32]byte{1}, 5, []byte("change"), nil),
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
		Outputs: []*bc.TxOutput{
			bc.NewTxOutput([32]byte{254}, 5, nil, nil),
		},
	}

	unsigned2 := &bc.TxData{
		Version: 1,
		Inputs:  []*bc.TxInput{{Previous: bc.Outpoint{Hash: bc.Hash{}, Index: 0}}},
		Outputs: []*bc.TxOutput{
			bc.NewTxOutput([32]byte{255}, 6, nil, nil),
		},
	}

	combined := &bc.TxData{
		Version: 1,
		Inputs: []*bc.TxInput{
			{Previous: bc.Outpoint{Hash: bc.Hash{}, Index: 0}},
			{Previous: bc.Outpoint{Hash: bc.Hash{}, Index: 0}},
		},
		Outputs: []*bc.TxOutput{
			bc.NewTxOutput([32]byte{254}, 5, nil, nil),
			bc.NewTxOutput([32]byte{255}, 6, nil, nil),
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

func TestCombineMetadata(t *testing.T) {
	cases := []struct {
		m1, m2 []byte
		want   error
	}{
		{
			m1: nil,
			m2: nil,
		},
		{
			m1: nil,
			m2: []byte("test"),
		},
		{
			m1: []byte("test"),
			m2: nil,
		},
		{
			m1:   []byte("test"),
			m2:   []byte("diff"),
			want: ErrBadBuildRequest,
		},
	}

	for _, c := range cases {
		tpl1 := &Template{Unsigned: &bc.TxData{Metadata: c.m1}}
		tpl2 := &Template{Unsigned: &bc.TxData{Metadata: c.m2}}

		_, err := combine(tpl1, tpl2)
		if errors.Root(err) != c.want {
			t.Fatalf("got err = %v want %v", errors.Root(err), c.want)
		}
	}
}

func TestAssembleSignatures(t *testing.T) {
	outscript := mustDecodeHex("76a914c5d128911c28776f56baaac550963f7b88501dc388c0")
	unsigned := &bc.TxData{
		Version: 1,
		Inputs:  []*bc.TxInput{{Previous: bc.Outpoint{Index: bc.InvalidOutputIndex}}},
		Outputs: []*bc.TxOutput{
			bc.NewTxOutput([32]byte{255}, 5, outscript, nil),
		},
	}
	sigData, _ := bc.ParseHash("b64d968745f18a5da6d5dd4ec750f7e6da5204000a9ee90ba9187ec85c25032c")

	tpl := &Template{
		Unsigned: unsigned,
		Inputs: []*Input{{
			SignatureData: sigData,
			Sigs: []*Signature{{
				XPub:           "xpub661MyMwAqRbcGZNqeB27ae2nQLWoWd9Ffx8NEXrVDFgFPe6Jdzw53p5m3ewA3K2z5nPmcJK7r1nykAwkoNHWgHr5kLCWi777ShtKwLdy55a",
				DerivationPath: []uint32{0, 0, 0, 0},
				DER:            mustDecodeHex("304402202ece2c2dfd0ca44b27c5e03658c7eaac4d61d5c2668940da1bdcf53b312db0fc0220670c520b67b6fd4f4efcfbe55e82dc4a4624059b51594889d664bea445deee6b01"),
			}},
			SigScriptSuffix: mustDecodeHex("4c695221033dda0a756db51f76a4f394161614f01df4061644c514fde3994adbe4a3a2d21621038a0f0a8d593773abcd8c878f8777c57986f9f84886c8dde0cf00fdc2c89f0c592103b9e805011523bb28eedb3fcfff8924684a91116a76408fe0972805295e50e15d53ae"),
		}},
	}

	tx, err := AssembleSignatures(tpl)
	if err != nil {
		t.Fatal(withStack(err))
	}

	want := "563350e56f596b093d0b7b8503f1cbf0854ed10956798e67992035b4b602ce98" // TODO(bobg): verify this is the right hash
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

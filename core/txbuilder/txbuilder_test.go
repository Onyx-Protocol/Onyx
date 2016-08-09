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
	"chain/encoding/json"
	"chain/errors"
	"chain/testutil"
)

type testAction bc.AssetAmount

func (t testAction) Build(ctx context.Context) ([]*bc.TxInput, []*bc.TxOutput, []*Input, error) {
	in := bc.NewSpendInput([32]byte{255}, 0, nil, t.AssetID, t.Amount, nil, nil)
	tplIn := &Input{
		SigComponents: []*SigScriptComponent{{
			Type: "data",
			Data: []byte("redeem"),
		}}}
	change := bc.NewTxOutput(t.AssetID, t.Amount, []byte("change"), nil)
	return []*bc.TxInput{in}, []*bc.TxOutput{change}, []*Input{tplIn}, nil
}

func newControlProgramAction(assetAmt bc.AssetAmount, script []byte) *ControlProgramAction {
	return &ControlProgramAction{
		Params: struct {
			bc.AssetAmount
			Program json.HexBytes `json:"control_program"`
		}{assetAmt, script},
	}
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

	actions := []Action{
		newControlProgramAction(bc.AssetAmount{AssetID: [32]byte{2}, Amount: 6}, []byte("dest")),
		testAction(bc.AssetAmount{AssetID: [32]byte{1}, Amount: 5}),
	}

	got, err := Build(ctx, nil, actions, nil)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	want := &Template{
		BlockChain: "sandbox",
		Unsigned: &bc.TxData{
			Version: 1,
			Inputs: []*bc.TxInput{
				bc.NewSpendInput([32]byte{255}, 0, nil, [32]byte{1}, 5, nil, nil),
			},
			Outputs: []*bc.TxOutput{
				bc.NewTxOutput([32]byte{2}, 6, []byte("dest"), nil),
				bc.NewTxOutput([32]byte{1}, 5, []byte("change"), nil),
			},
		},
		Inputs: []*Input{{
			SigComponents: []*SigScriptComponent{
				{
					Type: "data",
					Data: []byte("redeem"),
				},
			},
		}},
	}

	ComputeSigHashes(want)
	if !reflect.DeepEqual(got.Unsigned, want.Unsigned) {
		t.Errorf("got tx:\n\t%#v\nwant tx:\n\t%#v", got.Unsigned, want.Unsigned)
		t.Errorf("got tx inputs:\n\t%#v\nwant tx inputs:\n\t%#v", got.Unsigned.Inputs, want.Unsigned.Inputs)
		t.Errorf("got tx outputs:\n\t%#v\nwant tx outputs:\n\t%#v", got.Unsigned.Outputs, want.Unsigned.Outputs)
	}

	if !reflect.DeepEqual(got.Inputs, want.Inputs) {
		t.Errorf("got inputs:\n\t%#v\nwant inputs:\n\t%#v", got.Inputs, want.Inputs)
	}
}

func TestAssembleSignatures(t *testing.T) {
	var genesisHash bc.Hash
	issuanceProg := []byte{1}
	assetID := bc.ComputeAssetID(issuanceProg, genesisHash, 1)
	outscript := mustDecodeHex("76a914c5d128911c28776f56baaac550963f7b88501dc388c0")
	now := time.Unix(233400000, 0)
	unsigned := &bc.TxData{
		Version: 1,
		Inputs: []*bc.TxInput{
			bc.NewIssuanceInput(now, now.Add(time.Hour), genesisHash, 5, issuanceProg, nil, nil),
		},
		Outputs: []*bc.TxOutput{
			bc.NewTxOutput(assetID, 5, outscript, nil),
		},
	}
	sigData, _ := bc.ParseHash("b64d968745f18a5da6d5dd4ec750f7e6da5204000a9ee90ba9187ec85c25032c")

	tpl := &Template{
		Unsigned: unsigned,
		Inputs: []*Input{{
			SigComponents: []*SigScriptComponent{
				{
					Type:          "signature",
					Required:      2,
					SignatureData: sigData,
					Signatures: []*Signature{{
						XPub:           "xpub661MyMwAqRbcGZNqeB27ae2nQLWoWd9Ffx8NEXrVDFgFPe6Jdzw53p5m3ewA3K2z5nPmcJK7r1nykAwkoNHWgHr5kLCWi777ShtKwLdy55a",
						DerivationPath: []uint32{0, 0, 0, 0},
						Bytes:          mustDecodeHex("304402202ece2c2dfd0ca44b27c5e03658c7eaac4d61d5c2668940da1bdcf53b312db0fc0220670c520b67b6fd4f4efcfbe55e82dc4a4624059b51594889d664bea445deee6b01"),
					}},
				},
				{
					Type: "data",
					Data: mustDecodeHex("5221033dda0a756db51f76a4f394161614f01df4061644c514fde3994adbe4a3a2d21621038a0f0a8d593773abcd8c878f8777c57986f9f84886c8dde0cf00fdc2c89f0c592103b9e805011523bb28eedb3fcfff8924684a91116a76408fe0972805295e50e15d53ae"),
				},
			},
		}},
	}

	tx, err := AssembleSignatures(tpl)
	if err != nil {
		t.Fatal(withStack(err))
	}

	want := "1b30687452cf0cfbcf7ca744ebae0fe5b4173a73d91461613c48224fa18d42f9"
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

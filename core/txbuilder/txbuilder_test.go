package txbuilder

import (
	"context"
	"encoding/hex"
	"fmt"
	"reflect"
	"testing"
	"time"

	"golang.org/x/crypto/sha3"

	"chain/crypto/ed25519"
	"chain/crypto/ed25519/hd25519"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/encoding/json"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/mempool"
	"chain/protocol/vm"
	"chain/protocol/vmutil"
	"chain/testutil"
)

type testAction bc.AssetAmount

func (t testAction) Build(ctx context.Context, _ time.Time) (
	[]*bc.TxInput,
	[]*bc.TxOutput,
	[]*SigningInstruction,
	error,
) {
	in := bc.NewSpendInput([32]byte{255}, 0, nil, t.AssetID, t.Amount, nil, nil)
	tplIn := &SigningInstruction{}
	change := bc.NewTxOutput(t.AssetID, t.Amount, []byte("change"), nil)
	return []*bc.TxInput{in}, []*bc.TxOutput{change}, []*SigningInstruction{tplIn}, nil
}

func newControlProgramAction(assetAmt bc.AssetAmount, script []byte) *ControlProgramAction {
	return &ControlProgramAction{
		AssetAmount: assetAmt,
		Program:     script,
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
	expiryTime := time.Now().Add(time.Minute)
	got, err := Build(ctx, nil, actions, nil, expiryTime)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	want := &Template{
		Transaction: &bc.TxData{
			Version: 1,
			MaxTime: bc.Millis(expiryTime),
			Inputs: []*bc.TxInput{
				bc.NewSpendInput([32]byte{255}, 0, nil, [32]byte{1}, 5, nil, nil),
			},
			Outputs: []*bc.TxOutput{
				bc.NewTxOutput([32]byte{2}, 6, []byte("dest"), nil),
				bc.NewTxOutput([32]byte{1}, 5, []byte("change"), nil),
			},
		},
		SigningInstructions: []*SigningInstruction{{
			WitnessComponents: []WitnessComponent{},
		}},
	}

	if !reflect.DeepEqual(got.Transaction, want.Transaction) {
		t.Errorf("got tx:\n\t%#v\nwant tx:\n\t%#v", got.Transaction, want.Transaction)
		t.Errorf("got tx inputs:\n\t%#v\nwant tx inputs:\n\t%#v", got.Transaction.Inputs, want.Transaction.Inputs)
		t.Errorf("got tx outputs:\n\t%#v\nwant tx outputs:\n\t%#v", got.Transaction.Outputs, want.Transaction.Outputs)
	}

	if !reflect.DeepEqual(got.SigningInstructions, want.SigningInstructions) {
		t.Errorf("got signing instructions:\n\t%#v\nwant signing instructions:\n\t%#v", got.SigningInstructions, want.SigningInstructions)
	}
}

func TestMaterializeWitnesses(t *testing.T) {
	var initialBlockHash bc.Hash
	privkey, pubkey, err := hd25519.NewXKeys(nil)
	if err != nil {
		t.Fatal(err)
	}
	issuanceProg, _ := vmutil.P2DPMultiSigProgram([]ed25519.PublicKey{pubkey.Key}, 1)
	assetID := bc.ComputeAssetID(issuanceProg, initialBlockHash, 1)
	outscript := mustDecodeHex("76a914c5d128911c28776f56baaac550963f7b88501dc388c0")
	unsigned := &bc.TxData{
		Version: 1,
		Inputs: []*bc.TxInput{
			bc.NewIssuanceInput(nil, 5, nil, initialBlockHash, issuanceProg, nil),
		},
		Outputs: []*bc.TxOutput{
			bc.NewTxOutput(assetID, 5, outscript, nil),
		},
	}

	prog, err := vm.Compile(fmt.Sprintf("MAXTIME 0x804cf05736 LESSTHAN VERIFY 0 5 0x%x 1 0x76a914c5d128911c28776f56baaac550963f7b88501dc388c0 FINDOUTPUT", assetID[:]))
	h := sha3.Sum256(prog)
	sig := ed25519.Sign(privkey.Key, h[:])
	if err != nil {
		t.Fatal(err)
	}

	tpl := &Template{
		Transaction: unsigned,
		SigningInstructions: []*SigningInstruction{{
			WitnessComponents: []WitnessComponent{
				&SignatureWitness{
					Quorum: 1,
					Keys: []KeyID{{
						XPub:           pubkey.String(),
						DerivationPath: []uint32{0, 0, 0, 0},
					}},
					Program: prog,
					Sigs:    []json.HexBytes{sig},
				},
			},
		}},
	}

	want := [][]byte{
		sig,
		prog,
	}

	err = materializeWitnesses(tpl)
	if err != nil {
		t.Fatal(withStack(err))
	}

	got := tpl.Transaction.Inputs[0].Arguments()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got input witness %v, want input witness %v", got, want)
	}
}

func TestSignatureWitnessMaterialize(t *testing.T) {
	var initialBlockHash bc.Hash
	privkey1, pubkey1, err := hd25519.NewXKeys(nil)
	if err != nil {
		t.Fatal(err)
	}
	privkey2, pubkey2, err := hd25519.NewXKeys(nil)
	if err != nil {
		t.Fatal(err)
	}
	privkey3, pubkey3, err := hd25519.NewXKeys(nil)
	if err != nil {
		t.Fatal(err)
	}
	issuanceProg, _ := vmutil.P2DPMultiSigProgram([]ed25519.PublicKey{pubkey1.Key, pubkey2.Key, pubkey3.Key}, 2)
	assetID := bc.ComputeAssetID(issuanceProg, initialBlockHash, 1)
	outscript := mustDecodeHex("76a914c5d128911c28776f56baaac550963f7b88501dc388c0")
	unsigned := &bc.TxData{
		Version: 1,
		Inputs: []*bc.TxInput{
			bc.NewIssuanceInput(nil, 100, nil, initialBlockHash, issuanceProg, nil),
		},
		Outputs: []*bc.TxOutput{
			bc.NewTxOutput(assetID, 100, outscript, nil),
		},
	}

	tpl := &Template{
		Transaction: unsigned,
	}
	h := tpl.Hash(0)
	builder := vmutil.NewBuilder()
	builder.AddData(h[:])
	builder.AddOp(vm.OP_TXSIGHASH).AddOp(vm.OP_EQUAL)
	prog := builder.Program
	msg := sha3.Sum256(prog)
	sig1 := ed25519.Sign(privkey1.Key, msg[:])
	sig2 := ed25519.Sign(privkey2.Key, msg[:])
	sig3 := ed25519.Sign(privkey3.Key, msg[:])
	want := [][]byte{
		sig1,
		sig2,
		prog,
	}

	// Test with more signatures than required, in correct order
	tpl.SigningInstructions = []*SigningInstruction{{
		WitnessComponents: []WitnessComponent{
			&SignatureWitness{
				Quorum: 2,
				Keys: []KeyID{
					{
						XPub:           pubkey1.String(),
						DerivationPath: []uint32{0, 0, 0, 0},
					},
					{
						XPub:           pubkey2.String(),
						DerivationPath: []uint32{0, 0, 0, 0},
					},
					{
						XPub:           pubkey3.String(),
						DerivationPath: []uint32{0, 0, 0, 0},
					},
				},
				Program: prog,
				Sigs:    []json.HexBytes{sig1, sig2, sig3},
			},
		},
	}}
	err = materializeWitnesses(tpl)
	if err != nil {
		t.Fatal(withStack(err))
	}
	got := tpl.Transaction.Inputs[0].Arguments()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got input witness %v, want input witness %v", got, want)
	}

	// Test with more signatures than required, in incorrect order
	component, ok := tpl.SigningInstructions[0].WitnessComponents[0].(*SignatureWitness)
	if !ok {
		t.Fatal("expecting WitnessComponent of type SignatureWitness")
	}
	component.Sigs = []json.HexBytes{sig3, sig2, sig1}
	err = materializeWitnesses(tpl)
	if err != nil {
		t.Fatal(withStack(err))
	}

	got = tpl.Transaction.Inputs[0].Arguments()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got input witness %v, want input witness %v", got, want)
	}

	// Test with exact amount of signatures required, in correct order
	component.Sigs = []json.HexBytes{sig1, sig2}
	err = materializeWitnesses(tpl)
	if err != nil {
		t.Fatal(withStack(err))
	}
	got = tpl.Transaction.Inputs[0].Arguments()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got input witness %v, want input witness %v", got, want)
	}

	// Test with exact amount of signatures required, in incorrect order
	component.Sigs = []json.HexBytes{sig2, sig1}
	err = materializeWitnesses(tpl)
	if err != nil {
		t.Fatal(withStack(err))
	}
	got = tpl.Transaction.Inputs[0].Arguments()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got input witness %v, want input witness %v", got, want)
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

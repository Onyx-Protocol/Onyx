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

func (t testAction) Build(ctx context.Context) ([]*bc.TxInput, []*bc.TxOutput, []*Input, error) {
	in := bc.NewSpendInput([32]byte{255}, 0, nil, t.AssetID, t.Amount, nil, nil)
	tplIn := &Input{
		WitnessComponents: []WitnessComponent{
			DataWitness("redeem"),
		},
	}
	change := bc.NewTxOutput(t.AssetID, t.Amount, []byte("change"), nil)
	return []*bc.TxInput{in}, []*bc.TxOutput{change}, []*Input{tplIn}, nil
}

func newControlProgramAction(assetAmt bc.AssetAmount, script []byte) *ControlProgramAction {
	return &ControlProgramAction{
		Params: struct {
			bc.AssetAmount
			Program    json.HexBytes `json:"control_program"`
			AssetAlias string        `json:"asset_alias"`
		}{assetAmt, script, ""},
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
	expiryTime := bc.Millis(time.Now().Add(time.Minute))
	got, err := Build(ctx, nil, actions, nil, expiryTime)
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	want := &Template{
		Unsigned: &bc.TxData{
			Version: 1,
			MaxTime: expiryTime,
			Inputs: []*bc.TxInput{
				bc.NewSpendInput([32]byte{255}, 0, nil, [32]byte{1}, 5, nil, nil),
			},
			Outputs: []*bc.TxOutput{
				bc.NewTxOutput([32]byte{2}, 6, []byte("dest"), nil),
				bc.NewTxOutput([32]byte{1}, 5, []byte("change"), nil),
			},
		},
		Inputs: []*Input{{
			WitnessComponents: []WitnessComponent{
				DataWitness("redeem"),
			},
		}},
	}

	if !reflect.DeepEqual(got.Unsigned, want.Unsigned) {
		t.Errorf("got tx:\n\t%#v\nwant tx:\n\t%#v", got.Unsigned, want.Unsigned)
		t.Errorf("got tx inputs:\n\t%#v\nwant tx inputs:\n\t%#v", got.Unsigned.Inputs, want.Unsigned.Inputs)
		t.Errorf("got tx outputs:\n\t%#v\nwant tx outputs:\n\t%#v", got.Unsigned.Outputs, want.Unsigned.Outputs)
	}

	if !reflect.DeepEqual(got.Inputs, want.Inputs) {
		t.Errorf("got inputs:\n\t%#v\nwant inputs:\n\t%#v", got.Inputs, want.Inputs)
	}
}

func TestMaterializeWitnesses(t *testing.T) {
	var initialBlockHash bc.Hash
	privkey, pubkey, err := hd25519.NewXKeys(nil)
	if err != nil {
		t.Fatal(err)
	}
	issuanceProg := vmutil.P2DPMultiSigProgram([]ed25519.PublicKey{pubkey.Key}, 1)
	assetID := bc.ComputeAssetID(issuanceProg, initialBlockHash, 1)
	outscript := mustDecodeHex("76a914c5d128911c28776f56baaac550963f7b88501dc388c0")
	now := time.Unix(233400000, 0)
	unsigned := &bc.TxData{
		Version: 1,
		Inputs: []*bc.TxInput{
			bc.NewIssuanceInput(now, now.Add(time.Hour), initialBlockHash, 5, issuanceProg, nil, nil),
		},
		Outputs: []*bc.TxOutput{
			bc.NewTxOutput(assetID, 5, outscript, nil),
		},
	}

	witnessData := mustDecodeHex("5221033dda0a756db51f76a4f394161614f01df4061644c514fde3994adbe4a3a2d21621038a0f0a8d593773abcd8c878f8777c57986f9f84886c8dde0cf00fdc2c89f0c592103b9e805011523bb28eedb3fcfff8924684a91116a76408fe0972805295e50e15d53ae")
	prog, err := vm.Compile(fmt.Sprintf("0x804cf05736 MAXTIME LESSTHAN VERIFY 0 5 0x%x 1 0x76a914c5d128911c28776f56baaac550963f7b88501dc388c0 FINDOUTPUT", assetID[:]))
	h := sha3.Sum256(prog)
	sig := ed25519.Sign(privkey.Key, h[:])
	if err != nil {
		t.Fatal(err)
	}

	tpl := &Template{
		Unsigned: unsigned,
		Inputs: []*Input{{
			WitnessComponents: []WitnessComponent{
				&SignatureWitness{
					Quorum: 1,
					Keys: []KeyID{{
						XPub:           pubkey.String(),
						DerivationPath: []uint32{0, 0, 0, 0},
					}},
					Constraints: []Constraint{
						TTLConstraint(bc.Millis(now.Add(time.Hour))),
						&PayConstraint{
							AssetAmount: bc.AssetAmount{
								AssetID: assetID,
								Amount:  5,
							},
							Program: outscript,
						},
					},
					Sigs: []json.HexBytes{sig},
				},
				DataWitness(witnessData),
			},
		}},
	}

	want := [][]byte{
		sig,
		prog,
		witnessData,
	}

	tx, err := MaterializeWitnesses(tpl)
	if err != nil {
		t.Fatal(withStack(err))
	}

	got := tx.Inputs[0].InputWitness
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
	issuanceProg := vmutil.P2DPMultiSigProgram([]ed25519.PublicKey{pubkey1.Key, pubkey2.Key, pubkey3.Key}, 2)
	assetID := bc.ComputeAssetID(issuanceProg, initialBlockHash, 1)
	outscript := mustDecodeHex("76a914c5d128911c28776f56baaac550963f7b88501dc388c0")
	now := time.Unix(233400000, 0)
	unsigned := &bc.TxData{
		Version: 1,
		Inputs: []*bc.TxInput{
			bc.NewIssuanceInput(now, now.Add(time.Hour), initialBlockHash, 100, issuanceProg, nil, nil),
		},
		Outputs: []*bc.TxOutput{
			bc.NewTxOutput(assetID, 100, outscript, nil),
		},
	}

	tpl := &Template{
		Unsigned: unsigned,
	}
	h := tpl.Hash(0, bc.SigHashAll)
	builder := vmutil.NewBuilder()
	builder.AddData(h[:])
	builder.AddInt64(1).AddOp(vm.OP_TXSIGHASH).AddOp(vm.OP_EQUAL)
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
	tpl.Inputs = []*Input{{
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
				Constraints: []Constraint{},
				Sigs:        []json.HexBytes{sig1, sig2, sig3},
			},
		},
	}}
	tx, err := MaterializeWitnesses(tpl)
	if err != nil {
		t.Fatal(withStack(err))
	}
	got := tx.Inputs[0].InputWitness
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got input witness %v, want input witness %v", got, want)
	}

	// Test with more signatures than required, in incorrect order
	component, ok := tpl.Inputs[0].WitnessComponents[0].(*SignatureWitness)
	if !ok {
		t.Fatal("expecting WitnessComponent of type SignatureWitness")
	}
	component.Sigs = []json.HexBytes{sig3, sig2, sig1}
	tx, err = MaterializeWitnesses(tpl)
	if err != nil {
		t.Fatal(withStack(err))
	}

	got = tx.Inputs[0].InputWitness
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got input witness %v, want input witness %v", got, want)
	}

	// Test with exact amount of signatures required, in correct order
	component.Sigs = []json.HexBytes{sig1, sig2}
	tx, err = MaterializeWitnesses(tpl)
	if err != nil {
		t.Fatal(withStack(err))
	}
	got = tx.Inputs[0].InputWitness
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got input witness %v, want input witness %v", got, want)
	}

	// Test with exact amount of signatures required, in incorrect order
	component.Sigs = []json.HexBytes{sig2, sig1}
	tx, err = MaterializeWitnesses(tpl)
	if err != nil {
		t.Fatal(withStack(err))
	}
	got = tx.Inputs[0].InputWitness
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got input witness %v, want input witness %v", got, want)
	}

	// Test with insufficient amount of signatures required
	component.Sigs = []json.HexBytes{sig2}
	tx, err = MaterializeWitnesses(tpl)
	if errors.Root(err) != ErrMissingSig {
		t.Errorf("got %v, want ErrMissingSig", err)
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

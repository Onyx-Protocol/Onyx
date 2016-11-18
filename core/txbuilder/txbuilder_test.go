package txbuilder

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"reflect"
	"testing"
	"time"

	"golang.org/x/crypto/sha3"

	"chain/crypto/ed25519"
	"chain/crypto/ed25519/chainkd"
	"chain/encoding/json"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/mempool"
	"chain/protocol/vm"
	"chain/protocol/vmutil"
	"chain/testutil"
)

type testAction bc.AssetAmount

func (t testAction) Build(ctx context.Context, maxTime time.Time, b *TemplateBuilder) error {
	in := bc.NewSpendInput([32]byte{255}, 0, nil, t.AssetID, t.Amount, nil, nil)
	tplIn := &SigningInstruction{}

	err := b.AddInput(in, tplIn)
	if err != nil {
		return err
	}
	return b.AddOutput(bc.NewTxOutput(t.AssetID, t.Amount, []byte("change"), nil))
}

func newControlProgramAction(assetAmt bc.AssetAmount, script []byte) *controlProgramAction {
	return &controlProgramAction{
		AssetAmount: assetAmt,
		Program:     script,
	}
}

func TestBuild(t *testing.T) {
	ctx := context.Background()
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
		&setTxRefDataAction{Data: []byte("xyz")},
	}
	expiryTime := time.Now().Add(time.Minute)
	got, err := Build(ctx, nil, actions, expiryTime)
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
			ReferenceData: []byte("xyz"),
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

	// setting tx refdata twice should fail
	actions = append(actions, &setTxRefDataAction{Data: []byte("lmnop")})
	_, err = Build(ctx, nil, actions, expiryTime)
	if errors.Root(err) != ErrAction {
		t.Errorf("got error %#v, want ErrAction", err)
	}
	errs := errors.Data(err)["actions"].([]error)
	if len(errs) != 1 {
		t.Errorf("got error %v action errors, want 1", len(errs))
	}
	if errors.Root(errs[0]) != ErrBadRefData {
		t.Errorf("got error %v in action error, want ErrBadRefData", errs[0])
	}
}

func TestMaterializeWitnesses(t *testing.T) {
	var initialBlockHash bc.Hash
	privkey, pubkey, err := chainkd.NewXKeys(nil)
	if err != nil {
		t.Fatal(err)
	}
	issuanceProg, _ := vmutil.P2SPMultiSigProgram([]ed25519.PublicKey{pubkey.PublicKey()}, 1)
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

	prog, err := vm.Assemble(fmt.Sprintf("MAXTIME 0x804cf05736 LESSTHAN VERIFY 0 0 5 0x%x 1 0x76a914c5d128911c28776f56baaac550963f7b88501dc388c0 CHECKOUTPUT", assetID[:]))
	h := sha3.Sum256(prog)
	sig := privkey.Sign(h[:])
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
						DerivationPath: []json.HexBytes{{0, 0, 0, 0}},
					}},
					Program: prog,
					Sigs:    []json.HexBytes{sig},
				},
			},
		}},
	}

	want := [][]byte{
		vm.Int64Bytes(0),
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
	privkey1, pubkey1, err := chainkd.NewXKeys(nil)
	if err != nil {
		t.Fatal(err)
	}
	privkey2, pubkey2, err := chainkd.NewXKeys(nil)
	if err != nil {
		t.Fatal(err)
	}
	privkey3, pubkey3, err := chainkd.NewXKeys(nil)
	if err != nil {
		t.Fatal(err)
	}
	issuanceProg, _ := vmutil.P2SPMultiSigProgram([]ed25519.PublicKey{pubkey1.PublicKey(), pubkey2.PublicKey(), pubkey3.PublicKey()}, 2)
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
	sig1 := privkey1.Sign(msg[:])
	sig2 := privkey2.Sign(msg[:])
	sig3 := privkey3.Sign(msg[:])
	want := [][]byte{
		vm.Int64Bytes(0),
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
						DerivationPath: []json.HexBytes{{0, 0, 0, 0}},
					},
					{
						XPub:           pubkey2.String(),
						DerivationPath: []json.HexBytes{{0, 0, 0, 0}},
					},
					{
						XPub:           pubkey3.String(),
						DerivationPath: []json.HexBytes{{0, 0, 0, 0}},
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

	// Test with exact amount of signatures required, in correct order
	component, ok := tpl.SigningInstructions[0].WitnessComponents[0].(*SignatureWitness)
	if !ok {
		t.Fatal("expecting WitnessComponent of type SignatureWitness")
	}
	component.Sigs = []json.HexBytes{sig1, sig2}
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

func TestTxSighashCommitment(t *testing.T) {
	var initialBlockHash bc.Hash

	issuanceProg := []byte{byte(vm.OP_TRUE)}
	assetID := bc.ComputeAssetID(issuanceProg, initialBlockHash, 1)

	// Tx with only issuance inputs is OK
	tx := bc.NewTx(bc.TxData{
		Version: 1,
		Inputs: []*bc.TxInput{
			{
				AssetVersion: 1,
				TypedInput: &bc.IssuanceInput{
					Nonce:           []byte{1},
					Amount:          1,
					InitialBlock:    initialBlockHash,
					VMVersion:       1,
					IssuanceProgram: issuanceProg,
				},
			},
			{
				AssetVersion: 1,
				TypedInput: &bc.IssuanceInput{
					Nonce:           []byte{2},
					Amount:          1,
					InitialBlock:    initialBlockHash,
					VMVersion:       1,
					IssuanceProgram: issuanceProg,
				},
			},
		},
		Outputs: []*bc.TxOutput{
			{
				AssetVersion: 1,
				OutputCommitment: bc.OutputCommitment{
					AssetAmount: bc.AssetAmount{
						AssetID: assetID,
						Amount:  2,
					},
					VMVersion:      1,
					ControlProgram: []byte{3},
				},
			},
		},
		MinTime: bc.Millis(time.Now()),
		MaxTime: bc.Millis(time.Now().Add(time.Hour)),
	})
	err := checkTxSighashCommitment(tx)
	if err != nil {
		t.Errorf("issuances-only: got error %s, want no error", err)
	}

	// Tx with at any spend inputs, none committing to the txsighash, is not OK
	tx.Inputs = append(tx.Inputs, &bc.TxInput{
		AssetVersion: 1,
		TypedInput: &bc.SpendInput{
			OutputCommitment: bc.OutputCommitment{
				AssetAmount: bc.AssetAmount{
					AssetID: assetID,
					Amount:  2,
				},
				VMVersion:      1,
				ControlProgram: []byte{byte(vm.OP_TRUE)},
			},
		},
	})
	tx.Outputs[0].Amount = 4
	tx = bc.NewTx(tx.TxData) // recompute the tx hash
	err = checkTxSighashCommitment(tx)
	if err != ErrNoTxSighashCommitment {
		t.Errorf("no spend inputs committing to txsighash: got error %s, want ErrNoTxSighashCommitment", err)
	}

	// Tx with a spend input committing to the wrong txsighash is not OK
	spendInput := &bc.SpendInput{
		OutputCommitment: bc.OutputCommitment{
			AssetAmount: bc.AssetAmount{
				AssetID: assetID,
				Amount:  3,
			},
			VMVersion:      1,
			ControlProgram: []byte{byte(vm.OP_TRUE)},
		},
	}
	tx.Inputs = append(tx.Inputs, &bc.TxInput{
		AssetVersion: 1,
		TypedInput:   spendInput,
	})
	tx.Outputs[0].Amount = 7
	tx = bc.NewTx(tx.TxData) // recompute the tx hash
	spendInput.Arguments = make([][]byte, 3)
	prog, err := vm.Assemble("0x0000000000000000000000000000000000000000000000000000000000000000 TXSIGHASH EQUAL")
	if err != nil {
		t.Fatal(err)
	}
	spendInput.Arguments[2] = prog
	err = checkTxSighashCommitment(tx)
	if err != ErrNoTxSighashCommitment {
		t.Errorf("spend input committing to the wrong txsighash: got error %s, want ErrNoTxSighashCommitment", err)
	}

	// Tx with a spend input committing to the right txsighash is OK
	spendInput = &bc.SpendInput{
		OutputCommitment: bc.OutputCommitment{
			AssetAmount: bc.AssetAmount{
				AssetID: assetID,
				Amount:  4,
			},
			VMVersion:      1,
			ControlProgram: []byte{byte(vm.OP_TRUE)},
		},
	}
	tx.Inputs = append(tx.Inputs, &bc.TxInput{
		AssetVersion: 1,
		TypedInput:   spendInput,
	})
	tx.Outputs[0].Amount = 11
	tx = bc.NewTx(tx.TxData) // recompute the tx hash
	spendInput.Arguments = make([][]byte, 3)
	h := tx.HashForSig(4)
	prog, err = vm.Assemble(fmt.Sprintf("0x%x TXSIGHASH EQUAL", h[:]))
	if err != nil {
		t.Fatal(err)
	}
	spendInput.Arguments[2] = prog
	err = checkTxSighashCommitment(tx)
	if err != nil {
		t.Errorf("spend input committing to the right txsighash: got error %s, want no error", err)
	}
}

func TestCheckBlankCheck(t *testing.T) {
	cases := []struct {
		tx   *bc.TxData
		want error
	}{{
		tx: &bc.TxData{
			Inputs: []*bc.TxInput{bc.NewSpendInput(bc.Hash{}, 0, nil, bc.AssetID{0}, 5, nil, nil)},
		},
		want: ErrBlankCheck,
	}, {
		tx: &bc.TxData{
			Inputs:  []*bc.TxInput{bc.NewSpendInput(bc.Hash{}, 0, nil, bc.AssetID{0}, 5, nil, nil)},
			Outputs: []*bc.TxOutput{bc.NewTxOutput(bc.AssetID{0}, 3, nil, nil)},
		},
		want: ErrBlankCheck,
	}, {
		tx: &bc.TxData{
			Inputs: []*bc.TxInput{
				bc.NewSpendInput(bc.Hash{}, 0, nil, bc.AssetID{0}, 5, nil, nil),
				bc.NewSpendInput(bc.Hash{}, 0, nil, bc.AssetID{1}, 5, nil, nil),
			},
			Outputs: []*bc.TxOutput{bc.NewTxOutput(bc.AssetID{0}, 5, nil, nil)},
		},
		want: ErrBlankCheck,
	}, {
		tx: &bc.TxData{
			Inputs: []*bc.TxInput{bc.NewSpendInput(bc.Hash{}, 0, nil, bc.AssetID{0}, 5, nil, nil)},
			Outputs: []*bc.TxOutput{
				bc.NewTxOutput(bc.AssetID{0}, math.MaxInt64, nil, nil),
				bc.NewTxOutput(bc.AssetID{0}, 7, nil, nil),
			},
		},
		want: ErrBadAmount,
	}, {
		tx: &bc.TxData{
			Inputs: []*bc.TxInput{
				bc.NewSpendInput(bc.Hash{}, 0, nil, bc.AssetID{0}, 5, nil, nil),
				bc.NewSpendInput(bc.Hash{}, 0, nil, bc.AssetID{0}, math.MaxInt64, nil, nil),
			},
		},
		want: ErrBadAmount,
	}, {
		tx: &bc.TxData{
			Inputs:  []*bc.TxInput{bc.NewSpendInput(bc.Hash{}, 0, nil, bc.AssetID{0}, 5, nil, nil)},
			Outputs: []*bc.TxOutput{bc.NewTxOutput(bc.AssetID{0}, 5, nil, nil)},
		},
		want: nil,
	}, {
		tx: &bc.TxData{
			Outputs: []*bc.TxOutput{bc.NewTxOutput(bc.AssetID{0}, 5, nil, nil)},
		},
		want: nil,
	}, {
		tx: &bc.TxData{
			Inputs:  []*bc.TxInput{bc.NewSpendInput(bc.Hash{}, 0, nil, bc.AssetID{0}, 5, nil, nil)},
			Outputs: []*bc.TxOutput{bc.NewTxOutput(bc.AssetID{1}, 5, nil, nil)},
		},
		want: nil,
	}}

	for _, c := range cases {
		got := checkBlankCheck(c.tx)
		if errors.Root(got) != c.want {
			t.Errorf("checkUnsafe(%+v) err = %v want %v", c.tx, errors.Root(got), c.want)
		}
	}
}

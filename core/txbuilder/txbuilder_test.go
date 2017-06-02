package txbuilder

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"

	"golang.org/x/crypto/sha3"

	"chain/crypto/ed25519"
	"chain/crypto/ed25519/chainkd"
	"chain/encoding/json"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/bc/legacy"
	"chain/protocol/vm"
	"chain/protocol/vm/vmutil"
	"chain/testutil"
)

type testAction bc.AssetAmount

func (t testAction) Build(ctx context.Context, b *TemplateBuilder) error {
	in := legacy.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), *t.AssetId, t.Amount, 0, nil, bc.Hash{}, nil)
	tplIn := &SigningInstruction{}

	err := b.AddInput(in, tplIn)
	if err != nil {
		return err
	}
	return b.AddOutput(legacy.NewTxOutput(*t.AssetId, t.Amount, []byte("change"), nil))
}

func newControlProgramAction(assetAmt bc.AssetAmount, script []byte) *controlProgramAction {
	return &controlProgramAction{
		AssetAmount: assetAmt,
		Program:     script,
	}
}

func TestBuild(t *testing.T) {
	ctx := context.Background()

	assetID1 := bc.NewAssetID([32]byte{1})
	assetID2 := bc.NewAssetID([32]byte{2})

	actions := []Action{
		newControlProgramAction(bc.AssetAmount{AssetId: &assetID2, Amount: 6}, []byte("dest")),
		testAction(bc.AssetAmount{AssetId: &assetID1, Amount: 5}),
		&setTxRefDataAction{Data: []byte("xyz")},
	}
	expiryTime := time.Now().Add(time.Minute)
	got, err := Build(ctx, nil, actions, expiryTime)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	want := &Template{
		Transaction: legacy.NewTx(legacy.TxData{
			Version: 1,
			MaxTime: bc.Millis(expiryTime),
			Inputs: []*legacy.TxInput{
				legacy.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), assetID1, 5, 0, nil, bc.Hash{}, nil),
			},
			Outputs: []*legacy.TxOutput{
				legacy.NewTxOutput(assetID2, 6, []byte("dest"), nil),
				legacy.NewTxOutput(assetID1, 5, []byte("change"), nil),
			},
			ReferenceData: []byte("xyz"),
		}),
		SigningInstructions: []*SigningInstruction{{
			SignatureWitnesses: []*signatureWitness{},
		}},
	}

	if !testutil.DeepEqual(got.Transaction.TxData, want.Transaction.TxData) {
		t.Errorf("got tx:\n%s\nwant tx:\n%s", spew.Sdump(got.Transaction.TxData), spew.Sdump(want.Transaction.TxData))
	}

	if !testutil.DeepEqual(got.SigningInstructions, want.SigningInstructions) {
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
	assetID := bc.ComputeAssetID(issuanceProg, &initialBlockHash, 1, &bc.EmptyStringHash)
	nonce := []byte{1}
	outscript := mustDecodeHex("76a914c5d128911c28776f56baaac550963f7b88501dc388c0")
	unsigned := legacy.NewTx(legacy.TxData{
		Version: 1,
		Inputs: []*legacy.TxInput{
			legacy.NewIssuanceInput(nonce, 5, nil, initialBlockHash, issuanceProg, nil, nil),
		},
		Outputs: []*legacy.TxOutput{
			legacy.NewTxOutput(assetID, 5, outscript, nil),
		},
	})

	prog, err := vm.Assemble(fmt.Sprintf("MAXTIME 0x804cf05736 LESSTHAN VERIFY 0 0 5 0x%x 1 0x76a914c5d128911c28776f56baaac550963f7b88501dc388c0 CHECKOUTPUT", assetID.Bytes()))
	h := sha3.Sum256(prog)
	sig := privkey.Sign(h[:])
	if err != nil {
		t.Fatal(err)
	}

	tpl := &Template{
		Transaction: unsigned,
		SigningInstructions: []*SigningInstruction{{
			SignatureWitnesses: []*signatureWitness{
				&signatureWitness{
					Quorum: 1,
					Keys: []keyID{{
						XPub:           pubkey,
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
		testutil.FatalErr(t, err)
	}

	got := tpl.Transaction.Inputs[0].Arguments()
	if !testutil.DeepEqual(got, want) {
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
	assetID := bc.ComputeAssetID(issuanceProg, &initialBlockHash, 1, &bc.EmptyStringHash)
	outscript := mustDecodeHex("76a914c5d128911c28776f56baaac550963f7b88501dc388c0")
	unsigned := legacy.NewTx(legacy.TxData{
		Version: 1,
		Inputs: []*legacy.TxInput{
			legacy.NewIssuanceInput([]byte{1}, 100, nil, initialBlockHash, issuanceProg, nil, nil),
		},
		Outputs: []*legacy.TxOutput{
			legacy.NewTxOutput(assetID, 100, outscript, nil),
		},
	})

	tpl := &Template{
		Transaction: unsigned,
	}
	h := tpl.Hash(0)
	builder := vmutil.NewBuilder()
	builder.AddData(h.Bytes())
	builder.AddOp(vm.OP_TXSIGHASH).AddOp(vm.OP_EQUAL)
	prog, _ := builder.Build()
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
		SignatureWitnesses: []*signatureWitness{
			&signatureWitness{
				Quorum: 2,
				Keys: []keyID{
					{
						XPub:           pubkey1,
						DerivationPath: []json.HexBytes{{0, 0, 0, 0}},
					},
					{
						XPub:           pubkey2,
						DerivationPath: []json.HexBytes{{0, 0, 0, 0}},
					},
					{
						XPub:           pubkey3,
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
		testutil.FatalErr(t, err)
	}
	got := tpl.Transaction.Inputs[0].Arguments()
	if !testutil.DeepEqual(got, want) {
		t.Errorf("got input witness %v, want input witness %v", got, want)
	}

	// Test with exact amount of signatures required, in correct order
	component := tpl.SigningInstructions[0].SignatureWitnesses[0]
	component.Sigs = []json.HexBytes{sig1, sig2}
	err = materializeWitnesses(tpl)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	got = tpl.Transaction.Inputs[0].Arguments()
	if !testutil.DeepEqual(got, want) {
		t.Errorf("got input witness %v, want input witness %v", got, want)
	}
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
	assetID := bc.ComputeAssetID(issuanceProg, &initialBlockHash, 1, &bc.EmptyStringHash)

	// all-issuance input tx should fail if none of the inputs commit to the tx signature
	tx := legacy.NewTx(legacy.TxData{
		Version: 1,
		Inputs: []*legacy.TxInput{
			{
				AssetVersion: 1,
				TypedInput: &legacy.IssuanceInput{
					Nonce:  []byte{1},
					Amount: 1,
					IssuanceWitness: legacy.IssuanceWitness{
						InitialBlock:    initialBlockHash,
						VMVersion:       1,
						IssuanceProgram: issuanceProg,
					},
				},
			},
			{
				AssetVersion: 1,
				TypedInput: &legacy.IssuanceInput{
					Nonce:  []byte{2},
					Amount: 1,
					IssuanceWitness: legacy.IssuanceWitness{
						InitialBlock:    initialBlockHash,
						VMVersion:       1,
						IssuanceProgram: issuanceProg,
					},
				},
			},
		},
		Outputs: []*legacy.TxOutput{
			{
				AssetVersion: 1,
				OutputCommitment: legacy.OutputCommitment{
					AssetAmount: bc.AssetAmount{
						AssetId: &assetID,
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
	if err != ErrNoTxSighashAttempt {
		t.Errorf("no issuance inputs committing to txsighash: got error %s, want ErrNoTxSighashAttempt", err)
	}

	// Tx with any spend inputs, none committing to the txsighash, is not OK
	tx.Inputs = append(tx.Inputs, &legacy.TxInput{
		AssetVersion: 1,
		TypedInput: &legacy.SpendInput{
			SpendCommitment: legacy.SpendCommitment{
				AssetAmount: bc.AssetAmount{
					AssetId: &assetID,
					Amount:  2,
				},
				VMVersion:      1,
				ControlProgram: []byte{byte(vm.OP_TRUE)},
			},
		},
	})
	tx.Outputs[0].Amount = 4
	tx = legacy.NewTx(tx.TxData) // recompute the tx hash
	err = checkTxSighashCommitment(tx)
	if err != ErrNoTxSighashAttempt {
		t.Errorf("no spend inputs committing to txsighash: got error %s, want ErrNoTxSighashAttempt", err)
	}

	// Tx with a spend input committing to the wrong txsighash is not OK
	spendInput := &legacy.SpendInput{
		SpendCommitment: legacy.SpendCommitment{
			AssetAmount: bc.AssetAmount{
				AssetId: &assetID,
				Amount:  3,
			},
			VMVersion:      1,
			ControlProgram: []byte{byte(vm.OP_TRUE)},
		},
	}
	tx.Inputs = append(tx.Inputs, &legacy.TxInput{
		AssetVersion: 1,
		TypedInput:   spendInput,
	})
	tx.Outputs[0].Amount = 7
	tx = legacy.NewTx(tx.TxData) // recompute the tx hash
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
	spendInput = &legacy.SpendInput{
		SpendCommitment: legacy.SpendCommitment{
			AssetAmount: bc.AssetAmount{
				AssetId: &assetID,
				Amount:  4,
			},
			VMVersion:      1,
			ControlProgram: []byte{byte(vm.OP_TRUE)},
		},
	}
	tx.Inputs = append(tx.Inputs, &legacy.TxInput{
		AssetVersion: 1,
		TypedInput:   spendInput,
	})
	tx.Outputs[0].Amount = 11
	tx = legacy.NewTx(tx.TxData) // recompute the tx hash
	spendInput.Arguments = make([][]byte, 3)
	h := tx.SigHash(4)
	prog, err = vm.Assemble(fmt.Sprintf("0x%x TXSIGHASH EQUAL", h.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	spendInput.Arguments[2] = prog
	err = checkTxSighashCommitment(tx)
	if err != nil {
		t.Errorf("spend input committing to the right txsighash: got error %s, want no error", err)
	}

	//Tx with a spend input missing signature argument is not OK
	spendInput = &legacy.SpendInput{
		SpendCommitment: legacy.SpendCommitment{
			AssetAmount: bc.AssetAmount{
				AssetId: &assetID,
				Amount:  5,
			},
			VMVersion:      1,
			ControlProgram: []byte{byte(vm.OP_TRUE)},
		},
	}
	tx.Inputs = append(tx.Inputs, &legacy.TxInput{
		AssetVersion: 1,
		TypedInput:   spendInput,
	})
	tx.Outputs[0].Amount = 16
	tx = legacy.NewTx(tx.TxData) // recompute the tx hash
	spendInput.Arguments = make([][]byte, 2)
	h = tx.SigHash(5)
	prog, err = vm.Assemble(fmt.Sprintf("0x%x TXSIGHASH EQUAL", h.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	spendInput.Arguments[1] = prog
	err = checkTxSighashCommitment(tx)
	if err != ErrTxSignatureFailure {
		t.Errorf("spend input missing siguature: got error %s, want ErrTxSignatureFailure", err)
	}
}

func TestCheckBlankCheck(t *testing.T) {
	cases := []struct {
		tx   *legacy.TxData
		want error
	}{{
		tx: &legacy.TxData{
			Inputs: []*legacy.TxInput{legacy.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), bc.AssetID{}, 5, 0, nil, bc.Hash{}, nil)},
		},
		want: ErrBlankCheck,
	}, {
		tx: &legacy.TxData{
			Inputs:  []*legacy.TxInput{legacy.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), bc.AssetID{}, 5, 0, nil, bc.Hash{}, nil)},
			Outputs: []*legacy.TxOutput{legacy.NewTxOutput(bc.AssetID{}, 3, nil, nil)},
		},
		want: ErrBlankCheck,
	}, {
		tx: &legacy.TxData{
			Inputs: []*legacy.TxInput{
				legacy.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), bc.AssetID{}, 5, 0, nil, bc.Hash{}, nil),
				legacy.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), bc.NewAssetID([32]byte{1}), 5, 0, nil, bc.Hash{}, nil),
			},
			Outputs: []*legacy.TxOutput{legacy.NewTxOutput(bc.AssetID{}, 5, nil, nil)},
		},
		want: ErrBlankCheck,
	}, {
		tx: &legacy.TxData{
			Inputs: []*legacy.TxInput{legacy.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), bc.AssetID{}, 5, 0, nil, bc.Hash{}, nil)},
			Outputs: []*legacy.TxOutput{
				legacy.NewTxOutput(bc.AssetID{}, math.MaxInt64, nil, nil),
				legacy.NewTxOutput(bc.AssetID{}, 7, nil, nil),
			},
		},
		want: ErrBadAmount,
	}, {
		tx: &legacy.TxData{
			Inputs: []*legacy.TxInput{
				legacy.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), bc.AssetID{}, 5, 0, nil, bc.Hash{}, nil),
				legacy.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), bc.AssetID{}, math.MaxInt64, 0, nil, bc.Hash{}, nil),
			},
		},
		want: ErrBadAmount,
	}, {
		tx: &legacy.TxData{
			Inputs:  []*legacy.TxInput{legacy.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), bc.AssetID{}, 5, 0, nil, bc.Hash{}, nil)},
			Outputs: []*legacy.TxOutput{legacy.NewTxOutput(bc.AssetID{}, 5, nil, nil)},
		},
		want: nil,
	}, {
		tx: &legacy.TxData{
			Outputs: []*legacy.TxOutput{legacy.NewTxOutput(bc.AssetID{}, 5, nil, nil)},
		},
		want: nil,
	}, {
		tx: &legacy.TxData{
			Inputs:  []*legacy.TxInput{legacy.NewSpendInput(nil, bc.NewHash([32]byte{0xff}), bc.AssetID{}, 5, 0, nil, bc.Hash{}, nil)},
			Outputs: []*legacy.TxOutput{legacy.NewTxOutput(bc.NewAssetID([32]byte{1}), 5, nil, nil)},
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

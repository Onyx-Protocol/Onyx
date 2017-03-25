package vm_test

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/quick"

	"chain/errors"
	"chain/protocol/bc"
	. "chain/protocol/vm"
	"chain/testutil"
)

type tracebuf struct {
	bytes.Buffer
}

func (t tracebuf) dump() {
	os.Stdout.Write(t.Bytes())
}

// Programs that run without error.
func TestProgramOK(t *testing.T) {
	doOKNotOK(t, true)
}

// Programs that return an ErrFalseVMResult.
func TestProgramNotOK(t *testing.T) {
	doOKNotOK(t, false)
}

func doOKNotOK(t *testing.T, expectOK bool) {
	cases := []struct {
		prog string
		args [][]byte
	}{
		{"TRUE", nil},

		// bitwise ops
		{"INVERT 0xfef0 EQUAL", [][]byte{{0x01, 0x0f}}},

		{"AND 0x02 EQUAL", [][]byte{{0x03}, {0x06}}},
		{"AND 0x02 EQUAL", [][]byte{{0x03, 0xff}, {0x06}}},

		{"OR 0x07 EQUAL", [][]byte{{0x03}, {0x06}}},
		{"OR 0x07ff EQUAL", [][]byte{{0x03, 0xff}, {0x06}}},

		{"XOR 0x05 EQUAL", [][]byte{{0x03}, {0x06}}},
		{"XOR 0x05ff EQUAL", [][]byte{{0x03, 0xff}, {0x06}}},

		// numeric and logical ops
		{"1ADD 2 NUMEQUAL", [][]byte{Int64Bytes(1)}},
		{"1ADD 0 NUMEQUAL", [][]byte{Int64Bytes(-1)}},

		{"1SUB 1 NUMEQUAL", [][]byte{Int64Bytes(2)}},
		{"1SUB -1 NUMEQUAL", [][]byte{Int64Bytes(0)}},

		{"2MUL 2 NUMEQUAL", [][]byte{Int64Bytes(1)}},
		{"2MUL 0 NUMEQUAL", [][]byte{Int64Bytes(0)}},
		{"2MUL -2 NUMEQUAL", [][]byte{Int64Bytes(-1)}},

		{"2DIV 1 NUMEQUAL", [][]byte{Int64Bytes(2)}},
		{"2DIV 0 NUMEQUAL", [][]byte{Int64Bytes(1)}},
		{"2DIV 0 NUMEQUAL", [][]byte{Int64Bytes(0)}},
		{"2DIV -1 NUMEQUAL", [][]byte{Int64Bytes(-1)}},
		{"2DIV -1 NUMEQUAL", [][]byte{Int64Bytes(-2)}},

		{"NEGATE -1 NUMEQUAL", [][]byte{Int64Bytes(1)}},
		{"NEGATE 1 NUMEQUAL", [][]byte{Int64Bytes(-1)}},
		{"NEGATE 0 NUMEQUAL", [][]byte{Int64Bytes(0)}},

		{"ABS 1 NUMEQUAL", [][]byte{Int64Bytes(1)}},
		{"ABS 1 NUMEQUAL", [][]byte{Int64Bytes(-1)}},
		{"ABS 0 NUMEQUAL", [][]byte{Int64Bytes(0)}},

		{"0NOTEQUAL", [][]byte{Int64Bytes(1)}},
		{"0NOTEQUAL NOT", [][]byte{Int64Bytes(0)}},

		{"ADD 5 NUMEQUAL", [][]byte{Int64Bytes(2), Int64Bytes(3)}},

		{"SUB 2 NUMEQUAL", [][]byte{Int64Bytes(5), Int64Bytes(3)}},

		{"MUL 6 NUMEQUAL", [][]byte{Int64Bytes(2), Int64Bytes(3)}},

		{"DIV 2 NUMEQUAL", [][]byte{Int64Bytes(6), Int64Bytes(3)}},

		{"MOD 0 NUMEQUAL", [][]byte{Int64Bytes(6), Int64Bytes(2)}},
		{"MOD 0 NUMEQUAL", [][]byte{Int64Bytes(-6), Int64Bytes(2)}},
		{"MOD 0 NUMEQUAL", [][]byte{Int64Bytes(6), Int64Bytes(-2)}},
		{"MOD 0 NUMEQUAL", [][]byte{Int64Bytes(-6), Int64Bytes(-2)}},
		{"MOD 2 NUMEQUAL", [][]byte{Int64Bytes(12), Int64Bytes(10)}},
		{"MOD 8 NUMEQUAL", [][]byte{Int64Bytes(-12), Int64Bytes(10)}},
		{"MOD -8 NUMEQUAL", [][]byte{Int64Bytes(12), Int64Bytes(-10)}},
		{"MOD -2 NUMEQUAL", [][]byte{Int64Bytes(-12), Int64Bytes(-10)}},

		{"LSHIFT 2 NUMEQUAL", [][]byte{Int64Bytes(1), Int64Bytes(1)}},
		{"LSHIFT 4 NUMEQUAL", [][]byte{Int64Bytes(1), Int64Bytes(2)}},
		{"LSHIFT -2 NUMEQUAL", [][]byte{Int64Bytes(-1), Int64Bytes(1)}},
		{"LSHIFT -4 NUMEQUAL", [][]byte{Int64Bytes(-1), Int64Bytes(2)}},

		{"1 1 BOOLAND", nil},
		{"1 0 BOOLAND NOT", nil},
		{"0 1 BOOLAND NOT", nil},
		{"0 0 BOOLAND NOT", nil},

		{"1 1 BOOLOR", nil},
		{"1 0 BOOLOR", nil},
		{"0 1 BOOLOR", nil},
		{"0 0 BOOLOR NOT", nil},

		{"1 2 OR 3 EQUAL", nil},

		// splice ops
		{"0 CATPUSHDATA 0x0000 EQUAL", [][]byte{{0x00}}},
		{"0 0xff CATPUSHDATA 0x01ff EQUAL", nil},
		{"CATPUSHDATA 0x050105 EQUAL", [][]byte{{0x05}, {0x05}}},
		{"CATPUSHDATA 0xff01ff EQUAL", [][]byte{{0xff}, {0xff}}},
		{"0 0xcccccc CATPUSHDATA 0x03cccccc EQUAL", nil},
		{"0x05 0x05 SWAP 0xdeadbeef CATPUSHDATA DROP 0x05 EQUAL", nil},
		{"0x05 0x05 SWAP 0xdeadbeef CATPUSHDATA DROP 0x05 EQUAL", nil},

		// // control flow ops
		{"1 JUMP:7 0 1 EQUAL", nil},                                                       // jumps over 0
		{"1 JUMP:$target 0 $target 1 EQUAL", nil},                                         // jumps over 0
		{"1 1 JUMPIF:8 0 1 EQUAL", nil},                                                   // jumps over 0
		{"1 1 JUMPIF:$target 0 $target 1 EQUAL", nil},                                     // jumps over 0
		{"1 0 JUMPIF:8 0 1 EQUAL NOT", nil},                                               // doesn't jump over 0
		{"1 0 JUMPIF:$target 0 $target 1 EQUAL NOT", nil},                                 // doesn't jump over 0
		{"1 0 JUMPIF:1", nil},                                                             // doesn't jump, so no infinite loop
		{"1 $target 0 JUMPIF:$target", nil},                                               // doesn't jump, so no infinite loop
		{"4 1 JUMPIF:14 5 EQUAL JUMP:16 4 EQUAL", nil},                                    // if (true) { return x == 4; } else { return x == 5; }
		{"4 1 JUMPIF:$true 5 EQUAL JUMP:$end $true 4 EQUAL $end", nil},                    // if (true) { return x == 4; } else { return x == 5; }
		{"5 0 JUMPIF:14 5 EQUAL JUMP:16 4 EQUAL", nil},                                    // if (false) { return x == 4; } else { return x == 5; }
		{"5 0 JUMPIF:$true 5 EQUAL JUMP:$end $true 4 $test EQUAL $end", nil},              // if (false) { return x == 4; } else { return x == 5; }
		{"0 1 2 3 4 5 6 JUMP:13 DROP DUP 0 NUMNOTEQUAL JUMPIF:12 1", nil},                 // same as "0 1 2 3 4 5 6 WHILE DROP ENDWHILE 1"
		{"0 1 2 3 4 5 6 JUMP:$dup $drop DROP $dup DUP 0 NUMNOTEQUAL JUMPIF:$drop 1", nil}, // same as "0 1 2 3 4 5 6 WHILE DROP ENDWHILE 1"
		{"0 JUMP:7 1ADD DUP 10 LESSTHAN JUMPIF:6 10 NUMEQUAL", nil},                       // fixed version of "0 1 WHILE DROP 1ADD DUP 10 LESSTHAN ENDWHILE 10 NUMEQUAL"
		{"0 JUMP:$dup $add 1ADD $dup DUP 10 LESSTHAN JUMPIF:$add 10 NUMEQUAL", nil},       // fixed version of "0 1 WHILE DROP 1ADD DUP 10 LESSTHAN ENDWHILE 10 NUMEQUAL"

	}
	for i, c := range cases {
		progSrc := c.prog
		if !expectOK {
			progSrc += " NOT"
		}
		prog, err := Assemble(progSrc)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("* case %d, prog [%s] [%x]\n", i, progSrc, prog)
		trace := new(tracebuf)
		TraceOut = trace
		vm := &VirtualMachine{
			Program:   prog,
			RunLimit:  int64(InitialRunLimit),
			DataStack: append([][]byte{}, c.args...),
		}
		gotVM, err := vm.Run()
		if err == nil && gotVM.FalseResult() {
			err = ErrFalseVMResult
		}
		if expectOK && err != nil {
			trace.dump()
			t.Errorf("case %d [%s]: expected success, got error %s", i, progSrc, err)
		} else if !expectOK && err != ErrFalseVMResult {
			trace.dump()
			t.Errorf("case %d [%s]: expected ErrFalseVMResult, got %s", i, progSrc, err)
		}
	}
}

func TestVerifyTxInput(t *testing.T) {
	cases := []struct {
		input   *bc.TxInput
		wantErr error
	}{{
		input: bc.NewSpendInput(
			[][]byte{{2}, {3}},
			bc.Hash{},
			bc.AssetID{},
			1,
			0,
			[]byte{byte(OP_ADD), byte(OP_5), byte(OP_NUMEQUAL)},
			bc.Hash{},
			nil,
		),
	}, {
		input: bc.NewIssuanceInput(
			[]byte{12},
			1,
			nil,
			bc.Hash{},
			[]byte{byte(OP_ADD), byte(OP_5), byte(OP_NUMEQUAL)},
			[][]byte{{2}, {3}},
			nil,
		),
	}, {
		input: &bc.TxInput{
			TypedInput: &bc.IssuanceInput{
				Nonce: []byte{34},
				IssuanceWitness: bc.IssuanceWitness{
					VMVersion: 2,
				},
			},
		},
		wantErr: ErrUnsupportedVM,
	}, {
		input: &bc.TxInput{
			TypedInput: &bc.SpendInput{
				SpendCommitment: bc.SpendCommitment{
					VMVersion: 2,
				},
			},
		},
		wantErr: ErrUnsupportedVM,
	}, {
		input: bc.NewIssuanceInput(
			[]byte{56},
			1,
			nil,
			bc.Hash{},
			[]byte{byte(OP_ADD), byte(OP_5), byte(OP_NUMEQUAL)},
			[][]byte{make([]byte, 50001)},
			nil,
		),
		wantErr: ErrRunLimitExceeded,
	}}

	for i, c := range cases {
		tx := bc.NewTx(bc.TxData{
			Inputs: []*bc.TxInput{c.input},
		})

		gotErr := verifyTx(tx, 0)

		if errors.Root(gotErr) != c.wantErr {
			t.Errorf("VerifyTxInput(%d) err = %v want %v", i, gotErr, c.wantErr)
		}
	}
}

func TestVerifyBlockHeader(t *testing.T) {
	block := bc.MapBlock(&bc.Block{
		BlockHeader: bc.BlockHeader{
			BlockWitness: bc.BlockWitness{
				Witness: [][]byte{{2}, {3}},
			},
		},
	})
	consensusProg := []byte{byte(OP_ADD), byte(OP_5), byte(OP_NUMEQUAL)}

	gotErr := Verify(bc.NewBlockVMContext(block, consensusProg, block.Witness.Arguments))
	if gotErr != nil {
		t.Errorf("unexpected error: %v", gotErr)
	}

	block = bc.MapBlock(&bc.Block{
		BlockHeader: bc.BlockHeader{
			BlockWitness: bc.BlockWitness{
				Witness: [][]byte{make([]byte, 50000)},
			},
		},
	})

	gotErr = Verify(bc.NewBlockVMContext(block, consensusProg, block.Witness.Arguments))
	if errors.Root(gotErr) != ErrRunLimitExceeded {
		t.Error("expected block to exceed run limit")
	}
}

func TestRun(t *testing.T) {
	cases := []struct {
		vm      *VirtualMachine
		wantErr error
	}{{
		vm: &VirtualMachine{RunLimit: 50000, Program: []byte{byte(OP_TRUE)}},
	}, {
		vm:      &VirtualMachine{RunLimit: 50000, Program: []byte{byte(OP_ADD)}},
		wantErr: ErrDataStackUnderflow,
	}}

	for i, c := range cases {
		_, gotErr := c.vm.Run()

		if gotErr != c.wantErr {
			t.Errorf("run test %d: got err = %v want %v", i, gotErr, c.wantErr)
			continue
		}

		if c.wantErr != nil {
			continue
		}
	}
}

func TestStep(t *testing.T) {
	tx := bc.NewTx(bc.TxData{
		Version: 1,
		Inputs: []*bc.TxInput{
			bc.NewSpendInput(nil, bc.Hash{}, bc.AssetID{}, 1, 0, nil, bc.Hash{}, nil),
		},
	})
	txVMContext := bc.NewTxVMContext(tx.TxEntries, tx.TxEntries.TxInputs[0], bc.Program{}, nil)
	cases := []struct {
		startVM *VirtualMachine
		wantVM  *VirtualMachine
		wantErr error
	}{{
		startVM: &VirtualMachine{
			Program:  []byte{byte(OP_TRUE)},
			RunLimit: 50000,
		},
		wantVM: &VirtualMachine{
			Program:   []byte{byte(OP_TRUE)},
			RunLimit:  49990,
			DataStack: [][]byte{{1}},
			PC:        1,
			NextPC:    1,
			Data:      []byte{1},
		},
	}, {
		startVM: &VirtualMachine{
			Program:   []byte{byte(OP_TRUE), byte(OP_JUMP), byte(0xff), byte(0x00), byte(0x00), byte(0x00)},
			RunLimit:  49990,
			DataStack: [][]byte{},
			PC:        1,
		},
		wantVM: &VirtualMachine{
			Program:      []byte{byte(OP_TRUE), byte(OP_JUMP), byte(0xff), byte(0x00), byte(0x00), byte(0x00)},
			RunLimit:     49989,
			DataStack:    [][]byte{},
			Data:         []byte{byte(0xff), byte(0x00), byte(0x00), byte(0x00)},
			PC:           255,
			NextPC:       255,
			DeferredCost: 0,
		},
	}, {
		startVM: &VirtualMachine{
			Program:   []byte{byte(OP_TRUE), byte(OP_JUMPIF), byte(0x00), byte(0x00), byte(0x00), byte(0x00)},
			RunLimit:  49995,
			DataStack: [][]byte{{1}},
			PC:        1,
		},
		wantVM: &VirtualMachine{
			Program:      []byte{byte(OP_TRUE), byte(OP_JUMPIF), byte(0x00), byte(0x00), byte(0x00), byte(0x00)},
			RunLimit:     50003,
			DataStack:    [][]byte{},
			PC:           0,
			NextPC:       0,
			Data:         []byte{byte(0x00), byte(0x00), byte(0x00), byte(0x00)},
			DeferredCost: -9,
		},
	}, {
		startVM: &VirtualMachine{
			Program:   []byte{byte(OP_FALSE), byte(OP_JUMPIF), byte(0x00), byte(0x00), byte(0x00), byte(0x00)},
			RunLimit:  49995,
			DataStack: [][]byte{{}},
			PC:        1,
		},
		wantVM: &VirtualMachine{
			Program:      []byte{byte(OP_FALSE), byte(OP_JUMPIF), byte(0x00), byte(0x00), byte(0x00), byte(0x00)},
			RunLimit:     50002,
			DataStack:    [][]byte{},
			PC:           6,
			NextPC:       6,
			Data:         []byte{byte(0x00), byte(0x00), byte(0x00), byte(0x00)},
			DeferredCost: -8,
		},
	}, {
		startVM: &VirtualMachine{
			Program:   []byte{255},
			RunLimit:  50000,
			DataStack: [][]byte{},
		},
		wantVM: &VirtualMachine{
			Program:   []byte{255},
			RunLimit:  49999,
			PC:        1,
			NextPC:    1,
			DataStack: [][]byte{},
		},
	}, {
		startVM: &VirtualMachine{
			Program:  []byte{byte(OP_ADD)},
			RunLimit: 50000,
		},
		wantErr: ErrDataStackUnderflow,
	}, {
		startVM: &VirtualMachine{
			Program:  []byte{byte(OP_INDEX)},
			RunLimit: 1,
			Context:  txVMContext,
		},
		wantErr: ErrRunLimitExceeded,
	}, {
		startVM: &VirtualMachine{
			Program:           []byte{255},
			RunLimit:          100,
			ExpansionReserved: true,
		},
		wantErr: ErrDisallowedOpcode,
	}, {
		startVM: &VirtualMachine{
			Program:  []byte{255},
			RunLimit: 100,
		},
		wantVM: &VirtualMachine{
			Program:  []byte{255},
			RunLimit: 99,
			PC:       1,
			NextPC:   1,
		},
	}}

	for i, c := range cases {
		gotVM, gotErr := c.startVM.Step()

		if gotErr != c.wantErr {
			t.Errorf("step test %d: got err = %v want %v", i, gotErr, c.wantErr)
			continue
		}

		if c.wantErr != nil {
			continue
		}

		if !testutil.DeepEqual(gotVM, c.wantVM) {
			t.Errorf("step test %d:\n\tgot vm:  %+v\n\twant vm: %+v", i, gotVM, c.wantVM)
		}
	}
}

func decompile(prog []byte) string {
	var strs []string
	for i := uint32(0); i < uint32(len(prog)); { // update i inside the loop
		inst, err := ParseOp(prog, i)
		if err != nil {
			strs = append(strs, fmt.Sprintf("<%x>", prog[i]))
			i++
			continue
		}
		var str string
		if len(inst.Data) > 0 {
			str = fmt.Sprintf("0x%x", inst.Data)
		} else {
			str = inst.Op.String()
		}
		strs = append(strs, str)
		i += inst.Len
	}
	return strings.Join(strs, " ")
}

func TestVerifyTxInputQuickCheck(t *testing.T) {
	f := func(program []byte, witnesses [][]byte) (ok bool) {
		defer func() {
			if err := recover(); err != nil {
				t.Log(decompile(program))
				for i := range witnesses {
					t.Logf("witness %d: %x\n", i, witnesses[i])
				}
				t.Log(err)
				ok = false
			}
		}()
		tx := bc.NewTx(bc.TxData{
			Inputs: []*bc.TxInput{bc.NewSpendInput(witnesses, bc.Hash{}, bc.AssetID{}, 10, 0, program, bc.Hash{}, nil)},
		})
		verifyTx(tx, 0)
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestVerifyBlockHeaderQuickCheck(t *testing.T) {
	f := func(program []byte, witnesses [][]byte) (ok bool) {
		defer func() {
			if err := recover(); err != nil {
				t.Log(decompile(program))
				for i := range witnesses {
					t.Logf("witness %d: %x\n", i, witnesses[i])
				}
				t.Log(err)
				ok = false
			}
		}()
		block := bc.MapBlock(&bc.Block{BlockHeader: bc.BlockHeader{
			BlockWitness: bc.BlockWitness{
				Witness: witnesses,
			},
		}})
		Verify(bc.NewBlockVMContext(block, program, witnesses))
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func verifyTx(tx *bc.Tx, index uint32) error {
	var (
		prog bc.Program
		args [][]byte
	)
	inp := tx.TxInputs[index]
	switch inp := inp.(type) {
	case *bc.Issuance:
		prog = inp.Witness.AssetDefinition.IssuanceProgram
		args = inp.Witness.Arguments
	case *bc.Spend:
		prog = inp.SpentOutput.Body.ControlProgram
		args = inp.Witness.Arguments
	}
	return Verify(bc.NewTxVMContext(tx.TxEntries, inp, prog, args))
}

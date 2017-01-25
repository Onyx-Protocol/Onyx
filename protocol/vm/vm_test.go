package vm

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/quick"

	"chain/errors"
	"chain/protocol/bc"
	"chain/testutil"
)

type tracebuf struct {
	bytes.Buffer
}

func (t tracebuf) dump() {
	os.Stdout.Write(t.Bytes())
}

// Programs that run without error and return a true result.
func TestProgramOK(t *testing.T) {
	doOKNotOK(t, true)
}

// Programs that run without error and return a false result.
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
		vm := &virtualMachine{
			program:   prog,
			runLimit:  initialRunLimit,
			dataStack: append([][]byte{}, c.args...),
		}
		ok, err := vm.run()
		if err == nil {
			if ok != expectOK {
				trace.dump()
				t.Errorf("case %d [%s]: expected %v result, got %v", i, progSrc, expectOK, ok)
			}
		} else {
			trace.dump()
			t.Errorf("case %d [%s]: unexpected error: %s", i, progSrc, err)
		}
		if testing.Verbose() && (ok == expectOK) && err == nil {
			trace.dump()
			fmt.Println("")
		}
	}
}

func TestVerifyTxInput(t *testing.T) {
	cases := []struct {
		input   *bc.TxInput
		want    bool
		wantErr error
	}{{
		input: bc.NewSpendInput(
			bc.Hash{},
			0,
			[][]byte{{2}, {3}},
			bc.AssetID{},
			1,
			[]byte{byte(OP_ADD), byte(OP_5), byte(OP_NUMEQUAL)},
			nil,
		),
		want: true,
	}, {
		input: bc.NewIssuanceInput(
			nil,
			1,
			nil,
			bc.Hash{},
			[]byte{byte(OP_ADD), byte(OP_5), byte(OP_NUMEQUAL)},
			[][]byte{{2}, {3}},
			nil,
		),
		want: true,
	}, {
		input: &bc.TxInput{
			TypedInput: &bc.IssuanceInput{
				IssuanceWitness: bc.IssuanceWitness{
					VMVersion: 2,
				},
			},
		},
		wantErr: ErrUnsupportedVM,
	}, {
		input: &bc.TxInput{
			TypedInput: &bc.SpendInput{
				OutputCommitment: bc.OutputCommitment{
					VMVersion: 2,
				},
			},
		},
		wantErr: ErrUnsupportedVM,
	}, {
		input: bc.NewIssuanceInput(
			nil,
			1,
			nil,
			bc.Hash{},
			[]byte{byte(OP_ADD), byte(OP_5), byte(OP_NUMEQUAL)},
			[][]byte{make([]byte, 50001)},
			nil,
		),
		wantErr: ErrRunLimitExceeded,
	}, {
		input:   &bc.TxInput{},
		wantErr: ErrUnsupportedTx,
	}}

	for i, c := range cases {
		tx := &bc.Tx{TxData: bc.TxData{
			Inputs: []*bc.TxInput{c.input},
		}}

		got, gotErr := VerifyTxInput(tx, 0)

		if errors.Root(gotErr) != c.wantErr {
			t.Errorf("VerifyTxInput(%d) err = %v want %v", i, gotErr, c.wantErr)
		}

		if got != c.want {
			t.Errorf("VerifyTxInput(%d) = %v want %v", i, got, c.want)
		}
	}
}

func TestVerifyBlockHeader(t *testing.T) {
	block := &bc.Block{
		BlockHeader: bc.BlockHeader{
			BlockWitness: bc.BlockWitness{
				Witness: [][]byte{{2}, {3}},
			},
		},
	}
	prevBlock := &bc.Block{
		BlockHeader: bc.BlockHeader{
			BlockCommitment: bc.BlockCommitment{
				ConsensusProgram: []byte{byte(OP_ADD), byte(OP_5), byte(OP_NUMEQUAL)},
			},
		},
	}

	got, gotErr := VerifyBlockHeader(&prevBlock.BlockHeader, block)
	if gotErr != nil {
		t.Errorf("unexpected error: %v", gotErr)
	}

	if !got {
		t.Error("expected true result")
	}

	block = &bc.Block{
		BlockHeader: bc.BlockHeader{
			BlockWitness: bc.BlockWitness{
				Witness: [][]byte{make([]byte, 50000)},
			},
		},
	}

	_, gotErr = VerifyBlockHeader(&prevBlock.BlockHeader, block)
	if errors.Root(gotErr) != ErrRunLimitExceeded {
		t.Error("expected block to exceed run limit")
	}
}

func TestRun(t *testing.T) {
	cases := []struct {
		vm      *virtualMachine
		want    bool
		wantErr error
	}{{
		vm:   &virtualMachine{runLimit: 50000, program: []byte{byte(OP_TRUE)}},
		want: true,
	}, {
		vm:      &virtualMachine{runLimit: 50000, program: []byte{byte(OP_ADD)}},
		wantErr: ErrDataStackUnderflow,
	}}

	for i, c := range cases {
		got, gotErr := c.vm.run()

		if gotErr != c.wantErr {
			t.Errorf("run test %d: got err = %v want %v", i, gotErr, c.wantErr)
			continue
		}

		if c.wantErr != nil {
			continue
		}

		if got != c.want {
			t.Errorf("run test %d: got = %v want %v", i, got, c.want)
		}
	}
}

func TestStep(t *testing.T) {
	cases := []struct {
		startVM *virtualMachine
		wantVM  *virtualMachine
		wantErr error
	}{{
		startVM: &virtualMachine{
			program:  []byte{byte(OP_TRUE)},
			runLimit: 50000,
		},
		wantVM: &virtualMachine{
			program:   []byte{byte(OP_TRUE)},
			runLimit:  49990,
			dataStack: [][]byte{{1}},
			pc:        1,
			nextPC:    1,
			data:      []byte{1},
		},
	}, {
		startVM: &virtualMachine{
			program:   []byte{byte(OP_TRUE), byte(OP_JUMP), byte(0xff), byte(0x00), byte(0x00), byte(0x00)},
			runLimit:  49990,
			dataStack: [][]byte{},
			pc:        1,
		},
		wantVM: &virtualMachine{
			program:      []byte{byte(OP_TRUE), byte(OP_JUMP), byte(0xff), byte(0x00), byte(0x00), byte(0x00)},
			runLimit:     49989,
			dataStack:    [][]byte{},
			data:         []byte{byte(0xff), byte(0x00), byte(0x00), byte(0x00)},
			pc:           255,
			nextPC:       255,
			deferredCost: 0,
		},
	}, {
		startVM: &virtualMachine{
			program:   []byte{byte(OP_TRUE), byte(OP_JUMPIF), byte(0x00), byte(0x00), byte(0x00), byte(0x00)},
			runLimit:  49995,
			dataStack: [][]byte{{1}},
			pc:        1,
		},
		wantVM: &virtualMachine{
			program:      []byte{byte(OP_TRUE), byte(OP_JUMPIF), byte(0x00), byte(0x00), byte(0x00), byte(0x00)},
			runLimit:     50003,
			dataStack:    [][]byte{},
			pc:           0,
			nextPC:       0,
			data:         []byte{byte(0x00), byte(0x00), byte(0x00), byte(0x00)},
			deferredCost: -9,
		},
	}, {
		startVM: &virtualMachine{
			program:   []byte{byte(OP_FALSE), byte(OP_JUMPIF), byte(0x00), byte(0x00), byte(0x00), byte(0x00)},
			runLimit:  49995,
			dataStack: [][]byte{{}},
			pc:        1,
		},
		wantVM: &virtualMachine{
			program:      []byte{byte(OP_FALSE), byte(OP_JUMPIF), byte(0x00), byte(0x00), byte(0x00), byte(0x00)},
			runLimit:     50002,
			dataStack:    [][]byte{},
			pc:           6,
			nextPC:       6,
			data:         []byte{byte(0x00), byte(0x00), byte(0x00), byte(0x00)},
			deferredCost: -8,
		},
	}, {
		startVM: &virtualMachine{
			program:   []byte{255},
			runLimit:  50000,
			dataStack: [][]byte{},
		},
		wantVM: &virtualMachine{
			program:   []byte{255},
			runLimit:  49999,
			pc:        1,
			nextPC:    1,
			dataStack: [][]byte{},
		},
	}, {
		startVM: &virtualMachine{
			program:  []byte{byte(OP_ADD)},
			runLimit: 50000,
		},
		wantErr: ErrDataStackUnderflow,
	}, {
		startVM: &virtualMachine{
			program:  []byte{byte(OP_INDEX)},
			runLimit: 1,
			tx:       &bc.Tx{},
		},
		wantErr: ErrRunLimitExceeded,
	}, {
		startVM: &virtualMachine{
			program:           []byte{255},
			runLimit:          100,
			expansionReserved: true,
		},
		wantErr: ErrDisallowedOpcode,
	}, {
		startVM: &virtualMachine{
			program:  []byte{255},
			runLimit: 100,
		},
		wantVM: &virtualMachine{
			program:  []byte{255},
			runLimit: 99,
			pc:       1,
			nextPC:   1,
		},
	}}

	for i, c := range cases {
		gotErr := c.startVM.step()

		if gotErr != c.wantErr {
			t.Errorf("step test %d: got err = %v want %v", i, gotErr, c.wantErr)
			continue
		}

		if c.wantErr != nil {
			continue
		}

		if !testutil.DeepEqual(c.startVM, c.wantVM) {
			t.Errorf("step test %d:\n\tgot vm:  %+v\n\twant vm: %+v", i, c.startVM, c.wantVM)
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
			Inputs: []*bc.TxInput{bc.NewSpendInput(bc.Hash{}, 0, witnesses, bc.AssetID{}, 10, program, nil)},
		})
		verifyTxInput(tx, 0)
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
		prev := &bc.BlockHeader{
			BlockCommitment: bc.BlockCommitment{
				ConsensusProgram: program,
			},
		}
		block := &bc.Block{BlockHeader: bc.BlockHeader{
			BlockWitness: bc.BlockWitness{
				Witness: witnesses,
			},
		}}
		verifyBlockHeader(prev, block)
		return true
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

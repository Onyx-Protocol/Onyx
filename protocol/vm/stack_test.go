package vm

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"chain/testutil"
)

func TestStackOps(t *testing.T) {
	type testStruct struct {
		op      Op
		startVM *virtualMachine
		wantErr error
		wantVM  *virtualMachine
	}

	cases := []testStruct{{
		op: OP_TOALTSTACK,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{1}},
		},
		wantVM: &virtualMachine{
			runLimit:  49998,
			dataStack: [][]byte{},
			altStack:  [][]byte{{1}},
		},
	}, {
		op: OP_FROMALTSTACK,
		startVM: &virtualMachine{
			runLimit: 50000,
			altStack: [][]byte{{1}},
		},
		wantVM: &virtualMachine{
			runLimit:  49998,
			altStack:  [][]byte{},
			dataStack: [][]byte{{1}},
		},
	}, {
		op: OP_FROMALTSTACK,
		startVM: &virtualMachine{
			runLimit: 50000,
			altStack: [][]byte{},
		},
		wantErr: ErrAltStackUnderflow,
	}, {
		op: OP_2DROP,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{1}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:  50016,
			dataStack: [][]byte{},
		},
	}, {
		op: OP_2DUP,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:  49980,
			dataStack: [][]byte{{2}, {1}, {2}, {1}},
		},
	}, {
		op: OP_3DUP,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{3}, {2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:  49970,
			dataStack: [][]byte{{3}, {2}, {1}, {3}, {2}, {1}},
		},
	}, {
		op: OP_2OVER,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{4}, {3}, {2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:  49980,
			dataStack: [][]byte{{4}, {3}, {2}, {1}, {4}, {3}},
		},
	}, {
		op: OP_2OVER,
		startVM: &virtualMachine{
			runLimit:  2,
			dataStack: [][]byte{{4}, {3}, {2}, {1}},
		},
		wantErr: ErrRunLimitExceeded,
	}, {
		op: OP_2ROT,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{6}, {5}, {4}, {3}, {2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:  49998,
			dataStack: [][]byte{{4}, {3}, {2}, {1}, {6}, {5}},
		},
	}, {
		op: OP_2SWAP,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{4}, {3}, {2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:  49998,
			dataStack: [][]byte{{2}, {1}, {4}, {3}},
		},
	}, {
		op: OP_IFDUP,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{1}},
		},
		wantVM: &virtualMachine{
			runLimit:  49990,
			dataStack: [][]byte{{1}, {1}},
		},
	}, {
		op: OP_IFDUP,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{}},
		},
		wantVM: &virtualMachine{
			runLimit:  49999,
			dataStack: [][]byte{{}},
		},
	}, {
		op: OP_IFDUP,
		startVM: &virtualMachine{
			runLimit:  1,
			dataStack: [][]byte{{1}},
		},
		wantErr: ErrRunLimitExceeded,
	}, {
		op: OP_DEPTH,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{1}},
		},
		wantVM: &virtualMachine{
			runLimit:  49990,
			dataStack: [][]byte{{1}, {1}},
		},
	}, {
		op: OP_DEPTH,
		startVM: &virtualMachine{
			runLimit:  1,
			dataStack: [][]byte{{1}},
		},
		wantErr: ErrRunLimitExceeded,
	}, {
		op: OP_DROP,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{1}},
		},
		wantVM: &virtualMachine{
			runLimit:  50008,
			dataStack: [][]byte{},
		},
	}, {
		op: OP_DUP,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{1}},
		},
		wantVM: &virtualMachine{
			runLimit:  49990,
			dataStack: [][]byte{{1}, {1}},
		},
	}, {
		op: OP_DUP,
		startVM: &virtualMachine{
			runLimit:  1,
			dataStack: [][]byte{{1}},
		},
		wantErr: ErrRunLimitExceeded,
	}, {
		op: OP_NIP,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:  50008,
			dataStack: [][]byte{{1}},
		},
	}, {
		op: OP_OVER,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:  49990,
			dataStack: [][]byte{{2}, {1}, {2}},
		},
	}, {
		op: OP_OVER,
		startVM: &virtualMachine{
			runLimit:  1,
			dataStack: [][]byte{{2}, {1}},
		},
		wantErr: ErrRunLimitExceeded,
	}, {
		op: OP_PICK,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{3}, {2}, {1}, {2}},
		},
		wantVM: &virtualMachine{
			runLimit:  49998,
			dataStack: [][]byte{{3}, {2}, {1}, {3}},
		},
	}, {
		op: OP_PICK,
		startVM: &virtualMachine{
			runLimit:  2,
			dataStack: [][]byte{{0xff, 0xff}, {2}, {1}, {2}},
		},
		wantErr: ErrRunLimitExceeded,
	}, {
		op: OP_ROLL,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{3}, {2}, {1}, {2}},
		},
		wantVM: &virtualMachine{
			runLimit:  50007,
			dataStack: [][]byte{{2}, {1}, {3}},
		},
	}, {
		op: OP_ROT,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{3}, {2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:  49998,
			dataStack: [][]byte{{2}, {1}, {3}},
		},
	}, {
		op: OP_SWAP,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:  49999,
			dataStack: [][]byte{{1}, {2}},
		},
	}, {
		op: OP_TUCK,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:  49990,
			dataStack: [][]byte{{1}, {2}, {1}},
		},
	}, {
		op: OP_TUCK,
		startVM: &virtualMachine{
			runLimit:  1,
			dataStack: [][]byte{{2}, {1}},
		},
		wantErr: ErrRunLimitExceeded,
	}}
	stackops := []Op{
		OP_DEPTH, OP_FROMALTSTACK, OP_TOALTSTACK, OP_2DROP, OP_2DUP, OP_3DUP,
		OP_2OVER, OP_2ROT, OP_2SWAP, OP_IFDUP, OP_DROP, OP_DUP, OP_NIP,
		OP_OVER, OP_PICK, OP_ROLL, OP_ROT, OP_SWAP, OP_TUCK,
	}
	for _, op := range stackops {
		cases = append(cases, testStruct{
			op:      op,
			startVM: &virtualMachine{runLimit: 0},
			wantErr: ErrRunLimitExceeded,
		})
	}

	for i, c := range cases {
		err := ops[c.op].fn(c.startVM)

		if err != c.wantErr {
			t.Errorf("case %d, op %s: got err = %v want %v", i, ops[c.op].name, err, c.wantErr)
			continue
		}
		if c.wantErr != nil {
			continue
		}

		if !testutil.DeepEqual(c.startVM, c.wantVM) {
			t.Errorf("case %d, op %s: unexpected vm result\n\tgot:  %+v\n\twant: %+v\n", i, ops[c.op].name, c.startVM, c.wantVM)
		}
	}
}

func TestStackUnderflow(t *testing.T) {
	cases := []struct {
		narg int // number of stack items required
		op   func(*virtualMachine) error
	}{
		// bitwise
		{1, opInvert},
		{2, opAnd},
		{2, opOr},
		{2, opXor},
		{2, opEqual},
		{2, opEqualVerify},

		// control
		{1, opVerify},
		{3, opCheckPredicate},
		{1, opJumpIf},

		// crypto
		{1, opSha256},
		{1, opSha3},
		{3, opCheckSig},
		{3, opCheckMultiSig}, // special, see also TestCryptoOps

		// introspection
		{6, opCheckOutput},

		// numeric
		{1, op1Add},
		{1, op1Sub},
		{1, op2Mul},
		{1, op2Div},
		{1, opNegate},
		{1, opAbs},
		{1, opNot},
		{1, op0NotEqual},
		{2, opAdd},
		{2, opSub},
		{2, opMul},
		{2, opDiv},
		{2, opMod},
		{2, opLshift},
		{2, opRshift},
		{2, opBoolAnd},
		{2, opBoolOr},
		{2, opNumEqual},
		{2, opNumEqualVerify},
		{2, opNumNotEqual},
		{2, opLessThan},
		{2, opGreaterThan},
		{2, opLessThanOrEqual},
		{2, opGreaterThanOrEqual},
		{2, opMin},
		{2, opMax},
		{3, opWithin},

		// splice
		{2, opCat},
		{3, opSubstr},
		{2, opLeft},
		{2, opRight},
		{1, opSize},
		{2, opCatpushdata},

		// stack
		{1, opToAltStack},
		{2, op2Drop},
		{2, op2Dup},
		{3, op3Dup},
		{4, op2Over},
		{6, op2Rot},
		{4, op2Swap},
		{1, opIfDup},
		{1, opDrop},
		{1, opDup},
		{2, opNip},
		{2, opOver},
		{2, opPick}, // TODO(kr): special; check data-dependent # of pops
		{2, opRoll}, // TODO(kr): special; check data-dependent # of pops
		{3, opRot},
		{2, opSwap},
		{2, opTuck},
	}

	for _, test := range cases {
		t.Run(funcName(test.op), func(t *testing.T) {

			for i := 0; i < test.narg; i++ {
				t.Run(fmt.Sprintf("%d args", i), func(t *testing.T) {

					vm := &virtualMachine{
						runLimit:  50000,
						dataStack: make([][]byte, i),
					}
					err := test.op(vm)
					if err != ErrDataStackUnderflow {
						t.Errorf("err = %v, want ErrStackUnderflow", err)
					}

				})
			}

		})
	}
}

func funcName(f interface{}) string {
	v := reflect.ValueOf(f)
	if v.Kind() != reflect.Func {
		return ""
	}
	s := runtime.FuncForPC(v.Pointer()).Name()
	return s[strings.LastIndex(s, ".")+1:]
}

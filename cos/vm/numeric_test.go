package vm

import (
	"reflect"
	"testing"
)

func TestNumericOps(t *testing.T) {
	type testStruct struct {
		op      Op
		startVM *virtualMachine
		wantErr error
		wantVM  *virtualMachine
	}
	cases := []testStruct{{
		op: OP_1ADD,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}},
		},
		wantVM: &virtualMachine{
			runLimit:  49998,
			dataStack: [][]byte{{3}},
		},
	}, {
		op: OP_1SUB,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}},
		},
		wantVM: &virtualMachine{
			runLimit:  49998,
			dataStack: [][]byte{{1}},
		},
	}, {
		op: OP_2MUL,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}},
		},
		wantVM: &virtualMachine{
			runLimit:  49998,
			dataStack: [][]byte{{4}},
		},
	}, {
		op: OP_2DIV,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}},
		},
		wantVM: &virtualMachine{
			runLimit:  49998,
			dataStack: [][]byte{{1}},
		},
	}, {
		op: OP_2DIV,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{Int64Bytes(-2)},
		},
		wantVM: &virtualMachine{
			runLimit:  49998,
			dataStack: [][]byte{Int64Bytes(-1)},
		},
	}, {
		op: OP_NEGATE,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: 7,
			dataStack:    [][]byte{Int64Bytes(-2)},
		},
	}, {
		op: OP_ABS,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}},
		},
		wantVM: &virtualMachine{
			runLimit:  49998,
			dataStack: [][]byte{{2}},
		},
	}, {
		op: OP_ABS,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{Int64Bytes(-2)},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -7,
			dataStack:    [][]byte{{2}},
		},
	}, {
		op: OP_NOT,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -1,
			dataStack:    [][]byte{{}},
		},
	}, {
		op: OP_0NOTEQUAL,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}},
		},
		wantVM: &virtualMachine{
			runLimit:  49998,
			dataStack: [][]byte{{1}},
		},
	}, {
		op: OP_ADD,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -9,
			dataStack:    [][]byte{{3}},
		},
	}, {
		op: OP_SUB,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -9,
			dataStack:    [][]byte{{1}},
		},
	}, {
		op: OP_MUL,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49992,
			deferredCost: -9,
			dataStack:    [][]byte{{2}},
		},
	}, {
		op: OP_DIV,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49992,
			deferredCost: -9,
			dataStack:    [][]byte{{2}},
		},
	}, {
		op: OP_DIV,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{Int64Bytes(-2), {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49992,
			deferredCost: -9,
			dataStack:    [][]byte{Int64Bytes(-2)},
		},
	}, {
		op: OP_DIV,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{Int64Bytes(-2), Int64Bytes(-1)},
		},
		wantVM: &virtualMachine{
			runLimit:     49992,
			deferredCost: -23,
			dataStack:    [][]byte{{2}},
		},
	}, {
		op: OP_DIV,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {}},
		},
		wantErr: ErrDivZero,
	}, {
		op: OP_MOD,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49992,
			deferredCost: -10,
			dataStack:    [][]byte{{}},
		},
	}, {
		op: OP_MOD,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{Int64Bytes(-12), {10}},
		},
		wantVM: &virtualMachine{
			runLimit:     49992,
			deferredCost: -16,
			dataStack:    [][]byte{{8}},
		},
	}, {
		op: OP_MOD,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {0}},
		},
		wantErr: ErrDivZero,
	}, {
		op: OP_LSHIFT,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49992,
			deferredCost: -9,
			dataStack:    [][]byte{{4}},
		},
	}, {
		op: OP_LSHIFT,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{Int64Bytes(-2), {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49992,
			deferredCost: -9,
			dataStack:    [][]byte{Int64Bytes(-4)},
		},
	}, {
		op: OP_LSHIFT,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, Int64Bytes(-1)},
		},
		wantVM: &virtualMachine{
			runLimit:     49992,
			deferredCost: -16,
			dataStack:    [][]byte{{1}},
		},
	}, {
		op: OP_RSHIFT,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49992,
			deferredCost: -9,
			dataStack:    [][]byte{{1}},
		},
	}, {
		op: OP_RSHIFT,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{Int64Bytes(-2), {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49992,
			deferredCost: -9,
			dataStack:    [][]byte{Int64Bytes(-1)},
		},
	}, {
		op: OP_RSHIFT,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, Int64Bytes(-1)},
		},
		wantVM: &virtualMachine{
			runLimit:     49992,
			deferredCost: -16,
			dataStack:    [][]byte{{4}},
		},
	}, {
		op: OP_BOOLAND,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -9,
			dataStack:    [][]byte{{1}},
		},
	}, {
		op: OP_BOOLOR,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -9,
			dataStack:    [][]byte{{1}},
		},
	}, {
		op: OP_NUMEQUAL,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -10,
			dataStack:    [][]byte{{}},
		},
	}, {
		op: OP_NUMEQUALVERIFY,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {2}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -18,
			dataStack:    [][]byte{},
		},
	}, {
		op: OP_NUMEQUALVERIFY,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{1}, {2}},
		},
		wantErr: ErrVerifyFailed,
	}, {
		op: OP_NUMNOTEQUAL,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -9,
			dataStack:    [][]byte{{1}},
		},
	}, {
		op: OP_LESSTHAN,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -10,
			dataStack:    [][]byte{{}},
		},
	}, {
		op: OP_LESSTHANOREQUAL,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -10,
			dataStack:    [][]byte{{}},
		},
	}, {
		op: OP_GREATERTHAN,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -9,
			dataStack:    [][]byte{{1}},
		},
	}, {
		op: OP_GREATERTHANOREQUAL,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -9,
			dataStack:    [][]byte{{1}},
		},
	}, {
		op: OP_MIN,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -9,
			dataStack:    [][]byte{{1}},
		},
	}, {
		op: OP_MIN,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{1}, {2}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -9,
			dataStack:    [][]byte{{1}},
		},
	}, {
		op: OP_MAX,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{2}, {1}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -9,
			dataStack:    [][]byte{{2}},
		},
	}, {
		op: OP_MAX,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{1}, {2}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -9,
			dataStack:    [][]byte{{2}},
		},
	}, {
		op: OP_WITHIN,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{1}, {1}, {2}},
		},
		wantVM: &virtualMachine{
			runLimit:     49996,
			deferredCost: -18,
			dataStack:    [][]byte{{1}},
		},
	}, {
		op: OP_WITHIN,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{1}, {2}},
		},
		wantErr: ErrDataStackUnderflow,
	}}

	numops := []Op{
		OP_1ADD, OP_1SUB, OP_2MUL, OP_2DIV, OP_NEGATE, OP_ABS, OP_NOT, OP_0NOTEQUAL,
		OP_ADD, OP_SUB, OP_MUL, OP_DIV, OP_MOD, OP_LSHIFT, OP_RSHIFT, OP_BOOLAND,
		OP_BOOLOR, OP_NUMEQUAL, OP_NUMEQUALVERIFY, OP_NUMNOTEQUAL, OP_LESSTHAN,
		OP_LESSTHANOREQUAL, OP_GREATERTHAN, OP_GREATERTHANOREQUAL, OP_MIN, OP_MAX, OP_WITHIN,
	}
	twopop := numops[8:]

	for _, op := range numops {
		cases = append(cases, testStruct{
			op: op,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{},
			},
			wantErr: ErrDataStackUnderflow,
		}, testStruct{
			op: op,
			startVM: &virtualMachine{
				runLimit:  0,
				dataStack: [][]byte{{2}, {2}, {2}},
			},
			wantErr: ErrRunLimitExceeded,
		})
	}

	for _, op := range twopop {
		cases = append(cases, testStruct{
			op: op,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{{2}},
			},
			wantErr: ErrDataStackUnderflow,
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

		if !reflect.DeepEqual(c.startVM, c.wantVM) {
			t.Errorf("case %d, op %s: unexpected vm result\n\tgot:  %+v\n\twant: %+v\n", i, ops[c.op].name, c.startVM, c.wantVM)
		}
	}
}

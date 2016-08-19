package vm

import (
	"reflect"
	"testing"
)

func TestStackOps(t *testing.T) {
	type testStruct struct {
		op      uint8
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
		op: OP_NIP,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{1}},
		},
		wantErr: ErrDataStackUnderflow,
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
			runLimit:  50000,
			dataStack: [][]byte{{3}, {2}, {1}, {3}},
		},
		wantErr: ErrDataStackUnderflow,
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
		op: OP_ROLL,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{3}, {2}, {1}, {3}},
		},
		wantErr: ErrDataStackUnderflow,
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
	stackops := []uint8{
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
	for _, op := range stackops[2:] {
		cases = append(cases, testStruct{
			op:      op,
			startVM: &virtualMachine{runLimit: 50000, dataStack: [][]byte{}},
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

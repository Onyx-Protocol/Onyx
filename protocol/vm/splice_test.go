package vm

import (
	"testing"

	"chain/testutil"
)

func TestSpliceOps(t *testing.T) {
	type testStruct struct {
		op      Op
		startVM *virtualMachine
		wantErr error
		wantVM  *virtualMachine
	}
	cases := []testStruct{{
		op: OP_CAT,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{[]byte("hello"), []byte("world")},
		},
		wantVM: &virtualMachine{
			runLimit:     49986,
			deferredCost: -18,
			dataStack:    [][]byte{[]byte("helloworld")},
		},
	}, {
		op: OP_CAT,
		startVM: &virtualMachine{
			runLimit:  4,
			dataStack: [][]byte{[]byte("hello"), []byte("world")},
		},
		wantErr: ErrRunLimitExceeded,
	}, {
		op: OP_SUBSTR,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{[]byte("helloworld"), {3}, {5}},
		},
		wantVM: &virtualMachine{
			runLimit:     49991,
			deferredCost: -28,
			dataStack:    [][]byte{[]byte("lowor")},
		},
	}, {
		op: OP_SUBSTR,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{[]byte("helloworld"), {3}, Int64Bytes(-1)},
		},
		wantErr: ErrBadValue,
	}, {
		op: OP_SUBSTR,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{[]byte("helloworld"), Int64Bytes(-1), {5}},
		},
		wantErr: ErrBadValue,
	}, {
		op: OP_SUBSTR,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{[]byte("helloworld"), {6}, {5}},
		},
		wantErr: ErrBadValue,
	}, {
		op: OP_SUBSTR,
		startVM: &virtualMachine{
			runLimit:  4,
			dataStack: [][]byte{[]byte("helloworld"), {3}, {5}},
		},
		wantErr: ErrRunLimitExceeded,
	}, {
		op: OP_LEFT,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{[]byte("helloworld"), {5}},
		},
		wantVM: &virtualMachine{
			runLimit:     49991,
			deferredCost: -19,
			dataStack:    [][]byte{[]byte("hello")},
		},
	}, {
		op: OP_LEFT,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{[]byte("helloworld"), Int64Bytes(-1)},
		},
		wantErr: ErrBadValue,
	}, {
		op: OP_LEFT,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{[]byte("helloworld"), {11}},
		},
		wantErr: ErrBadValue,
	}, {
		op: OP_LEFT,
		startVM: &virtualMachine{
			runLimit:  4,
			dataStack: [][]byte{[]byte("helloworld"), {5}},
		},
		wantErr: ErrRunLimitExceeded,
	}, {
		op: OP_RIGHT,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{[]byte("helloworld"), {5}},
		},
		wantVM: &virtualMachine{
			runLimit:     49991,
			deferredCost: -19,
			dataStack:    [][]byte{[]byte("world")},
		},
	}, {
		op: OP_RIGHT,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{[]byte("helloworld"), Int64Bytes(-1)},
		},
		wantErr: ErrBadValue,
	}, {
		op: OP_RIGHT,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{[]byte("helloworld"), {11}},
		},
		wantErr: ErrBadValue,
	}, {
		op: OP_RIGHT,
		startVM: &virtualMachine{
			runLimit:  4,
			dataStack: [][]byte{[]byte("helloworld"), {5}},
		},
		wantErr: ErrRunLimitExceeded,
	}, {
		op: OP_SIZE,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{[]byte("helloworld")},
		},
		wantVM: &virtualMachine{
			runLimit:     49999,
			deferredCost: 9,
			dataStack:    [][]byte{[]byte("helloworld"), {10}},
		},
	}, {
		op: OP_CATPUSHDATA,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{0xff}, {0xab, 0xcd}},
		},
		wantVM: &virtualMachine{
			runLimit:     49993,
			deferredCost: -10,
			dataStack:    [][]byte{{0xff, 0x02, 0xab, 0xcd}},
		},
	}, {
		op: OP_CATPUSHDATA,
		startVM: &virtualMachine{
			runLimit:  4,
			dataStack: [][]byte{{0xff}, {0xab, 0xcd}},
		},
		wantErr: ErrRunLimitExceeded,
	}}

	spliceops := []Op{OP_CAT, OP_SUBSTR, OP_LEFT, OP_RIGHT, OP_CATPUSHDATA, OP_SIZE}
	for _, op := range spliceops {
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

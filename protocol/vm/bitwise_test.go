package vm

import (
	"testing"

	"chain/testutil"
)

func TestBitwiseOps(t *testing.T) {
	type testStruct struct {
		op      Op
		startVM *virtualMachine
		wantErr error
		wantVM  *virtualMachine
	}
	cases := []testStruct{{
		op: OP_INVERT,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{255}},
		},
		wantVM: &virtualMachine{
			runLimit:  49998,
			dataStack: [][]byte{{0}},
		},
	}, {
		op: OP_INVERT,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{255, 0}},
		},
		wantVM: &virtualMachine{
			runLimit:  49997,
			dataStack: [][]byte{{0, 255}},
		},
	}, {
		op: OP_AND,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{0xff}, {0x80}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -9,
			dataStack:    [][]byte{{0x80}},
		},
	}, {
		op: OP_AND,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{0xff}, {0x80, 0xff}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -10,
			dataStack:    [][]byte{{0x80}},
		},
	}, {
		op: OP_AND,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{0x80, 0xff}, {0xff}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -10,
			dataStack:    [][]byte{{0x80}},
		},
	}, {
		op: OP_OR,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{0xff}, {0x80}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -9,
			dataStack:    [][]byte{{0xff}},
		},
	}, {
		op: OP_OR,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{0xff}, {0x80, 0x10}},
		},
		wantVM: &virtualMachine{
			runLimit:     49997,
			deferredCost: -9,
			dataStack:    [][]byte{{0xff, 0x10}},
		},
	}, {
		op: OP_OR,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{0xff, 0x10}, {0x80}},
		},
		wantVM: &virtualMachine{
			runLimit:     49997,
			deferredCost: -9,
			dataStack:    [][]byte{{0xff, 0x10}},
		},
	}, {
		op: OP_XOR,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{0xff}, {0x80}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -9,
			dataStack:    [][]byte{{0x7f}},
		},
	}, {
		op: OP_XOR,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{0xff}, {0x80, 0x10}},
		},
		wantVM: &virtualMachine{
			runLimit:     49997,
			deferredCost: -9,
			dataStack:    [][]byte{{0x7f, 0x10}},
		},
	}, {
		op: OP_XOR,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{0xff, 0x10}, {0x80}},
		},
		wantVM: &virtualMachine{
			runLimit:     49997,
			deferredCost: -9,
			dataStack:    [][]byte{{0x7f, 0x10}},
		},
	}, {
		op: OP_EQUAL,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{0xff}, {0xff}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -9,
			dataStack:    [][]byte{{1}},
		},
	}, {
		op: OP_EQUAL,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{0xff, 0x10}, {0xff, 0x10}},
		},
		wantVM: &virtualMachine{
			runLimit:     49997,
			deferredCost: -11,
			dataStack:    [][]byte{{1}},
		},
	}, {
		op: OP_EQUAL,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{0xff}, {0x80}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -10,
			dataStack:    [][]byte{{}},
		},
	}, {
		op: OP_EQUAL,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{0xff}, {0xff, 0x80}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -11,
			dataStack:    [][]byte{{}},
		},
	}, {
		op: OP_EQUAL,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{0xff, 0x80}, {0xff}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -11,
			dataStack:    [][]byte{{}},
		},
	}, {
		op: OP_EQUALVERIFY,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{0xff}, {0xff}},
		},
		wantVM: &virtualMachine{
			runLimit:     49998,
			deferredCost: -18,
			dataStack:    [][]byte{},
		},
	}, {
		op: OP_EQUALVERIFY,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{0xff, 0x10}, {0xff, 0x10}},
		},
		wantVM: &virtualMachine{
			runLimit:     49997,
			deferredCost: -20,
			dataStack:    [][]byte{},
		},
	}, {
		op: OP_EQUALVERIFY,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{0xff}, {0x80}},
		},
		wantErr: ErrVerifyFailed,
	}, {
		op: OP_EQUALVERIFY,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{0xff}, {0xff, 0x80}},
		},
		wantErr: ErrVerifyFailed,
	}, {
		op: OP_EQUALVERIFY,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{0xff, 0x80}, {0xff}},
		},
		wantErr: ErrVerifyFailed,
	}}

	bitops := []Op{OP_INVERT, OP_AND, OP_OR, OP_XOR, OP_EQUAL, OP_EQUALVERIFY}
	for _, op := range bitops {
		cases = append(cases, testStruct{
			op: op,
			startVM: &virtualMachine{
				runLimit:  0,
				dataStack: [][]byte{{0xff}, {0xff}},
			},
			wantErr: ErrRunLimitExceeded,
		}, testStruct{
			op: op,
			startVM: &virtualMachine{
				runLimit:  1,
				dataStack: [][]byte{{0xff}, {0xff}},
			},
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

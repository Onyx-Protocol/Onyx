package vm

import (
	"testing"

	"chain/testutil"
)

func TestControlOps(t *testing.T) {
	type testStruct struct {
		op      Op
		startVM *virtualMachine
		wantErr error
		wantVM  *virtualMachine
	}
	cases := []testStruct{{
		op: OP_JUMP,
		startVM: &virtualMachine{
			runLimit: 50000,
			pc:       0,
			nextPC:   1,
			data:     []byte{0x05, 0x00, 0x00, 0x00},
		},
		wantErr: nil,
		wantVM: &virtualMachine{
			runLimit: 49999,
			pc:       0,
			nextPC:   5,
			data:     []byte{0x05, 0x00, 0x00, 0x00},
		},
	}, {
		op: OP_JUMP,
		startVM: &virtualMachine{
			runLimit: 50000,
			pc:       0,
			nextPC:   1,
			data:     []byte{0xff, 0xff, 0xff, 0xff},
		},
		wantErr: nil,
		wantVM: &virtualMachine{
			runLimit: 49999,
			pc:       0,
			nextPC:   4294967295,
			data:     []byte{0xff, 0xff, 0xff, 0xff},
		},
	}, {
		op: OP_JUMPIF,
		startVM: &virtualMachine{
			runLimit:     50000,
			pc:           0,
			nextPC:       1,
			deferredCost: 0,
			dataStack:    [][]byte{{1}},
			data:         []byte{0x05, 0x00, 0x00, 0x00},
		},
		wantErr: nil,
		wantVM: &virtualMachine{
			runLimit:     49999,
			pc:           0,
			nextPC:       5,
			deferredCost: -9,
			dataStack:    [][]byte{},
			data:         []byte{0x05, 0x00, 0x00, 0x00},
		},
	}, {
		op: OP_JUMPIF,
		startVM: &virtualMachine{
			runLimit:     50000,
			pc:           0,
			nextPC:       1,
			deferredCost: 0,
			dataStack:    [][]byte{{}},
			data:         []byte{0x05, 0x00, 0x00, 0x00},
		},
		wantErr: nil,
		wantVM: &virtualMachine{
			runLimit:     49999,
			pc:           0,
			nextPC:       1,
			deferredCost: -8,
			dataStack:    [][]byte{},
			data:         []byte{0x05, 0x00, 0x00, 0x00},
		},
	}, {
		op: OP_VERIFY,
		startVM: &virtualMachine{
			pc:           0,
			runLimit:     50000,
			deferredCost: 0,
			dataStack:    [][]byte{{1}},
		},
		wantErr: nil,
		wantVM: &virtualMachine{
			runLimit:     49999,
			deferredCost: -9,
			dataStack:    [][]byte{},
		},
	}, {
		op: OP_VERIFY,
		startVM: &virtualMachine{
			runLimit:     50000,
			deferredCost: 0,
			dataStack:    [][]byte{{1, 1}},
		},
		wantErr: nil,
		wantVM: &virtualMachine{
			runLimit:     49999,
			deferredCost: -10,
			dataStack:    [][]byte{},
		},
	}, {
		op: OP_VERIFY,
		startVM: &virtualMachine{
			runLimit:     50000,
			deferredCost: 0,
			dataStack:    [][]byte{{}},
		},
		wantErr: ErrVerifyFailed,
	}, {
		startVM: &virtualMachine{runLimit: 50000},
		op:      OP_FAIL,
		wantErr: ErrReturn,
	}, {
		op: OP_CHECKPREDICATE,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{}, {byte(OP_TRUE)}, {}},
		},
		wantVM: &virtualMachine{
			runLimit:     0,
			deferredCost: -49951,
			dataStack:    [][]byte{{1}},
		},
	}, {
		op: OP_CHECKPREDICATE,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{}, {}, {}},
		},
		wantVM: &virtualMachine{
			runLimit:     0,
			deferredCost: -49952,
			dataStack:    [][]byte{{}},
		},
	}, {
		op: OP_CHECKPREDICATE,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{}, {byte(OP_FAIL)}, {}},
		},
		wantVM: &virtualMachine{
			runLimit:     0,
			deferredCost: -49952,
			dataStack:    [][]byte{{}},
		},
	}, {
		op: OP_CHECKPREDICATE,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{}, {}, Int64Bytes(-1)},
		},
		wantErr: ErrBadValue,
	}, {
		op: OP_CHECKPREDICATE,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{}, {}, Int64Bytes(50000)},
		},
		wantErr: ErrRunLimitExceeded,
	}, {
		op: OP_CHECKPREDICATE,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{0x05}, {0x07}, {0x02}, {byte(OP_ADD), byte(OP_12), byte(OP_NUMEQUAL)}, {}},
		},
		wantVM: &virtualMachine{
			deferredCost: -49968,
			dataStack:    [][]byte{{0x01}},
		},
	}, {
		// stack underflow in child vm should produce false result in parent vm
		op: OP_CHECKPREDICATE,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{0x05}, {0x07}, {0x01}, {byte(OP_ADD), byte(OP_DATA_12), byte(OP_NUMEQUAL)}, {}},
		},
		wantVM: &virtualMachine{
			deferredCost: -49954,
			dataStack:    [][]byte{{0x05}, {}},
		},
	}}

	limitChecks := []Op{
		OP_CHECKPREDICATE, OP_VERIFY, OP_FAIL,
	}

	for _, op := range limitChecks {
		cases = append(cases, testStruct{
			op:      op,
			startVM: &virtualMachine{runLimit: 0},
			wantErr: ErrRunLimitExceeded,
		})
	}

	for i, c := range cases {
		err := ops[c.op].fn(c.startVM)

		if err != c.wantErr {
			t.Errorf("case %d, op %s: got err = %v want %v", i, c.op.String(), err, c.wantErr)
			continue
		}
		if c.wantErr != nil {
			continue
		}

		if !testutil.DeepEqual(c.startVM, c.wantVM) {
			t.Errorf("case %d, op %s: unexpected vm result\n\tgot:  %+v\n\twant: %+v\n", i, c.op.String(), c.startVM, c.wantVM)
		}
	}
}

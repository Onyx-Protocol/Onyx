package vm

import (
	"reflect"
	"testing"
)

func TestControlOps(t *testing.T) {
	type testStruct struct {
		op      Op
		startVM *virtualMachine
		wantErr error
		wantVM  *virtualMachine
	}
	cases := []testStruct{{
		op: OP_IF,
		startVM: &virtualMachine{
			runLimit:     50000,
			deferredCost: 0,
			dataStack:    [][]byte{{1}},
		},
		wantErr: nil,
		wantVM: &virtualMachine{
			runLimit:     49996,
			deferredCost: -9,
			dataStack:    [][]byte{},
			controlStack: []controlTuple{{optype: cfIf, flag: true}},
		},
	}, {
		op: OP_IF,
		startVM: &virtualMachine{
			runLimit:     50000,
			deferredCost: 0,
			dataStack:    [][]byte{{1, 1}},
		},
		wantErr: nil,
		wantVM: &virtualMachine{
			runLimit:     49996,
			deferredCost: -10,
			dataStack:    [][]byte{},
			controlStack: []controlTuple{{optype: cfIf, flag: true}},
		},
	}, {
		op: OP_IF,
		startVM: &virtualMachine{
			runLimit:     50000,
			deferredCost: 0,
			dataStack:    [][]byte{{}},
		},
		wantErr: nil,
		wantVM: &virtualMachine{
			runLimit:     49996,
			deferredCost: -8,
			dataStack:    [][]byte{},
			controlStack: []controlTuple{{optype: cfIf, flag: false}},
		},
	}, {
		op: OP_IF,
		startVM: &virtualMachine{
			runLimit:     50000,
			dataStack:    [][]byte{{1}},
			controlStack: []controlTuple{{optype: cfIf, flag: false}},
		},
		wantErr: nil,
		wantVM: &virtualMachine{
			runLimit:     49996,
			dataStack:    [][]byte{{1}},
			controlStack: []controlTuple{{optype: cfIf, flag: false}, {optype: cfIf, flag: false}},
		},
	}, {
		op: OP_IF,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{},
		},
		wantErr: ErrDataStackUnderflow,
	}, {
		op: OP_ELSE,
		startVM: &virtualMachine{
			runLimit:     50000,
			controlStack: []controlTuple{{optype: cfIf, flag: true}},
		},
		wantErr: nil,
		wantVM: &virtualMachine{
			runLimit:     49996,
			controlStack: []controlTuple{{optype: cfElse, flag: false}},
		},
	}, {
		op: OP_ELSE,
		startVM: &virtualMachine{
			runLimit:     50000,
			controlStack: []controlTuple{{optype: cfIf, flag: false}},
		},
		wantErr: nil,
		wantVM: &virtualMachine{
			runLimit:     49996,
			controlStack: []controlTuple{{optype: cfElse, flag: true}},
		},
	}, {
		op: OP_ELSE,
		startVM: &virtualMachine{
			runLimit:     50000,
			controlStack: []controlTuple{{optype: cfIf, flag: false}, {optype: cfIf, flag: false}},
		},
		wantErr: nil,
		wantVM: &virtualMachine{
			runLimit:     49996,
			controlStack: []controlTuple{{optype: cfIf, flag: false}, {optype: cfElse, flag: false}},
		},
	}, {
		op: OP_ELSE,
		startVM: &virtualMachine{
			runLimit:     50000,
			controlStack: []controlTuple{{optype: cfElse, flag: true}},
		},
		wantErr: ErrBadControlSyntax,
	}, {
		op: OP_ELSE,
		startVM: &virtualMachine{
			runLimit:     50000,
			controlStack: []controlTuple{{optype: cfWhile, flag: true}},
		},
		wantErr: ErrBadControlSyntax,
	}, {
		op: OP_ELSE,
		startVM: &virtualMachine{
			runLimit:     50000,
			controlStack: []controlTuple{},
		},
		wantErr: ErrControlStackUnderflow,
	}, {
		op: OP_ENDIF,
		startVM: &virtualMachine{
			runLimit:     50000,
			controlStack: []controlTuple{{optype: cfIf, flag: true}},
		},
		wantErr: nil,
		wantVM: &virtualMachine{
			runLimit:     49999,
			controlStack: []controlTuple{},
		},
	}, {
		op: OP_ENDIF,
		startVM: &virtualMachine{
			runLimit:     50000,
			controlStack: []controlTuple{{optype: cfElse, flag: false}},
		},
		wantErr: nil,
		wantVM: &virtualMachine{
			runLimit:     49999,
			controlStack: []controlTuple{},
		},
	}, {
		op: OP_ENDIF,
		startVM: &virtualMachine{
			runLimit:     50000,
			controlStack: []controlTuple{{optype: cfWhile, flag: true}},
		},
		wantErr: ErrBadControlSyntax,
	}, {
		op: OP_ENDIF,
		startVM: &virtualMachine{
			runLimit:     50000,
			controlStack: []controlTuple{},
		},
		wantErr: ErrControlStackUnderflow,
	}, {
		op: OP_VERIFY,
		startVM: &virtualMachine{
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
		op: OP_VERIFY,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{},
		},
		wantErr: ErrDataStackUnderflow,
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
			runLimit: 50000,
		},
		wantErr: ErrDataStackUnderflow,
	}, {
		op: OP_CHECKPREDICATE,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{}},
		},
		wantErr: ErrDataStackUnderflow,
	}, {
		op: OP_CHECKPREDICATE,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{}, {}},
		},
		wantErr: ErrDataStackUnderflow,
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
	}, {
		op: OP_WHILE,
		startVM: &virtualMachine{
			runLimit:     50000,
			deferredCost: 0,
			dataStack:    [][]byte{{1}},
			pc:           5,
		},
		wantVM: &virtualMachine{
			runLimit:     49996,
			dataStack:    [][]byte{{1}},
			controlStack: []controlTuple{{optype: cfWhile, flag: true, pc: 5}},
			pc:           5,
		},
	}, {
		op: OP_WHILE,
		startVM: &virtualMachine{
			runLimit:     50000,
			deferredCost: 0,
			dataStack:    [][]byte{{}},
			pc:           5,
		},
		wantVM: &virtualMachine{
			runLimit:     49996,
			deferredCost: -8,
			dataStack:    [][]byte{},
			controlStack: []controlTuple{{optype: cfWhile, flag: false, pc: 5}},
			pc:           5,
		},
	}, {
		op: OP_WHILE,
		startVM: &virtualMachine{
			runLimit:     50000,
			deferredCost: 0,
			dataStack:    [][]byte{{}},
			controlStack: []controlTuple{{optype: cfIf, flag: false}},
			pc:           5,
		},
		wantVM: &virtualMachine{
			runLimit:     49996,
			deferredCost: 0,
			dataStack:    [][]byte{{}},
			controlStack: []controlTuple{{optype: cfIf, flag: false}, {optype: cfWhile, flag: false}},
			pc:           5,
		},
	}, {
		op: OP_WHILE,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{},
		},
		wantErr: ErrDataStackUnderflow,
	}, {
		op: OP_ENDWHILE,
		startVM: &virtualMachine{
			runLimit:     50000,
			controlStack: []controlTuple{{optype: cfWhile, flag: true, pc: 5}},
		},
		wantVM: &virtualMachine{
			runLimit:     49999,
			controlStack: []controlTuple{},
			nextPC:       5,
		},
	}, {
		op: OP_ENDWHILE,
		startVM: &virtualMachine{
			runLimit:     50000,
			controlStack: []controlTuple{{optype: cfWhile, flag: false, pc: 5}},
		},
		wantVM: &virtualMachine{
			runLimit:     49999,
			controlStack: []controlTuple{},
		},
	}, {
		op: OP_ENDWHILE,
		startVM: &virtualMachine{
			runLimit:     50000,
			controlStack: []controlTuple{},
		},
		wantErr: ErrControlStackUnderflow,
	}, {
		op: OP_ENDWHILE,
		startVM: &virtualMachine{
			runLimit:     50000,
			controlStack: []controlTuple{{optype: cfIf, flag: true}},
		},
		wantErr: ErrBadControlSyntax,
	}, {
		op: OP_ENDWHILE,
		startVM: &virtualMachine{
			runLimit:     50000,
			controlStack: []controlTuple{{optype: cfElse, flag: true}},
		},
		wantErr: ErrBadControlSyntax,
	}}

	limitChecks := []Op{
		OP_IF, OP_NOTIF, OP_ELSE, OP_ENDIF, OP_WHILE, OP_ENDWHILE,
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

		if !reflect.DeepEqual(c.startVM, c.wantVM) {
			t.Errorf("case %d, op %s: unexpected vm result\n\tgot:  %+v\n\twant: %+v\n", i, c.op.String(), c.startVM, c.wantVM)
		}
	}
}

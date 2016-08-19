package vm

import (
	"reflect"
	"testing"
)

func TestPushdataOps(t *testing.T) {
	type testStruct struct {
		op      uint8
		startVM *virtualMachine
		wantErr error
		wantVM  *virtualMachine
	}
	cases := []testStruct{{
		op: OP_FALSE,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{},
		},
		wantVM: &virtualMachine{
			runLimit:  49991,
			dataStack: [][]byte{{}},
		},
	}, {
		op: OP_FALSE,
		startVM: &virtualMachine{
			runLimit:  1,
			dataStack: [][]byte{},
		},
		wantErr: ErrRunLimitExceeded,
	}, {
		op: OP_1NEGATE,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{},
		},
		wantVM: &virtualMachine{
			runLimit:  49983,
			dataStack: [][]byte{Int64Bytes(-1)},
		},
	}, {
		op: OP_1NEGATE,
		startVM: &virtualMachine{
			runLimit:  1,
			dataStack: [][]byte{},
		},
		wantErr: ErrRunLimitExceeded,
	}}

	pushdataops := []uint8{OP_PUSHDATA1, OP_PUSHDATA2, OP_PUSHDATA4}
	for i := 1; i <= 75; i++ {
		pushdataops = append(pushdataops, uint8(i))
	}
	for _, op := range pushdataops {
		cases = append(cases, testStruct{
			op: op,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{},
				data:      []byte("data"),
			},
			wantVM: &virtualMachine{
				runLimit:  49987,
				dataStack: [][]byte{[]byte("data")},
				data:      []byte("data"),
			},
		}, testStruct{
			op: op,
			startVM: &virtualMachine{
				runLimit:  1,
				dataStack: [][]byte{},
				data:      []byte("data"),
			},
			wantErr: ErrRunLimitExceeded,
		})
	}

	pushops := append(pushdataops, OP_FALSE, OP_1NEGATE, OP_NOP)
	for _, op := range pushops {
		cases = append(cases, testStruct{
			op: op,
			startVM: &virtualMachine{
				runLimit:  0,
				dataStack: [][]byte{},
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

		if !reflect.DeepEqual(c.startVM, c.wantVM) {
			t.Errorf("case %d, op %s: unexpected vm result\n\tgot:  %+v\n\twant: %+v\n", i, ops[c.op].name, c.startVM, c.wantVM)
		}
	}
}

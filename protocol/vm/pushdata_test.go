package vm

import (
	"bytes"
	"testing"

	"chain/testutil"
)

func TestPushdataOps(t *testing.T) {
	type testStruct struct {
		op      Op
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

	pushdataops := []Op{OP_PUSHDATA1, OP_PUSHDATA2, OP_PUSHDATA4}
	for i := 1; i <= 75; i++ {
		pushdataops = append(pushdataops, Op(i))
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

		if !testutil.DeepEqual(c.startVM, c.wantVM) {
			t.Errorf("case %d, op %s: unexpected vm result\n\tgot:  %+v\n\twant: %+v\n", i, ops[c.op].name, c.startVM, c.wantVM)
		}
	}
}

func TestPushDataBytes(t *testing.T) {
	type test struct {
		data []byte
		want []byte
	}
	cases := []test{{
		data: nil,
		want: []byte{byte(OP_0)},
	}, {
		data: make([]byte, 255),
		want: append([]byte{byte(OP_PUSHDATA1), 0xff}, make([]byte, 255)...),
	}, {
		data: make([]byte, 1<<8),
		want: append([]byte{byte(OP_PUSHDATA2), 0, 1}, make([]byte, 1<<8)...),
	}, {
		data: make([]byte, 1<<16),
		want: append([]byte{byte(OP_PUSHDATA4), 0, 0, 1, 0}, make([]byte, 1<<16)...),
	}}

	for i := 1; i <= 75; i++ {
		cases = append(cases, test{
			data: make([]byte, i),
			want: append([]byte{byte(OP_DATA_1) - 1 + byte(i)}, make([]byte, i)...),
		})
	}

	for _, c := range cases {
		got := PushdataBytes(c.data)

		dl := len(c.data)
		if dl > 10 {
			dl = 10
		}
		if !bytes.Equal(got, c.want) {
			t.Errorf("PushdataBytes(%x...) = %x...[%d] want %x...[%d]", c.data[:dl], got[:dl], len(got), c.want[:dl], len(c.want))
		}
	}
}

func TestPushdataInt64(t *testing.T) {
	type test struct {
		num  int64
		want []byte
	}
	cases := []test{{
		num:  0,
		want: []byte{byte(OP_0)},
	}, {
		num:  17,
		want: []byte{byte(OP_DATA_1), 0x11},
	}, {
		num:  255,
		want: []byte{byte(OP_DATA_1), 0xff},
	}, {
		num:  256,
		want: []byte{byte(OP_DATA_2), 0x00, 0x01},
	}, {
		num:  -1,
		want: []byte{byte(OP_DATA_8), 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
	}, {
		num:  -2,
		want: []byte{byte(OP_DATA_8), 0xfe, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
	}}

	for i := 1; i <= 16; i++ {
		cases = append(cases, test{
			num:  int64(i),
			want: []byte{byte(OP_1) - 1 + byte(i)},
		})
	}

	for _, c := range cases {
		got := PushdataInt64(c.num)

		if !bytes.Equal(got, c.want) {
			t.Errorf("PushdataInt64(%d) = %x want %x", c.num, got, c.want)
		}
	}
}

package vm

import (
	"chain/cos/bc"
	"reflect"
	"testing"
	"time"
)

func TestIntrospectionOps(t *testing.T) {
	now := time.Now()
	tx := bc.NewTx(bc.TxData{
		Inputs: []*bc.TxInput{
			bc.NewSpendInput(bc.Hash{}, 0, nil, bc.AssetID{1}, 5, []byte("spendprog"), []byte("ref")),
			bc.NewIssuanceInput(now, now.Add(time.Minute), bc.Hash{}, 6, []byte("issueprog"), nil, nil),
		},
		Outputs: []*bc.TxOutput{
			bc.NewTxOutput(bc.AssetID{3}, 8, []byte("wrongprog"), nil),
			bc.NewTxOutput(bc.AssetID{3}, 8, []byte("controlprog"), nil),
			bc.NewTxOutput(bc.AssetID{2}, 8, []byte("controlprog"), nil),
			bc.NewTxOutput(bc.AssetID{2}, 7, []byte("controlprog"), nil),
			bc.NewTxOutput(bc.AssetID{2}, 7, []byte("controlprog"), []byte("outref")),
		},
		MinTime: 0,
		MaxTime: 20,
	})

	type testStruct struct {
		op      uint8
		startVM *virtualMachine
		wantErr error
		wantVM  *virtualMachine
	}
	cases := []testStruct{{
		op: OP_FINDOUTPUT,
		startVM: &virtualMachine{
			runLimit: 50000,
			tx:       tx,
			dataStack: [][]byte{
				mustDecodeHex("1f2a05f881ed9fa0c9068a84823677409f863891a2196eb55dbfbb677a566374"),
				{7},
				append([]byte{2}, make([]byte, 31)...),
				[]byte("controlprog"),
			},
		},
		wantVM: &virtualMachine{
			runLimit:     49984,
			deferredCost: -99,
			tx:           tx,
			dataStack:    [][]byte{{1}},
		},
	}, {
		op: OP_FINDOUTPUT,
		startVM: &virtualMachine{
			runLimit: 50000,
			tx:       tx,
			dataStack: [][]byte{
				[]byte{},
				{1},
				append([]byte{9}, make([]byte, 31)...),
				[]byte("missingprog"),
			},
		},
		wantVM: &virtualMachine{
			runLimit:     49984,
			deferredCost: -68,
			tx:           tx,
			dataStack:    [][]byte{{}},
		},
	}, {
		op: OP_FINDOUTPUT,
		startVM: &virtualMachine{
			runLimit: 50000,
			tx:       nil,
			dataStack: [][]byte{
				mustDecodeHex("1f2a05f881ed9fa0c9068a84823677409f863891a2196eb55dbfbb677a566374"),
				{7},
				append([]byte{2}, make([]byte, 31)...),
				[]byte("controlprog"),
			},
		},
		wantErr: ErrContext,
	}, {
		op: OP_FINDOUTPUT,
		startVM: &virtualMachine{
			runLimit:  50000,
			tx:        tx,
			dataStack: [][]byte{},
		},
		wantErr: ErrDataStackUnderflow,
	}, {
		op: OP_FINDOUTPUT,
		startVM: &virtualMachine{
			runLimit: 50000,
			tx:       tx,
			dataStack: [][]byte{
				[]byte("controlprog"),
			},
		},
		wantErr: ErrDataStackUnderflow,
	}, {
		op: OP_FINDOUTPUT,
		startVM: &virtualMachine{
			runLimit: 50000,
			tx:       tx,
			dataStack: [][]byte{
				append([]byte{2}, make([]byte, 31)...),
				[]byte("controlprog"),
			},
		},
		wantErr: ErrDataStackUnderflow,
	}, {
		op: OP_FINDOUTPUT,
		startVM: &virtualMachine{
			runLimit: 50000,
			tx:       tx,
			dataStack: [][]byte{
				{7},
				append([]byte{2}, make([]byte, 31)...),
				[]byte("controlprog"),
			},
		},
		wantErr: ErrDataStackUnderflow,
	}, {
		op: OP_FINDOUTPUT,
		startVM: &virtualMachine{
			runLimit: 50000,
			tx:       tx,
			dataStack: [][]byte{
				mustDecodeHex("1f2a05f881ed9fa0c9068a84823677409f863891a2196eb55dbfbb677a566374"),
				Int64Bytes(-1),
				append([]byte{2}, make([]byte, 31)...),
				[]byte("controlprog"),
			},
		},
		wantErr: ErrBadValue,
	}, {
		op: OP_FINDOUTPUT,
		startVM: &virtualMachine{
			runLimit: 0,
			tx:       tx,
			dataStack: [][]byte{
				mustDecodeHex("1f2a05f881ed9fa0c9068a84823677409f863891a2196eb55dbfbb677a566374"),
				{7},
				append([]byte{2}, make([]byte, 31)...),
				[]byte("controlprog"),
			},
		},
		wantErr: ErrRunLimitExceeded,
	}, {
		op: OP_ASSET,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{},
			tx:        tx,
		},
		wantVM: &virtualMachine{
			runLimit:     49999,
			deferredCost: 40,
			dataStack:    [][]byte{append([]byte{1}, make([]byte, 31)...)},
			tx:           tx,
		},
	}, {
		op: OP_AMOUNT,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{},
			tx:        tx,
		},
		wantVM: &virtualMachine{
			runLimit:     49999,
			deferredCost: 9,
			dataStack:    [][]byte{{5}},
			tx:           tx,
		},
	}, {
		op: OP_PROGRAM,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{},
			tx:        tx,
		},
		wantVM: &virtualMachine{
			runLimit:     49999,
			deferredCost: 17,
			dataStack:    [][]byte{[]byte("spendprog")},
			tx:           tx,
		},
	}, {
		op: OP_PROGRAM,
		startVM: &virtualMachine{
			runLimit:   50000,
			dataStack:  [][]byte{},
			tx:         tx,
			inputIndex: 1,
		},
		wantVM: &virtualMachine{
			runLimit:     49999,
			deferredCost: 17,
			dataStack:    [][]byte{[]byte("issueprog")},
			tx:           tx,
			inputIndex:   1,
		},
	}, {
		op: OP_MINTIME,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{},
			tx:        tx,
		},
		wantVM: &virtualMachine{
			runLimit:     49999,
			deferredCost: 8,
			dataStack:    [][]byte{{}},
			tx:           tx,
		},
	}, {
		op: OP_MAXTIME,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{},
			tx:        tx,
		},
		wantVM: &virtualMachine{
			runLimit:     49999,
			deferredCost: 9,
			dataStack:    [][]byte{{20}},
			tx:           tx,
		},
	}, {
		op: OP_REFDATAHASH,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{},
			tx:        tx,
		},
		wantVM: &virtualMachine{
			runLimit:     49999,
			deferredCost: 40,
			dataStack: [][]byte{{
				68, 190, 94, 20, 206, 33, 111, 75, 44, 53, 165, 235, 11, 53, 208, 120,
				189, 165, 92, 240, 91, 93, 54, 238, 14, 122, 1, 251, 198, 239, 98, 183,
			}},
			tx: tx,
		},
	}, {
		op: OP_INDEX,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{},
			tx:        tx,
		},
		wantVM: &virtualMachine{
			runLimit:     49999,
			deferredCost: 8,
			dataStack:    [][]byte{{}},
			tx:           tx,
		},
	}}

	txops := []uint8{
		OP_FINDOUTPUT, OP_ASSET, OP_AMOUNT, OP_PROGRAM,
		OP_MINTIME, OP_MAXTIME, OP_REFDATAHASH, OP_INDEX,
	}

	for _, op := range txops {
		cases = append(cases, testStruct{
			op: op,
			startVM: &virtualMachine{
				runLimit:  0,
				dataStack: [][]byte{},
				tx:        tx,
			},
			wantErr: ErrRunLimitExceeded,
		}, testStruct{
			op: op,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{},
				tx:        nil,
			},
			wantErr: ErrContext,
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

		c.wantVM.sigHasher = c.startVM.sigHasher
		if !reflect.DeepEqual(c.startVM, c.wantVM) {
			t.Errorf("case %d, op %s: unexpected vm result\n\tgot:  %+v\n\twant: %+v\n", i, ops[c.op].name, c.startVM, c.wantVM)
		}
	}
}

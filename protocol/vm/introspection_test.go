package vm

import (
	"reflect"
	"testing"

	"chain/protocol/bc"
)

func TestOutpointAndNonceOp(t *testing.T) {
	var zeroHash bc.Hash
	nonce := []byte{36, 37, 38}
	tx := bc.NewTx(bc.TxData{
		Inputs: []*bc.TxInput{
			bc.NewSpendInput(zeroHash, 0, nil, bc.AssetID{1}, 5, []byte("spendprog"), []byte("ref")),
			bc.NewIssuanceInput(nonce, 6, nil, zeroHash, []byte("issueprog"), nil),
		},
	})
	vm := &virtualMachine{
		runLimit:   50000,
		tx:         tx,
		inputIndex: 0,
		program:    []byte{uint8(OP_OUTPOINT)},
	}
	err := vm.step()
	if err != nil {
		t.Fatal(err)
	}
	expectedStack := [][]byte{zeroHash[:], []byte{}}
	if !reflect.DeepEqual(vm.dataStack, expectedStack) {
		t.Errorf("expected stack %v, got %v", expectedStack, vm.dataStack)
	}

	vm = &virtualMachine{
		runLimit:   50000,
		tx:         tx,
		inputIndex: 1,
		program:    []byte{uint8(OP_OUTPOINT)},
	}
	err = vm.step()
	if err != ErrContext {
		t.Errorf("expected ErrContext, got %v", err)
	}

	vm = &virtualMachine{
		runLimit:   50000,
		tx:         tx,
		inputIndex: 0,
		program:    []byte{uint8(OP_NONCE)},
	}
	err = vm.step()
	if err != ErrContext {
		t.Errorf("expected ErrContext, got %v", err)
	}
	vm = &virtualMachine{
		runLimit:   50000,
		tx:         tx,
		inputIndex: 1,
		program:    []byte{uint8(OP_NONCE)},
	}
	err = vm.step()
	if err != nil {
		t.Fatal(err)
	}
	expectedStack = [][]byte{nonce}
	if !reflect.DeepEqual(vm.dataStack, expectedStack) {
		t.Errorf("expected stack %v, got %v", expectedStack, vm.dataStack)
	}
}

func TestIntrospectionOps(t *testing.T) {
	tx := bc.NewTx(bc.TxData{
		ReferenceData: []byte("txref"),
		Inputs: []*bc.TxInput{
			bc.NewSpendInput(bc.Hash{}, 0, nil, bc.AssetID{1}, 5, []byte("spendprog"), []byte("ref")),
			bc.NewIssuanceInput(nil, 6, nil, bc.Hash{}, []byte("issueprog"), nil),
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
		op      Op
		startVM *virtualMachine
		wantErr error
		wantVM  *virtualMachine
	}
	cases := []testStruct{{
		op: OP_CHECKOUTPUT,
		startVM: &virtualMachine{
			runLimit: 50000,
			tx:       tx,
			dataStack: [][]byte{
				{4},
				mustDecodeHex("1f2a05f881ed9fa0c9068a84823677409f863891a2196eb55dbfbb677a566374"),
				{7},
				append([]byte{2}, make([]byte, 31)...),
				{1},
				[]byte("controlprog"),
			},
		},
		wantVM: &virtualMachine{
			runLimit:     49984,
			deferredCost: -117,
			tx:           tx,
			dataStack:    [][]byte{{1}},
		},
	}, {
		op: OP_CHECKOUTPUT,
		startVM: &virtualMachine{
			runLimit: 50000,
			tx:       tx,
			dataStack: [][]byte{
				{3},
				mustDecodeHex("1f2a05f881ed9fa0c9068a84823677409f863891a2196eb55dbfbb677a566374"),
				{7},
				append([]byte{2}, make([]byte, 31)...),
				{1},
				[]byte("controlprog"),
			},
		},
		wantVM: &virtualMachine{
			runLimit:     49984,
			deferredCost: -118,
			tx:           tx,
			dataStack:    [][]byte{{}},
		},
	}, {
		op: OP_CHECKOUTPUT,
		startVM: &virtualMachine{
			runLimit: 50000,
			tx:       tx,
			dataStack: [][]byte{
				{0},
				[]byte{},
				{1},
				append([]byte{9}, make([]byte, 31)...),
				{1},
				[]byte("missingprog"),
			},
		},
		wantVM: &virtualMachine{
			runLimit:     49984,
			deferredCost: -86,
			tx:           tx,
			dataStack:    [][]byte{{}},
		},
	}, {
		op: OP_CHECKOUTPUT,
		startVM: &virtualMachine{
			runLimit: 50000,
			tx:       nil,
			dataStack: [][]byte{
				{0},
				mustDecodeHex("1f2a05f881ed9fa0c9068a84823677409f863891a2196eb55dbfbb677a566374"),
				{7},
				append([]byte{2}, make([]byte, 31)...),
				{1},
				[]byte("controlprog"),
			},
		},
		wantErr: ErrContext,
	}, {
		op: OP_CHECKOUTPUT,
		startVM: &virtualMachine{
			runLimit:  50000,
			tx:        tx,
			dataStack: [][]byte{},
		},
		wantErr: ErrDataStackUnderflow,
	}, {
		op: OP_CHECKOUTPUT,
		startVM: &virtualMachine{
			runLimit: 50000,
			tx:       tx,
			dataStack: [][]byte{
				[]byte("controlprog"),
			},
		},
		wantErr: ErrDataStackUnderflow,
	}, {
		op: OP_CHECKOUTPUT,
		startVM: &virtualMachine{
			runLimit: 50000,
			tx:       tx,
			dataStack: [][]byte{
				append([]byte{2}, make([]byte, 31)...),
				{1},
				[]byte("controlprog"),
			},
		},
		wantErr: ErrDataStackUnderflow,
	}, {
		op: OP_CHECKOUTPUT,
		startVM: &virtualMachine{
			runLimit: 50000,
			tx:       tx,
			dataStack: [][]byte{
				{7},
				append([]byte{2}, make([]byte, 31)...),
				{1},
				[]byte("controlprog"),
			},
		},
		wantErr: ErrDataStackUnderflow,
	}, {
		op: OP_CHECKOUTPUT,
		startVM: &virtualMachine{
			runLimit: 50000,
			tx:       tx,
			dataStack: [][]byte{
				mustDecodeHex("1f2a05f881ed9fa0c9068a84823677409f863891a2196eb55dbfbb677a566374"),
				{7},
				append([]byte{2}, make([]byte, 31)...),
				{1},
				[]byte("controlprog"),
			},
		},
		wantErr: ErrDataStackUnderflow,
	}, {
		op: OP_CHECKOUTPUT,
		startVM: &virtualMachine{
			runLimit: 50000,
			tx:       tx,
			dataStack: [][]byte{
				{4},
				mustDecodeHex("1f2a05f881ed9fa0c9068a84823677409f863891a2196eb55dbfbb677a566374"),
				{7},
				append([]byte{2}, make([]byte, 31)...),
				Int64Bytes(-1),
				[]byte("controlprog"),
			},
		},
		wantErr: ErrBadValue,
	}, {
		op: OP_CHECKOUTPUT,
		startVM: &virtualMachine{
			runLimit: 50000,
			tx:       tx,
			dataStack: [][]byte{
				{4},
				mustDecodeHex("1f2a05f881ed9fa0c9068a84823677409f863891a2196eb55dbfbb677a566374"),
				Int64Bytes(-1),
				append([]byte{2}, make([]byte, 31)...),
				{1},
				[]byte("controlprog"),
			},
		},
		wantErr: ErrBadValue,
	}, {
		op: OP_CHECKOUTPUT,
		startVM: &virtualMachine{
			runLimit: 50000,
			tx:       tx,
			dataStack: [][]byte{
				Int64Bytes(-1),
				mustDecodeHex("1f2a05f881ed9fa0c9068a84823677409f863891a2196eb55dbfbb677a566374"),
				{7},
				append([]byte{2}, make([]byte, 31)...),
				{1},
				[]byte("controlprog"),
			},
		},
		wantErr: ErrBadValue,
	}, {
		op: OP_CHECKOUTPUT,
		startVM: &virtualMachine{
			runLimit: 50000,
			tx:       tx,
			dataStack: [][]byte{
				{5},
				mustDecodeHex("1f2a05f881ed9fa0c9068a84823677409f863891a2196eb55dbfbb677a566374"),
				{7},
				append([]byte{2}, make([]byte, 31)...),
				{1},
				[]byte("controlprog"),
			},
		},
		wantErr: ErrBadValue,
	}, {
		op: OP_CHECKOUTPUT,
		startVM: &virtualMachine{
			runLimit: 0,
			tx:       tx,
			dataStack: [][]byte{
				{4},
				mustDecodeHex("1f2a05f881ed9fa0c9068a84823677409f863891a2196eb55dbfbb677a566374"),
				{7},
				append([]byte{2}, make([]byte, 31)...),
				{1},
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
		op: OP_TXREFDATAHASH,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{},
			tx:        tx,
		},
		wantVM: &virtualMachine{
			runLimit:     49999,
			deferredCost: 40,
			dataStack: [][]byte{{
				62, 81, 144, 242, 105, 30, 109, 69, 28, 80, 237, 249, 169, 166, 106, 122,
				103, 121, 199, 135, 103, 100, 82, 129, 13, 191, 79, 110, 64, 83, 104, 44,
			}},
			tx: tx,
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

	txops := []Op{
		OP_CHECKOUTPUT, OP_ASSET, OP_AMOUNT, OP_PROGRAM,
		OP_MINTIME, OP_MAXTIME, OP_TXREFDATAHASH, OP_REFDATAHASH,
		OP_INDEX, OP_OUTPOINT,
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

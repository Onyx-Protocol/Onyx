package vm

import (
	"bytes"
	"reflect"
	"testing"

	"chain-stealth/crypto/ca"
	"chain-stealth/protocol/bc"
)

func TestNextProgram(t *testing.T) {
	block := &bc.Block{
		BlockHeader: bc.BlockHeader{
			ConsensusProgram: []byte{0x1, 0x2, 0x3},
		},
	}
	prog, err := Assemble("NEXTPROGRAM 0x010203 EQUAL")
	if err != nil {
		t.Fatal(err)
	}
	vm := &virtualMachine{
		runLimit: 50000,
		block:    block,
		program:  prog,
	}
	err = vm.run()
	if err != nil {
		t.Errorf("got error %s, expected none", err)
	}

	prog, err = Assemble("NEXTPROGRAM 0x0102 EQUAL")
	if err != nil {
		t.Fatal(err)
	}
	vm = &virtualMachine{
		runLimit: 50000,
		block:    block,
		program:  prog,
	}
	err = vm.run()
	switch err {
	case ErrFalseVMResult:
		// ok
	case nil:
		t.Error("got ok result, expected failure")
	default:
		t.Errorf("got error %s, expected none", err)
	}
}

func TestBlockTime(t *testing.T) {
	block := &bc.Block{
		BlockHeader: bc.BlockHeader{
			TimestampMS: 3263827,
		},
	}
	prog, err := Assemble("BLOCKTIME 3263827 NUMEQUAL")
	if err != nil {
		t.Fatal(err)
	}
	vm := &virtualMachine{
		runLimit: 50000,
		block:    block,
		program:  prog,
	}
	err = vm.run()
	if err != nil {
		t.Errorf("got error %s, expected none", err)
	}

	prog, err = Assemble("BLOCKTIME 3263826 NUMEQUAL")
	if err != nil {
		t.Fatal(err)
	}
	vm = &virtualMachine{
		runLimit: 50000,
		block:    block,
		program:  prog,
	}
	err = vm.run()
	switch err {
	case ErrFalseVMResult:
		// ok
	case nil:
		t.Error("got ok result, expected failure")
	default:
		t.Errorf("got error %s, expected none", err)
	}
}

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
			tx: tx,
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
			runLimit:     50101,
			deferredCost: -117,
			tx:           tx,
			dataStack:    [][]byte{{1}},
		},
	}, {
		op: OP_CHECKOUTPUT,
		startVM: &virtualMachine{
			tx: tx,
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
			runLimit:     50102,
			deferredCost: -118,
			tx:           tx,
			dataStack:    [][]byte{{}},
		},
		wantErr: ErrFalseVMResult,
	}, {
		op: OP_CHECKOUTPUT,
		startVM: &virtualMachine{
			tx: tx,
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
			runLimit:     50070,
			deferredCost: -86,
			tx:           tx,
			dataStack:    [][]byte{{}},
		},
		wantErr: ErrFalseVMResult,
	}, {
		op: OP_CHECKOUTPUT,
		startVM: &virtualMachine{
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
			tx: tx,
		},
		wantErr: ErrDataStackUnderflow,
	}, {
		op: OP_CHECKOUTPUT,
		startVM: &virtualMachine{
			tx: tx,
			dataStack: [][]byte{
				[]byte("controlprog"),
			},
		},
		wantErr: ErrDataStackUnderflow,
	}, {
		op: OP_CHECKOUTPUT,
		startVM: &virtualMachine{
			tx: tx,
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
			tx: tx,
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
			tx: tx,
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
			tx: tx,
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
			tx: tx,
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
			tx: tx,
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
			tx: tx,
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
			tx: tx,
		},
		wantVM: &virtualMachine{
			runLimit:     49959,
			deferredCost: 40,
			dataStack:    [][]byte{append([]byte{1}, make([]byte, 31)...)},
			tx:           tx,
		},
	}, {
		op: OP_AMOUNT,
		startVM: &virtualMachine{
			tx: tx,
		},
		wantVM: &virtualMachine{
			runLimit:     49990,
			deferredCost: 9,
			dataStack:    [][]byte{{5}},
			tx:           tx,
		},
	}, {
		op: OP_PROGRAM,
		startVM: &virtualMachine{
			mainprog: []byte("spendprog"),
			tx:       tx,
		},
		wantVM: &virtualMachine{
			runLimit:     49982,
			deferredCost: 17,
			dataStack:    [][]byte{[]byte("spendprog")},
			tx:           tx,
		},
	}, {
		op: OP_PROGRAM,
		startVM: &virtualMachine{
			mainprog:   []byte("issueprog"),
			runLimit:   50000,
			tx:         tx,
			inputIndex: 1,
		},
		wantVM: &virtualMachine{
			runLimit:     49982,
			deferredCost: 17,
			dataStack:    [][]byte{[]byte("issueprog")},
			tx:           tx,
			inputIndex:   1,
		},
	}, {
		op: OP_MINTIME,
		startVM: &virtualMachine{
			tx: tx,
		},
		wantVM: &virtualMachine{
			runLimit:     49991,
			deferredCost: 8,
			tx:           tx,
			dataStack:    [][]byte{[]byte{}},
		},
		wantErr: ErrFalseVMResult,
	}, {
		op: OP_MAXTIME,
		startVM: &virtualMachine{
			tx: tx,
		},
		wantVM: &virtualMachine{
			runLimit:     49990,
			deferredCost: 9,
			dataStack:    [][]byte{{20}},
			tx:           tx,
		},
	}, {
		op: OP_TXREFDATAHASH,
		startVM: &virtualMachine{
			tx: tx,
		},
		wantVM: &virtualMachine{
			runLimit:     49959,
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
			tx: tx,
		},
		wantVM: &virtualMachine{
			runLimit:     49959,
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
			tx: tx,
		},
		wantVM: &virtualMachine{
			runLimit:     49991,
			deferredCost: 8,
			tx:           tx,
			dataStack:    [][]byte{[]byte{}},
		},
		wantErr: ErrFalseVMResult,
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
				runLimit: 0,
				tx:       tx,
			},
			wantErr: ErrRunLimitExceeded,
		}, testStruct{
			op: op,
			startVM: &virtualMachine{
				tx: nil,
			},
			wantErr: ErrContext,
		})
	}

	for i, c := range cases {
		prog := []byte{byte(c.op)}
		vm := c.startVM
		if c.wantErr != ErrRunLimitExceeded {
			vm.runLimit = 50000
		}
		if vm.mainprog == nil {
			vm.mainprog = prog
		}
		vm.program = prog
		err := vm.run()
		if err != c.wantErr {
			t.Errorf("case %d, op %s: got err = %v want %v", i, ops[c.op].name, err, c.wantErr)
			continue
		}
		if c.wantErr != nil {
			continue
		}
		c.wantVM.mainprog = vm.mainprog
		c.wantVM.program = prog
		c.wantVM.pc = 1
		c.wantVM.nextPC = 1
		c.wantVM.sigHasher = c.startVM.sigHasher
		if !reflect.DeepEqual(vm, c.wantVM) {
			t.Errorf("case %d, op %s: unexpected vm result\n\tgot:  %+v\n\twant: %+v\n", i, ops[c.op].name, c.startVM, c.wantVM)
		}
	}
}

func TestCAIntrospection(t *testing.T) {
	cases := []struct {
		opcode            Op
		expansionreserved bool
		assetversion      uint64
		input             bc.TypedInput
		issuancekey       *ca.Point
		wanterr           error
		wantval           []byte
	}{
		{
			opcode:       OP_ASSETCOMMITMENT,
			assetversion: 2,
			input:        &bc.SpendInput{TypedOutput: &bc.Outputv1{}},
			wantval:      mustDecodeHex("118236b5545d2ea79ccd83b43193a68843cdbcf5395d1fc03cd851d3dbdd972f"),
		},
		{
			opcode:       OP_ASSETCOMMITMENT,
			assetversion: 1,
			input:        &bc.SpendInput{TypedOutput: &bc.Outputv1{}},
		},
		{
			opcode:            OP_ASSETCOMMITMENT,
			expansionreserved: true,
			assetversion:      1,
			input:             &bc.SpendInput{TypedOutput: &bc.Outputv1{}},
			wanterr:           ErrDisallowedOpcode,
		},
		{
			opcode:       OP_ASSETCOMMITMENT,
			assetversion: 2,
			input:        &bc.SpendInput{TypedOutput: &bc.Outputv2{}},
			wantval:      mustDecodeHex("118236b5545d2ea79ccd83b43193a68843cdbcf5395d1fc03cd851d3dbdd972f"),
		},
		{
			opcode:       OP_ASSETCOMMITMENT,
			assetversion: 2,
			input:        &bc.IssuanceInput1{},
			wantval:      mustDecodeHex("eb46c912f195d67b836ab6e2392e409869e914c5420cbf59d5ed6d55dc259cc2"),
		},
		{
			opcode:       OP_ASSETCOMMITMENT,
			assetversion: 1,
			input:        &bc.IssuanceInput1{},
		},
		{
			opcode:            OP_ASSETCOMMITMENT,
			expansionreserved: true,
			assetversion:      1,
			input:             &bc.IssuanceInput1{},
			wanterr:           ErrDisallowedOpcode,
		},
		{
			opcode:       OP_ASSETCOMMITMENT,
			assetversion: 2,
			input:        &bc.IssuanceInput2{},
			wantval:      mustDecodeHex("118236b5545d2ea79ccd83b43193a68843cdbcf5395d1fc03cd851d3dbdd972f"),
		},
		{
			opcode:       OP_ASSETCOMMITMENT,
			assetversion: 1,
			input:        &bc.IssuanceInput2{},
		},
		{
			opcode:            OP_ASSETCOMMITMENT,
			expansionreserved: true,
			assetversion:      1,
			input:             &bc.IssuanceInput2{},
			wanterr:           ErrDisallowedOpcode,
		},
		{
			opcode:       OP_VALUECOMMITMENT,
			assetversion: 2,
			input:        &bc.SpendInput{TypedOutput: &bc.Outputv1{}},
			wantval:      mustDecodeHex("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			opcode:       OP_VALUECOMMITMENT,
			assetversion: 1,
			input:        &bc.SpendInput{TypedOutput: &bc.Outputv1{}},
		},
		{
			opcode:            OP_VALUECOMMITMENT,
			expansionreserved: true,
			assetversion:      1,
			input:             &bc.SpendInput{TypedOutput: &bc.Outputv1{}},
			wanterr:           ErrDisallowedOpcode,
		},
		{
			opcode:       OP_VALUECOMMITMENT,
			assetversion: 2,
			input:        &bc.SpendInput{TypedOutput: &bc.Outputv2{}},
			wantval:      mustDecodeHex("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			opcode:       OP_VALUECOMMITMENT,
			assetversion: 2,
			input:        &bc.IssuanceInput1{},
			wantval:      mustDecodeHex("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			opcode:       OP_VALUECOMMITMENT,
			assetversion: 1,
			input:        &bc.IssuanceInput1{},
		},
		{
			opcode:            OP_VALUECOMMITMENT,
			expansionreserved: true,
			assetversion:      1,
			input:             &bc.IssuanceInput1{},
			wanterr:           ErrDisallowedOpcode,
		},
		{
			opcode:       OP_VALUECOMMITMENT,
			assetversion: 2,
			input:        &bc.IssuanceInput2{},
			wantval:      mustDecodeHex("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			opcode:       OP_VALUECOMMITMENT,
			assetversion: 1,
			input:        &bc.IssuanceInput2{},
		},
		{
			opcode:            OP_VALUECOMMITMENT,
			expansionreserved: true,
			assetversion:      1,
			input:             &bc.IssuanceInput2{},
			wanterr:           ErrDisallowedOpcode,
		},
		{
			opcode:       OP_ISSUANCEKEY,
			assetversion: 2,
			issuancekey:  &ca.ZeroPoint,
			wantval:      mustDecodeHex("0100000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			opcode:       OP_ISSUANCEKEY,
			assetversion: 1,
			issuancekey:  &ca.ZeroPoint,
		},
		{
			opcode:            OP_ISSUANCEKEY,
			expansionreserved: true,
			assetversion:      1,
			issuancekey:       &ca.ZeroPoint,
			wanterr:           ErrDisallowedOpcode,
		},
	}
	for i, c := range cases {
		tx := &bc.Tx{
			TxData: bc.TxData{
				Version: 2,
				Inputs: []*bc.TxInput{
					{
						AssetVersion: c.assetversion,
						TypedInput:   c.input,
					},
				},
			},
		}
		vm := &virtualMachine{
			tx:                tx,
			program:           []byte{byte(c.opcode)},
			issuanceKey:       c.issuancekey,
			runLimit:          100,
			expansionReserved: c.expansionreserved,
		}
		err := vm.step()
		if c.wanterr != nil {
			if err != c.wanterr {
				t.Errorf("case %d: want error %s, got %s", i, c.wanterr, err)
			}
			continue
		}
		if c.wantval == nil {
			continue
		}
		if len(vm.dataStack) == 0 {
			t.Errorf("case %d: empty stack", i)
			continue
		}
		got := vm.dataStack[len(vm.dataStack)-1]
		if !bytes.Equal(got, c.wantval) {
			t.Errorf("case %d: got %x, want %x", i, got, c.wantval)
		}
	}
}

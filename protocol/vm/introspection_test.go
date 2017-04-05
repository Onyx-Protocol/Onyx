package vm_test

import (
	"testing"

	"github.com/davecgh/go-spew/spew"

	"chain/crypto/sha3pool"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/validation"
	. "chain/protocol/vm"
	"chain/testutil"
)

func TestNextProgram(t *testing.T) {
	prog, err := Assemble("NEXTPROGRAM 0x010203 EQUAL")
	if err != nil {
		t.Fatal(err)
	}
	context := &Context{
		VMVersion:            1,
		Code:                 []byte{0xcd, 0x3, 0x1, 0x2, 0x3, 0x87},
		Arguments:            nil,
		BlockHash:            &[]byte{0x90, 0xf6, 0x1f, 0xd7, 0x35, 0xcd, 0x28, 0x7c, 0x78, 0xf, 0x43, 0xdc, 0xa7, 0xe7, 0xaf, 0x63, 0xad, 0x83, 0xa, 0x63, 0xcf, 0x56, 0x65, 0x7, 0x21, 0xcd, 0xb4, 0x54, 0xa3, 0xee, 0xf9, 0x2f},
		BlockTimeMS:          new(uint64),
		NextConsensusProgram: &[]byte{1, 2, 3},
	}

	vm := &VirtualMachine{
		RunLimit: 50000,
		Program:  prog,
		Context:  context,
	}
	_, err = vm.Run()
	if err != nil {
		t.Errorf("got error %s, expected none", err)
	}

	prog, err = Assemble("NEXTPROGRAM 0x0102 EQUAL")
	if err != nil {
		t.Fatal(err)
	}
	vm = &VirtualMachine{
		RunLimit: 50000,
		Program:  prog,
		Context:  context,
	}
	_, err = vm.Run()
	if err == nil && vm.FalseResult() {
		err = ErrFalseVMResult
	}
	switch err {
	case nil:
		t.Error("got ok result, expected failure")
	case ErrFalseVMResult:
		// ok
	default:
		t.Errorf("got error %s, expected ErrFalseVMResult", err)
	}
}

func TestBlockTime(t *testing.T) {
	prog, err := Assemble("BLOCKTIME 3263827 NUMEQUAL")
	if err != nil {
		t.Fatal(err)
	}
	var blockTimeMS uint64 = 3263826
	context := &Context{
		VMVersion:            1,
		Code:                 []byte{0xce, 0x3, 0x53, 0xcd, 0x31, 0x9c},
		Arguments:            nil,
		BlockHash:            &[]byte{0xc5, 0x5d, 0x64, 0xec, 0x18, 0xec, 0x3e, 0xb6, 0x30, 0xb5, 0x69, 0xd4, 0xb1, 0x23, 0xb9, 0x2c, 0x80, 0x64, 0xce, 0x8d, 0x7d, 0x7c, 0xc0, 0xbf, 0xaf, 0x57, 0x8, 0xb0, 0x62, 0x71, 0xa3, 0xed},
		BlockTimeMS:          &blockTimeMS,
		NextConsensusProgram: &[]byte{},
	}

	vm := &VirtualMachine{
		RunLimit: 50000,
		Program:  prog,
		Context:  context,
	}
	_, err = vm.Run()
	if err != nil {
		t.Errorf("got error %s, expected none", err)
	}

	prog, err = Assemble("BLOCKTIME 3263826 NUMEQUAL")
	if err != nil {
		t.Fatal(err)
	}
	vm = &VirtualMachine{
		RunLimit: 50000,
		Program:  prog,
		Context:  context,
	}
	_, err = vm.Run()
	if err == nil && vm.FalseResult() {
		err = ErrFalseVMResult
	}
	switch err {
	case nil:
		t.Error("got ok result, expected failure")
	case ErrFalseVMResult:
		// ok
	default:
		t.Errorf("got error %s, expected ErrFalseVMResult", err)
	}
}

func TestOutputIDAndNonceOp(t *testing.T) {
	var zeroHash bc.Hash
	nonceBytes := []byte{36, 37, 38}
	issuanceProgram := []byte("issueprog")
	var emptyHash bc.Hash
	sha3pool.Sum256(emptyHash[:], nil)
	assetID := bc.ComputeAssetID(issuanceProgram, zeroHash, 1, emptyHash)
	tx := bc.NewTx(bc.TxData{
		Inputs: []*bc.TxInput{
			bc.NewSpendInput(nil, bc.Hash{}, assetID, 5, 0, []byte("spendprog"), bc.Hash{}, []byte("ref")),
			bc.NewIssuanceInput(nonceBytes, 6, nil, zeroHash, issuanceProgram, nil, nil),
		},
	})
	outputID, err := tx.Inputs[0].SpentOutputID()
	if err != nil {
		t.Fatal(err)
	}
	prog := []byte{uint8(OP_OUTPUTID)}
	vm := &VirtualMachine{
		RunLimit: 50000,
		Program:  prog,
		Context:  validation.NewTxVMContext(tx.TxEntries, tx.TxEntries.TxInputs[0], bc.Program{VMVersion: 1, Code: prog}, nil),
	}
	gotVM, err := vm.Step()
	if err != nil {
		t.Fatal(err)
	}

	expectedStack := [][]byte{outputID[:]}
	if !testutil.DeepEqual(gotVM.DataStack, expectedStack) {
		t.Errorf("expected stack %v, got %v; vm is:\n%s", expectedStack, gotVM.DataStack, spew.Sdump(vm))
	}

	prog = []byte{uint8(OP_OUTPUTID)}
	vm = &VirtualMachine{
		RunLimit: 50000,
		Program:  prog,
		Context:  validation.NewTxVMContext(tx.TxEntries, tx.TxEntries.TxInputs[1], bc.Program{VMVersion: 1, Code: prog}, nil),
	}
	_, err = vm.Step()
	if err != ErrContext {
		t.Errorf("expected ErrContext, got %v", err)
	}

	prog = []byte{uint8(OP_NONCE)}
	vm = &VirtualMachine{
		RunLimit: 50000,
		Program:  prog,
		Context:  validation.NewTxVMContext(tx.TxEntries, tx.TxEntries.TxInputs[0], bc.Program{VMVersion: 1, Code: prog}, nil),
	}
	_, err = vm.Step()
	if err != ErrContext {
		t.Errorf("expected ErrContext, got %v", err)
	}

	prog = []byte{uint8(OP_NONCE)}
	vm = &VirtualMachine{
		RunLimit: 50000,
		Program:  prog,
		Context:  validation.NewTxVMContext(tx.TxEntries, tx.TxEntries.TxInputs[1], bc.Program{VMVersion: 1, Code: prog}, nil),
	}
	gotVM, err = vm.Step()
	if err != nil {
		t.Fatal(err)
	}

	expectedNonceProgCode := append([]byte{0x3}, nonceBytes...)
	expectedNonceProgCode = append(expectedNonceProgCode, byte(OP_DROP), byte(OP_ASSET))
	expectedNonceProgCode = append(expectedNonceProgCode, 0x20)
	expectedNonceProgCode = append(expectedNonceProgCode, assetID[:]...)
	expectedNonceProgCode = append(expectedNonceProgCode, byte(OP_EQUAL))
	expectedNonceProg := bc.Program{
		VMVersion: 1,
		Code:      expectedNonceProgCode,
	}
	expectedNonceTimeRange := bc.NewTimeRange(tx.Body.MinTimeMS, tx.Body.MaxTimeMS)
	expectedNonce := bc.NewNonce(expectedNonceProg, expectedNonceTimeRange)
	expectedNonceID := bc.EntryID(expectedNonce)

	expectedStack = [][]byte{expectedNonceID[:]}
	if !testutil.DeepEqual(gotVM.DataStack, expectedStack) {
		t.Errorf("expected stack %v, got %v", expectedStack, gotVM.DataStack)
	}
}

func TestIntrospectionOps(t *testing.T) {
	tx := bc.NewTx(bc.TxData{
		ReferenceData: []byte("txref"),
		Inputs: []*bc.TxInput{
			bc.NewSpendInput(nil, bc.Hash{}, bc.AssetID{1}, 5, 1, []byte("spendprog"), bc.Hash{}, []byte("ref")),
			bc.NewIssuanceInput(nil, 6, nil, bc.Hash{}, []byte("issueprog"), nil, nil),
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

	context0 := validation.NewTxVMContext(tx.TxEntries, tx.TxEntries.TxInputs[0], bc.Program{VMVersion: 1}, nil)

	type testStruct struct {
		op      Op
		startVM *VirtualMachine
		wantErr error
		wantVM  *VirtualMachine
	}
	cases := []testStruct{{
		op: OP_CHECKOUTPUT,
		startVM: &VirtualMachine{
			DataStack: [][]byte{
				{0},
				[]byte{},
				{1},
				append([]byte{9}, make([]byte, 31)...),
				{1},
				[]byte("missingprog"),
			},
			Context: context0,
		},
		wantVM: &VirtualMachine{
			RunLimit:     50070,
			DeferredCost: -86,
			DataStack:    [][]byte{{}},
			Context:      context0,
		},
	}, {
		op: OP_CHECKOUTPUT,
		startVM: &VirtualMachine{
			DataStack: [][]byte{
				{4},
				mustDecodeHex("1f2a05f881ed9fa0c9068a84823677409f863891a2196eb55dbfbb677a566374"),
				{7},
				append([]byte{2}, make([]byte, 31)...),
				Int64Bytes(-1),
				[]byte("controlprog"),
			},
			Context: context0,
		},
		wantErr: ErrBadValue,
	}, {
		op: OP_CHECKOUTPUT,
		startVM: &VirtualMachine{
			DataStack: [][]byte{
				{4},
				mustDecodeHex("1f2a05f881ed9fa0c9068a84823677409f863891a2196eb55dbfbb677a566374"),
				Int64Bytes(-1),
				append([]byte{2}, make([]byte, 31)...),
				{1},
				[]byte("controlprog"),
			},
			Context: context0,
		},
		wantErr: ErrBadValue,
	}, {
		op: OP_CHECKOUTPUT,
		startVM: &VirtualMachine{
			DataStack: [][]byte{
				Int64Bytes(-1),
				mustDecodeHex("1f2a05f881ed9fa0c9068a84823677409f863891a2196eb55dbfbb677a566374"),
				{7},
				append([]byte{2}, make([]byte, 31)...),
				{1},
				[]byte("controlprog"),
			},
			Context: context0,
		},
		wantErr: ErrBadValue,
	}, {
		op: OP_CHECKOUTPUT,
		startVM: &VirtualMachine{
			DataStack: [][]byte{
				{5},
				mustDecodeHex("1f2a05f881ed9fa0c9068a84823677409f863891a2196eb55dbfbb677a566374"),
				{7},
				append([]byte{2}, make([]byte, 31)...),
				{1},
				[]byte("controlprog"),
			},
			Context: context0,
		},
		wantErr: ErrBadValue,
	}, {
		op: OP_CHECKOUTPUT,
		startVM: &VirtualMachine{
			RunLimit: 0,
			DataStack: [][]byte{
				{4},
				mustDecodeHex("1f2a05f881ed9fa0c9068a84823677409f863891a2196eb55dbfbb677a566374"),
				{7},
				append([]byte{2}, make([]byte, 31)...),
				{1},
				[]byte("controlprog"),
			},
			Context: context0,
		},
		wantErr: ErrRunLimitExceeded,
	}, {
		op: OP_ASSET,
		startVM: &VirtualMachine{
			Context: context0,
		},
		wantVM: &VirtualMachine{
			RunLimit:     49959,
			DeferredCost: 40,
			DataStack:    [][]byte{append([]byte{1}, make([]byte, 31)...)},
			Context:      context0,
		},
	}, {
		op: OP_AMOUNT,
		startVM: &VirtualMachine{
			Context: context0,
		},
		wantVM: &VirtualMachine{
			RunLimit:     49990,
			DeferredCost: 9,
			DataStack:    [][]byte{{5}},
			Context:      context0,
		},
	}, {
		op: OP_PROGRAM,
		startVM: &VirtualMachine{
			Program: []byte("spendprog"),
			Context: validation.NewTxVMContext(tx.TxEntries, tx.TxEntries.TxInputs[0], bc.Program{VMVersion: 1, Code: []byte("spendprog")}, nil),
		},
		wantVM: &VirtualMachine{
			RunLimit:     49982,
			DeferredCost: 17,
			DataStack:    [][]byte{[]byte("spendprog")},
			Context:      validation.NewTxVMContext(tx.TxEntries, tx.TxEntries.TxInputs[0], bc.Program{VMVersion: 1, Code: []byte("spendprog")}, nil),
		},
	}, {
		op: OP_PROGRAM,
		startVM: &VirtualMachine{
			Program:  []byte("issueprog"),
			RunLimit: 50000,
			Context:  validation.NewTxVMContext(tx.TxEntries, tx.TxEntries.TxInputs[1], bc.Program{VMVersion: 1, Code: []byte("issueprog")}, nil),
		},
		wantVM: &VirtualMachine{
			RunLimit:     49982,
			DeferredCost: 17,
			DataStack:    [][]byte{[]byte("issueprog")},
			Context:      validation.NewTxVMContext(tx.TxEntries, tx.TxEntries.TxInputs[1], bc.Program{VMVersion: 1, Code: []byte("issueprog")}, nil),
		},
	}, {
		op: OP_MINTIME,
		startVM: &VirtualMachine{
			Context: context0,
		},
		wantVM: &VirtualMachine{
			RunLimit:     49991,
			DeferredCost: 8,
			DataStack:    [][]byte{[]byte{}},
			Context:      context0,
		},
	}, {
		op: OP_MAXTIME,
		startVM: &VirtualMachine{
			Context: context0,
		},
		wantVM: &VirtualMachine{
			RunLimit:     49990,
			DeferredCost: 9,
			DataStack:    [][]byte{{20}},
			Context:      context0,
		},
	}, {
		op: OP_TXDATAHASH,
		startVM: &VirtualMachine{
			Context: context0,
		},
		wantVM: &VirtualMachine{
			RunLimit:     49959,
			DeferredCost: 40,
			DataStack: [][]byte{{
				62, 81, 144, 242, 105, 30, 109, 69, 28, 80, 237, 249, 169, 166, 106, 122,
				103, 121, 199, 135, 103, 100, 82, 129, 13, 191, 79, 110, 64, 83, 104, 44,
			}},
			Context: context0,
		},
	}, {
		op: OP_DATAHASH,
		startVM: &VirtualMachine{
			Context: context0,
		},
		wantVM: &VirtualMachine{
			RunLimit:     49959,
			DeferredCost: 40,
			DataStack: [][]byte{{
				68, 190, 94, 20, 206, 33, 111, 75, 44, 53, 165, 235, 11, 53, 208, 120,
				189, 165, 92, 240, 91, 93, 54, 238, 14, 122, 1, 251, 198, 239, 98, 183,
			}},
			Context: context0,
		},
	}, {
		op: OP_INDEX,
		startVM: &VirtualMachine{
			Context: context0,
		},
		wantVM: &VirtualMachine{
			RunLimit:     49991,
			DeferredCost: 8,
			DataStack:    [][]byte{[]byte{}},
			Context:      context0,
		},
	}, {
		// The current entry is input 0
		op: OP_ENTRYID,
		startVM: &VirtualMachine{
			Context: context0,
		},
		wantVM: &VirtualMachine{
			RunLimit:     49959,
			DeferredCost: 40,
			DataStack:    [][]byte{tx.TxEntries.TxInputIDs[0][:]},
			Context:      context0,
		},
	}, {
		// The current entry is input 1
		op: OP_ENTRYID,
		startVM: &VirtualMachine{
			Context: validation.NewTxVMContext(tx.TxEntries, tx.TxEntries.TxInputs[1], bc.Program{VMVersion: 1}, nil),
		},
		wantVM: &VirtualMachine{
			RunLimit:     49959,
			DeferredCost: 40,
			DataStack:    [][]byte{tx.TxEntries.TxInputIDs[1][:]},
			Context:      validation.NewTxVMContext(tx.TxEntries, tx.TxEntries.TxInputs[1], bc.Program{VMVersion: 1}, nil),
		},
	}, {
		// The current entry is the internal mux node
		op: OP_ENTRYID,
		startVM: &VirtualMachine{
			Context: validation.NewTxVMContext(tx.TxEntries, tx.TxEntries.TxInputs[0].(*bc.Spend).Witness.Destination.Entry, bc.Program{VMVersion: 1}, nil),
		},
		wantVM: &VirtualMachine{
			RunLimit:     49959,
			DeferredCost: 40,
			DataStack:    [][]byte{tx.TxEntries.TxInputs[0].(*bc.Spend).Witness.Destination.Ref[:]},
			Context:      validation.NewTxVMContext(tx.TxEntries, tx.TxEntries.TxInputs[0].(*bc.Spend).Witness.Destination.Entry, bc.Program{VMVersion: 1}, nil),
		},
	}}

	txops := []Op{
		OP_CHECKOUTPUT, OP_ASSET, OP_AMOUNT, OP_PROGRAM,
		OP_MINTIME, OP_MAXTIME, OP_TXDATAHASH, OP_DATAHASH,
		OP_INDEX, OP_OUTPUTID,
	}

	for _, op := range txops {
		cases = append(cases, testStruct{
			op: op,
			startVM: &VirtualMachine{
				RunLimit: 0,
				Context:  context0,
			},
			wantErr: ErrRunLimitExceeded,
		})
	}

	for i, c := range cases {
		t.Logf("case %d", i)
		prog := []byte{byte(c.op)}
		vm := c.startVM
		if c.wantErr != ErrRunLimitExceeded {
			vm.RunLimit = 50000
		}
		vm.Program = prog
		gotVM, err := vm.Run()
		switch errors.Root(err) {
		case c.wantErr:
			// ok
		case nil:
			t.Errorf("case %d, op %s: got no error, want %v", i, OpName(c.op), c.wantErr)
		default:
			t.Errorf("case %d, op %s: got err = %v want %v", i, OpName(c.op), err, c.wantErr)
		}
		if c.wantErr != nil {
			continue
		}

		c.wantVM.Program = prog
		c.wantVM.PC = 1
		c.wantVM.NextPC = 1
		c.wantVM.Context = gotVM.Context

		if !testutil.DeepEqual(gotVM, c.wantVM) {
			t.Errorf("case %d, op %s: unexpected vm result\n\tgot:  %+v\n\twant: %+v\nstartVM is:\n%s", i, OpName(c.op), gotVM, c.wantVM, spew.Sdump(c.startVM))
		}
	}
}

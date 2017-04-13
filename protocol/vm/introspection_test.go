package vm_test

import (
	"testing"

	"github.com/davecgh/go-spew/spew"

	"chain/errors"
	. "chain/protocol/vm"
	"chain/testutil"
)

func TestNextProgram(t *testing.T) {
	context := &Context{
		NextConsensusProgram: &[]byte{1, 2, 3},
	}

	prog, err := Assemble("NEXTPROGRAM 0x010203 EQUAL")
	if err != nil {
		t.Fatal(err)
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
	var blockTimeMS uint64 = 3263826
	context := &Context{
		BlockTimeMS: &blockTimeMS,
	}

	prog, err := Assemble("BLOCKTIME 3263827 NUMEQUAL")
	if err != nil {
		t.Fatal(err)
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
	// arbitrary
	outputID := mustDecodeHex("0a60f9b12950c84c221012a808ef7782823b7e16b71fe2ba01811cda96a217df")
	nonceID := mustDecodeHex("c4a6e6256debfca379595e444b91af56846397e8007ea87c40c622170dd13ff7")

	prog := []byte{uint8(OP_OUTPUTID)}
	vm := &VirtualMachine{
		RunLimit: 50000,
		Program:  prog,
		Context:  &Context{SpentOutputID: &outputID},
	}
	gotVM, err := vm.Step()
	if err != nil {
		t.Fatal(err)
	}

	expectedStack := [][]byte{outputID}
	if !testutil.DeepEqual(gotVM.DataStack, expectedStack) {
		t.Errorf("expected stack %v, got %v; vm is:\n%s", expectedStack, gotVM.DataStack, spew.Sdump(vm))
	}

	prog = []byte{uint8(OP_OUTPUTID)}
	vm = &VirtualMachine{
		RunLimit: 50000,
		Program:  prog,
		Context:  &Context{SpentOutputID: nil},
	}
	_, err = vm.Step()
	if err != ErrContext {
		t.Errorf("expected ErrContext, got %v", err)
	}

	prog = []byte{uint8(OP_NONCE)}
	vm = &VirtualMachine{
		RunLimit: 50000,
		Program:  prog,
		Context:  &Context{AnchorID: nil},
	}
	_, err = vm.Step()
	if err != ErrContext {
		t.Errorf("expected ErrContext, got %v", err)
	}

	prog = []byte{uint8(OP_NONCE)}
	vm = &VirtualMachine{
		RunLimit: 50000,
		Program:  prog,
		Context:  &Context{AnchorID: &nonceID},
	}
	gotVM, err = vm.Step()
	if err != nil {
		t.Fatal(err)
	}

	expectedStack = [][]byte{nonceID}
	if !testutil.DeepEqual(gotVM.DataStack, expectedStack) {
		t.Errorf("expected stack %v, got %v", expectedStack, gotVM.DataStack)
	}
}

func TestIntrospectionOps(t *testing.T) {
	// arbitrary
	entryID := mustDecodeHex("2e68d78cdeaa98944c12512cf9c719eb4881e9afb61e4b766df5f369aee6392c")
	entryData := mustDecodeHex("44be5e14ce216f4b2c35a5eb0b35d078bda55cf05b5d36ee0e7a01fbc6ef62b7")
	assetID := mustDecodeHex("0100000000000000000000000000000000000000000000000000000000000000")
	txData := mustDecodeHex("3e5190f2691e6d451c50edf9a9a66a7a6779c787676452810dbf4f6e4053682c")

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
			Context: &Context{
				CheckOutput: func(uint64, []byte, uint64, []byte, uint64, []byte, bool) (bool, error) {
					return false, nil
				},
			},
		},
		wantVM: &VirtualMachine{
			RunLimit:     50070,
			DeferredCost: -86,
			DataStack:    [][]byte{{}},
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
			Context: &Context{},
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
			Context: &Context{},
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
			Context: &Context{},
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
			Context: &Context{
				CheckOutput: func(uint64, []byte, uint64, []byte, uint64, []byte, bool) (bool, error) {
					return false, ErrBadValue
				},
			},
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
			Context: &Context{},
		},
		wantErr: ErrRunLimitExceeded,
	}, {
		op: OP_ASSET,
		startVM: &VirtualMachine{
			Context: &Context{AssetID: &assetID},
		},
		wantVM: &VirtualMachine{
			RunLimit:     49959,
			DeferredCost: 40,
			DataStack:    [][]byte{assetID},
		},
	}, {
		op: OP_AMOUNT,
		startVM: &VirtualMachine{
			Context: &Context{Amount: uint64ptr(5)},
		},
		wantVM: &VirtualMachine{
			RunLimit:     49990,
			DeferredCost: 9,
			DataStack:    [][]byte{{5}},
		},
	}, {
		op: OP_PROGRAM,
		startVM: &VirtualMachine{
			Program: []byte("spendprog"),
			Context: &Context{Code: []byte("spendprog")},
		},
		wantVM: &VirtualMachine{
			RunLimit:     49982,
			DeferredCost: 17,
			DataStack:    [][]byte{[]byte("spendprog")},
		},
	}, {
		op: OP_PROGRAM,
		startVM: &VirtualMachine{
			Program:  []byte("issueprog"),
			RunLimit: 50000,
			Context:  &Context{Code: []byte("issueprog")},
		},
		wantVM: &VirtualMachine{
			RunLimit:     49982,
			DeferredCost: 17,
			DataStack:    [][]byte{[]byte("issueprog")},
		},
	}, {
		op: OP_MINTIME,
		startVM: &VirtualMachine{
			Context: &Context{MinTimeMS: new(uint64)},
		},
		wantVM: &VirtualMachine{
			RunLimit:     49991,
			DeferredCost: 8,
			DataStack:    [][]byte{[]byte{}},
		},
	}, {
		op: OP_MAXTIME,
		startVM: &VirtualMachine{
			Context: &Context{MaxTimeMS: uint64ptr(20)},
		},
		wantVM: &VirtualMachine{
			RunLimit:     49990,
			DeferredCost: 9,
			DataStack:    [][]byte{{20}},
		},
	}, {
		op: OP_TXDATA,
		startVM: &VirtualMachine{
			Context: &Context{TxData: &txData},
		},
		wantVM: &VirtualMachine{
			RunLimit:     49959,
			DeferredCost: 40,
			DataStack:    [][]byte{txData},
		},
	}, {
		op: OP_ENTRYDATA,
		startVM: &VirtualMachine{
			Context: &Context{EntryData: &entryData},
		},
		wantVM: &VirtualMachine{
			RunLimit:     49959,
			DeferredCost: 40,
			DataStack:    [][]byte{entryData},
		},
	}, {
		op: OP_INDEX,
		startVM: &VirtualMachine{
			Context: &Context{DestPos: new(uint64)},
		},
		wantVM: &VirtualMachine{
			RunLimit:     49991,
			DeferredCost: 8,
			DataStack:    [][]byte{[]byte{}},
		},
	}, {
		op: OP_ENTRYID,
		startVM: &VirtualMachine{
			Context: &Context{EntryID: entryID},
		},
		wantVM: &VirtualMachine{
			RunLimit:     49959,
			DeferredCost: 40,
			DataStack:    [][]byte{entryID},
		},
	}}

	txops := []Op{
		OP_CHECKOUTPUT, OP_ASSET, OP_AMOUNT, OP_PROGRAM,
		OP_MINTIME, OP_MAXTIME, OP_TXDATA, OP_ENTRYDATA,
		OP_INDEX, OP_OUTPUTID,
	}

	for _, op := range txops {
		cases = append(cases, testStruct{
			op: op,
			startVM: &VirtualMachine{
				RunLimit: 0,
				Context:  &Context{},
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

func uint64ptr(n uint64) *uint64 { return &n }

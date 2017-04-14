package vm

import (
	"testing"

	"github.com/davecgh/go-spew/spew"

	"chain/errors"
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
	vm := &virtualMachine{
		runLimit: 50000,
		program:  prog,
		context:  context,
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
		program:  prog,
		context:  context,
	}
	err = vm.run()
	if err == nil && vm.falseResult() {
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

	prog, err := Assemble("BLOCKTIME 3263826 NUMEQUAL")
	if err != nil {
		t.Fatal(err)
	}
	vm := &virtualMachine{
		runLimit: 50000,
		program:  prog,
		context:  &Context{BlockTimeMS: &blockTimeMS},
	}
	err = vm.run()
	if err != nil {
		t.Errorf("got error %s, expected none", err)
	}
	if vm.falseResult() {
		t.Error("result is false, want success")
	}

	prog, err = Assemble("BLOCKTIME 3263827 NUMEQUAL")
	if err != nil {
		t.Fatal(err)
	}
	vm = &virtualMachine{
		runLimit: 50000,
		program:  prog,
		context:  &Context{BlockTimeMS: &blockTimeMS},
	}
	err = vm.run()
	if err == nil && vm.falseResult() {
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
	vm := &virtualMachine{
		runLimit: 50000,
		program:  prog,
		context:  &Context{SpentOutputID: &outputID},
	}
	err := vm.step()
	if err != nil {
		t.Fatal(err)
	}
	gotVM := vm

	expectedStack := [][]byte{outputID}
	if !testutil.DeepEqual(gotVM.dataStack, expectedStack) {
		t.Errorf("expected stack %v, got %v; vm is:\n%s", expectedStack, gotVM.dataStack, spew.Sdump(vm))
	}

	prog = []byte{uint8(OP_OUTPUTID)}
	vm = &virtualMachine{
		runLimit: 50000,
		program:  prog,
		context:  &Context{SpentOutputID: nil},
	}
	err = vm.step()
	if err != ErrContext {
		t.Errorf("expected ErrContext, got %v", err)
	}

	prog = []byte{uint8(OP_NONCE)}
	vm = &virtualMachine{
		runLimit: 50000,
		program:  prog,
		context:  &Context{AnchorID: nil},
	}
	err = vm.step()
	if err != ErrContext {
		t.Errorf("expected ErrContext, got %v", err)
	}

	prog = []byte{uint8(OP_NONCE)}
	vm = &virtualMachine{
		runLimit: 50000,
		program:  prog,
		context:  &Context{AnchorID: &nonceID},
	}
	err = vm.step()
	if err != nil {
		t.Fatal(err)
	}
	gotVM = vm

	expectedStack = [][]byte{nonceID}
	if !testutil.DeepEqual(gotVM.dataStack, expectedStack) {
		t.Errorf("expected stack %v, got %v", expectedStack, gotVM.dataStack)
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
		startVM *virtualMachine
		wantErr error
		wantVM  *virtualMachine
	}
	cases := []testStruct{{
		op: OP_CHECKOUTPUT,
		startVM: &virtualMachine{
			dataStack: [][]byte{
				{0},
				[]byte{},
				{1},
				append([]byte{9}, make([]byte, 31)...),
				{1},
				[]byte("missingprog"),
			},
			context: &Context{
				CheckOutput: func(uint64, []byte, uint64, []byte, uint64, []byte, bool) (bool, error) {
					return false, nil
				},
			},
		},
		wantVM: &virtualMachine{
			runLimit:     50070,
			deferredCost: -86,
			dataStack:    [][]byte{{}},
		},
	}, {
		op: OP_CHECKOUTPUT,
		startVM: &virtualMachine{
			dataStack: [][]byte{
				{4},
				mustDecodeHex("1f2a05f881ed9fa0c9068a84823677409f863891a2196eb55dbfbb677a566374"),
				{7},
				append([]byte{2}, make([]byte, 31)...),
				Int64Bytes(-1),
				[]byte("controlprog"),
			},
			context: &Context{},
		},
		wantErr: ErrBadValue,
	}, {
		op: OP_CHECKOUTPUT,
		startVM: &virtualMachine{
			dataStack: [][]byte{
				{4},
				mustDecodeHex("1f2a05f881ed9fa0c9068a84823677409f863891a2196eb55dbfbb677a566374"),
				Int64Bytes(-1),
				append([]byte{2}, make([]byte, 31)...),
				{1},
				[]byte("controlprog"),
			},
			context: &Context{},
		},
		wantErr: ErrBadValue,
	}, {
		op: OP_CHECKOUTPUT,
		startVM: &virtualMachine{
			dataStack: [][]byte{
				Int64Bytes(-1),
				mustDecodeHex("1f2a05f881ed9fa0c9068a84823677409f863891a2196eb55dbfbb677a566374"),
				{7},
				append([]byte{2}, make([]byte, 31)...),
				{1},
				[]byte("controlprog"),
			},
			context: &Context{},
		},
		wantErr: ErrBadValue,
	}, {
		op: OP_CHECKOUTPUT,
		startVM: &virtualMachine{
			dataStack: [][]byte{
				{5},
				mustDecodeHex("1f2a05f881ed9fa0c9068a84823677409f863891a2196eb55dbfbb677a566374"),
				{7},
				append([]byte{2}, make([]byte, 31)...),
				{1},
				[]byte("controlprog"),
			},
			context: &Context{
				CheckOutput: func(uint64, []byte, uint64, []byte, uint64, []byte, bool) (bool, error) {
					return false, ErrBadValue
				},
			},
		},
		wantErr: ErrBadValue,
	}, {
		op: OP_CHECKOUTPUT,
		startVM: &virtualMachine{
			runLimit: 0,
			dataStack: [][]byte{
				{4},
				mustDecodeHex("1f2a05f881ed9fa0c9068a84823677409f863891a2196eb55dbfbb677a566374"),
				{7},
				append([]byte{2}, make([]byte, 31)...),
				{1},
				[]byte("controlprog"),
			},
			context: &Context{},
		},
		wantErr: ErrRunLimitExceeded,
	}, {
		op: OP_ASSET,
		startVM: &virtualMachine{
			context: &Context{AssetID: &assetID},
		},
		wantVM: &virtualMachine{
			runLimit:     49959,
			deferredCost: 40,
			dataStack:    [][]byte{assetID},
		},
	}, {
		op: OP_AMOUNT,
		startVM: &virtualMachine{
			context: &Context{Amount: uint64ptr(5)},
		},
		wantVM: &virtualMachine{
			runLimit:     49990,
			deferredCost: 9,
			dataStack:    [][]byte{{5}},
		},
	}, {
		op: OP_PROGRAM,
		startVM: &virtualMachine{
			program: []byte("spendprog"),
			context: &Context{Code: []byte("spendprog")},
		},
		wantVM: &virtualMachine{
			runLimit:     49982,
			deferredCost: 17,
			dataStack:    [][]byte{[]byte("spendprog")},
		},
	}, {
		op: OP_PROGRAM,
		startVM: &virtualMachine{
			program:  []byte("issueprog"),
			runLimit: 50000,
			context:  &Context{Code: []byte("issueprog")},
		},
		wantVM: &virtualMachine{
			runLimit:     49982,
			deferredCost: 17,
			dataStack:    [][]byte{[]byte("issueprog")},
		},
	}, {
		op: OP_MINTIME,
		startVM: &virtualMachine{
			context: &Context{MinTimeMS: new(uint64)},
		},
		wantVM: &virtualMachine{
			runLimit:     49991,
			deferredCost: 8,
			dataStack:    [][]byte{[]byte{}},
		},
	}, {
		op: OP_MAXTIME,
		startVM: &virtualMachine{
			context: &Context{MaxTimeMS: uint64ptr(20)},
		},
		wantVM: &virtualMachine{
			runLimit:     49990,
			deferredCost: 9,
			dataStack:    [][]byte{{20}},
		},
	}, {
		op: OP_TXDATA,
		startVM: &virtualMachine{
			context: &Context{TxData: &txData},
		},
		wantVM: &virtualMachine{
			runLimit:     49959,
			deferredCost: 40,
			dataStack:    [][]byte{txData},
		},
	}, {
		op: OP_ENTRYDATA,
		startVM: &virtualMachine{
			context: &Context{EntryData: &entryData},
		},
		wantVM: &virtualMachine{
			runLimit:     49959,
			deferredCost: 40,
			dataStack:    [][]byte{entryData},
		},
	}, {
		op: OP_INDEX,
		startVM: &virtualMachine{
			context: &Context{DestPos: new(uint64)},
		},
		wantVM: &virtualMachine{
			runLimit:     49991,
			deferredCost: 8,
			dataStack:    [][]byte{[]byte{}},
		},
	}, {
		op: OP_ENTRYID,
		startVM: &virtualMachine{
			context: &Context{EntryID: entryID},
		},
		wantVM: &virtualMachine{
			runLimit:     49959,
			deferredCost: 40,
			dataStack:    [][]byte{entryID},
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
			startVM: &virtualMachine{
				runLimit: 0,
				context:  &Context{},
			},
			wantErr: ErrRunLimitExceeded,
		})
	}

	for i, c := range cases {
		t.Logf("case %d", i)
		prog := []byte{byte(c.op)}
		vm := c.startVM
		if c.wantErr != ErrRunLimitExceeded {
			vm.runLimit = 50000
		}
		vm.program = prog
		err := vm.run()
		switch errors.Root(err) {
		case c.wantErr:
			// ok
		case nil:
			t.Errorf("case %d, op %s: got no error, want %v", i, ops[c.op].name, c.wantErr)
		default:
			t.Errorf("case %d, op %s: got err = %v want %v", i, ops[c.op].name, err, c.wantErr)
		}
		if c.wantErr != nil {
			continue
		}
		gotVM := vm

		c.wantVM.program = prog
		c.wantVM.pc = 1
		c.wantVM.nextPC = 1
		c.wantVM.context = gotVM.context

		if !testutil.DeepEqual(gotVM, c.wantVM) {
			t.Errorf("case %d, op %s: unexpected vm result\n\tgot:  %+v\n\twant: %+v\nstartVM is:\n%s", i, ops[c.op].name, gotVM, c.wantVM, spew.Sdump(c.startVM))
		}
	}
}

func uint64ptr(n uint64) *uint64 { return &n }

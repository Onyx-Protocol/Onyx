package vm

import (
	"testing"

	"chain/testutil"
)

var emptyBlockVMContext = &Context{
	BlockHash: &[]uint8{0xf0, 0x85, 0x4f, 0x88, 0xb4, 0x89, 0x0, 0x99, 0x2f, 0xec, 0x40, 0x43, 0xf9, 0x65, 0xfa, 0x2, 0x9d, 0xeb, 0x8a, 0xd6, 0x93, 0xcf, 0x37, 0x11, 0xfe, 0x83, 0x9, 0xb3, 0x90, 0x6a, 0x5a, 0x86},
}

func TestCheckSig(t *testing.T) {
	cases := []struct {
		prog    string
		ok, err bool
	}{
		{
			// This one's OK
			"0x26ced30b1942b89ef5332a9f22f1a61e5a6a3f8a5bc33b2fc58b1daf78c81bf1d5c8add19cea050adeb37da3a7bf8f813c6a6922b42934a6441fa6bb1c7fc208 0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20 0xdbca6fb13badb7cfdf76510070ffad15b85f9934224a9e11202f5e8f86b584a6 CHECKSIG",
			true, false,
		},
		{
			// This one has a wrong-length signature
			"0x26ced30b1942b89ef5332a9f22f1a61e5a6a3f8a5bc33b2fc58b1daf78c81bf1d5c8add19cea050adeb37da3a7bf8f813c6a6922b42934a6441fa6bb1c7fc2 0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20 0xdbca6fb13badb7cfdf76510070ffad15b85f9934224a9e11202f5e8f86b584a6 CHECKSIG",
			false, false,
		},
		{
			// This one has a wrong-length message
			"0x26ced30b1942b89ef5332a9f22f1a61e5a6a3f8a5bc33b2fc58b1daf78c81bf1d5c8add19cea050adeb37da3a7bf8f813c6a6922b42934a6441fa6bb1c7fc208 0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f 0xdbca6fb13badb7cfdf76510070ffad15b85f9934224a9e11202f5e8f86b584a6 CHECKSIG",
			false, true,
		},
		{
			// This one has a wrong-length pubkey
			"0x26ced30b1942b89ef5332a9f22f1a61e5a6a3f8a5bc33b2fc58b1daf78c81bf1d5c8add19cea050adeb37da3a7bf8f813c6a6922b42934a6441fa6bb1c7fc208 0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20 0xdbca6fb13badb7cfdf76510070ffad15b85f9934224a9e11202f5e8f86b584 CHECKSIG",
			false, false,
		},
		{
			// This one has a wrong byte in the signature
			"0x00ced30b1942b89ef5332a9f22f1a61e5a6a3f8a5bc33b2fc58b1daf78c81bf1d5c8add19cea050adeb37da3a7bf8f813c6a6922b42934a6441fa6bb1c7fc208 0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20 0xdbca6fb13badb7cfdf76510070ffad15b85f9934224a9e11202f5e8f86b584a6 CHECKSIG",
			false, false,
		},
		{
			// This one has a wrong byte in the message
			"0x26ced30b1942b89ef5332a9f22f1a61e5a6a3f8a5bc33b2fc58b1daf78c81bf1d5c8add19cea050adeb37da3a7bf8f813c6a6922b42934a6441fa6bb1c7fc208 0x0002030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20 0xdbca6fb13badb7cfdf76510070ffad15b85f9934224a9e11202f5e8f86b584a6 CHECKSIG",
			false, false,
		},
		{
			// This one has a wrong byte in the pubkey
			"0x26ced30b1942b89ef5332a9f22f1a61e5a6a3f8a5bc33b2fc58b1daf78c81bf1d5c8add19cea050adeb37da3a7bf8f813c6a6922b42934a6441fa6bb1c7fc208 0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20 0x00ca6fb13badb7cfdf76510070ffad15b85f9934224a9e11202f5e8f86b584a6 CHECKSIG",
			false, false,
		},
		{
			"0x010203 0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20 0x040506 1 1 CHECKMULTISIG",
			false, false,
		},
		{
			"0x010203 0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f 0x040506 1 1 CHECKMULTISIG",
			false, true,
		},
		{
			"0x26ced30b1942b89ef5332a9f22f1a61e5a6a3f8a5bc33b2fc58b1daf78c81bf1d5c8add19cea050adeb37da3a7bf8f813c6a6922b42934a6441fa6bb1c7fc208 0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20 0xdbca6fb13badb7cfdf76510070ffad15b85f9934224a9e11202f5e8f86b584a6 1 1 CHECKMULTISIG",
			true, false,
		},
	}

	for i, c := range cases {
		prog, err := Assemble(c.prog)
		if err != nil {
			t.Fatalf("case %d: %s", i, err)
		}
		vm := &virtualMachine{
			program:  prog,
			runLimit: 50000,
		}
		err = vm.run()
		if c.err {
			if err == nil {
				t.Errorf("case %d: expected error, got ok result", i)
			}
		} else if c.ok {
			if err != nil {
				t.Errorf("case %d: expected ok result, got error %s", i, err)
			}
		} else if !vm.falseResult() {
			t.Errorf("case %d: expected false VM result, got error %s", i, err)
		}
	}
}

func TestCryptoOps(t *testing.T) {
	type testStruct struct {
		op      Op
		startVM *virtualMachine
		wantErr error
		wantVM  *virtualMachine
	}
	cases := []testStruct{{
		op: OP_SHA256,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{1}},
		},
		wantVM: &virtualMachine{
			runLimit: 49905,
			dataStack: [][]byte{{
				75, 245, 18, 47, 52, 69, 84, 197, 59, 222, 46, 187, 140, 210, 183, 227,
				209, 96, 10, 214, 49, 195, 133, 165, 215, 204, 226, 60, 119, 133, 69, 154,
			}},
		},
	}, {
		op: OP_SHA256,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{make([]byte, 65)},
		},
		wantVM: &virtualMachine{
			runLimit: 49968,
			dataStack: [][]byte{{
				152, 206, 66, 222, 239, 81, 212, 2, 105, 213, 66, 245, 49, 75, 239, 44,
				116, 104, 212, 1, 173, 93, 133, 22, 139, 250, 180, 192, 16, 143, 117, 247,
			}},
		},
	}, {
		op: OP_SHA3,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{1}},
		},
		wantVM: &virtualMachine{
			runLimit: 49905,
			dataStack: [][]byte{{
				39, 103, 241, 92, 138, 242, 242, 199, 34, 93, 82, 115, 253, 214, 131, 237,
				199, 20, 17, 10, 152, 125, 16, 84, 105, 124, 52, 138, 237, 78, 108, 199,
			}},
		},
	}, {
		op: OP_SHA3,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{make([]byte, 65)},
		},
		wantVM: &virtualMachine{
			runLimit: 49968,
			dataStack: [][]byte{{
				65, 106, 167, 181, 192, 224, 101, 48, 102, 167, 198, 77, 189, 208, 0, 157,
				190, 132, 56, 97, 81, 254, 3, 159, 217, 66, 250, 162, 219, 97, 114, 235,
			}},
		},
	}, {
		op: OP_CHECKSIG,
		startVM: &virtualMachine{
			runLimit: 50000,
			dataStack: [][]byte{
				mustDecodeHex("af5abdf4bbb34f4a089efc298234f84fd909def662a8df03b4d7d40372728851" +
					"fbd3bf59920af5a7c361a4851967714271d1727e3be417a60053c30969d8860c"),
				mustDecodeHex("916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"),
				mustDecodeHex("ab3220d065dc875c6a5b4ecc39809b5f24eb0a605e9eef5190457edbf1e3b866"),
			},
		},
		wantVM: &virtualMachine{
			deferredCost: -143,
			runLimit:     48976,
			dataStack:    [][]byte{{1}},
		},
	}, {
		op: OP_CHECKSIG,
		startVM: &virtualMachine{
			runLimit: 50000,
			dataStack: [][]byte{
				mustDecodeHex("af5abdf4bbb34f4a089efc298234f84fd909def662a8df03b4d7d40372728851" +
					"fbd3bf59920af5a7c361a4851967714271d1727e3be417a60053c30969d8860c"),
				mustDecodeHex("badda7a7a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"),
				mustDecodeHex("ab3220d065dc875c6a5b4ecc39809b5f24eb0a605e9eef5190457edbf1e3b866"),
			},
		},
		wantVM: &virtualMachine{
			deferredCost: -144,
			runLimit:     48976,
			dataStack:    [][]byte{{}},
		},
	}, {
		op: OP_CHECKSIG,
		startVM: &virtualMachine{
			runLimit: 50000,
			dataStack: [][]byte{
				mustDecodeHex("af5abdf4bbb34f4a089efc298234f84fd909def662a8df03b4d7d40372728851" +
					"fbd3bf59920af5a7c361a4851967714271d1727e3be417a60053c30969d8860c"),
				mustDecodeHex("916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"),
				mustDecodeHex("bad220d065dc875c6a5b4ecc39809b5f24eb0a605e9eef5190457edbf1e3b866"),
			},
		},
		wantVM: &virtualMachine{
			deferredCost: -144,
			runLimit:     48976,
			dataStack:    [][]byte{{}},
		},
	}, {
		op: OP_CHECKSIG,
		startVM: &virtualMachine{
			runLimit: 50000,
			dataStack: [][]byte{
				mustDecodeHex("badabdf4bbb34f4a089efc298234f84fd909def662a8df03b4d7d40372728851" +
					"fbd3bf59920af5a7c361a4851967714271d1727e3be417a60053c30969d8860c"),
				mustDecodeHex("916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"),
				mustDecodeHex("ab3220d065dc875c6a5b4ecc39809b5f24eb0a605e9eef5190457edbf1e3b866"),
			},
		},
		wantVM: &virtualMachine{
			deferredCost: -144,
			runLimit:     48976,
			dataStack:    [][]byte{{}},
		},
	}, {
		op: OP_CHECKSIG,
		startVM: &virtualMachine{
			runLimit: 50000,
			dataStack: [][]byte{
				mustDecodeHex("af5abdf4bbb34f4a089efc298234f84fd909def662a8df03b4d7d40372728851" +
					"fbd3bf59920af5a7c361a4851967714271d1727e3be417a60053c30969d8860c"),
				mustDecodeHex("badbad"),
				mustDecodeHex("ab3220d065dc875c6a5b4ecc39809b5f24eb0a605e9eef5190457edbf1e3b866"),
			},
		},
		wantErr: ErrBadValue,
	}, {
		op: OP_CHECKSIG,
		startVM: &virtualMachine{
			runLimit: 0,
		},
		wantErr: ErrRunLimitExceeded,
	}, {
		op: OP_CHECKMULTISIG,
		startVM: &virtualMachine{
			runLimit: 50000,
			dataStack: [][]byte{
				mustDecodeHex("af5abdf4bbb34f4a089efc298234f84fd909def662a8df03b4d7d40372728851fbd3bf59920af5a7c361a4851967714271d1727e3be417a60053c30969d8860c"),
				mustDecodeHex("916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"),
				mustDecodeHex("ab3220d065dc875c6a5b4ecc39809b5f24eb0a605e9eef5190457edbf1e3b866"),
				{1},
				{1},
			},
		},
		wantVM: &virtualMachine{
			deferredCost: -161,
			runLimit:     48976,
			dataStack:    [][]byte{{1}},
		},
	}, {
		op: OP_CHECKMULTISIG,
		startVM: &virtualMachine{
			runLimit: 50000,
			dataStack: [][]byte{
				mustDecodeHex("badabdf4bbb34f4a089efc298234f84fd909def662a8df03b4d7d40372728851fbd3bf59920af5a7c361a4851967714271d1727e3be417a60053c30969d8860c"),
				mustDecodeHex("916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"),
				mustDecodeHex("ab3220d065dc875c6a5b4ecc39809b5f24eb0a605e9eef5190457edbf1e3b866"),
				{1},
				{1},
			},
		},
		wantVM: &virtualMachine{
			deferredCost: -162,
			runLimit:     48976,
			dataStack:    [][]byte{{}},
		},
	}, {
		op: OP_CHECKMULTISIG,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{},
		},
		wantErr: ErrDataStackUnderflow,
	}, {
		op: OP_CHECKMULTISIG,
		startVM: &virtualMachine{
			runLimit: 50000,
			dataStack: [][]byte{
				{1},
				{1},
			},
		},
		wantErr: ErrDataStackUnderflow,
	}, {
		op: OP_CHECKMULTISIG,
		startVM: &virtualMachine{
			runLimit: 50000,
			dataStack: [][]byte{
				mustDecodeHex("916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"),
				{1},
				{1},
			},
		},
		wantErr: ErrDataStackUnderflow,
	}, {
		op: OP_CHECKMULTISIG,
		startVM: &virtualMachine{
			runLimit: 50000,
			dataStack: [][]byte{
				mustDecodeHex("ab3220d065dc875c6a5b4ecc39809b5f24eb0a605e9eef5190457edbf1e3b866"),
				mustDecodeHex("916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"),
				{1},
				{1},
			},
		},
		wantErr: ErrDataStackUnderflow,
	}, {
		op: OP_CHECKMULTISIG,
		startVM: &virtualMachine{
			runLimit: 50000,
			dataStack: [][]byte{
				mustDecodeHex("af5abdf4bbb34f4a089efc298234f84fd909def662a8df03b4d7d40372728851fbd3bf59920af5a7c361a4851967714271d1727e3be417a60053c30969d8860c"),
				mustDecodeHex("badbad"),
				mustDecodeHex("ab3220d065dc875c6a5b4ecc39809b5f24eb0a605e9eef5190457edbf1e3b866"),
				{1},
				{1},
			},
		},
		wantErr: ErrBadValue,
	}, {
		op: OP_CHECKMULTISIG,
		startVM: &virtualMachine{
			runLimit: 50000,
			dataStack: [][]byte{
				mustDecodeHex("af5abdf4bbb34f4a089efc298234f84fd909def662a8df03b4d7d40372728851fbd3bf59920af5a7c361a4851967714271d1727e3be417a60053c30969d8860c"),
				mustDecodeHex("916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"),
				mustDecodeHex("ab3220d065dc875c6a5b4ecc39809b5f24eb0a605e9eef5190457edbf1e3b866"),
				{1},
				{0},
			},
		},
		wantErr: ErrBadValue,
	}, {
		op: OP_CHECKMULTISIG,
		startVM: &virtualMachine{
			runLimit: 50000,
			dataStack: [][]byte{
				mustDecodeHex("af5abdf4bbb34f4a089efc298234f84fd909def662a8df03b4d7d40372728851fbd3bf59920af5a7c361a4851967714271d1727e3be417a60053c30969d8860c"),
				mustDecodeHex("916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"),
				mustDecodeHex("ab3220d065dc875c6a5b4ecc39809b5f24eb0a605e9eef5190457edbf1e3b866"),
				{0},
				{1},
			},
		},
		wantErr: ErrBadValue,
	}, {
		op: OP_CHECKMULTISIG,
		startVM: &virtualMachine{
			runLimit: 50000,
			dataStack: [][]byte{
				mustDecodeHex("af5abdf4bbb34f4a089efc298234f84fd909def662a8df03b4d7d40372728851fbd3bf59920af5a7c361a4851967714271d1727e3be417a60053c30969d8860c"),
				mustDecodeHex("916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"),
				mustDecodeHex("ab3220d065dc875c6a5b4ecc39809b5f24eb0a605e9eef5190457edbf1e3b866"),
				{2},
				{1},
			},
		},
		wantErr: ErrBadValue,
	}, {
		op: OP_CHECKMULTISIG,
		startVM: &virtualMachine{
			runLimit: 0,
			dataStack: [][]byte{
				mustDecodeHex("af5abdf4bbb34f4a089efc298234f84fd909def662a8df03b4d7d40372728851fbd3bf59920af5a7c361a4851967714271d1727e3be417a60053c30969d8860c"),
				mustDecodeHex("916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"),
				mustDecodeHex("ab3220d065dc875c6a5b4ecc39809b5f24eb0a605e9eef5190457edbf1e3b866"),
				{1},
				{1},
			},
		},
		wantErr: ErrRunLimitExceeded,
	}, {
		op: OP_TXSIGHASH,
		startVM: &virtualMachine{
			runLimit: 50000,
			context: &Context{
				TxSigHash: func() []byte {
					return []byte{
						0x2f, 0x00, 0x3c, 0xdd, 0x64, 0x42, 0x7b, 0x5e,
						0xed, 0xd6, 0xcc, 0xb5, 0x85, 0x47, 0x02, 0x0b,
						0x02, 0xde, 0xf2, 0x2d, 0xc5, 0x99, 0x7e, 0x9d,
						0xa9, 0xac, 0x40, 0x49, 0xc3, 0x4a, 0x58, 0xd8,
					}
				},
			},
		},
		wantVM: &virtualMachine{
			runLimit: 49704,
			dataStack: [][]byte{{
				47, 0, 60, 221, 100, 66, 123, 94,
				237, 214, 204, 181, 133, 71, 2, 11,
				2, 222, 242, 45, 197, 153, 126, 157,
				169, 172, 64, 73, 195, 74, 88, 216,
			}},
		},
	}, {
		op: OP_TXSIGHASH,
		startVM: &virtualMachine{
			runLimit: 0,
			context:  &Context{},
		},
		wantErr: ErrRunLimitExceeded,
	}, {
		op: OP_BLOCKHASH,
		startVM: &virtualMachine{
			runLimit: 50000,
			context:  emptyBlockVMContext,
		},
		wantVM: &virtualMachine{
			runLimit: 49959,
			dataStack: [][]byte{{
				240, 133, 79, 136, 180, 137, 0, 153,
				47, 236, 64, 67, 249, 101, 250, 2,
				157, 235, 138, 214, 147, 207, 55, 17,
				254, 131, 9, 179, 144, 106, 90, 134,
			}},
			context: emptyBlockVMContext,
		},
	}, {
		op: OP_BLOCKHASH,
		startVM: &virtualMachine{
			runLimit: 0,
			context:  emptyBlockVMContext,
		},
		wantErr: ErrRunLimitExceeded,
	}}

	hashOps := []Op{OP_SHA256, OP_SHA3}
	for _, op := range hashOps {
		cases = append(cases, testStruct{
			op: op,
			startVM: &virtualMachine{
				runLimit:  0,
				dataStack: [][]byte{{1}},
			},
			wantErr: ErrRunLimitExceeded,
		})
	}

	for i, c := range cases {
		t.Logf("case %d", i)

		err := ops[c.op].fn(c.startVM)
		gotVM := c.startVM

		if err != c.wantErr {
			t.Errorf("case %d, op %s: got err = %v want %v", i, ops[c.op].name, err, c.wantErr)
			continue
		}
		if c.wantErr != nil {
			continue
		}

		// Hack: the context objects will otherwise compare unequal
		// sometimes (because of the function pointer within?) and we
		// don't care
		c.wantVM.context = gotVM.context

		if !testutil.DeepEqual(gotVM, c.wantVM) {
			t.Errorf("case %d, op %s: unexpected vm result\n\tgot:  %+v\n\twant: %+v\n", i, ops[c.op].name, gotVM, c.wantVM)
		}
	}
}

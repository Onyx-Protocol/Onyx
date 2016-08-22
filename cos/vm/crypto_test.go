package vm

import (
	"chain/cos/bc"
	"chain/crypto/ed25519/hd25519"
	"encoding/hex"
	"reflect"
	"testing"
)

func TestCryptoOps(t *testing.T) {
	tx := bc.NewTx(bc.TxData{
		Inputs:  []*bc.TxInput{bc.NewSpendInput(bc.Hash{}, 0, nil, bc.AssetID{}, 5, nil, nil)},
		Outputs: []*bc.TxOutput{},
	})

	type testStruct struct {
		op      Op
		startVM *virtualMachine
		wantErr error
		wantVM  *virtualMachine
	}
	cases := []testStruct{{
		op: OP_RIPEMD160,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{1}},
		},
		wantVM: &virtualMachine{
			runLimit: 49917,
			dataStack: [][]byte{{
				242, 145, 186, 80, 21, 223, 52, 140, 128, 133,
				63, 165, 187, 15, 121, 70, 245, 201, 225, 179,
			}},
		},
	}, {
		op: OP_RIPEMD160,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{make([]byte, 65)},
		},
		wantVM: &virtualMachine{
			runLimit: 49980,
			dataStack: [][]byte{{
				171, 60, 102, 205, 10, 63, 18, 180, 244, 250,
				235, 84, 138, 85, 22, 7, 148, 250, 215, 6,
			}},
		},
	}, {
		op: OP_SHA1,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{{1}},
		},
		wantVM: &virtualMachine{
			runLimit: 49917,
			dataStack: [][]byte{{
				191, 139, 69, 48, 216, 210, 70, 221, 116, 172,
				83, 161, 52, 113, 187, 161, 121, 65, 223, 247,
			}},
		},
	}, {
		op: OP_SHA1,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{make([]byte, 65)},
		},
		wantVM: &virtualMachine{
			runLimit: 49980,
			dataStack: [][]byte{{
				240, 250, 69, 144, 107, 208, 244, 195, 102, 143,
				205, 13, 143, 104, 212, 178, 152, 179, 14, 91,
			}},
		},
	}, {
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
			runLimit:  49119,
			dataStack: [][]byte{{1}},
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
			runLimit:  49120,
			dataStack: [][]byte{{}},
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
			runLimit:  49120,
			dataStack: [][]byte{{}},
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
			runLimit:  49120,
			dataStack: [][]byte{{}},
		},
	}, {
		op: OP_CHECKSIG,
		startVM: &virtualMachine{
			runLimit:  50000,
			dataStack: [][]byte{},
		},
		wantErr: ErrDataStackUnderflow,
	}, {
		op: OP_CHECKSIG,
		startVM: &virtualMachine{
			runLimit: 50000,
			dataStack: [][]byte{
				mustDecodeHex("ab3220d065dc875c6a5b4ecc39809b5f24eb0a605e9eef5190457edbf1e3b866"),
			},
		},
		wantErr: ErrDataStackUnderflow,
	}, {
		op: OP_CHECKSIG,
		startVM: &virtualMachine{
			runLimit: 50000,
			dataStack: [][]byte{
				mustDecodeHex("916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"),
				mustDecodeHex("ab3220d065dc875c6a5b4ecc39809b5f24eb0a605e9eef5190457edbf1e3b866"),
			},
		},
		wantErr: ErrDataStackUnderflow,
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
			runLimit: 50000,
			dataStack: [][]byte{
				mustDecodeHex("af5abdf4bbb34f4a089efc298234f84fd909def662a8df03b4d7d40372728851" +
					"fbd3bf59920af5a7c361a4851967714271d1727e3be417a60053c30969d8860c"),
				mustDecodeHex("916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"),
				mustDecodeHex("badbad"),
			},
		},
		wantErr: hd25519.ErrEOT,
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
				mustDecodeHex("af5abdf4bbb34f4a089efc298234f84fd909def662a8df03b4d7d40372728851" +
					"fbd3bf59920af5a7c361a4851967714271d1727e3be417a60053c30969d8860c"),
				{1},
				mustDecodeHex("ab3220d065dc875c6a5b4ecc39809b5f24eb0a605e9eef5190457edbf1e3b866"),
				{1},
				mustDecodeHex("916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"),
			},
		},
		wantVM: &virtualMachine{
			runLimit:  49137,
			dataStack: [][]byte{{1}},
		},
	}, {
		op: OP_CHECKMULTISIG,
		startVM: &virtualMachine{
			runLimit: 50000,
			dataStack: [][]byte{
				mustDecodeHex("badabdf4bbb34f4a089efc298234f84fd909def662a8df03b4d7d40372728851" +
					"fbd3bf59920af5a7c361a4851967714271d1727e3be417a60053c30969d8860c"),
				{1},
				mustDecodeHex("ab3220d065dc875c6a5b4ecc39809b5f24eb0a605e9eef5190457edbf1e3b866"),
				{1},
				mustDecodeHex("916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"),
			},
		},
		wantVM: &virtualMachine{
			runLimit:  49138,
			dataStack: [][]byte{{}},
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
				mustDecodeHex("916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"),
			},
		},
		wantErr: ErrDataStackUnderflow,
	}, {
		op: OP_CHECKMULTISIG,
		startVM: &virtualMachine{
			runLimit: 50000,
			dataStack: [][]byte{
				{1},
				mustDecodeHex("916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"),
			},
		},
		wantErr: ErrDataStackUnderflow,
	}, {
		op: OP_CHECKMULTISIG,
		startVM: &virtualMachine{
			runLimit: 50000,
			dataStack: [][]byte{
				mustDecodeHex("ab3220d065dc875c6a5b4ecc39809b5f24eb0a605e9eef5190457edbf1e3b866"),
				{1},
				mustDecodeHex("916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"),
			},
		},
		wantErr: ErrDataStackUnderflow,
	}, {
		op: OP_CHECKMULTISIG,
		startVM: &virtualMachine{
			runLimit: 50000,
			dataStack: [][]byte{
				{1},
				mustDecodeHex("ab3220d065dc875c6a5b4ecc39809b5f24eb0a605e9eef5190457edbf1e3b866"),
				{1},
				mustDecodeHex("916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"),
			},
		},
		wantErr: ErrDataStackUnderflow,
	}, {
		op: OP_CHECKMULTISIG,
		startVM: &virtualMachine{
			runLimit: 50000,
			dataStack: [][]byte{
				mustDecodeHex("af5abdf4bbb34f4a089efc298234f84fd909def662a8df03b4d7d40372728851" +
					"fbd3bf59920af5a7c361a4851967714271d1727e3be417a60053c30969d8860c"),
				{1},
				mustDecodeHex("ab3220d065dc875c6a5b4ecc39809b5f24eb0a605e9eef5190457edbf1e3b866"),
				{1},
				mustDecodeHex("badbad"),
			},
		},
		wantErr: ErrBadValue,
	}, {
		op: OP_CHECKMULTISIG,
		startVM: &virtualMachine{
			runLimit: 50000,
			dataStack: [][]byte{
				mustDecodeHex("af5abdf4bbb34f4a089efc298234f84fd909def662a8df03b4d7d40372728851" +
					"fbd3bf59920af5a7c361a4851967714271d1727e3be417a60053c30969d8860c"),
				{1},
				mustDecodeHex("ab3220d065dc875c6a5b4ecc39809b5f24eb0a605e9eef5190457edbf1e3b866"),
				{0},
				mustDecodeHex("916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"),
			},
		},
		wantErr: ErrBadValue,
	}, {
		op: OP_CHECKMULTISIG,
		startVM: &virtualMachine{
			runLimit: 50000,
			dataStack: [][]byte{
				mustDecodeHex("af5abdf4bbb34f4a089efc298234f84fd909def662a8df03b4d7d40372728851" +
					"fbd3bf59920af5a7c361a4851967714271d1727e3be417a60053c30969d8860c"),
				{0},
				mustDecodeHex("ab3220d065dc875c6a5b4ecc39809b5f24eb0a605e9eef5190457edbf1e3b866"),
				{1},
				mustDecodeHex("916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"),
			},
		},
		wantErr: ErrBadValue,
	}, {
		op: OP_CHECKMULTISIG,
		startVM: &virtualMachine{
			runLimit: 50000,
			dataStack: [][]byte{
				mustDecodeHex("af5abdf4bbb34f4a089efc298234f84fd909def662a8df03b4d7d40372728851" +
					"fbd3bf59920af5a7c361a4851967714271d1727e3be417a60053c30969d8860c"),
				{2},
				mustDecodeHex("ab3220d065dc875c6a5b4ecc39809b5f24eb0a605e9eef5190457edbf1e3b866"),
				{1},
				mustDecodeHex("916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"),
			},
		},
		wantErr: ErrBadValue,
	}, {
		op: OP_CHECKMULTISIG,
		startVM: &virtualMachine{
			runLimit: 50000,
			dataStack: [][]byte{
				mustDecodeHex("af5abdf4bbb34f4a089efc298234f84fd909def662a8df03b4d7d40372728851" +
					"fbd3bf59920af5a7c361a4851967714271d1727e3be417a60053c30969d8860c"),
				{1},
				mustDecodeHex("badbad"),
				{1},
				mustDecodeHex("916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"),
			},
		},
		wantErr: hd25519.ErrEOT,
	}, {
		op: OP_CHECKMULTISIG,
		startVM: &virtualMachine{
			runLimit: 0,
			dataStack: [][]byte{
				mustDecodeHex("af5abdf4bbb34f4a089efc298234f84fd909def662a8df03b4d7d40372728851" +
					"fbd3bf59920af5a7c361a4851967714271d1727e3be417a60053c30969d8860c"),
				{1},
				mustDecodeHex("ab3220d065dc875c6a5b4ecc39809b5f24eb0a605e9eef5190457edbf1e3b866"),
				{1},
				mustDecodeHex("916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"),
			},
		},
		wantErr: ErrRunLimitExceeded,
	}, {
		op: OP_TXSIGHASH,
		startVM: &virtualMachine{
			runLimit:  50000,
			tx:        tx,
			sigHasher: bc.NewSigHasher(&tx.TxData),
			dataStack: [][]byte{{byte(bc.SigHashAll)}},
		},
		wantVM: &virtualMachine{
			runLimit: 49841,
			tx:       tx,
			dataStack: [][]byte{{
				229, 93, 29, 146, 0, 210, 76, 184, 119, 144, 206, 162, 80, 95, 125, 125,
				239, 185, 25, 190, 4, 197, 2, 114, 27, 237, 98, 106, 8, 68, 152, 3,
			}},
		},
	}, {
		op: OP_TXSIGHASH,
		startVM: &virtualMachine{
			runLimit:  0,
			tx:        tx,
			sigHasher: bc.NewSigHasher(&tx.TxData),
			dataStack: [][]byte{{byte(bc.SigHashAll)}},
		},
		wantErr: ErrRunLimitExceeded,
	}, {
		op: OP_TXSIGHASH,
		startVM: &virtualMachine{
			runLimit:  50000,
			tx:        tx,
			sigHasher: bc.NewSigHasher(&tx.TxData),
			dataStack: [][]byte{},
		},
		wantErr: ErrDataStackUnderflow,
	}, {
		op: OP_TXSIGHASH,
		startVM: &virtualMachine{
			runLimit:  50000,
			tx:        nil,
			sigHasher: bc.NewSigHasher(&tx.TxData),
			dataStack: [][]byte{{byte(bc.SigHashAll)}},
		},
		wantErr: ErrContext,
	}, {
		op: OP_BLOCKSIGHASH,
		startVM: &virtualMachine{
			runLimit: 50000,
			block:    &bc.Block{},
		},
		wantVM: &virtualMachine{
			runLimit: 49832,
			dataStack: [][]byte{{
				46, 87, 204, 195, 74, 20, 1, 41, 253, 183, 90, 121, 57, 8, 151, 70,
				184, 65, 6, 185, 30, 180, 112, 95, 211, 21, 21, 49, 218, 27, 166, 88,
			}},
			block: &bc.Block{},
		},
	}, {
		op: OP_BLOCKSIGHASH,
		startVM: &virtualMachine{
			runLimit: 0,
			block:    &bc.Block{},
		},
		wantErr: ErrRunLimitExceeded,
	}, {
		op: OP_BLOCKSIGHASH,
		startVM: &virtualMachine{
			runLimit: 50000,
			block:    nil,
		},
		wantErr: ErrContext,
	}}

	hashOps := []Op{OP_RIPEMD160, OP_SHA1, OP_SHA256, OP_SHA3}
	for _, op := range hashOps {
		cases = append(cases, testStruct{
			op: op,
			startVM: &virtualMachine{
				runLimit:  0,
				dataStack: [][]byte{{1}},
			},
			wantErr: ErrRunLimitExceeded,
		}, testStruct{
			op: op,
			startVM: &virtualMachine{
				runLimit:  50000,
				dataStack: [][]byte{},
			},
			wantErr: ErrDataStackUnderflow,
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

func mustDecodeHex(h string) []byte {
	bits, err := hex.DecodeString(h)
	if err != nil {
		panic(err)
	}
	return bits
}

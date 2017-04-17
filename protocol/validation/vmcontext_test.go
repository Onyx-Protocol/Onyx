package validation

import (
	"encoding/hex"
	"fmt"
	"testing"

	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/bc/legacy"
	"chain/protocol/vm"
)

func TestCheckOutput(t *testing.T) {
	tx := legacy.NewTx(legacy.TxData{
		ReferenceData: []byte("txref"),
		Inputs: []*legacy.TxInput{
			legacy.NewSpendInput(nil, bc.Hash{}, bc.NewAssetID([32]byte{1}), 5, 1, []byte("spendprog"), bc.Hash{}, []byte("ref")),
			legacy.NewIssuanceInput(nil, 6, nil, bc.Hash{}, []byte("issueprog"), nil, nil),
		},
		Outputs: []*legacy.TxOutput{
			legacy.NewTxOutput(bc.NewAssetID([32]byte{3}), 8, []byte("wrongprog"), nil),
			legacy.NewTxOutput(bc.NewAssetID([32]byte{3}), 8, []byte("controlprog"), nil),
			legacy.NewTxOutput(bc.NewAssetID([32]byte{2}), 8, []byte("controlprog"), nil),
			legacy.NewTxOutput(bc.NewAssetID([32]byte{2}), 7, []byte("controlprog"), nil),
			legacy.NewTxOutput(bc.NewAssetID([32]byte{2}), 7, []byte("controlprog"), []byte("outref")),
		},
		MinTime: 0,
		MaxTime: 20,
	})

	txCtx := &entryContext{
		entry:   tx.Tx.Entries[tx.Tx.InputIDs[0]],
		entries: tx.Tx.Entries,
	}

	cases := []struct {
		// args to CheckOutput
		index     uint64
		data      []byte
		amount    uint64
		assetID   []byte
		vmVersion uint64
		code      []byte

		wantErr error
		wantOk  bool
	}{
		{
			index:     4,
			data:      mustDecodeHex("1f2a05f881ed9fa0c9068a84823677409f863891a2196eb55dbfbb677a566374"),
			amount:    7,
			assetID:   append([]byte{2}, make([]byte, 31)...),
			vmVersion: 1,
			code:      []byte("controlprog"),
			wantOk:    true,
		},
		{
			index:     3,
			data:      mustDecodeHex("1f2a05f881ed9fa0c9068a84823677409f863891a2196eb55dbfbb677a566374"),
			amount:    7,
			assetID:   append([]byte{2}, make([]byte, 31)...),
			vmVersion: 1,
			code:      []byte("controlprog"),
			wantOk:    false,
		},
		{
			index:     0,
			data:      []byte{},
			amount:    1,
			assetID:   append([]byte{9}, make([]byte, 31)...),
			vmVersion: 1,
			code:      []byte("missingprog"),
			wantOk:    false,
		},
		{
			index:     5,
			data:      mustDecodeHex("1f2a05f881ed9fa0c9068a84823677409f863891a2196eb55dbfbb677a566374"),
			amount:    7,
			assetID:   append([]byte{2}, make([]byte, 31)...),
			vmVersion: 1,
			code:      []byte("controlprog"),
			wantErr:   vm.ErrBadValue,
		},
	}

	for i, test := range cases {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			gotOk, err := txCtx.checkOutput(test.index, test.data, test.amount, test.assetID, test.vmVersion, test.code, false)
			if g := errors.Root(err); g != test.wantErr {
				t.Errorf("checkOutput(%v, %v, %v, %x, %v, %x) err = %v, want %v",
					test.index, test.data, test.amount, test.assetID, test.vmVersion, test.code,
					g, test.wantErr)
				return
			}
			if gotOk != test.wantOk {
				t.Errorf("checkOutput(%v, %v, %v, %x, %v, %x) ok = %v, want %v",
					test.index, test.data, test.amount, test.assetID, test.vmVersion, test.code,
					gotOk, test.wantOk)
			}

		})
	}
}

func mustDecodeHex(h string) []byte {
	bits, err := hex.DecodeString(h)
	if err != nil {
		panic(err)
	}
	return bits
}

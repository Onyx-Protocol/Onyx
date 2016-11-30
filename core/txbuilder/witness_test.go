package txbuilder

import (
	"bytes"
	"testing"

	"chain/core/pb"
	"chain/protocol/bc"
	"chain/protocol/vm"
)

func TestInferConstraints(t *testing.T) {
	tpl := &Template{
		Tx: &bc.TxData{
			Inputs: []*bc.TxInput{
				bc.NewSpendInput(bc.Hash{}, 1, nil, bc.AssetID{}, 123, nil, []byte{1}),
			},
			Outputs: []*bc.TxOutput{
				bc.NewTxOutput(bc.AssetID{}, 123, []byte{10, 11, 12}, nil),
			},
			MinTime: 1,
			MaxTime: 2,
		},
		TxTemplate: &pb.TxTemplate{
			AllowAdditionalActions: true,
		},
	}
	prog := buildSigProgram(tpl, 0)
	want, err := vm.Assemble("MINTIME 1 GREATERTHANOREQUAL VERIFY MAXTIME 2 LESSTHANOREQUAL VERIFY 0x0000000000000000000000000000000000000000000000000000000000000000 1 OUTPOINT ROT NUMEQUAL VERIFY EQUAL VERIFY 0x2767f15c8af2f2c7225d5273fdd683edc714110a987d1054697c348aed4e6cc7 REFDATAHASH EQUAL VERIFY 0 0 123 0x0000000000000000000000000000000000000000000000000000000000000000 1 0x0a0b0c CHECKOUTPUT")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(want, prog) {
		t.Errorf("expected sig witness program %x, got %x", want, prog)
	}
}

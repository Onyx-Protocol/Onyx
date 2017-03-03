package txbuilder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/davecgh/go-spew/spew"

	chainjson "chain/encoding/json"
	"chain/protocol/bc"
	"chain/protocol/vm"
	"chain/testutil"
)

func TestInferConstraints(t *testing.T) {
	tpl := &Template{
		Transaction: bc.NewTx(bc.TxData{
			Inputs: []*bc.TxInput{
				bc.NewSpendInput(nil, bc.Hash{}, bc.AssetID{}, 123, 0, nil, bc.Hash{}, []byte{1}),
			},
			Outputs: []*bc.TxOutput{
				bc.NewTxOutput(bc.AssetID{}, 123, []byte{10, 11, 12}, nil),
			},
			MinTime: 1,
			MaxTime: 2,
		}),
		AllowAdditional: true,
	}
	prog := buildSigProgram(tpl, 0)
	wantSrc := fmt.Sprintf("MINTIME 1 GREATERTHANOREQUAL VERIFY MAXTIME 2 LESSTHANOREQUAL VERIFY 0x%x OUTPUTID EQUAL VERIFY 0x2767f15c8af2f2c7225d5273fdd683edc714110a987d1054697c348aed4e6cc7 REFDATAHASH EQUAL VERIFY 0 0 123 0x0000000000000000000000000000000000000000000000000000000000000000 1 0x0a0b0c CHECKOUTPUT", tpl.Transaction.SpentOutputIDs[0].Bytes())
	want, err := vm.Assemble(wantSrc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(want, prog) {
		progSrc, _ := vm.Disassemble(prog)
		t.Errorf("expected sig witness program %x [%s], got %x [%s]", want, wantSrc, prog, progSrc)
	}
}

func TestWitnessJSON(t *testing.T) {
	si := &SigningInstruction{
		AssetAmount: bc.AssetAmount{
			AssetID: bc.AssetID{0xff},
			Amount:  21,
		},
		Position: 17,
		WitnessComponents: []WitnessComponent{
			&SignatureWitness{
				Quorum: 4,
				Keys: []KeyID{{
					XPub:           testutil.TestXPub,
					DerivationPath: []chainjson.HexBytes{{5, 6, 7}},
				}},
				Sigs: []chainjson.HexBytes{{8, 9, 10}},
			},
		},
	}

	b, err := json.MarshalIndent(si, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	var got SigningInstruction
	err = json.Unmarshal(b, &got)
	if err != nil {
		t.Fatal(err)
	}

	if !testutil.DeepEqual(si, &got) {
		t.Errorf("got:\n%s\nwant:\n%s\nJSON was: %s", spew.Sdump(&got), spew.Sdump(si), string(b))
	}
}

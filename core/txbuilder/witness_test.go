package txbuilder

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"

	chainjson "chain-stealth/encoding/json"
	"chain-stealth/protocol/bc"
	"chain-stealth/protocol/vm"
)

func TestInferConstraints(t *testing.T) {
	tpl := &Template{
		Transaction: &bc.TxData{
			Inputs: []*bc.TxInput{
				bc.NewSpendInput(bc.Hash{}, 1, nil, bc.AssetID{}, 123, nil, []byte{1}),
			},
			Outputs: []*bc.TxOutput{
				bc.NewTxOutput(bc.AssetID{}, 123, []byte{10, 11, 12}, nil),
			},
			MinTime: 1,
			MaxTime: 2,
		},
		AllowAdditional: true,
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
					XPub:           "fd",
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

	if !reflect.DeepEqual(si, &got) {
		t.Errorf("got:\n%s\nwant:\n%s\nJSON was: %s", spew.Sdump(&got), spew.Sdump(si), string(b))
	}
}

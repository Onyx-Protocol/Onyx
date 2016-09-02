package txbuilder

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"

	chainjson "chain/encoding/json"
	"chain/protocol/bc"
)

func TestWitnessJSON(t *testing.T) {
	inp := &Input{
		AssetAmount: bc.AssetAmount{
			AssetID: bc.AssetID{0xff},
			Amount:  21,
		},
		Position: 17,
		WitnessComponents: []WitnessComponent{
			DataWitness{1, 2, 3},
			&SignatureWitness{
				Quorum:        4,
				SignatureData: bc.Hash{0xfe},
				Signatures: []*Signature{
					{
						XPub:           "fd",
						DerivationPath: []uint32{5, 6, 7},
						Bytes:          chainjson.HexBytes{8, 9, 10},
					},
				},
			},
		},
	}

	b, err := json.Marshal(inp)
	if err != nil {
		t.Fatal(err)
	}

	var got Input
	err = json.Unmarshal(b, &got)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(inp, &got) {
		t.Errorf("got:\n%s\nwant:\n%s", spew.Sdump(&got), spew.Sdump(inp))
	}
}

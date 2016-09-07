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
				Quorum: 4,
				Keys: []KeyID{{
					XPub:           "fd",
					DerivationPath: []uint32{5, 6, 7},
				}},
				Constraints: []Constraint{
					TTLConstraint(11),
					OutpointConstraint(bc.Outpoint{
						Hash:  bc.Hash{0xfc},
						Index: 38,
					}),
					&PayConstraint{
						AssetAmount: bc.AssetAmount{
							AssetID: bc.AssetID{0xfa},
							Amount:  39,
						},
						Program:     chainjson.HexBytes{40, 41, 42},
						RefDataHash: &bc.Hash{43, 44},
					},
				},
				Sigs: []chainjson.HexBytes{{8, 9, 10}},
			},
		},
	}

	b, err := json.MarshalIndent(inp, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	var got Input
	err = json.Unmarshal(b, &got)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(inp, &got) {
		t.Errorf("got:\n%s\nwant:\n%s\nJSON was: %s", spew.Sdump(&got), spew.Sdump(inp), string(b))
	}
}

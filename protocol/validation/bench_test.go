package validation

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"testing"

	"chain/protocol/bc"
	"chain/protocol/state"
)

func BenchmarkValidateBlock(b *testing.B) {
	b.StopTimer()
	ctx := context.Background()
	jsonBlocks, err := ioutil.ReadFile("./blocks.json")
	if err != nil {
		b.Fatal(err)
	}

	var blocks []*bc.Block
	err = json.Unmarshal(jsonBlocks, &blocks)
	if err != nil {
		b.Fatal(err)
	}
	if blocks[0].Height != 1 {
		b.Fatal("first test block must have height 1")
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		var current *bc.Block
		snapshot := state.NewSnapshot(blocks[0].Hash())
		for _, block := range blocks {
			err := ValidateBlockForAccept(ctx, snapshot, current, block, CheckTxWellFormed)
			if err != nil {
				b.Fatal(err)
			}
			current = block
		}
	}
}

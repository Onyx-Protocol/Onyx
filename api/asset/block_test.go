package asset

import (
	"reflect"
	"testing"
	"time"

	"golang.org/x/net/context"

	"chain/fedchain/bc"
)

func TestGenerateBlock(t *testing.T) {
	const fix = `
		INSERT INTO blocks (block_hash, height, data)
		VALUES(
			'341fb89912be0110b527375998810c99ac96a317c63b071ccf33b7514cf0f0a5',
			1,
			decode('0000000100000000000000013132330000000000000000000000000000000000000000000000000000000000414243000000000000000000000000000000000000000000000000000000000058595a000000000000000000000000000000000000000000000000000000000000000000000000640f746573742d7369672d73637269707412746573742d6f75747075742d73637269707401000000010000000000000000000007746573742d7478', 'hex')
		);

		INSERT INTO pool_txs (tx_hash, data, sort_id)
		VALUES (
			'37383ebfffe807d694343a9004a42f605592e0dc7f7d5de76857fb46a7050410',
			decode('0000000101341fb89912be0110b527375998810c99ac96a317c63b071ccf33b7514cf0f0a5ffffffff6f00473045022100c561a9b4854742bc36c805513b872b2c0a1a367da24710eadd4f3fbc3b1ab41302207cf9eec4e5db694831fe43cf193f23d869291025ac6062199dd6b8998e93e15825512103623fb1fe38ce7e43cf407ec99b061c6d2da0278e80ce094393875c5b94f1ed9051ae0001df03f294bd08930f542a42b91199a8afe1b45c28eeb058cc5e8c8d600e0dd42f0000000000000001000000000000000000000474782d31', 'hex'),
			1
		), (
			'5b3864897b701f217ae956c7ce2bbfb9ac415da38430b7d56acd104ca9b03ed6',
			decode('0000000101341fb89912be0110b527375998810c99ac96a317c63b071ccf33b7514cf0f0a5ffffffff6e0046304402206ac2db5b49c8f9059d7ecad4f08a1d29e851e321720f590f5426cfbb19840d4402206aacef503d7c3cd065a17c2553b372ca2de0613eba3debc70896c9ab6545029b25512103b050bdde9880d9e8634f12798748cb26e9435a778305f3ae1ddba759d6479b2a51ae00015abad6dfb0de611046ebda5de05bfebc6a08d9a71831b43f2acd554bf54f33180000000000000001000000000000000000000474782d32', 'hex'),
			2
		);
	`

	withContext(t, fix, func(t *testing.T, ctx context.Context) {
		now := time.Now()
		got, err := GenerateBlock(ctx, now)
		if err != nil {
			t.Fatalf("err got = %v want nil", err)
		}

		want := &bc.Block{
			BlockHeader: bc.BlockHeader{
				Version:           bc.NewBlockVersion,
				Height:            2,
				PreviousBlockHash: mustParseHash("341fb89912be0110b527375998810c99ac96a317c63b071ccf33b7514cf0f0a5"),
				Timestamp:         uint64(now.Unix()),
			},
			Transactions: []*bc.Tx{
				{
					Version:  1,
					Metadata: []byte("tx-1"),
					Inputs: []*bc.TxInput{{
						Previous: bc.Outpoint{
							Hash:  mustParseHash("341fb89912be0110b527375998810c99ac96a317c63b071ccf33b7514cf0f0a5"),
							Index: bc.InvalidOutputIndex,
						},
						SignatureScript: mustDecodeHex("00473045022100c561a9b4854742bc36c805513b872b2c0a1a367da24710eadd4f3fbc3b1ab41302207cf9eec4e5db694831fe43cf193f23d869291025ac6062199dd6b8998e93e15825512103623fb1fe38ce7e43cf407ec99b061c6d2da0278e80ce094393875c5b94f1ed9051ae"),
					}},
					Outputs: []*bc.TxOutput{{
						Value:   1,
						AssetID: mustParseHash("df03f294bd08930f542a42b91199a8afe1b45c28eeb058cc5e8c8d600e0dd42f"),
					}},
				},
				{
					Version:  1,
					Metadata: []byte("tx-2"),
					Inputs: []*bc.TxInput{{
						Previous: bc.Outpoint{
							Hash:  mustParseHash("341fb89912be0110b527375998810c99ac96a317c63b071ccf33b7514cf0f0a5"),
							Index: bc.InvalidOutputIndex,
						},
						SignatureScript: mustDecodeHex("0046304402206ac2db5b49c8f9059d7ecad4f08a1d29e851e321720f590f5426cfbb19840d4402206aacef503d7c3cd065a17c2553b372ca2de0613eba3debc70896c9ab6545029b25512103b050bdde9880d9e8634f12798748cb26e9435a778305f3ae1ddba759d6479b2a51ae"),
					}},
					Outputs: []*bc.TxOutput{{
						Value:   1,
						AssetID: mustParseHash("5abad6dfb0de611046ebda5de05bfebc6a08d9a71831b43f2acd554bf54f3318"),
					}},
				},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("generated block:\ngot:  %+v\nwant: %+v", got, want)
		}
	})
}

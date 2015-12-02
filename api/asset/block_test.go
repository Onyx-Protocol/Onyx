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
			'92b34025babea306bdf67cfe9a2576d8475ea9476caeb1fbdea43bf3d56d011a',
			11,
			decode('00000001000000000000000b95a00b5cd11f577a461e6bb884899ee0aa1662088097b644af7a50d76e1a243f0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000565f8903000001000000010195a00b5cd11f577a461e6bb884899ee0aa1662088097b644af7a50d76e1a243fffffffff7000483045022100c80b4deb9aae29da4e8768a5fbe0ac6ccca1d020f4b924005cc066f09b18e14e02206acd491a84eda9c15bed01a7648b191974a2ef47f7cefda3bde06092cd144e68012551210210b002870438af79b829bc22c4505e14779ef0080c411ad497d7a0846ee0af6f51ae00000125fbb43a93c290fde3997d92c416d3cc7ff40a13aa309d051406978635085c8d000000000000003217a9144bbae13b661c0cd9cf89271fd96dadb65e7a80378700000000000000000000', 'hex')
		);

		INSERT INTO pool_txs (tx_hash, data, sort_id)
		VALUES (
			'd8d804a9fae1dc447779eb9826116f32f22c83bef4ef228d6423e99a546deebd',
			decode('000000010192b34025babea306bdf67cfe9a2576d8475ea9476caeb1fbdea43bf3d56d011affffffff70004830450221009037e1d39b7d59d24eba8012baddd5f4ab886a51b46f52b7c479ddfa55eeb5c5022076008409243475b25dfba6db85e15cf3d74561a147375941e4830baa69769b51012551210210b002870438af79b829bc22c4505e14779ef0080c411ad497d7a0846ee0af6f51ae00137b0a2020226b6579223a2022636c616d220a7d0125fbb43a93c290fde3997d92c416d3cc7ff40a13aa309d051406978635085c8d000000000000003217a9145881cd104f8d64635751ac0f3c0decf9150c11068700000000000000000000', 'hex'),
			1
		), (
			'27764579c4cf0395c91c6941011b3e9a627b02f29b259e8f6bc5ca9c50c5f256',
			decode('000000010192b34025babea306bdf67cfe9a2576d8475ea9476caeb1fbdea43bf3d56d011affffffff7000483045022100f3bcffcfd6a1ce9542b653500386cd0ee7b9c86c59390ca0fc0238c0ebe3f1d6022065ac468a51a016842660c3a616c99a9aa5109a3bad1877ba3e0f010f3972472e012551210210b002870438af79b829bc22c4505e14779ef0080c411ad497d7a0846ee0af6f51ae00137b0a2020226b6579223a2022636c616d220a7d0125fbb43a93c290fde3997d92c416d3cc7ff40a13aa309d051406978635085c8d000000000000003217a914c171e443e05b953baa7b7d834028ed91e47b4d0b8700000000000000000000', 'hex'),
			2
		);
	`

	withContext(t, fix, func(ctx context.Context) {
		now := time.Now()
		got, err := GenerateBlock(ctx, now)
		if err != nil {
			t.Fatalf("err got = %v want nil", err)
		}

		want := &bc.Block{
			BlockHeader: bc.BlockHeader{
				Version:           bc.NewBlockVersion,
				Height:            12,
				PreviousBlockHash: mustParseHash("92b34025babea306bdf67cfe9a2576d8475ea9476caeb1fbdea43bf3d56d011a"),
				Timestamp:         uint64(now.Unix()),
			},
			Transactions: []*bc.Tx{
				{
					Version: 1,
					Inputs: []*bc.TxInput{{
						Previous: bc.Outpoint{
							Hash:  mustParseHash("92b34025babea306bdf67cfe9a2576d8475ea9476caeb1fbdea43bf3d56d011a"),
							Index: bc.InvalidOutputIndex,
						},
						SignatureScript: mustDecodeHex("004830450221009037e1d39b7d59d24eba8012baddd5f4ab886a51b46f52b7c479ddfa55eeb5c5022076008409243475b25dfba6db85e15cf3d74561a147375941e4830baa69769b51012551210210b002870438af79b829bc22c4505e14779ef0080c411ad497d7a0846ee0af6f51ae"),
						AssetDefinition: []byte(`{
  "key": "clam"
}`),
					}},
					Outputs: []*bc.TxOutput{{
						Value:   50,
						AssetID: mustParseHash("25fbb43a93c290fde3997d92c416d3cc7ff40a13aa309d051406978635085c8d"),
						Script:  mustDecodeHex("a9145881cd104f8d64635751ac0f3c0decf9150c110687"),
					}},
				},
				{
					Version: 1,
					Inputs: []*bc.TxInput{{
						Previous: bc.Outpoint{
							Hash:  mustParseHash("92b34025babea306bdf67cfe9a2576d8475ea9476caeb1fbdea43bf3d56d011a"),
							Index: bc.InvalidOutputIndex,
						},
						SignatureScript: mustDecodeHex("00483045022100f3bcffcfd6a1ce9542b653500386cd0ee7b9c86c59390ca0fc0238c0ebe3f1d6022065ac468a51a016842660c3a616c99a9aa5109a3bad1877ba3e0f010f3972472e012551210210b002870438af79b829bc22c4505e14779ef0080c411ad497d7a0846ee0af6f51ae"),
						AssetDefinition: []byte(`{
  "key": "clam"
}`),
					}},
					Outputs: []*bc.TxOutput{{
						Value:   50,
						AssetID: mustParseHash("25fbb43a93c290fde3997d92c416d3cc7ff40a13aa309d051406978635085c8d"),
						Script:  mustDecodeHex("a914c171e443e05b953baa7b7d834028ed91e47b4d0b87"),
					}},
				},
			},
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("generated block:\ngot:  %+v\nwant: %+v", got, want)
		}
	})
}

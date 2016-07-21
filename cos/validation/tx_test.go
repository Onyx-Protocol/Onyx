package validation

import (
	"chain/cos/bc"
	"chain/cos/patricia"
	"chain/cos/state"
	"chain/errors"
	"fmt"
	"testing"
	"time"
)

func TestNoUpdateEmptyAD(t *testing.T) {
	tree := patricia.NewTree(nil)
	tx := bc.NewTx(bc.TxData{Inputs: []*bc.TxInput{{
		SignatureScript: []byte("foo"),
		Previous:        bc.Outpoint{Index: bc.InvalidOutputIndex},
	}}})
	err := ApplyTx(tree, tx)
	if err != nil {
		t.Fatal(err)
	}

	k, _ := state.ADPTreeItem(bc.AssetID{}, bc.Hash{})
	if tree.Lookup(k) != nil {
		// If metadata field is empty, no update of ADP takes place.
		t.Fatal("apply tx should not save an empty asset definition")
	}
}

func TestTxIsWellFormed(t *testing.T) {
	aid1, aid2 := bc.AssetID([32]byte{1}), bc.AssetID([32]byte{2})
	prevout1 := bc.Outpoint{Hash: [32]byte{10}, Index: 0}
	prevout2 := bc.Outpoint{Hash: [32]byte{11}, Index: 0}
	issuance := bc.Outpoint{Index: bc.InvalidOutputIndex}

	testCases := []struct {
		badTx  bool
		detail string
		tx     bc.Tx
	}{
		{
			badTx:  true,
			detail: "inputs are missing",
			tx:     bc.Tx{}, // empty
		},
		{
			badTx:  true,
			detail: fmt.Sprintf("amounts for asset %s are not balanced on inputs and outputs", aid1),
			tx: bc.Tx{
				TxData: bc.TxData{
					Inputs: []*bc.TxInput{
						{
							Previous:    prevout1,
							AssetAmount: bc.AssetAmount{AssetID: aid1, Amount: 1000},
						},
					},
					Outputs: []*bc.TxOutput{
						bc.NewTxOutput(aid1, 999, nil, nil),
					},
				},
			},
		},
		{
			badTx:  true,
			detail: fmt.Sprintf("amounts for asset %s are not balanced on inputs and outputs", aid2),
			tx: bc.Tx{
				TxData: bc.TxData{
					Inputs: []*bc.TxInput{
						{
							Previous:    prevout1,
							AssetAmount: bc.AssetAmount{AssetID: aid1, Amount: 500},
						},
						{
							Previous:    prevout2,
							AssetAmount: bc.AssetAmount{AssetID: aid2, Amount: 500},
						},
					},
					Outputs: []*bc.TxOutput{
						bc.NewTxOutput(aid1, 500, nil, nil),
						bc.NewTxOutput(aid2, 1000, nil, nil),
					},
				},
			},
		},
		{
			// can have zero-amount outputs to re-publish asset definitions
			badTx: false,
			tx: bc.Tx{
				TxData: bc.TxData{
					Inputs: []*bc.TxInput{
						{
							Previous:    issuance,
							AssetAmount: bc.AssetAmount{AssetID: aid1, Amount: 0},
						},
					},
					Outputs: []*bc.TxOutput{
						bc.NewTxOutput(aid1, 0, nil, nil),
					},
				},
			},
		},
		{
			badTx:  true,
			detail: "non-issuance output value must be greater than 0",
			tx: bc.Tx{
				TxData: bc.TxData{
					Inputs: []*bc.TxInput{
						{
							Previous:    issuance,
							AssetAmount: bc.AssetAmount{AssetID: aid1, Amount: 0},
						},
						{
							Previous:    prevout1,
							AssetAmount: bc.AssetAmount{AssetID: aid2, Amount: 0},
						},
					},
					Outputs: []*bc.TxOutput{
						bc.NewTxOutput(aid1, 0, nil, nil),
						bc.NewTxOutput(aid2, 0, nil, nil),
					},
				},
			},
		},
		{
			badTx: false,
			tx: bc.Tx{
				TxData: bc.TxData{
					Inputs: []*bc.TxInput{
						{AssetAmount: bc.AssetAmount{AssetID: aid1, Amount: 1000}},
					},
					Outputs: []*bc.TxOutput{
						bc.NewTxOutput(aid1, 1000, nil, nil),
					},
				},
			},
		},
		{
			badTx: false,
			tx: bc.Tx{
				TxData: bc.TxData{
					Inputs: []*bc.TxInput{
						{
							Previous:    prevout1,
							AssetAmount: bc.AssetAmount{AssetID: aid1, Amount: 500},
						},
						{
							Previous:    prevout2,
							AssetAmount: bc.AssetAmount{AssetID: aid2, Amount: 500},
						},
					},
					Outputs: []*bc.TxOutput{
						bc.NewTxOutput(aid1, 500, nil, nil),
						bc.NewTxOutput(aid2, 100, nil, nil),
						bc.NewTxOutput(aid2, 200, nil, nil),
						bc.NewTxOutput(aid2, 200, nil, nil),
					},
				},
			},
		},
		{
			badTx: false,
			tx: bc.Tx{
				TxData: bc.TxData{
					Inputs: []*bc.TxInput{
						{
							Previous:    prevout1,
							AssetAmount: bc.AssetAmount{AssetID: aid1, Amount: 500},
						},
						{
							Previous:    prevout2,
							AssetAmount: bc.AssetAmount{AssetID: aid1, Amount: 500},
						},
					},
					Outputs: []*bc.TxOutput{
						bc.NewTxOutput(aid1, 1000, nil, nil),
					},
				},
			},
		},
		{
			badTx:  true,
			detail: "positive maxtime must be >= mintime",
			tx: bc.Tx{
				TxData: bc.TxData{
					MinTime: 2,
					MaxTime: 1,
					Inputs: []*bc.TxInput{
						{AssetAmount: bc.AssetAmount{AssetID: aid1, Amount: 1000}},
					},
					Outputs: []*bc.TxOutput{
						bc.NewTxOutput(aid1, 1000, nil, nil),
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		err := ValidateTx(&tc.tx)
		if tc.badTx && errors.Root(err) != ErrBadTx {
			t.Errorf("got = %s, want ErrBadTx", err)
			continue
		}

		if tc.detail != "" && tc.detail != errors.Detail(err) {
			t.Errorf("errors.Detail: got = %s, want = %s", errors.Detail(err), tc.detail)
		}
	}
}

func TestValidateInvalidTimestamps(t *testing.T) {
	aid := bc.AssetID(mustParseHash("59999b124d0787b27f6ac4aeecb08dda3021720081c98988c074b5a8bc2e9c41"))
	cases := []struct {
		ok        bool
		tx        bc.Tx
		timestamp uint64
	}{
		{
			ok: true,
			tx: bc.Tx{
				TxData: bc.TxData{
					MinTime: 1,
					MaxTime: 100,
					Inputs: []*bc.TxInput{
						{
							Previous: bc.Outpoint{Index: bc.InvalidOutputIndex},
						},
					},
					Outputs: []*bc.TxOutput{
						bc.NewTxOutput(aid, 1000, nil, nil),
					},
				},
			},
			timestamp: 50,
		},
		{
			ok: false,
			tx: bc.Tx{
				TxData: bc.TxData{
					MinTime: 1,
					MaxTime: 100,
					Inputs: []*bc.TxInput{
						{
							Previous: bc.Outpoint{Index: bc.InvalidOutputIndex},
						},
					},
					Outputs: []*bc.TxOutput{
						bc.NewTxOutput(aid, 1000, nil, nil),
					},
				},
			},
			timestamp: 150,
		},
		{
			ok: false,
			tx: bc.Tx{
				TxData: bc.TxData{
					MinTime: 100,
					MaxTime: 200,
					Inputs: []*bc.TxInput{
						{
							Previous: bc.Outpoint{Index: bc.InvalidOutputIndex},
						},
					},
					Outputs: []*bc.TxOutput{
						bc.NewTxOutput(aid, 1000, nil, nil),
					},
				},
			},
			timestamp: 50,
		},
	}

	for i, c := range cases {
		err := ConfirmTx(patricia.NewTree(nil), &c.tx, c.timestamp)
		if !c.ok && errors.Root(err) != ErrBadTx {
			t.Errorf("test %d: got = %s, want ErrBadTx", i, err)
			continue
		}

		if c.ok && err != nil {
			t.Errorf("test %d: unexpected error: %s", i, err.Error())
			continue
		}
	}
}

func BenchmarkConfirmTx(b *testing.B) {
	tree := patricia.NewTree(nil)
	tx := txFromHex("0000000101341fb89912be0110b527375998810c99ac96a317c63b071ccf33b7514cf0f0a5ffffffff6f00473045022100c561a9b4854742bc36c805513b872b2c0a1a367da24710eadd4f3fbc3b1ab41302207cf9eec4e5db694831fe43cf193f23d869291025ac6062199dd6b8998e93e15825512103623fb1fe38ce7e43cf407ec99b061c6d2da0278e80ce094393875c5b94f1ed9051ae0001df03f294bd08930f542a42b91199a8afe1b45c28eeb058cc5e8c8d600e0dd42f0000000000000001000000000000000000000474782d31")
	ts := uint64(time.Now().Unix())
	for i := 0; i < b.N; i++ {
		ConfirmTx(tree, tx, ts)
	}
}

func txFromHex(s string) *bc.Tx {
	tx := new(bc.Tx)
	err := tx.UnmarshalText([]byte(s))
	if err != nil {
		panic(err)
	}
	return tx
}

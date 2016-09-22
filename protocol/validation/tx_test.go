package validation

import (
	"fmt"
	"testing"
	"time"

	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/state"
	"chain/protocol/vm"
)

func TestUniqueIssuance(t *testing.T) {
	var initialBlockHash bc.Hash
	trueProg := []byte{byte(vm.OP_TRUE)}
	assetID := bc.ComputeAssetID(trueProg, initialBlockHash, 1)
	now := time.Now()
	issuanceInp := bc.NewIssuanceInput(nil, 1, nil, initialBlockHash, trueProg, nil)

	// Transaction with empty nonce (and no other inputs) is invalid
	tx := bc.NewTx(bc.TxData{
		Version: 1,
		Inputs:  []*bc.TxInput{issuanceInp},
		Outputs: []*bc.TxOutput{bc.NewTxOutput(assetID, 1, trueProg, nil)},
		MinTime: bc.Millis(now),
		MaxTime: bc.Millis(now.Add(time.Hour)),
	})
	if ValidateTx(tx) == nil {
		t.Errorf("expected tx with only issuance inputs with empty nonces to fail validation")
	}

	issuanceInp.TypedInput.(*bc.IssuanceInput).Nonce = []byte{1}

	// Transaction with non-empty nonce and unbounded time window is invalid
	tx = bc.NewTx(bc.TxData{
		Version: 1,
		Inputs:  []*bc.TxInput{issuanceInp},
		Outputs: []*bc.TxOutput{bc.NewTxOutput(assetID, 1, trueProg, nil)},
		MinTime: bc.Millis(now),
	})
	if ValidateTx(tx) == nil {
		t.Errorf("expected tx with unbounded time window to fail validation")
	}

	tx = bc.NewTx(bc.TxData{
		Version: 1,
		Inputs:  []*bc.TxInput{issuanceInp},
		Outputs: []*bc.TxOutput{bc.NewTxOutput(assetID, 1, trueProg, nil)},
		MaxTime: bc.Millis(now.Add(time.Hour)),
	})
	if ValidateTx(tx) == nil {
		t.Errorf("expected tx with unbounded time window to fail validation")
	}

	// Transaction with the issuance twice is invalid
	tx = bc.NewTx(bc.TxData{
		Version: 1,
		Inputs:  []*bc.TxInput{issuanceInp, issuanceInp},
		Outputs: []*bc.TxOutput{bc.NewTxOutput(assetID, 2, trueProg, nil)},
		MinTime: bc.Millis(now),
		MaxTime: bc.Millis(now.Add(time.Hour)),
	})
	if ValidateTx(tx) == nil {
		t.Errorf("expected tx with duplicate inputs to fail validation")
	}

	// Transaction with the issuance just once is valid
	tx = bc.NewTx(bc.TxData{
		Version: 1,
		Inputs:  []*bc.TxInput{issuanceInp},
		Outputs: []*bc.TxOutput{bc.NewTxOutput(assetID, 1, trueProg, nil)},
		MinTime: bc.Millis(now),
		MaxTime: bc.Millis(now.Add(time.Hour)),
	})
	err := ValidateTx(tx)
	if err != nil {
		t.Errorf("expected tx with unique issuance to pass validation, got: %s", err)
	}

	snapshot := state.Empty()

	// Add tx to the state tree so we can spend it in the next tx
	err = ApplyTx(snapshot, tx)
	if err != nil {
		t.Fatal(err)
	}

	true2Prog := []byte{byte(vm.OP_TRUE), byte(vm.OP_TRUE)}
	asset2ID := bc.ComputeAssetID(true2Prog, initialBlockHash, 1)
	issuance2Inp := bc.NewIssuanceInput(nil, 1, nil, initialBlockHash, true2Prog, nil)

	// Transaction with empty nonce does not get added to issuance memory
	tx = bc.NewTx(bc.TxData{
		Version: 1,
		Inputs: []*bc.TxInput{
			bc.NewSpendInput(tx.Hash, 0, nil, assetID, 1, trueProg, nil),
			issuance2Inp,
		},
		Outputs: []*bc.TxOutput{
			bc.NewTxOutput(assetID, 1, trueProg, nil),
			bc.NewTxOutput(asset2ID, 1, trueProg, nil),
		},
	})
	err = ValidateTx(tx)
	if err != nil {
		t.Fatal(err)
	}
	err = ConfirmTx(snapshot, tx, bc.Millis(now))
	if err != nil {
		t.Fatal(err)
	}
	iHash, err := tx.IssuanceHash(1)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := snapshot.Issuances[iHash]; ok {
		t.Errorf("expected input with empty nonce to be omitted from issuance memory")
	}

	issuance2Inp.TypedInput.(*bc.IssuanceInput).Nonce = []byte{2}

	// This one _is_ added to the issuance memory
	tx = bc.NewTx(bc.TxData{
		Version: 1,
		Inputs: []*bc.TxInput{
			issuance2Inp,
		},
		Outputs: []*bc.TxOutput{
			bc.NewTxOutput(asset2ID, 1, trueProg, nil),
		},
		MinTime: bc.Millis(now),
		MaxTime: bc.Millis(now.Add(time.Hour)),
	})
	err = ValidateTx(tx)
	if err != nil {
		t.Fatal(err)
	}
	err = ConfirmTx(snapshot, tx, bc.Millis(now))
	if err != nil {
		t.Fatal(err)
	}
	err = ApplyTx(snapshot, tx)
	if err != nil {
		t.Fatal(err)
	}
	iHash, err = tx.IssuanceHash(0)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := snapshot.Issuances[iHash]; !ok {
		t.Errorf("expected input with non-empty nonce to be added to issuance memory")
	}
	// Adding it again should fail
	if ConfirmTx(snapshot, tx, bc.Millis(now)) == nil {
		t.Errorf("expected adding duplicate issuance tx to fail")
	}
}

func TestTxIsWellFormed(t *testing.T) {
	var initialBlockHash bc.Hash
	issuanceProg := []byte{1}
	aid1 := bc.ComputeAssetID(issuanceProg, initialBlockHash, 1)
	aid2 := bc.AssetID([32]byte{2})
	txhash1 := bc.Hash{10}
	txhash2 := bc.Hash{11}

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
						bc.NewSpendInput(txhash1, 0, nil, aid1, 1000, nil, nil),
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
						bc.NewSpendInput(txhash1, 0, nil, aid1, 500, nil, nil),
						bc.NewSpendInput(txhash2, 0, nil, aid2, 500, nil, nil),
					},
					Outputs: []*bc.TxOutput{
						bc.NewTxOutput(aid1, 500, nil, nil),
						bc.NewTxOutput(aid2, 1000, nil, nil),
					},
				},
			},
		},
		{
			badTx:  true,
			detail: "output value must be greater than 0",
			tx: bc.Tx{
				TxData: bc.TxData{
					Inputs: []*bc.TxInput{
						bc.NewIssuanceInput(nil, 0, nil, initialBlockHash, issuanceProg, nil),
						bc.NewSpendInput(txhash1, 0, nil, aid2, 0, nil, nil),
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
						bc.NewSpendInput(bc.Hash{}, 0, nil, aid1, 1000, nil, nil),
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
						bc.NewSpendInput(txhash1, 0, nil, aid1, 500, nil, nil),
						bc.NewSpendInput(txhash2, 0, nil, aid2, 500, nil, nil),
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
						bc.NewSpendInput(txhash1, 0, nil, aid1, 500, nil, nil),
						bc.NewSpendInput(txhash2, 0, nil, aid1, 500, nil, nil),
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
						bc.NewSpendInput(bc.Hash{}, 0, nil, aid1, 1000, nil, nil),
					},
					Outputs: []*bc.TxOutput{
						bc.NewTxOutput(aid1, 1000, nil, nil),
					},
				},
			},
		},
	}

	for i, tc := range testCases {
		err := ValidateTx(&tc.tx)
		if tc.badTx && errors.Root(err) != ErrBadTx {
			t.Errorf("test %d: got = %s, want ErrBadTx", i, err)
			continue
		}

		if tc.detail != "" && tc.detail != errors.Detail(err) {
			t.Errorf("errors.Detail: got = %s, want = %s", errors.Detail(err), tc.detail)
		}
	}
}

func TestValidateInvalidTimestamps(t *testing.T) {
	var initialBlockHash bc.Hash
	issuanceProg := []byte{1}
	aid := bc.ComputeAssetID(issuanceProg, initialBlockHash, 1)
	now := time.Now()
	cases := []struct {
		ok        bool
		tx        bc.Tx
		timestamp uint64
	}{
		{
			ok: true,
			tx: bc.Tx{
				TxData: bc.TxData{
					MinTime: bc.Millis(now),
					MaxTime: bc.Millis(now.Add(time.Hour)),
					Inputs: []*bc.TxInput{
						bc.NewIssuanceInput(nil, 1000, nil, initialBlockHash, issuanceProg, nil),
					},
					Outputs: []*bc.TxOutput{
						bc.NewTxOutput(aid, 1000, nil, nil),
					},
				},
			},
			timestamp: bc.Millis(now.Add(time.Minute)),
		},
		{
			ok: false,
			tx: bc.Tx{
				TxData: bc.TxData{
					MinTime: bc.Millis(now),
					MaxTime: bc.Millis(now.Add(time.Minute)),
					Inputs: []*bc.TxInput{
						bc.NewIssuanceInput(nil, 1000, nil, initialBlockHash, issuanceProg, nil),
					},
					Outputs: []*bc.TxOutput{
						bc.NewTxOutput(aid, 1000, nil, nil),
					},
				},
			},
			timestamp: bc.Millis(now.Add(time.Hour)),
		},
		{
			ok: false,
			tx: bc.Tx{
				TxData: bc.TxData{
					MinTime: bc.Millis(now),
					MaxTime: bc.Millis(now.Add(time.Minute)),
					Inputs: []*bc.TxInput{
						bc.NewIssuanceInput(nil, 1000, nil, initialBlockHash, issuanceProg, nil),
					},
					Outputs: []*bc.TxOutput{
						bc.NewTxOutput(aid, 1000, nil, nil),
					},
				},
			},
			timestamp: bc.Millis(now.Add(-time.Hour)),
		},
	}

	for i, c := range cases {
		err := ConfirmTx(state.Empty(), &c.tx, c.timestamp)
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
	tx := txFromHex("0000000101341fb89912be0110b527375998810c99ac96a317c63b071ccf33b7514cf0f0a5ffffffff6f00473045022100c561a9b4854742bc36c805513b872b2c0a1a367da24710eadd4f3fbc3b1ab41302207cf9eec4e5db694831fe43cf193f23d869291025ac6062199dd6b8998e93e15825512103623fb1fe38ce7e43cf407ec99b061c6d2da0278e80ce094393875c5b94f1ed9051ae0001df03f294bd08930f542a42b91199a8afe1b45c28eeb058cc5e8c8d600e0dd42f0000000000000001000000000000000000000474782d31")
	ts := uint64(time.Now().Unix())
	for i := 0; i < b.N; i++ {
		ConfirmTx(state.Empty(), tx, ts)
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

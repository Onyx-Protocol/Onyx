package validation

import (
	"fmt"
	"strings"
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
	if CheckTxWellFormed(tx) == nil {
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
	if CheckTxWellFormed(tx) == nil {
		t.Errorf("expected tx with unbounded time window to fail validation")
	}

	tx = bc.NewTx(bc.TxData{
		Version: 1,
		Inputs:  []*bc.TxInput{issuanceInp},
		Outputs: []*bc.TxOutput{bc.NewTxOutput(assetID, 1, trueProg, nil)},
		MaxTime: bc.Millis(now.Add(time.Hour)),
	})
	if CheckTxWellFormed(tx) == nil {
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
	if CheckTxWellFormed(tx) == nil {
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
	err := CheckTxWellFormed(tx)
	if err != nil {
		t.Errorf("expected tx with unique issuance to pass validation, got: %s", err)
	}

	snapshot := state.NewSnapshot(initialBlockHash)

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
	err = CheckTxWellFormed(tx)
	if err != nil {
		t.Fatal(err)
	}

	block := &bc.Block{
		BlockHeader: bc.BlockHeader{
			Version:     1,
			TimestampMS: bc.Millis(now),
		},
	}

	err = ConfirmTx(snapshot, block, tx)
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
	err = CheckTxWellFormed(tx)
	if err != nil {
		t.Fatal(err)
	}
	err = ConfirmTx(snapshot, block, tx)
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
	if ConfirmTx(snapshot, block, tx) == nil {
		t.Errorf("expected adding duplicate issuance tx to fail")
	}
}

func TestTxWellFormed(t *testing.T) {
	var initialBlockHash bc.Hash
	issuanceProg := []byte{1}
	aid1 := bc.ComputeAssetID(issuanceProg, initialBlockHash, 1)
	aid2 := bc.AssetID([32]byte{2})
	txhash1 := bc.Hash{10}
	txhash2 := bc.Hash{11}
	trueProg := []byte{byte(vm.OP_TRUE)}

	testCases := []struct {
		badTx  bool
		detail string
		tx     bc.TxData
	}{
		{
			badTx:  true,
			detail: "inputs are missing",
			tx: bc.TxData{
				Version: 1,
			}, // empty
		},
		{
			badTx:  true,
			detail: fmt.Sprintf("amounts for asset %s are not balanced on inputs and outputs", aid1),
			tx: bc.TxData{
				Version: 1,
				Inputs: []*bc.TxInput{
					bc.NewSpendInput(txhash1, 0, nil, aid1, 1000, nil, nil),
				},
				Outputs: []*bc.TxOutput{
					bc.NewTxOutput(aid1, 999, nil, nil),
				},
			},
		},
		{
			badTx:  true,
			detail: fmt.Sprintf("amounts for asset %s are not balanced on inputs and outputs", aid2),
			tx: bc.TxData{
				Version: 1,
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
		{
			badTx:  true,
			detail: "output value must be greater than 0",
			tx: bc.TxData{
				Version: 1,
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
		{
			badTx: false,
			tx: bc.TxData{
				Version: 1,
				Inputs: []*bc.TxInput{
					bc.NewSpendInput(bc.Hash{}, 0, nil, aid1, 1000, nil, nil),
				},
				Outputs: []*bc.TxOutput{
					bc.NewTxOutput(aid1, 1000, nil, nil),
				},
			},
		},
		{
			badTx: false,
			tx: bc.TxData{
				Version: 1,
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
		{
			badTx: false,
			tx: bc.TxData{
				Version: 1,
				Inputs: []*bc.TxInput{
					bc.NewSpendInput(txhash1, 0, nil, aid1, 500, nil, nil),
					bc.NewSpendInput(txhash2, 0, nil, aid1, 500, nil, nil),
				},
				Outputs: []*bc.TxOutput{
					bc.NewTxOutput(aid1, 1000, nil, nil),
				},
			},
		},
		{
			badTx:  true,
			detail: "positive maxtime must be >= mintime",
			tx: bc.TxData{
				Version: 1,
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
		{
			badTx: false,
			tx: bc.TxData{
				Version: 1,
				Inputs: []*bc.TxInput{
					{
						AssetVersion: 1,
						TypedInput: &bc.SpendInput{
							OutputCommitment: bc.OutputCommitment{
								AssetAmount: bc.AssetAmount{
									Amount: 1,
								},
								VMVersion:      1,
								ControlProgram: trueProg,
							},
						},
					},
				},
				Outputs: []*bc.TxOutput{
					{
						AssetVersion: 1,
						OutputCommitment: bc.OutputCommitment{
							AssetAmount: bc.AssetAmount{
								Amount: 1,
							},
							VMVersion:      1,
							ControlProgram: trueProg,
						},
					},
				},
			},
		},
		{
			// unknown tx version is still well-formed
			badTx: false,
			tx: bc.TxData{
				Version: 2,
				Inputs: []*bc.TxInput{
					{
						AssetVersion: 1,
						TypedInput: &bc.SpendInput{
							OutputCommitment: bc.OutputCommitment{
								AssetAmount: bc.AssetAmount{
									Amount: 1,
								},
								VMVersion:      1,
								ControlProgram: trueProg,
							},
						},
					},
				},
				Outputs: []*bc.TxOutput{
					{
						AssetVersion: 1,
						OutputCommitment: bc.OutputCommitment{
							AssetAmount: bc.AssetAmount{
								Amount: 1,
							},
							VMVersion:      1,
							ControlProgram: trueProg,
						},
					},
				},
			},
		},
		{
			// unknown asset version in unknown tx version is ok
			badTx: false,
			tx: bc.TxData{
				Version: 2,
				Inputs: []*bc.TxInput{
					{
						AssetVersion: 2,
						TypedInput: &bc.SpendInput{
							OutputCommitment: bc.OutputCommitment{
								AssetAmount: bc.AssetAmount{
									Amount: 1,
								},
								VMVersion:      1,
								ControlProgram: trueProg,
							},
						},
					},
				},
				Outputs: []*bc.TxOutput{
					{
						AssetVersion: 1,
						OutputCommitment: bc.OutputCommitment{
							AssetAmount: bc.AssetAmount{
								Amount: 1,
							},
							VMVersion:      1,
							ControlProgram: trueProg,
						},
					},
				},
			},
		},
		{
			// unknown asset version in unknown tx version is ok
			badTx: false,
			tx: bc.TxData{
				Version: 2,
				Inputs: []*bc.TxInput{
					{
						AssetVersion: 1,
						TypedInput: &bc.SpendInput{
							OutputCommitment: bc.OutputCommitment{
								AssetAmount: bc.AssetAmount{
									Amount: 1,
								},
								VMVersion:      1,
								ControlProgram: trueProg,
							},
						},
					},
				},
				Outputs: []*bc.TxOutput{
					{
						AssetVersion: 2,
						OutputCommitment: bc.OutputCommitment{
							AssetAmount: bc.AssetAmount{
								Amount: 1,
							},
							VMVersion:      1,
							ControlProgram: trueProg,
						},
					},
				},
			},
		},
		{
			// unknown vm version in unknown tx version is ok
			badTx: false,
			tx: bc.TxData{
				Version: 2,
				Inputs: []*bc.TxInput{
					{
						AssetVersion: 1,
						TypedInput: &bc.SpendInput{
							OutputCommitment: bc.OutputCommitment{
								AssetAmount: bc.AssetAmount{
									Amount: 1,
								},
								VMVersion:      2,
								ControlProgram: trueProg,
							},
						},
					},
				},
				Outputs: []*bc.TxOutput{
					{
						AssetVersion: 1,
						OutputCommitment: bc.OutputCommitment{
							AssetAmount: bc.AssetAmount{
								Amount: 1,
							},
							VMVersion:      1,
							ControlProgram: trueProg,
						},
					},
				},
			},
		},
		{
			// unknown vm version in unknown tx version is ok
			badTx: false,
			tx: bc.TxData{
				Version: 2,
				Inputs: []*bc.TxInput{
					{
						AssetVersion: 1,
						TypedInput: &bc.SpendInput{
							OutputCommitment: bc.OutputCommitment{
								AssetAmount: bc.AssetAmount{
									Amount: 1,
								},
								VMVersion:      2,
								ControlProgram: trueProg,
							},
						},
					},
				},
				Outputs: []*bc.TxOutput{
					{
						AssetVersion: 1,
						OutputCommitment: bc.OutputCommitment{
							AssetAmount: bc.AssetAmount{
								Amount: 1,
							},
							VMVersion:      2,
							ControlProgram: trueProg,
						},
					},
				},
			},
		},
		{
			// expansion opcodes with unknown tx version are ok
			badTx: false,
			tx: bc.TxData{
				Version: 2,
				Inputs: []*bc.TxInput{
					{
						AssetVersion: 1,
						TypedInput: &bc.SpendInput{
							OutputCommitment: bc.OutputCommitment{
								AssetAmount: bc.AssetAmount{
									Amount: 1,
								},
								VMVersion:      1,
								ControlProgram: []byte{0x50, byte(vm.OP_TRUE)},
							},
						},
					},
				},
				Outputs: []*bc.TxOutput{
					{
						AssetVersion: 1,
						OutputCommitment: bc.OutputCommitment{
							AssetAmount: bc.AssetAmount{
								Amount: 1,
							},
							VMVersion:      1,
							ControlProgram: trueProg,
						},
					},
				},
			},
		},
		{
			// unknown asset version in tx version 1 is not ok
			badTx:  true,
			detail: "unknown asset version",
			tx: bc.TxData{
				Version: 1,
				Inputs: []*bc.TxInput{
					{
						AssetVersion: 2,
						TypedInput: &bc.SpendInput{
							OutputCommitment: bc.OutputCommitment{
								AssetAmount: bc.AssetAmount{
									Amount: 1,
								},
								VMVersion:      1,
								ControlProgram: trueProg,
							},
						},
					},
				},
				Outputs: []*bc.TxOutput{
					{
						AssetVersion: 1,
						OutputCommitment: bc.OutputCommitment{
							AssetAmount: bc.AssetAmount{
								Amount: 1,
							},
							VMVersion:      1,
							ControlProgram: trueProg,
						},
					},
				},
			},
		},
		{
			// unknown asset version in tx version 1 is not ok
			badTx:  true,
			detail: "unknown asset version",
			tx: bc.TxData{
				Version: 1,
				Inputs: []*bc.TxInput{
					{
						AssetVersion: 1,
						TypedInput: &bc.SpendInput{
							OutputCommitment: bc.OutputCommitment{
								AssetAmount: bc.AssetAmount{
									Amount: 1,
								},
								VMVersion:      1,
								ControlProgram: trueProg,
							},
						},
					},
				},
				Outputs: []*bc.TxOutput{
					{
						AssetVersion: 2,
						OutputCommitment: bc.OutputCommitment{
							AssetAmount: bc.AssetAmount{
								Amount: 1,
							},
							VMVersion:      1,
							ControlProgram: trueProg,
						},
					},
				},
			},
		},
		{
			// unknown vm version in tx version 1 is not ok
			badTx:  true,
			detail: "unknown vm version",
			tx: bc.TxData{
				Version: 1,
				Inputs: []*bc.TxInput{
					{
						AssetVersion: 1,
						TypedInput: &bc.SpendInput{
							OutputCommitment: bc.OutputCommitment{
								AssetAmount: bc.AssetAmount{
									Amount: 1,
								},
								VMVersion:      2,
								ControlProgram: trueProg,
							},
						},
					},
				},
				Outputs: []*bc.TxOutput{
					{
						AssetVersion: 1,
						OutputCommitment: bc.OutputCommitment{
							AssetAmount: bc.AssetAmount{
								Amount: 1,
							},
							VMVersion:      1,
							ControlProgram: trueProg,
						},
					},
				},
			},
		},
		{
			// unknown vm version in tx version 1 is not ok
			badTx: true,
			tx: bc.TxData{
				Version: 1,
				Inputs: []*bc.TxInput{
					{
						AssetVersion: 1,
						TypedInput: &bc.SpendInput{
							OutputCommitment: bc.OutputCommitment{
								AssetAmount: bc.AssetAmount{
									Amount: 1,
								},
								VMVersion:      1,
								ControlProgram: trueProg,
							},
						},
					},
				},
				Outputs: []*bc.TxOutput{
					{
						AssetVersion: 1,
						OutputCommitment: bc.OutputCommitment{
							AssetAmount: bc.AssetAmount{
								Amount: 1,
							},
							VMVersion:      2,
							ControlProgram: trueProg,
						},
					},
				},
			},
		},
		{
			// expansion opcodes in tx version 1 are not ok
			badTx:  true,
			detail: "disallowed opcode",
			tx: bc.TxData{
				Version: 1,
				Inputs: []*bc.TxInput{
					{
						AssetVersion: 1,
						TypedInput: &bc.SpendInput{
							OutputCommitment: bc.OutputCommitment{
								AssetAmount: bc.AssetAmount{
									Amount: 1,
								},
								VMVersion:      1,
								ControlProgram: []byte{0x50, byte(vm.OP_TRUE)},
							},
						},
					},
				},
				Outputs: []*bc.TxOutput{
					{
						AssetVersion: 1,
						OutputCommitment: bc.OutputCommitment{
							AssetAmount: bc.AssetAmount{
								Amount: 1,
							},
							VMVersion:      1,
							ControlProgram: trueProg,
						},
					},
				},
			},
		},
	}

	for i, tc := range testCases {
		tx := bc.NewTx(tc.tx)
		err := CheckTxWellFormed(tx)
		if tc.badTx && errors.Root(err) != ErrBadTx {
			t.Errorf("test %d: got = %s, want ErrBadTx", i, err)
			continue
		}

		if tc.detail != "" && !strings.Contains(errors.Detail(err), tc.detail) {
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
					Version: 1,
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
					Version: 1,
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
					Version: 1,
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
		block := &bc.Block{
			BlockHeader: bc.BlockHeader{
				Version:     1,
				TimestampMS: c.timestamp,
			},
		}
		err := ConfirmTx(state.NewSnapshot(initialBlockHash), block, &c.tx)
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

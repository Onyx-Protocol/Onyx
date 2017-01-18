package validation

import (
	"math"
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
	assetID := bc.ComputeAssetID(trueProg, initialBlockHash, 1, bc.EmptyStringHash)
	now := time.Now()
	issuanceInp := bc.NewIssuanceInput(nil, 1, nil, initialBlockHash, trueProg, nil, nil)

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

	snapshot := state.Empty()

	// Add tx to the state tree so we can spend it in the next tx
	err = ApplyTx(snapshot, tx)
	if err != nil {
		t.Fatal(err)
	}

	true2Prog := []byte{byte(vm.OP_TRUE), byte(vm.OP_TRUE)}
	asset2ID := bc.ComputeAssetID(true2Prog, initialBlockHash, 1, bc.EmptyStringHash)
	issuance2Inp := bc.NewIssuanceInput(nil, 1, nil, initialBlockHash, true2Prog, nil, nil)

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

	err = ConfirmTx(snapshot, initialBlockHash, block, tx)
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
	err = ConfirmTx(snapshot, initialBlockHash, block, tx)
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
	if ConfirmTx(snapshot, initialBlockHash, block, tx) == nil {
		t.Errorf("expected adding duplicate issuance tx to fail")
	}
}

func TestTxWellFormed(t *testing.T) {
	var initialBlockHash bc.Hash
	trueProg := []byte{byte(vm.OP_TRUE)}
	issuanceProg := trueProg
	aid1 := bc.ComputeAssetID(issuanceProg, initialBlockHash, 1, bc.EmptyStringHash)
	aid2 := bc.AssetID([32]byte{2})
	txhash1 := bc.Hash{10}
	txhash2 := bc.Hash{11}

	testCases := []struct {
		suberr error
		tx     bc.TxData
	}{
		{
			suberr: errNoInputs,
			tx: bc.TxData{
				Version: 1,
			}, // empty
		},
		{
			suberr: errUnbalancedV1,
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
			suberr: errUnbalancedV1,
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
			suberr: errEmptyOutput,
			tx: bc.TxData{
				Version: 1,
				Inputs: []*bc.TxInput{
					bc.NewIssuanceInput(nil, 0, nil, initialBlockHash, issuanceProg, nil, nil),
					bc.NewSpendInput(txhash1, 0, nil, aid2, 0, nil, nil),
				},
				Outputs: []*bc.TxOutput{
					bc.NewTxOutput(aid1, 0, nil, nil),
					bc.NewTxOutput(aid2, 0, nil, nil),
				},
			},
		},
		{
			tx: bc.TxData{
				Version: 1,
				Inputs: []*bc.TxInput{
					bc.NewSpendInput(bc.Hash{}, 0, nil, aid1, 1000, trueProg, nil),
				},
				Outputs: []*bc.TxOutput{
					bc.NewTxOutput(aid1, 1000, nil, nil),
				},
			},
		},
		{
			tx: bc.TxData{
				Version: 1,
				Inputs: []*bc.TxInput{
					bc.NewSpendInput(txhash1, 0, nil, aid1, 500, trueProg, nil),
					bc.NewSpendInput(txhash2, 0, nil, aid2, 500, trueProg, nil),
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
			tx: bc.TxData{
				Version: 1,
				Inputs: []*bc.TxInput{
					bc.NewSpendInput(txhash1, 0, nil, aid1, 500, trueProg, nil),
					bc.NewSpendInput(txhash2, 0, nil, aid1, 500, trueProg, nil),
				},
				Outputs: []*bc.TxOutput{
					bc.NewTxOutput(aid1, 1000, nil, nil),
				},
			},
		},
		{
			suberr: errMisorderedTime,
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
			suberr: errAssetVersion,
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
			// unknown asset version in unknown tx version is ok
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
			suberr: errVMVersion,
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
			suberr: errVMVersion,
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
							VMVersion:      2,
							ControlProgram: trueProg,
						},
					},
				},
			},
		},
		{
			// expansion opcodes with unknown tx version are ok
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
			suberr: errAssetVersion,
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
			suberr: errAssetVersion,
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
			suberr: errVMVersion,
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
			suberr: errVMVersion,
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
			suberr: vm.ErrDisallowedOpcode,
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
		{
			suberr: errInputTooBig,
			tx: bc.TxData{
				Version: 1,
				Inputs: []*bc.TxInput{
					{
						AssetVersion: 1,
						TypedInput: &bc.SpendInput{
							OutputCommitment: bc.OutputCommitment{
								AssetAmount: bc.AssetAmount{
									Amount: math.MaxInt64 + 1,
								},
							},
						},
					},
				},
			},
		},
		{
			suberr: errInputSumTooBig,
			tx: bc.TxData{
				Version: 1,
				Inputs: []*bc.TxInput{
					{
						AssetVersion: 1,
						TypedInput: &bc.SpendInput{
							OutputCommitment: bc.OutputCommitment{
								AssetAmount: bc.AssetAmount{
									Amount: math.MaxInt64,
								},
								VMVersion:      1,
								ControlProgram: trueProg,
							},
						},
					},
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
			},
		},
		{
			suberr: errDuplicateInput,
			tx: bc.TxData{
				Version: 1,
				Inputs: []*bc.TxInput{
					{
						AssetVersion: 1,
						TypedInput: &bc.SpendInput{
							OutputCommitment: bc.OutputCommitment{
								AssetAmount: bc.AssetAmount{
									Amount: 10,
								},
								VMVersion:      1,
								ControlProgram: trueProg,
							},
						},
					},
					{
						AssetVersion: 1,
						TypedInput: &bc.SpendInput{
							OutputCommitment: bc.OutputCommitment{
								AssetAmount: bc.AssetAmount{
									Amount: 10,
								},
								VMVersion:      1,
								ControlProgram: trueProg,
							},
						},
					},
				},
			},
		},
		{
			suberr: errOutputTooBig,
			tx: bc.TxData{
				Version: 1,
				Inputs: []*bc.TxInput{
					{
						AssetVersion: 1,
						TypedInput: &bc.SpendInput{
							OutputCommitment: bc.OutputCommitment{
								AssetAmount: bc.AssetAmount{
									Amount: 10,
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
								Amount: math.MaxInt64 + 1,
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
		if err == nil {
			if tc.suberr != nil {
				t.Errorf("case %d: got no error, want ErrBadTx with suberr %s", i, tc.suberr)
			}
			continue
		}
		if tc.suberr == nil {
			t.Errorf("case %d: got %s, want no error", i, err)
			continue
		}
		suberr, _ := errors.Data(err)["badtx"]
		if subsuberr, ok := suberr.(vm.Error); ok {
			suberr = subsuberr.Err
		}
		if suberr != tc.suberr {
			t.Errorf("case %d: got %s, want ErrBadTx with suberr %s", i, err, tc.suberr)
		}
	}
}

func TestValidateInvalidIssuances(t *testing.T) {
	var initialBlockHash bc.Hash
	issuanceProg := []byte{1}
	aid := bc.ComputeAssetID(issuanceProg, initialBlockHash, 1, bc.EmptyStringHash)
	now := time.Now()

	wrongInitialBlockHash := initialBlockHash
	wrongInitialBlockHash[0] ^= 1

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
						bc.NewIssuanceInput(nil, 1000, nil, initialBlockHash, issuanceProg, nil, nil),
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
						bc.NewIssuanceInput(nil, 1000, nil, initialBlockHash, issuanceProg, nil, nil),
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
						bc.NewIssuanceInput(nil, 1000, nil, initialBlockHash, issuanceProg, nil, nil),
					},
					Outputs: []*bc.TxOutput{
						bc.NewTxOutput(aid, 1000, nil, nil),
					},
				},
			},
			timestamp: bc.Millis(now.Add(-time.Hour)),
		},
		{
			ok: false,
			tx: bc.Tx{
				TxData: bc.TxData{
					Version: 1,
					MinTime: bc.Millis(now),
					MaxTime: bc.Millis(now.Add(time.Hour)),
					Inputs: []*bc.TxInput{
						bc.NewIssuanceInput(nil, 1000, nil, wrongInitialBlockHash, issuanceProg, nil, nil),
					},
					Outputs: []*bc.TxOutput{
						bc.NewTxOutput(aid, 1000, nil, nil),
					},
				},
			},
			timestamp: bc.Millis(now.Add(time.Minute)),
		},
	}

	for i, c := range cases {
		block := &bc.Block{
			BlockHeader: bc.BlockHeader{
				Version:     1,
				TimestampMS: c.timestamp,
			},
		}
		err := ConfirmTx(state.Empty(), initialBlockHash, block, &c.tx)
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

func TestConfirmTx(t *testing.T) {
	txhash1 := bc.Hash{1}

	outpoint1 := bc.Outpoint{Hash: txhash1}

	trueProg := []byte{0x51}

	assetID1 := bc.AssetID{10}

	out1 := bc.OutputCommitment{
		AssetAmount: bc.AssetAmount{
			AssetID: assetID1,
			Amount:  11,
		},
		VMVersion:      1,
		ControlProgram: trueProg,
	}
	txout := bc.TxOutput{
		AssetVersion:     1,
		OutputCommitment: out1,
	}

	stateout := state.NewOutput(txout, outpoint1)

	snapshot := state.Empty()
	err := snapshot.Tree.Insert(state.OutputTreeItem(stateout))
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		blockheader *bc.BlockHeader
		tx          *bc.TxData
		suberr      error
		doApply     bool
	}{
		{
			blockheader: &bc.BlockHeader{
				Version: 1,
			},
			tx: &bc.TxData{
				Version: 2,
			},
			suberr: errTxVersion,
		},
		{
			blockheader: &bc.BlockHeader{
				Version:     1,
				TimestampMS: 10,
			},
			tx: &bc.TxData{
				Version: 1,
				MinTime: 11,
			},
			suberr: errNotYet,
		},
		{
			blockheader: &bc.BlockHeader{
				Version:     1,
				TimestampMS: 10,
			},
			tx: &bc.TxData{
				Version: 1,
				MaxTime: 9,
			},
			suberr: errTooLate,
		},
		{
			blockheader: &bc.BlockHeader{
				Version: 1,
			},
			tx: &bc.TxData{
				Version: 1,
				Inputs: []*bc.TxInput{
					{
						AssetVersion: 1,
						TypedInput: &bc.IssuanceInput{
							IssuanceWitness: bc.IssuanceWitness{
								InitialBlock: bc.Hash{1},
							},
						},
					},
				},
			},
			suberr: errWrongBlockchain,
		},
		{
			blockheader: &bc.BlockHeader{
				Version: 1,
			},
			tx: &bc.TxData{
				Version: 1,
				Inputs: []*bc.TxInput{
					{
						AssetVersion: 1,
						TypedInput: &bc.IssuanceInput{
							Nonce: []byte{1},
						},
					},
				},
			},
			suberr: errTimelessIssuance,
		},
		{
			blockheader: &bc.BlockHeader{
				Version: 1,
			},
			tx: &bc.TxData{
				Version: 1,
				Inputs: []*bc.TxInput{
					{
						AssetVersion: 1,
						TypedInput: &bc.SpendInput{
							OutputCommitment: bc.OutputCommitment{},
						},
					},
				},
			},
			suberr: errInvalidOutput,
		},
		{
			blockheader: &bc.BlockHeader{
				Version: 1,
			},
			tx: &bc.TxData{
				Version: 1,
				Inputs: []*bc.TxInput{
					{
						AssetVersion: 1,
						TypedInput: &bc.SpendInput{
							Outpoint:         outpoint1,
							OutputCommitment: out1,
						},
					},
				},
			},
			doApply: true,
		},
	}
	for i, c := range cases {
		var initialBlockHash bc.Hash
		block := &bc.Block{
			BlockHeader: *c.blockheader,
		}
		tx := bc.NewTx(*c.tx)
		err := ConfirmTx(snapshot, initialBlockHash, block, tx)
		if c.suberr == nil {
			if err != nil {
				t.Errorf("case %d: got error %s, want no error", i, err)
			}

			if c.doApply {
				err = ApplyTx(snapshot, tx)
				if err != nil {
					t.Errorf("case %d: confirm succeeded but apply failed: %s", i, err)
					continue
				}
				// Apply succeeded, now try to confirm again - it should fail
				// with "invalid output."
				err = ConfirmTx(snapshot, initialBlockHash, block, tx)
				if err == nil {
					t.Errorf("case %d: confirm and apply succeeded, second confirm succeeded unexpectedly", i)
					continue
				}
				suberr, _ := errors.Data(err)["badtx"]
				if suberr != errInvalidOutput {
					t.Errorf("case %d: confirm and apply succeeded, second confirm failed but with the wrong error: %s", i, err)
				}
			}

			continue
		}
		if err == nil {
			t.Errorf("case %d: got no error, want badtx with suberr %s", i, c.suberr)
			continue
		}
		suberr, _ := errors.Data(err)["badtx"]
		if suberr != c.suberr {
			t.Errorf("case %d: got error %s, want badtx with suberr %s", i, err, suberr)
		}
	}
}

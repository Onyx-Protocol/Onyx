package validation

import (
	"math"
	"testing"
	"time"

	"chain/encoding/blockchain"
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
	nowMS := bc.Millis(now)
	issuanceInp := bc.NewIssuanceInput(nil, 1, nil, initialBlockHash, trueProg, nil, nil)

	// Transaction with empty nonce (and no other inputs) is invalid
	_, err := bc.ComputeTxEntries(&bc.TxData{
		Version: 1,
		Inputs:  []*bc.TxInput{issuanceInp},
		Outputs: []*bc.TxOutput{bc.NewTxOutput(assetID, 1, trueProg, nil)},
		MinTime: nowMS,
		MaxTime: bc.Millis(now.Add(time.Hour)),
	})
	if err == nil {
		t.Errorf("expected tx with only issuance inputs with empty nonces to fail in computing hashes")
	}

	issuanceInp.TypedInput.(*bc.IssuanceInput).Nonce = []byte{1}

	var tx *bc.Tx

	if false { // xxx disabled for now
		// Transaction with non-empty nonce and unbounded time window is invalid
		tx = bc.NewTx(bc.TxData{
			Version: 1,
			Inputs:  []*bc.TxInput{issuanceInp},
			Outputs: []*bc.TxOutput{bc.NewTxOutput(assetID, 1, trueProg, nil)},
			MinTime: nowMS,
		})
		if CheckTxWellFormed(tx.TxEntries) == nil {
			t.Errorf("expected tx with unbounded time window to fail validation")
		}
	}

	if false { // xxx disabled for now
		tx = bc.NewTx(bc.TxData{
			Version: 1,
			Inputs:  []*bc.TxInput{issuanceInp},
			Outputs: []*bc.TxOutput{bc.NewTxOutput(assetID, 1, trueProg, nil)},
			MaxTime: bc.Millis(now.Add(time.Hour)),
		})
		if CheckTxWellFormed(tx.TxEntries) == nil {
			t.Errorf("expected tx with unbounded time window to fail validation")
		}
	}

	// Transaction with the issuance twice is invalid
	tx = bc.NewTx(bc.TxData{
		Version: 1,
		Inputs:  []*bc.TxInput{issuanceInp, issuanceInp},
		Outputs: []*bc.TxOutput{bc.NewTxOutput(assetID, 2, trueProg, nil)},
		MinTime: nowMS,
		MaxTime: bc.Millis(now.Add(time.Hour)),
	})
	if CheckTxWellFormed(tx.TxEntries) == nil {
		t.Errorf("expected tx with duplicate inputs to fail validation")
	}

	// Transaction with the issuance just once is valid
	tx = bc.NewTx(bc.TxData{
		Version: 1,
		Inputs:  []*bc.TxInput{issuanceInp},
		Outputs: []*bc.TxOutput{bc.NewTxOutput(assetID, 1, trueProg, nil)},
		MinTime: nowMS,
		MaxTime: bc.Millis(now.Add(time.Hour)),
	})
	err = CheckTxWellFormed(tx.TxEntries)
	if err != nil {
		t.Errorf("expected tx with unique issuance to pass validation, got: %s", err)
	}

	snapshot := state.Empty()

	// Add tx to the state tree so we can spend it in the next tx
	err = ApplyTx(snapshot, tx.TxEntries)
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
			bc.NewSpendInput(nil, tx.Results[0].(*bc.Output).SourceID(), assetID, 1, tx.Results[0].(*bc.Output).SourcePosition(), trueProg, tx.Results[0].(*bc.Output).Data(), nil),
			issuance2Inp,
		},
		Outputs: []*bc.TxOutput{
			bc.NewTxOutput(assetID, 1, trueProg, nil),
			bc.NewTxOutput(asset2ID, 1, trueProg, nil),
		},
	})
	err = CheckTxWellFormed(tx.TxEntries)
	if err != nil {
		t.Fatal(err)
	}

	var iHash bc.Hash

	if false { // xxx disabled for now
		err = ConfirmTx(snapshot, initialBlockHash, 1, nowMS, tx.TxEntries)
		if err != nil {
			t.Fatal(err)
		}
		iHash = tx.IssuanceHash(1)
		if _, ok := snapshot.Issuances[iHash]; ok {
			t.Errorf("expected input with empty nonce to be omitted from issuance memory")
		}
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
		MinTime: nowMS,
		MaxTime: bc.Millis(now.Add(time.Hour)),
	})
	err = CheckTxWellFormed(tx.TxEntries)
	if err != nil {
		t.Fatal(err)
	}
	err = ConfirmTx(snapshot, initialBlockHash, 1, nowMS, tx.TxEntries)
	if err != nil {
		t.Fatal(err)
	}
	err = ApplyTx(snapshot, tx.TxEntries)
	if err != nil {
		t.Fatal(err)
	}
	iHash = tx.IssuanceHash(0)
	if _, ok := snapshot.Issuances[iHash]; !ok {
		t.Errorf("expected input with non-empty nonce to be added to issuance memory")
	}
	// Adding it again should fail
	if ConfirmTx(snapshot, initialBlockHash, 1, nowMS, tx.TxEntries) == nil {
		t.Errorf("expected adding duplicate issuance tx to fail")
	}
}

func TestTxWellFormed(t *testing.T) {
	var initialBlockHash bc.Hash
	trueProg := []byte{byte(vm.OP_TRUE)}
	issuanceProg := trueProg
	aid1 := bc.ComputeAssetID(issuanceProg, initialBlockHash, 1, bc.EmptyStringHash)
	aid2 := bc.AssetID([32]byte{2})

	tx1 := bc.NewTx(bc.TxData{
		Inputs: []*bc.TxInput{
			bc.NewIssuanceInput([]byte{1}, 10, nil, initialBlockHash, issuanceProg, nil, nil),
		},
		Outputs: []*bc.TxOutput{
			{
				OutputCommitment: bc.OutputCommitment{
					AssetAmount: bc.AssetAmount{
						AssetID: aid1,
						Amount:  10,
					},
				},
			},
		},
	})
	tx2 := bc.NewTx(bc.TxData{
		Outputs: []*bc.TxOutput{
			{
				OutputCommitment: bc.OutputCommitment{
					AssetAmount: bc.AssetAmount{
						AssetID: aid2,
						Amount:  10,
					},
				},
			},
		},
	})

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
					bc.NewSpendInput(nil, tx1.Results[0].(*bc.Output).SourceID(), aid1, 1000, tx1.Results[0].(*bc.Output).SourcePosition(), trueProg, tx1.Results[0].(*bc.Output).Data(), nil),
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
					bc.NewSpendInput(nil, tx1.Results[0].(*bc.Output).SourceID(), aid1, 500, tx1.Results[0].(*bc.Output).SourcePosition(), trueProg, tx1.Results[0].(*bc.Output).Data(), nil),
					bc.NewSpendInput(nil, tx2.Results[0].(*bc.Output).SourceID(), aid2, 500, tx2.Results[0].(*bc.Output).SourcePosition(), trueProg, tx2.Results[0].(*bc.Output).Data(), nil),
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
					bc.NewSpendInput(nil, tx1.Results[0].(*bc.Output).SourceID(), aid2, 0, tx1.Results[0].(*bc.Output).SourcePosition(), trueProg, tx1.Results[0].(*bc.Output).Data(), nil),
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
					bc.NewSpendInput(nil, tx1.Results[0].(*bc.Output).SourceID(), aid1, 1000, tx1.Results[0].(*bc.Output).SourcePosition(), trueProg, tx1.Results[0].(*bc.Output).Data(), nil),
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
					bc.NewSpendInput(nil, tx1.Results[0].(*bc.Output).SourceID(), aid1, 500, tx1.Results[0].(*bc.Output).SourcePosition(), trueProg, tx1.Results[0].(*bc.Output).Data(), nil),
					bc.NewSpendInput(nil, tx2.Results[0].(*bc.Output).SourceID(), aid2, 500, tx2.Results[0].(*bc.Output).SourcePosition(), trueProg, tx2.Results[0].(*bc.Output).Data(), nil),
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
					bc.NewSpendInput(nil, tx1.Results[0].(*bc.Output).SourceID(), aid1, 500, tx1.Results[0].(*bc.Output).SourcePosition(), trueProg, tx1.Results[0].(*bc.Output).Data(), nil),
					bc.NewSpendInput(nil, tx2.Results[0].(*bc.Output).SourceID(), aid1, 500, tx2.Results[0].(*bc.Output).SourcePosition(), trueProg, tx2.Results[0].(*bc.Output).Data(), nil),
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
					bc.NewSpendInput(nil, tx1.Results[0].(*bc.Output).SourceID(), aid1, 1000, tx1.Results[0].(*bc.Output).SourcePosition(), trueProg, tx1.Results[0].(*bc.Output).Data(), nil),
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
							SpendCommitment: bc.SpendCommitment{
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
							SpendCommitment: bc.SpendCommitment{
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
							SpendCommitment: bc.SpendCommitment{
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
							SpendCommitment: bc.SpendCommitment{
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
							SpendCommitment: bc.SpendCommitment{
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
							SpendCommitment: bc.SpendCommitment{
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
			// unknown vm version in tx version 1 is not ok
			suberr: errVMVersion,
			tx: bc.TxData{
				Version: 1,
				Inputs: []*bc.TxInput{
					{
						AssetVersion: 1,
						TypedInput: &bc.SpendInput{
							SpendCommitment: bc.SpendCommitment{
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
							SpendCommitment: bc.SpendCommitment{
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
							SpendCommitment: bc.SpendCommitment{
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
			suberr: errInputSumTooBig,
			tx: bc.TxData{
				Version: 1,
				Inputs: []*bc.TxInput{
					{
						AssetVersion: 1,
						TypedInput: &bc.SpendInput{
							SpendCommitment: bc.SpendCommitment{
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
							SpendCommitment: bc.SpendCommitment{
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
		// This case disabled because mapTx cannot produce duplicate entries!
		// {
		// 	suberr: errDuplicateInput,
		// 	tx: bc.TxData{
		// 		Version: 1,
		// 		Inputs: []*bc.TxInput{
		// 			{
		// 				AssetVersion: 1,
		// 				TypedInput: &bc.SpendInput{
		// 					SpendCommitment: bc.SpendCommitment{
		// 						AssetAmount: bc.AssetAmount{
		// 							Amount: 10,
		// 						},
		// 						VMVersion:      1,
		// 						ControlProgram: trueProg,
		// 					},
		// 				},
		// 			},
		// 			{
		// 				AssetVersion: 1,
		// 				TypedInput: &bc.SpendInput{
		// 					SpendCommitment: bc.SpendCommitment{
		// 						AssetAmount: bc.AssetAmount{
		// 							Amount: 10,
		// 						},
		// 						VMVersion:      1,
		// 						ControlProgram: trueProg,
		// 					},
		// 				},
		// 			},
		// 		},
		// 	},
		// },
	}

	for i, tc := range testCases {
		tx := bc.NewTx(tc.tx)

		err := CheckTxWellFormed(tx.TxEntries)
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

func TestTxRangeErrs(t *testing.T) {
	trueProg := []byte{byte(vm.OP_TRUE)}
	cases := []*bc.TxData{
		{
			Version: 1,
			Inputs: []*bc.TxInput{
				{
					AssetVersion: 1,
					TypedInput: &bc.SpendInput{
						SpendCommitment: bc.SpendCommitment{
							AssetAmount: bc.AssetAmount{
								Amount: math.MaxInt64 + 1,
							},
						},
					},
				},
			},
		},

		{
			Version: 1,
			Inputs: []*bc.TxInput{
				{
					AssetVersion: 1,
					TypedInput: &bc.SpendInput{
						SpendCommitment: bc.SpendCommitment{
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
	}

	for _, c := range cases {
		_, err := bc.ComputeTxEntries(c)
		switch errors.Root(err) {
		case nil:
			t.Errorf("got no error, want blockchain.ErrRange")
		case blockchain.ErrRange:
			// ok
		default:
			t.Errorf("got error %s, want blockchain.ErrRange", err)
		}
	}
}

func TestValidateInvalidIssuances(t *testing.T) {
	t.Skip() // xxx disabled for now

	var initialBlockHash bc.Hash
	issuanceProg := []byte{1}
	aid := bc.ComputeAssetID(issuanceProg, initialBlockHash, 1, bc.EmptyStringHash)
	now := time.Now()

	wrongInitialBlockHash := initialBlockHash
	wrongInitialBlockHash[0] ^= 1

	cases := []struct {
		ok        bool
		tx        *bc.Tx
		timestamp uint64
	}{
		{
			ok: true,
			tx: bc.NewTx(bc.TxData{
				Version: 1,
				MinTime: bc.Millis(now),
				MaxTime: bc.Millis(now.Add(time.Hour)),
				Inputs: []*bc.TxInput{
					bc.NewIssuanceInput(nil, 1000, nil, initialBlockHash, issuanceProg, nil, nil),
				},
				Outputs: []*bc.TxOutput{
					bc.NewTxOutput(aid, 1000, nil, nil),
				},
			}),
			timestamp: bc.Millis(now.Add(time.Minute)),
		},
		{
			ok: false,
			tx: bc.NewTx(bc.TxData{
				Version: 1,
				MinTime: bc.Millis(now),
				MaxTime: bc.Millis(now.Add(time.Minute)),
				Inputs: []*bc.TxInput{
					bc.NewIssuanceInput(nil, 1000, nil, initialBlockHash, issuanceProg, nil, nil),
				},
				Outputs: []*bc.TxOutput{
					bc.NewTxOutput(aid, 1000, nil, nil),
				},
			}),
			timestamp: bc.Millis(now.Add(time.Hour)),
		},
		{
			ok: false,
			tx: bc.NewTx(bc.TxData{
				Version: 1,
				MinTime: bc.Millis(now),
				MaxTime: bc.Millis(now.Add(time.Minute)),
				Inputs: []*bc.TxInput{
					bc.NewIssuanceInput(nil, 1000, nil, initialBlockHash, issuanceProg, nil, nil),
				},
				Outputs: []*bc.TxOutput{
					bc.NewTxOutput(aid, 1000, nil, nil),
				},
			}),
			timestamp: bc.Millis(now.Add(-time.Hour)),
		},
		{
			ok: false,
			tx: bc.NewTx(bc.TxData{
				Version: 1,
				MinTime: bc.Millis(now),
				MaxTime: bc.Millis(now.Add(time.Hour)),
				Inputs: []*bc.TxInput{
					bc.NewIssuanceInput(nil, 1000, nil, wrongInitialBlockHash, issuanceProg, nil, nil),
				},
				Outputs: []*bc.TxOutput{
					bc.NewTxOutput(aid, 1000, nil, nil),
				},
			}),
			timestamp: bc.Millis(now.Add(time.Minute)),
		},
	}

	for i, c := range cases {
		err := ConfirmTx(state.Empty(), initialBlockHash, 1, c.timestamp, c.tx.TxEntries)
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
	tx := bc.NewTx(bc.TxData{
		Outputs: []*bc.TxOutput{&txout},
	})

	outid1 := tx.OutputID(0)
	outres := tx.Results[0].(*bc.Output)

	snapshot := state.Empty()
	err := snapshot.Tree.Insert(outid1[:])
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		blockTimestampMS uint64
		tx               *bc.TxData
		suberr           error
		doApply          bool
	}{
		{
			tx: &bc.TxData{
				Version: 2,
			},
			suberr: errTxVersion,
		},
		{
			blockTimestampMS: 10,
			tx: &bc.TxData{
				Version: 1,
				MinTime: 11,
			},
			suberr: errNotYet,
		},
		{
			blockTimestampMS: 10,
			tx: &bc.TxData{
				Version: 1,
				MaxTime: 9,
			},
			suberr: errTooLate,
		},
		{
			tx: &bc.TxData{
				Version: 1,
				Inputs: []*bc.TxInput{
					{
						AssetVersion: 1,
						TypedInput: &bc.IssuanceInput{
							Nonce: []byte{1},
							IssuanceWitness: bc.IssuanceWitness{
								InitialBlock: bc.Hash{1},
							},
						},
					},
				},
			},
			suberr: errWrongBlockchain,
		},
		// xxx disabled for now
		// {
		// 	tx: &bc.TxData{
		// 		Version: 1,
		// 		Inputs: []*bc.TxInput{
		// 			{
		// 				AssetVersion: 1,
		// 				TypedInput: &bc.IssuanceInput{
		// 					Nonce: []byte{1},
		// 				},
		// 			},
		// 		},
		// 	},
		// 	suberr: errTimelessIssuance,
		// },
		{
			tx: &bc.TxData{
				Version: 1,
				Inputs: []*bc.TxInput{
					{
						AssetVersion: 1,
						TypedInput: &bc.SpendInput{
							SpendCommitment: bc.SpendCommitment{},
						},
					},
				},
			},
			suberr: errInvalidOutput,
		},
		{
			tx: &bc.TxData{
				Version: 1,
				Inputs: []*bc.TxInput{
					{
						AssetVersion: 1,
						TypedInput: &bc.SpendInput{
							SpendCommitment: bc.SpendCommitment{
								AssetAmount:    out1.AssetAmount,
								VMVersion:      out1.VMVersion,
								ControlProgram: out1.ControlProgram,
								SourceID:       outres.SourceID(),
								SourcePosition: outres.SourcePosition(),
								RefDataHash:    outres.Data(),
							},
						},
					},
				},
			},
			doApply: true,
		},
	}
	for i, c := range cases {
		var initialBlockHash bc.Hash
		tx := bc.NewTx(*c.tx)
		err := ConfirmTx(snapshot, initialBlockHash, 1, c.blockTimestampMS, tx.TxEntries)
		if c.suberr == nil {
			if err != nil {
				t.Errorf("case %d: got error %s, want no error", i, err)
			}

			if c.doApply {
				err = ApplyTx(snapshot, tx.TxEntries)
				if err != nil {
					t.Errorf("case %d: confirm succeeded but apply failed: %s", i, err)
					continue
				}
				// Apply succeeded, now try to confirm again - it should fail
				// with "invalid output."
				err = ConfirmTx(snapshot, initialBlockHash, 1, c.blockTimestampMS, tx.TxEntries)
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

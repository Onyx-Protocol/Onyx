// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package txscript_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"golang.org/x/crypto/sha3"
	"golang.org/x/net/context"

	"chain/cos/bc"
	. "chain/cos/txscript"
)

// testName returns a descriptive test name for the given reference test data.
func testName(test []string, num int) (string, error) {
	var name string

	if len(test) < 3 || len(test) > 4 {
		return name, fmt.Errorf("invalid test length %d", len(test))
	}

	if len(test) == 4 {
		name = fmt.Sprintf("test %d (%s)", num, test[3])
	} else {
		name = fmt.Sprintf("test %d ([%s, %s, %s])", num, test[0], test[1],
			test[2])
	}
	return name, nil
}

// parseScriptFlags parses the provided flags string from the format used in the
// reference tests into ScriptFlags suitable for use in the script engine.
func parseScriptFlags(flagStr string) (ScriptFlags, error) {
	var flags ScriptFlags

	sFlags := strings.Split(flagStr, ",")
	for _, flag := range sFlags {
		switch flag {
		case "":
			// Nothing.
		case "DERSIG":
			flags |= ScriptVerifyDERSignatures
		case "DISCOURAGE_UPGRADABLE_NOPS":
			flags |= ScriptDiscourageUpgradableNops
		case "LOW_S":
			flags |= ScriptVerifyLowS
		case "MINIMALDATA":
			flags |= ScriptVerifyMinimalData
		case "NONE":
			// Nothing.
		case "NULLDUMMY":
			flags |= ScriptStrictMultiSig
		case "SIGPUSHONLY":
			flags |= ScriptVerifySigPushOnly
		case "STRICTENC":
			flags |= ScriptVerifyStrictEncoding
		default:
			return flags, fmt.Errorf("invalid flag: %s", flag)
		}
	}
	return flags, nil
}

var testAssetID, testAssetID2 bc.AssetID

func newCoinbaseTx(val uint64, pkScript []byte, assetID bc.AssetID) *bc.TxData {
	if pkScript == nil {
		pkScript = []byte{OP_TRUE}
	}
	aa := bc.AssetAmount{Amount: val, AssetID: assetID}
	return &bc.TxData{
		Version:  bc.CurrentTransactionVersion,
		Inputs:   []*bc.TxInput{{AssetAmount: aa, SignatureScript: []byte{OP_0, OP_0}}},
		Outputs:  []*bc.TxOutput{{AssetAmount: aa, Script: pkScript}},
		LockTime: 2e9,
	}
}

// createSpendTx generates a basic spending transaction given the passed
// signature and public key scripts.
func createSpendingTx(sigScript, pkScript []byte) *bc.TxData {
	coinbaseTx1 := newCoinbaseTx(3, pkScript, testAssetID)
	coinbaseTx2 := newCoinbaseTx(4, pkScript, testAssetID)
	coinbaseTx3 := newCoinbaseTx(5, nil, testAssetID2)

	spendingTx := &bc.TxData{
		Version: bc.CurrentTransactionVersion,
		Inputs: []*bc.TxInput{
			{
				Previous:        bc.Outpoint{Hash: coinbaseTx1.Hash(), Index: 0},
				PrevScript:      coinbaseTx1.Outputs[0].Script,
				AssetAmount:     coinbaseTx1.Outputs[0].AssetAmount,
				SignatureScript: sigScript,
			},
			{
				Previous:        bc.Outpoint{Hash: coinbaseTx2.Hash(), Index: 0},
				PrevScript:      coinbaseTx2.Outputs[0].Script,
				AssetAmount:     coinbaseTx2.Outputs[0].AssetAmount,
				SignatureScript: sigScript,
			},
			{
				Previous:    bc.Outpoint{Hash: coinbaseTx3.Hash(), Index: 0},
				PrevScript:  coinbaseTx3.Outputs[0].Script,
				AssetAmount: coinbaseTx3.Outputs[0].AssetAmount,
			},
		},
		Outputs: []*bc.TxOutput{
			{
				AssetAmount: bc.AssetAmount{AssetID: testAssetID, Amount: 7},
				Script:      pkScript,
			},
			{
				AssetAmount: bc.AssetAmount{AssetID: testAssetID2, Amount: 5},
			},
		},
		LockTime: 2e9,
	}
	return spendingTx
}

// TestScriptInvalidTests ensures all of the tests in script_invalid.json fail
// as expected.
func TestScriptInvalidTests(t *testing.T) {
	testHelper(t, "script_invalid.json", func(t *testing.T, test []string, name string, testNum int) {
		scriptSig, err := ParseScriptString(test[0])
		if err != nil {
			t.Errorf("%s: can't parse scriptSig; %v", name, err)
			return
		}
		scriptPubKey, err := ParseScriptString(test[1])
		if err != nil {
			t.Errorf("%s: can't parse scriptPubkey; %v", name, err)
			return
		}
		flags, err := parseScriptFlags(test[2])
		if err != nil {
			t.Errorf("%s: %v", name, err)
			return
		}
		tx := createSpendingTx(scriptSig, scriptPubKey)
		vm, err := newTestEngine(scriptPubKey, tx, flags)
		if err == nil {
			if err := vm.Execute(); err == nil {
				t.Errorf("%s test succeeded when it "+
					"should have failed\n", name)
			}
			return
		}
	})
}

// TestScriptValidTests ensures all of the tests in script_valid.json pass as
// expected.
func TestScriptValidTests(t *testing.T) {
	testHelper(t, "script_valid.json", func(t *testing.T, test []string, name string, testNum int) {
		scriptSig, err := ParseScriptString(test[0])
		if err != nil {
			t.Errorf("%s: can't parse scriptSig; %v", name, err)
			return
		}
		scriptPubKey, err := ParseScriptString(test[1])
		if err != nil {
			t.Errorf("%s: can't parse scriptPubkey; %v", name, err)
			return
		}
		flags, err := parseScriptFlags(test[2])
		if err != nil {
			t.Errorf("%s: %v", name, err)
			return
		}
		tx := createSpendingTx(scriptSig, scriptPubKey)
		vm, err := newTestEngine(scriptPubKey, tx, flags)
		if err != nil {
			t.Errorf("%s failed to create script: %v", name, err)
			return
		}
		err = vm.Execute()
		if err != nil {
			t.Errorf("%s failed to execute: %v", name, err)
			return
		}
	})
}

// TestP2CValidTests ensures all of the tests in p2c_valid.json pass
// as expected.
func TestP2CValidTests(t *testing.T) {
	testHelper(t, "p2c_valid.json", func(t *testing.T, test []string, name string, testNum int) {
		scriptSig, scriptPubKey, err := prepareP2CTest(t, test, name, testNum)
		if err != nil {
			t.Errorf("Could not prepare P2C valid test %d (%s): %v\n", testNum, name, err)
			return
		}

		tx := createSpendingTx(scriptSig, scriptPubKey)

		vm, err := newReusableTestEngine(tx)
		if err != nil {
			t.Errorf("TestP2CValidTests: test %d (%s) failed to create engine: %v\n", testNum, name, err)
			return
		}

		err = vm.Prepare(scriptPubKey, 0)
		if err != nil {
			t.Errorf("TestP2CValidTests: Could not prepare engine for test %d (%s), input 0: %v\n", testNum, name, err)
			return
		}
		err = vm.Execute()
		if err != nil {
			t.Errorf("TestP2CValidTests: test %d (%s), input 0 failed to execute: %v\n", testNum, name, err)
			return
		}

		err = vm.Prepare(scriptPubKey, 1)
		if err != nil {
			t.Errorf("TestP2CValidTests: Could not prepare engine for test %d (%s), input 1: %v\n", testNum, name, err)
			return
		}
		err = vm.Execute()
		if err != nil {
			t.Errorf("TestP2CValidTests: test %d (%s), input 1 failed to execute: %v\n", testNum, name, err)
			return
		}
	})
}

// TestP2CValidTests ensures all of the tests in p2c_invalid.json fail
// as expected.
func TestP2CInvalidTests(t *testing.T) {
	testHelper(t, "p2c_invalid.json", func(t *testing.T, test []string, name string, testNum int) {
		scriptSig, scriptPubKey, err := prepareP2CTest(t, test, name, testNum)
		if err != nil {
			t.Errorf("Could not prepare P2C invalid test %d (%s): %v\n", testNum, name, err)
			return
		}

		tx := createSpendingTx(scriptSig, scriptPubKey)

		vm, err := newReusableTestEngine(tx)
		if err != nil {
			t.Errorf("TestP2CInvalidTests: test %d (%s) failed to create engine: %v\n", testNum, name, err)
			return
		}

		err = vm.Prepare(scriptPubKey, 0)
		if err != nil {
			t.Errorf("TestP2CInvalidTests: Could not prepare engine for test %d (%s), input 0: %v\n", testNum, name, err)
			return
		}
		err = vm.Execute()
		if err != nil {
			// Got an expected failure
			return
		}

		err = vm.Prepare(scriptPubKey, 1)
		if err != nil {
			t.Errorf("TestP2CInvalidTests: Could not prepare engine for test %d (%s), input 1: %v\n", testNum, name, err)
			return
		}
		err = vm.Execute()
		if err != nil {
			// Got an expected failure
			return
		}

		t.Errorf("TestP2CInvalidTests: test %d (%s) succeeded but was supposed to fail\n", testNum, name)
	})
}

func testHelper(t *testing.T, filename string, cb func(*testing.T, []string, string, int)) {
	file, err := ioutil.ReadFile("data/" + filename)
	if err != nil {
		t.Errorf("Could not read %s: %v\n", filename, err)
		return
	}

	var tests [][]string
	err = json.Unmarshal(file, &tests)
	if err != nil {
		t.Errorf("Could not unmarshal from %s: %v\n", filename, err)
		return
	}

	testNum := 1
	for _, test := range tests {
		// Skip comments
		if len(test) == 1 {
			continue
		}
		name, err := testName(test, testNum)
		if err != nil {
			t.Errorf("Could not get name of test %d: %v\n", testNum, err)
			continue
		}
		cb(t, test, name, testNum)
		testNum++
	}
}

func prepareP2CTest(t *testing.T, test []string, name string, testNum int) ([]byte, []byte, error) {
	contractScript, err := ParseScriptString(test[0])
	if err != nil {
		return nil, nil, err
	}
	scriptSig, err := ParseScriptString(test[1])
	if err != nil {
		return nil, nil, err
	}
	pkParamsBytes, err := ParseScriptString(test[2])
	if err != nil {
		return nil, nil, err
	}

	scriptSig = AddDataToScript(scriptSig, contractScript)

	pkParamsPops, err := TstParseScript(pkParamsBytes)
	if err != nil {
		return nil, nil, err
	}

	pkParams := make([]Item, 0, len(pkParamsPops))
	for _, pkParamsPop := range pkParamsPops {
		if !TstIsPushdataOp(pkParamsPop) {
			return nil, nil, ErrStackNonPushOnly
		}
		pkParams = append(pkParams, DataItem(TstPopData(pkParamsPop)))
	}

	contractHash := sha3.Sum256(contractScript)
	pkScript, err := PayToContractHash(contractHash, pkParams, ScriptVersion1)
	if err != nil {
		return nil, nil, err
	}

	return scriptSig, pkScript, nil
}

func testCircFunc(ctx context.Context, assets []bc.AssetID) (map[bc.AssetID]int64, error) {
	return map[bc.AssetID]int64{bc.AssetID{1}: 5}, nil
}

func newReusableTestEngine(tx *bc.TxData) (*Engine, error) {
	result, err := NewReusableEngine(nil, testCircFunc, tx, 0)
	if err != nil {
		return nil, err
	}
	result.TstSetTimestamp(11)
	return result, nil
}

func newTestEngine(scriptPubKey []byte, tx *bc.TxData, flags ScriptFlags) (*Engine, error) {
	result, err := NewEngine(nil, testCircFunc, scriptPubKey, tx, 0, flags)
	if err != nil {
		return nil, err
	}
	result.TstSetTimestamp(11)
	return result, nil
}

func init() {
	for i := 0; i < 32; i++ {
		testAssetID[i] = byte(i + 1)
		testAssetID2[i] = byte(i * 2)
	}
}

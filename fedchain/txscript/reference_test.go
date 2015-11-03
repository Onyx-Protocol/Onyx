// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package txscript_test

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/net/context"

	"chain/fedchain/bc"
	"chain/fedchain/state"
	. "chain/fedchain/txscript"
)

// testName returns a descriptive test name for the given reference test data.
func testName(test []string) (string, error) {
	var name string

	if len(test) < 3 || len(test) > 4 {
		return name, fmt.Errorf("invalid test length %d", len(test))
	}

	if len(test) == 4 {
		name = fmt.Sprintf("test (%s)", test[3])
	} else {
		name = fmt.Sprintf("test ([%s, %s, %s])", test[0], test[1],
			test[2])
	}
	return name, nil
}

// parse hex string into a []byte.
func parseHex(tok string) ([]byte, error) {
	if !strings.HasPrefix(tok, "0x") {
		return nil, errors.New("not a hex number")
	}
	return hex.DecodeString(tok[2:])
}

// shortFormOps holds a map of opcode names to values for use in short form
// parsing.  It is declared here so it only needs to be created once.
var shortFormOps map[string]byte

// parseShortForm parses a string as as used in the Bitcoin Core reference tests
// into the script it came from.
//
// The format used for these tests is pretty simple if ad-hoc:
//   - Opcodes other than the push opcodes and unknown are present as
//     either OP_NAME or just NAME
//   - Plain numbers are made into push operations
//   - Numbers beginning with 0x are inserted into the []byte as-is (so
//     0x14 is OP_DATA_20)
//   - Single quoted strings are pushed as data
//   - Anything else is an error
func parseShortForm(script string) ([]byte, error) {
	// Only create the short form opcode map once.
	if shortFormOps == nil {
		ops := make(map[string]byte)
		for opcodeName, opcodeValue := range OpcodeByName {
			if strings.Contains(opcodeName, "OP_UNKNOWN") {
				continue
			}
			ops[opcodeName] = opcodeValue

			// The opcodes named OP_# can't have the OP_ prefix
			// stripped or they would conflict with the plain
			// numbers.  Also, since OP_FALSE and OP_TRUE are
			// aliases for the OP_0, and OP_1, respectively, they
			// have the same value, so detect those by name and
			// allow them.
			if (opcodeName == "OP_FALSE" || opcodeName == "OP_TRUE") ||
				(opcodeValue != OP_0 && (opcodeValue < OP_1 ||
					opcodeValue > OP_16)) {

				ops[strings.TrimPrefix(opcodeName, "OP_")] = opcodeValue
			}
		}
		shortFormOps = ops
	}

	// Split only does one separator so convert all \n and tab into  space.
	script = strings.Replace(script, "\n", " ", -1)
	script = strings.Replace(script, "\t", " ", -1)
	tokens := strings.Split(script, " ")
	builder := NewScriptBuilder()

	for _, tok := range tokens {
		if len(tok) == 0 {
			continue
		}
		// if parses as a plain number
		if num, err := strconv.ParseInt(tok, 10, 64); err == nil {
			builder.AddInt64(num)
			continue
		} else if bts, err := parseHex(tok); err == nil {
			builder.TstConcatRawScript(bts)
		} else if len(tok) >= 2 &&
			tok[0] == '\'' && tok[len(tok)-1] == '\'' {
			builder.AddFullData([]byte(tok[1 : len(tok)-1]))
		} else if opcode, ok := shortFormOps[tok]; ok {
			builder.AddOp(opcode)
		} else {
			return nil, fmt.Errorf("bad token \"%s\"", tok)
		}

	}
	return builder.Script()
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
		case "CLEANSTACK":
			flags |= ScriptVerifyCleanStack
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
		case "P2SH":
			flags |= ScriptBip16
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

func newCoinbaseTx(val uint64, pkScript []byte, assetID bc.AssetID) *bc.Tx {
	if pkScript == nil {
		pkScript = []byte{OP_TRUE}
	}
	return &bc.Tx{
		Version: bc.CurrentTransactionVersion,
		Inputs:  []*bc.TxInput{{SignatureScript: []byte{OP_0, OP_0}}},
		Outputs: []*bc.TxOutput{{Value: val, Script: pkScript, AssetID: assetID}},
	}
}

// createSpendTx generates a basic spending transaction given the passed
// signature and public key scripts.
func createSpendingTx(sigScript, pkScript []byte) (*bc.Tx, *testViewReader) {
	coinbaseTx1 := newCoinbaseTx(3, pkScript, testAssetID)
	coinbaseTx2 := newCoinbaseTx(4, pkScript, testAssetID)
	coinbaseTx3 := newCoinbaseTx(5, nil, testAssetID2)

	spendingTx := &bc.Tx{
		Version: bc.CurrentTransactionVersion,
		Inputs: []*bc.TxInput{
			{
				Previous:        bc.Outpoint{Hash: coinbaseTx1.Hash(), Index: 0},
				SignatureScript: sigScript,
				AssetID:         testAssetID,
			},
			{
				Previous:        bc.Outpoint{Hash: coinbaseTx2.Hash(), Index: 0},
				SignatureScript: sigScript,
				AssetID:         testAssetID,
			},
			{
				Previous: bc.Outpoint{Hash: coinbaseTx3.Hash(), Index: 0},
				AssetID:  testAssetID2,
			},
		},
		Outputs: []*bc.TxOutput{
			{
				Value:   7,
				AssetID: testAssetID,
				Script:  pkScript,
			},
			{
				Value:   5,
				AssetID: testAssetID2,
			},
		},
	}

	return spendingTx, &testViewReader{spendingTx: spendingTx, coinbaseTxs: []*bc.Tx{coinbaseTx1, coinbaseTx2, coinbaseTx3}}
}

// TestScriptInvalidTests ensures all of the tests in script_invalid.json fail
// as expected.
func TestScriptInvalidTests(t *testing.T) {
	testHelper(t, "script_invalid.json", func(t *testing.T, test []string, name string, testNum int) {
		scriptSig, err := parseShortForm(test[0])
		if err != nil {
			t.Errorf("%s: can't parse scriptSig; %v", name, err)
			return
		}
		scriptPubKey, err := parseShortForm(test[1])
		if err != nil {
			t.Errorf("%s: can't parse scriptPubkey; %v", name, err)
			return
		}
		flags, err := parseScriptFlags(test[2])
		if err != nil {
			t.Errorf("%s: %v", name, err)
			return
		}
		tx, viewReader := createSpendingTx(scriptSig, scriptPubKey)
		vm, err := newTestEngine(*viewReader, scriptPubKey, tx, flags)
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
		scriptSig, err := parseShortForm(test[0])
		if err != nil {
			t.Errorf("%s: can't parse scriptSig; %v", name, err)
			return
		}
		scriptPubKey, err := parseShortForm(test[1])
		if err != nil {
			t.Errorf("%s: can't parse scriptPubkey; %v", name, err)
			return
		}
		flags, err := parseScriptFlags(test[2])
		if err != nil {
			t.Errorf("%s: %v", name, err)
			return
		}
		tx, viewReader := createSpendingTx(scriptSig, scriptPubKey)
		vm, err := newTestEngine(*viewReader, scriptPubKey, tx, flags)
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

const P2CFLAGS = ScriptBip16 | ScriptVerifyStrictEncoding

// TestP2CValidTests ensures all of the tests in p2c_valid.json pass
// as expected.
func TestP2CValidTests(t *testing.T) {
	testHelper(t, "p2c_valid.json", func(t *testing.T, test []string, name string, testNum int) {
		scriptSig, scriptPubKey, err := prepareP2CTest(t, test, name, testNum)
		if err != nil {
			t.Errorf("Could not prepare P2C valid test %d (%s): %v\n", testNum, name, err)
			return
		}

		tx, viewReader := createSpendingTx(scriptSig, scriptPubKey)

		vm, err := newReusableTestEngine(*viewReader, tx)
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

		tx, viewReader := createSpendingTx(scriptSig, scriptPubKey)

		vm, err := newReusableTestEngine(*viewReader, tx)
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
		name, err := testName(test)
		if err != nil {
			t.Errorf("Could not get name of test %d: %v\n", testNum, err)
			continue
		}
		cb(t, test, name, testNum)
		testNum++
	}
}

func prepareP2CTest(t *testing.T, test []string, name string, testNum int) ([]byte, []byte, error) {
	contractScript, err := parseShortForm(test[0])
	if err != nil {
		return nil, nil, err
	}
	scriptSig, err := parseShortForm(test[1])
	if err != nil {
		return nil, nil, err
	}
	scriptPubKey, err := parseShortForm(test[2])
	if err != nil {
		return nil, nil, err
	}

	scriptSig = AddDataToScript(scriptSig, contractScript)

	parsedScriptPubKey, err := TstParseScript(scriptPubKey)
	if err != nil {
		return nil, nil, err
	}
	numParams := len(parsedScriptPubKey)

	if numParams == 0 {
		scriptPubKey = append(scriptPubKey, byte(0))
	} else {
		scriptPubKey = append(scriptPubKey, byte((OP_1 + numParams - 1)))
	}
	scriptPubKey = append(scriptPubKey, OP_ROLL)
	scriptPubKey = append(scriptPubKey, OP_DUP)
	scriptPubKey = append(scriptPubKey, OP_HASH160)
	scriptPubKey = AddDataToScript(scriptPubKey, Hash160(contractScript))
	scriptPubKey = append(scriptPubKey, OP_EQUALVERIFY)

	return scriptSig, scriptPubKey, nil
}

type testViewReader struct {
	spendingTx  *bc.Tx
	coinbaseTxs []*bc.Tx
}

func (viewReader testViewReader) Output(ctx context.Context, outpoint bc.Outpoint) *state.Output {
	if outpoint.Hash == viewReader.spendingTx.Hash() {
		return state.NewOutput(*viewReader.spendingTx.Outputs[outpoint.Index], outpoint, false)
	}
	for _, coinbaseTx := range viewReader.coinbaseTxs {
		if outpoint.Hash == coinbaseTx.Hash() {
			return state.NewOutput(*coinbaseTx.Outputs[outpoint.Index], outpoint, true)
		}
	}
	return nil
}

func (viewReader testViewReader) UnspentP2COutputs(ctx context.Context, contractHash bc.ContractHash, assetID bc.AssetID) []*state.Output {
	result := make([]*state.Output, 0, len(viewReader.spendingTx.Outputs))
	txhash := viewReader.spendingTx.Hash()
	for i, output := range viewReader.spendingTx.Outputs {
		if output.AssetID == assetID {
			isPayToContract, outputContractHash := TestPayToContract(output.Script)
			if isPayToContract && *outputContractHash == contractHash {
				result = append(result, state.NewOutput(*output, *bc.NewOutpoint(txhash[:], uint32(i)), false))
			}
		}
	}
	return result
}

func newReusableTestEngine(viewReader testViewReader, tx *bc.Tx) (*Engine, error) {
	result, err := NewReusableEngine(nil, viewReader, tx, P2CFLAGS)
	if err != nil {
		return nil, err
	}
	result.TstSetTimestamp(11)
	return result, nil
}

func newTestEngine(viewReader testViewReader, scriptPubKey []byte, tx *bc.Tx, flags ScriptFlags) (*Engine, error) {
	result, err := NewEngine(nil, viewReader, scriptPubKey, tx, 0, flags)
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

// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package txscript_test

import (
	"bytes"
	"reflect"
	"testing"

	. "chain/cos/txscript"
	"chain/crypto/hash256"
)

// TestPushedData ensured the PushedData function extracts the expected data out
// of various scripts.
func TestPushedData(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		script string
		out    [][]byte
		valid  bool
	}{
		{
			"0 IF 0 ELSE 2 ENDIF",
			[][]byte{nil, nil, []byte{0x02}},
			true,
		},
		{
			"16777216 10000000",
			[][]byte{
				{0x00, 0x00, 0x00, 0x01}, // 16777216
				{0x80, 0x96, 0x98, 0x00}, // 10000000
			},
			true,
		},
		{
			"DUP HASH160 '17VZNX1SN5NtKa8UQFxwQbFeFc3iqRYhem' EQUALVERIFY CHECKSIG",
			[][]byte{
				// 17VZNX1SN5NtKa8UQFxwQbFeFc3iqRYhem
				{
					0x31, 0x37, 0x56, 0x5a, 0x4e, 0x58, 0x31, 0x53, 0x4e, 0x35,
					0x4e, 0x74, 0x4b, 0x61, 0x38, 0x55, 0x51, 0x46, 0x78, 0x77,
					0x51, 0x62, 0x46, 0x65, 0x46, 0x63, 0x33, 0x69, 0x71, 0x52,
					0x59, 0x68, 0x65, 0x6d,
				},
			},
			true,
		},
		{
			"PUSHDATA4 1000 EQUAL",
			nil,
			false,
		},
	}

	for i, test := range tests {
		script := mustParseScriptString(test.script)
		data, err := PushedData(script)
		if test.valid && err != nil {
			t.Errorf("TestPushedData failed test #%d: %v\n", i, err)
			continue
		} else if !test.valid && err == nil {
			t.Errorf("TestPushedData failed test #%d: test should "+
				"be invalid\n", i)
			continue
		}
		if !reflect.DeepEqual(data, test.out) {
			t.Errorf("TestPushedData failed test #%d: want: %x "+
				"got: %x\n", i, test.out, data)
		}
	}
}

// TestHasCanonicalPush ensures the canonicalPush function works as expected.
func TestHasCanonicalPush(t *testing.T) {
	t.Parallel()

	for i := 0; i < 65535; i++ {
		builder := NewScriptBuilder()
		builder.AddInt64(int64(i))
		script, err := builder.Script()
		if err != nil {
			t.Errorf("Script: test #%d unexpected error: %v\n", i,
				err)
			continue
		}
		if result := IsPushOnlyScript(script); !result {
			t.Errorf("IsPushOnlyScript: test #%d failed: %x\n", i,
				script)
			continue
		}
		pops, err := TstParseScript(script)
		if err != nil {
			t.Errorf("TstParseScript: #%d failed: %v", i, err)
			continue
		}
		for _, pop := range pops {
			if result := TstHasCanonicalPushes(pop); !result {
				t.Errorf("TstHasCanonicalPushes: test #%d "+
					"failed: %x\n", i, script)
				break
			}
		}
	}
	for i := 0; i <= MaxScriptElementSize; i++ {
		builder := NewScriptBuilder()
		builder.AddData(bytes.Repeat([]byte{0x49}, i))
		script, err := builder.Script()
		if err != nil {
			t.Errorf("StandardPushesTests test #%d unexpected error: %v\n", i, err)
			continue
		}
		if result := IsPushOnlyScript(script); !result {
			t.Errorf("StandardPushesTests IsPushOnlyScript test #%d failed: %x\n", i, script)
			continue
		}
		pops, err := TstParseScript(script)
		if err != nil {
			t.Errorf("StandardPushesTests #%d failed to TstParseScript: %v", i, err)
			continue
		}
		for _, pop := range pops {
			if result := TstHasCanonicalPushes(pop); !result {
				t.Errorf("StandardPushesTests TstHasCanonicalPushes test #%d failed: %x\n", i, script)
				break
			}
		}
	}
}

// TestRemoveOpcodes ensures that removing opcodes from scripts behaves as
// expected.
func TestRemoveOpcodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		before string
		remove byte
		err    error
		after  string
	}{
		{
			// Nothing to remove.
			name:   "nothing to remove",
			before: "NOP",
			remove: OP_CODESEPARATOR,
			after:  "NOP",
		},
		{
			// Test basic opcode removal.
			name:   "codeseparator 1",
			before: "NOP CODESEPARATOR TRUE",
			remove: OP_CODESEPARATOR,
			after:  "NOP TRUE",
		},
		{
			// The opcode in question is actually part of the data
			// in a previous opcode.
			name:   "codeseparator by coincidence",
			before: "NOP DATA_1 CODESEPARATOR TRUE",
			remove: OP_CODESEPARATOR,
			after:  "NOP DATA_1 CODESEPARATOR TRUE",
		},
		{
			name:   "invalid opcode",
			before: "CAT",
			remove: OP_CODESEPARATOR,
			after:  "CAT",
		},
		{
			name:   "invalid length (insruction)",
			before: "PUSHDATA1",
			remove: OP_CODESEPARATOR,
			err:    ErrStackShortScript,
		},
		{
			name:   "invalid length (data)",
			before: "PUSHDATA1 0xff 0xfe",
			remove: OP_CODESEPARATOR,
			err:    ErrStackShortScript,
		},
	}

	for _, test := range tests {
		before := mustParseScriptString(test.before)
		after := mustParseScriptString(test.after)
		result, err := TstRemoveOpcode(before, test.remove)
		if test.err != nil {
			if err != test.err {
				t.Errorf("%s: got unexpected error. exp: \"%v\" "+
					"got: \"%v\"", test.name, test.err, err)
			}
			return
		}
		if err != nil {
			t.Errorf("%s: unexpected failure: \"%v\"", test.name, err)
			return
		}
		if !bytes.Equal(after, result) {
			t.Errorf("%s: value does not equal expected: exp: \"%v\""+
				" got: \"%v\"", test.name, after, result)
		}
	}
}

// TestRemoveOpcodeByData ensures that removing data carrying opcodes based on
// the data they contain works as expected.
func TestRemoveOpcodeByData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		before []byte
		remove []byte
		err    error
		after  []byte
	}{
		{
			name:   "nothing to do",
			before: []byte{OP_NOP},
			remove: []byte{1, 2, 3, 4},
			after:  []byte{OP_NOP},
		},
		{
			name:   "simple case",
			before: []byte{OP_DATA_4, 1, 2, 3, 4},
			remove: []byte{1, 2, 3, 4},
			after:  nil,
		},
		{
			name:   "simple case (miss)",
			before: []byte{OP_DATA_4, 1, 2, 3, 4},
			remove: []byte{1, 2, 3, 5},
			after:  []byte{OP_DATA_4, 1, 2, 3, 4},
		},
		{
			// padded to keep it canonical.
			name: "simple case (pushdata1)",
			before: append(append([]byte{OP_PUSHDATA1, 76},
				bytes.Repeat([]byte{0}, 72)...),
				[]byte{1, 2, 3, 4}...),
			remove: []byte{1, 2, 3, 4},
			after:  nil,
		},
		{
			name: "simple case (pushdata1 miss)",
			before: append(append([]byte{OP_PUSHDATA1, 76},
				bytes.Repeat([]byte{0}, 72)...),
				[]byte{1, 2, 3, 4}...),
			remove: []byte{1, 2, 3, 5},
			after: append(append([]byte{OP_PUSHDATA1, 76},
				bytes.Repeat([]byte{0}, 72)...),
				[]byte{1, 2, 3, 4}...),
		},
		{
			name:   "simple case (pushdata1 miss noncanonical)",
			before: []byte{OP_PUSHDATA1, 4, 1, 2, 3, 4},
			remove: []byte{1, 2, 3, 4},
			after:  []byte{OP_PUSHDATA1, 4, 1, 2, 3, 4},
		},
		{
			name: "simple case (pushdata2)",
			before: append(append([]byte{OP_PUSHDATA2, 0, 1},
				bytes.Repeat([]byte{0}, 252)...),
				[]byte{1, 2, 3, 4}...),
			remove: []byte{1, 2, 3, 4},
			after:  nil,
		},
		{
			name: "simple case (pushdata2 miss)",
			before: append(append([]byte{OP_PUSHDATA2, 0, 1},
				bytes.Repeat([]byte{0}, 252)...),
				[]byte{1, 2, 3, 4}...),
			remove: []byte{1, 2, 3, 4, 5},
			after: append(append([]byte{OP_PUSHDATA2, 0, 1},
				bytes.Repeat([]byte{0}, 252)...),
				[]byte{1, 2, 3, 4}...),
		},
		{
			name:   "simple case (pushdata2 miss noncanonical)",
			before: []byte{OP_PUSHDATA2, 4, 0, 1, 2, 3, 4},
			remove: []byte{1, 2, 3, 4},
			after:  []byte{OP_PUSHDATA2, 4, 0, 1, 2, 3, 4},
		},
		{
			// This is padded to make the push canonical.
			name: "simple case (pushdata4)",
			before: append(append([]byte{OP_PUSHDATA4, 0, 0, 1, 0},
				bytes.Repeat([]byte{0}, 65532)...),
				[]byte{1, 2, 3, 4}...),
			remove: []byte{1, 2, 3, 4},
			after:  nil,
		},
		{
			name:   "simple case (pushdata4 miss noncanonical)",
			before: []byte{OP_PUSHDATA4, 4, 0, 0, 0, 1, 2, 3, 4},
			remove: []byte{1, 2, 3, 4},
			after:  []byte{OP_PUSHDATA4, 4, 0, 0, 0, 1, 2, 3, 4},
		},
		{
			// This is padded to make the push canonical.
			name: "simple case (pushdata4 miss)",
			before: append(append([]byte{OP_PUSHDATA4, 0, 0, 1, 0},
				bytes.Repeat([]byte{0}, 65532)...), []byte{1, 2, 3, 4}...),
			remove: []byte{1, 2, 3, 4, 5},
			after: append(append([]byte{OP_PUSHDATA4, 0, 0, 1, 0},
				bytes.Repeat([]byte{0}, 65532)...), []byte{1, 2, 3, 4}...),
		},
		{
			name:   "invalid opcode ",
			before: []byte{OP_UNKNOWN187},
			remove: []byte{1, 2, 3, 4},
			after:  []byte{OP_UNKNOWN187},
		},
		{
			name:   "invalid length (instruction)",
			before: []byte{OP_PUSHDATA1},
			remove: []byte{1, 2, 3, 4},
			err:    ErrStackShortScript,
		},
		{
			name:   "invalid length (data)",
			before: []byte{OP_PUSHDATA1, 255, 254},
			remove: []byte{1, 2, 3, 4},
			err:    ErrStackShortScript,
		},
	}

	for _, test := range tests {
		result, err := TstRemoveOpcodeByData(test.before,
			test.remove)
		if test.err != nil {
			if err != test.err {
				t.Errorf("%s: got unexpected error. exp: \"%v\" "+
					"got: \"%v\"", test.name, test.err, err)
			}
			return
		}
		if err != nil {
			t.Errorf("%s: unexpected failure: \"%v\"", test.name, err)
			return
		}
		if !bytes.Equal(test.after, result) {
			t.Errorf("%s: value does not equal expected: exp: \"%v\""+
				" got: \"%v\"", test.name, test.after, result)
		}
	}
}

// TestIsPayToScriptHash ensures the IsPayToScriptHash function returns the
// expected results for all the scripts in scriptClassTests.
func TestIsPayToScriptHash(t *testing.T) {
	t.Parallel()

	for _, test := range scriptClassTests {
		script := mustParseScriptString(test.script)
		shouldBe := (test.class == ScriptHashTy)
		p2sh := IsPayToScriptHash(script)
		if p2sh != shouldBe {
			t.Errorf("%s: epxected p2sh %v, got %v", test.name,
				shouldBe, p2sh)
		}
	}
}

// TestHasCanonicalPushes ensures the canonicalPush function properly determines
// what is considered a canonical push for the purposes of removeOpcodeByData.
func TestHasCanonicalPushes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		script   string
		expected bool
	}{
		{
			name: "does not parse",
			script: "0x046708afdb0fe5548271967f1a67130b7105cd6a82" +
				"8e03909a67962e0ea1f61d",
			expected: false,
		},
		{
			name:     "non-canonical push",
			script:   "PUSHDATA1 0x04 0x01020304",
			expected: false,
		},
	}

	for i, test := range tests {
		script := mustParseScriptString(test.script)
		pops, err := TstParseScript(script)
		if err != nil {
			if test.expected {
				t.Errorf("TstParseScript #%d failed: %v", i, err)
			}
			continue
		}
		for _, pop := range pops {
			if TstHasCanonicalPushes(pop) != test.expected {
				t.Errorf("TstHasCanonicalPushes: #%d (%s) "+
					"wrong result\ngot: %v\nwant: %v", i,
					test.name, true, test.expected)
				break
			}
		}
	}
}

// TestIsPushOnlyScript ensures the IsPushOnlyScript function returns the
// expected results.
func TestIsPushOnlyScript(t *testing.T) {
	t.Parallel()

	test := struct {
		name     string
		script   []byte
		expected bool
	}{
		name: "does not parse",
		script: mustParseScriptString("0x046708afdb0fe5548271967f1a67130" +
			"b7105cd6a828e03909a67962e0ea1f61d"),
		expected: false,
	}

	if IsPushOnlyScript(test.script) != test.expected {
		t.Errorf("IsPushOnlyScript (%s) wrong result\ngot: %v\nwant: "+
			"%v", test.name, true, test.expected)
	}
}

// TestIsUnspendable ensures the IsUnspendable function returns the expected
// results.
func TestIsUnspendable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pkScript []byte
		expected bool
	}{
		{
			// Unspendable
			pkScript: []byte{0x6a, 0x04, 0x74, 0x65, 0x73, 0x74},
			expected: true,
		},
		{
			// Spendable
			pkScript: []byte{0x76, 0xa9, 0x14, 0x29, 0x95, 0xa0,
				0xfe, 0x68, 0x43, 0xfa, 0x9b, 0x95, 0x45,
				0x97, 0xf0, 0xdc, 0xa7, 0xa4, 0x4d, 0xf6,
				0xfa, 0x0b, 0x5c, 0x88, 0xac},
			expected: false,
		},
	}

	for i, test := range tests {
		res := IsUnspendable(test.pkScript)
		if res != test.expected {
			t.Errorf("TestIsUnspendable #%d failed: got %v want %v",
				i, res, test.expected)
			continue
		}
	}
}

func TestPayToContract(t *testing.T) {
	t.Parallel()

	contract := mustParseScriptString("'abc' OP_DROP")
	params := []Item{
		DataItem(mustParseScriptString("1")),
		DataItem(mustParseScriptString("2")),
		DataItem(mustParseScriptString("3")),
	}

	script, err := PayToContractInline(contract, params, ScriptVersion1)
	if err != nil {
		t.Fatal(err)
	}

	// N.b. no OP_NOP separating params from contract script
	expected := []byte{OP_1, OP_DROP, OP_DATA_1, OP_3, OP_DATA_1, OP_2, OP_DATA_1, OP_1, OP_DATA_3, 'a', 'b', 'c', OP_DROP}
	if !bytes.Equal(script, expected) {
		t.Errorf("expected %v, got %v", expected, script)
	}

	scriptVersion, _, _, _ := ParseP2C(script, []byte{0x1})
	if scriptVersion != nil {
		t.Errorf("expected misleading contracthint to make ParseP2C fail, but it didn't")
	}

	scriptVersion, parsedContract, _, parsedParams := ParseP2C(script, contract)
	if !bytes.Equal(scriptVersion, ScriptVersion1) {
		t.Errorf("expected script version %v, got %v", ScriptVersion1, scriptVersion)
	}
	if !bytes.Equal(parsedContract, contract) {
		t.Errorf("expected contract %v, got %v", contract, parsedContract)
	}
	if len(parsedParams) != len(params) {
		t.Errorf("expected %d parsed params, got %d", len(params), len(parsedParams))
	}
	for i, param := range params {
		if !bytes.Equal(param.(DataItem), parsedParams[i]) {
			t.Errorf("expected param %d to be %v, got %v", i, param, parsedParams[i])
		}
	}

	contractHash := hash256.Sum(contract)
	script, err = PayToContractHash(contractHash, params, ScriptVersion1)
	if err != nil {
		t.Fatal(err)
	}

	expected = []byte{OP_1, OP_DROP, OP_DATA_1, OP_3, OP_DATA_1, OP_2, OP_DATA_1, OP_1, OP_3, OP_ROLL, OP_DUP, OP_HASH256, OP_DATA_32}
	expected = append(expected, contractHash[:]...)
	expected = append(expected, []byte{OP_EQUALVERIFY, OP_EVAL}...)
	if !bytes.Equal(script, expected) {
		t.Errorf("expected %v, got %v", expected, script)
	}

	for _, contractHint := range [][]byte{nil, contract} {
		var desc string
		if contractHint == nil {
			desc = "without contracthint"
		} else {
			desc = "with contracthint"
		}

		scriptVersion, _, parsedContractHash, parsedParams := ParseP2C(script, contractHint)
		if !bytes.Equal(scriptVersion, ScriptVersion1) {
			t.Errorf("[%s] expected script version %v, got %v", desc, ScriptVersion1, scriptVersion)
		}
		if parsedContractHash != contractHash {
			t.Errorf("[%s] expected contract hash %v, got %v", desc, contractHash, parsedContractHash)
		}
		if len(parsedParams) != len(params) {
			t.Errorf("[%s] expected %d parsed params, got %d", desc, len(params), len(parsedParams))
		}
		for i, param := range params {
			if !bytes.Equal(param.(DataItem), parsedParams[i]) {
				t.Errorf("[%s] expected param %d to be %v, got %v", desc, i, param, parsedParams[i])
			}
		}
	}
}

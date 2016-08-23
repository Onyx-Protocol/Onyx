package vmutil

import (
	"bytes"
	"testing"

	"golang.org/x/crypto/sha3"

	"chain/cos/vm"
	"chain/crypto/ed25519"
)

// TestIsPushOnlyScript ensures the IsPushOnlyScript function returns the
// expected results.
func TestIsPushOnly(t *testing.T) {
	cases := []struct {
		prog     string
		expected bool
	}{
		{
			prog:     "",
			expected: true,
		}, {
			prog:     "1",
			expected: true,
		}, {
			prog:     "0xfadedbed",
			expected: true,
		}, {
			prog:     "1 0xfadedbed TRUE FALSE",
			expected: true,
		}, {
			prog:     "1 0xfadedbed TRUE NOP FALSE",
			expected: false,
		},
	}

	for i, c := range cases {
		compiled, err := vm.Compile(c.prog)
		if err != nil {
			t.Fatal(err)
		}
		pops, err := vm.ParseProgram(compiled)
		if err != nil {
			t.Fatal(err)
		}
		got := isPushOnly(pops)
		if got != c.expected {
			t.Errorf("case %d (%s): expected %v, got %v", i, c.prog, c.expected, got)
		}
	}
}

// TestIsUnspendable ensures the IsUnspendable function returns the expected
// results.
func TestIsUnspendable(t *testing.T) {
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
	contract, err := vm.Compile("'abc' DROP")
	if err != nil {
		t.Fatal(err)
	}
	params := [][]byte{
		vm.Int64Bytes(1),
		vm.Int64Bytes(2),
		vm.Int64Bytes(3),
	}

	contractHash := sha3.Sum256(contract)
	script := PayToContractHash(contractHash, params)

	expected := []byte{byte(vm.OP_DATA_1), 3, byte(vm.OP_DATA_1), 2, byte(vm.OP_DATA_1), 1, byte(vm.OP_3), byte(vm.OP_ROLL), byte(vm.OP_DUP), byte(vm.OP_SHA3), byte(vm.OP_DATA_32)}
	expected = append(expected, contractHash[:]...)
	expected = append(expected, []byte{byte(vm.OP_EQUALVERIFY), byte(vm.OP_0), byte(vm.OP_CHECKPREDICATE)}...)
	if !bytes.Equal(script, expected) {
		t.Errorf("expected %v, got %v", expected, script)
	}
}

func Test00Multisig(t *testing.T) {
	prog, err := doMultiSigScript(nil, 0, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(prog) < 1 {
		t.Fatal("doMultiSigScript(0, 0) = {} want script")
	}
}

func Test01Multisig(t *testing.T) {
	pubkeys := []ed25519.PublicKey{{}}
	_, err := doMultiSigScript(pubkeys, 0, true)
	if err == nil {
		t.Fatal("doMultiSigScript(1, 0) = success want error")
	}
}

package vmutil

import (
	"bytes"
	"testing"

	"chain/crypto/ed25519"
)

// TestIsUnspendable ensures the IsUnspendable function returns the expected
// results.
func TestIsUnspendable(t *testing.T) {
	tests := []struct {
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

func Test00Multisig(t *testing.T) {
	prog, err := BlockMultiSigProgram(nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(prog) < 1 {
		t.Fatal("BlockMultiSigScript(0, 0) = {} want script")
	}
}

func Test01Multisig(t *testing.T) {
	pubkeys := []ed25519.PublicKey{{}}
	_, err := BlockMultiSigProgram(pubkeys, 0)
	if err == nil {
		t.Fatal("BlockMultiSigScript(1, 0) = success want error")
	}
}

func TestParse00Multisig(t *testing.T) {
	prog, err := BlockMultiSigProgram(nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	keys, quorum, err := ParseBlockMultiSigProgram(prog)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 0 || quorum != 0 {
		t.Fatalf("ParseBlockMultiSigScript(%x) = (%v, %d) want (nil, 0)", prog, keys, quorum)
	}
}

func TestP2SP(t *testing.T) {
	pub1, _, _ := ed25519.GenerateKey(nil)
	pub2, _, _ := ed25519.GenerateKey(nil)
	prog, _ := P2SPMultiSigProgram([]ed25519.PublicKey{pub1, pub2}, 1)
	pubs, n, err := ParseP2SPMultiSigProgram(prog)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("expected nrequired=1, got %d", n)
	}
	if !bytes.Equal(pubs[0], pub1) {
		t.Errorf("expected first pubkey to be %x, got %x", pub1, pubs[0])
	}
	if !bytes.Equal(pubs[1], pub2) {
		t.Errorf("expected second pubkey to be %x, got %x", pub2, pubs[1])
	}
}

func TestBlockMultisig(t *testing.T) {
	pub1, _, _ := ed25519.GenerateKey(nil)
	pub2, _, _ := ed25519.GenerateKey(nil)
	prog, _ := BlockMultiSigProgram([]ed25519.PublicKey{pub1, pub2}, 1)
	pubs, n, err := ParseBlockMultiSigProgram(prog)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("expected nrequired=1, got %d", n)
	}
	if !bytes.Equal(pubs[0], pub1) {
		t.Errorf("expected first pubkey to be %x, got %x", pub1, pubs[0])
	}
	if !bytes.Equal(pubs[1], pub2) {
		t.Errorf("expected second pubkey to be %x, got %x", pub2, pubs[1])
	}
}

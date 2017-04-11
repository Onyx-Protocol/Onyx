package chainkd

import (
	"bytes"
	"crypto"
	"testing"

	"chain/crypto/ed25519"
)

// Testing basic InnerSign+Verify and the invariants:
// 1) Expand(PrivateKey).Sign() == PrivateKey.Sign()
// 2) InnerSign(Expand(PrivateKey)) == Sign(PrivateKey)

type zeroReader struct{}

func (zeroReader) Read(buf []byte) (int, error) {
	for i := range buf {
		buf[i] = 0
	}
	return len(buf), nil
}

func TestInnerSignVerify(t *testing.T) {
	var zero zeroReader
	public, private, _ := ed25519.GenerateKey(zero)
	expprivate := ExpandEd25519PrivateKey(private)

	message := []byte("test message")
	sig := Ed25519InnerSign(expprivate, message)
	if !ed25519.Verify(public, message, sig) {
		t.Errorf("valid signature rejected")
	}

	wrongMessage := []byte("wrong message")
	if ed25519.Verify(public, wrongMessage, sig) {
		t.Errorf("signature of different message accepted")
	}
}

func TestExpandedKeySignerInterfaceInvariant(t *testing.T) {
	var zero zeroReader
	public, private, _ := ed25519.GenerateKey(zero)
	expprivate := ExpandEd25519PrivateKey(private)

	signer1 := crypto.Signer(private)
	signer2 := crypto.Signer(expprivate)

	publicInterface1 := signer1.Public()
	publicInterface2 := signer2.Public()
	public1, ok := publicInterface1.(ed25519.PublicKey)
	if !ok {
		t.Fatalf("expected PublicKey from Public() but got %T", publicInterface1)
	}
	public2, ok := publicInterface2.(ed25519.PublicKey)
	if !ok {
		t.Fatalf("expected PublicKey from Public() but got %T", publicInterface2)
	}

	if !bytes.Equal(public, public1) {
		t.Errorf("public keys do not match: original:%x vs Public():%x", public, public1)
	}
	if !bytes.Equal(public, public2) {
		t.Errorf("public keys do not match: original:%x vs Public():%x", public, public2)
	}

	message := []byte("message")
	var noHash crypto.Hash
	signature1, err := signer1.Sign(zero, message, noHash)
	if err != nil {
		t.Fatalf("error from Sign(): %s", err)
	}
	signature2, err := signer2.Sign(zero, message, noHash)
	if err != nil {
		t.Fatalf("error from Sign(): %s", err)
	}
	if !bytes.Equal(signature1[:], signature2[:]) {
		t.Errorf(".Sign() should return identical signatures for Signer(privkey) and Signer(Expand(privkey))")
	}
	if !ed25519.Verify(public, message, signature1) {
		t.Errorf("Verify failed on signature from Sign()")
	}
}

func TestInnerSignInvariant(t *testing.T) {
	var zero zeroReader
	_, private, _ := ed25519.GenerateKey(zero)
	expprivate := ExpandEd25519PrivateKey(private)

	message := []byte("test message")
	sig1 := ed25519.Sign(private, message)
	sig2 := Ed25519InnerSign(expprivate, message)

	if !bytes.Equal(sig1[:], sig2[:]) {
		t.Errorf("InnerSign(Expand(privkey)) must return the same as Sign(privkey)")
	}
}

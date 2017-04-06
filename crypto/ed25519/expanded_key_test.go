package ed25519

import (
	"bytes"
	"crypto"
	"testing"
)

// Testing basic InnerSign+Verify and the invariants:
// 1) PrivateKey.Expanded().Sign() == PrivateKey.Sign()
// 2) InnerSign(PrivateKey.Expanded()) == Sign(PrivateKey)

func TestInnerSignVerify(t *testing.T) {
	var zero zeroReader
	public, private, _ := GenerateKey(zero)
	expprivate := private.Expanded()

	message := []byte("test message")
	sig := InnerSign(expprivate, message)
	if !Verify(public, message, sig) {
		t.Errorf("valid signature rejected")
	}

	wrongMessage := []byte("wrong message")
	if Verify(public, wrongMessage, sig) {
		t.Errorf("signature of different message accepted")
	}
}

func TestExpandedKeySignerInterfaceInvariant(t *testing.T) {
	var zero zeroReader
	public, private, _ := GenerateKey(zero)
	expprivate := private.Expanded()

	signer1 := crypto.Signer(private)
	signer2 := crypto.Signer(expprivate)

	publicInterface1 := signer1.Public()
	publicInterface2 := signer2.Public()
	public1, ok := publicInterface1.(PublicKey)
	if !ok {
		t.Fatalf("expected PublicKey from Public() but got %T", publicInterface1)
	}
	public2, ok := publicInterface2.(PublicKey)
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
		t.Errorf(".Sign() should return identical signatures for Signer(privkey) and Signer(privkey.Expanded())")
	}
	if !Verify(public, message, signature1) {
		t.Errorf("Verify failed on signature from Sign()")
	}
}

func TestInnerSignInvariant(t *testing.T) {
	var zero zeroReader
	_, private, _ := GenerateKey(zero)
	expprivate := private.Expanded()

	message := []byte("test message")
	sig1 := Sign(private, message)
	sig2 := InnerSign(expprivate, message)

	if !bytes.Equal(sig1[:], sig2[:]) {
		t.Errorf("InnerSign(privkey.Expanded()) must return the same as Sign(privkey)")
	}
}

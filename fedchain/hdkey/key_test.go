package hdkey

import "testing"

func TestNew(t *testing.T) {
	pub, priv, err := New()
	if err != nil {
		t.Fatal(err)
	}

	validPub, err := priv.Neuter()
	if err != nil {
		t.Fatal(err)
	}

	if validPub.String() != pub.String() {
		t.Fatal("incorrect private/public key pair")
	}
}

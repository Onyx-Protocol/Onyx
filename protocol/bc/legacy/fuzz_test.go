package legacy

import "testing"

func TestFuzzUnknownAssetVersion(t *testing.T) {
	const rawTx = `07010700f785c1f1b72b0001f1b72b0001012b00089def834ab929327f3f479177e2d8c293f2f7fc4f251db8547896c0eeafb984261a73767178584c246400b50150935a092ffad7ec9fbac4f4486db6c3b8cd5b9f51cf697248584dde286a722000012b766baa20627e83fdad13dd98436fa7cbdd1412d50ef65528edb7e2ed8f2675b2a0b209235151ad696c00c0030040b984261ad6e71876ec4c2464012b766baa209d44ee5b6ebf6c408772ead7713f1a66b9de7655ff452513487be1fb10de7d985151ad696c00c02a7b2274657374223a225175657279546573742e7465737442616c616e636551756572792e74657374227d`

	var want Tx
	err := want.UnmarshalText([]byte(rawTx))
	if err != nil {
		t.Fatal(err)
	}

	b, err := want.MarshalText()
	if err != nil {
		t.Fatal(err)
	}

	// Make sure serialzing and deserializing gives the same tx
	var got Tx
	err = got.UnmarshalText(b)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID.String() != want.ID.String() {
		t.Errorf("tx id changed to %s", got.ID.String())
	}
}
